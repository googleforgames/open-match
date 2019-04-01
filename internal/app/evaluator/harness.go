// The evaluator is, generalized, a weighted graph problem
// https://www.google.co.jp/search?q=computer+science+weighted+graph&oq=computer+science+weighted+graph
// However, it's up to the developer to decide what values in their matchmaking
// decision process are the weights as well as what to prioritize (make as many
// groups as possible is a common goal).  The default evaluator makes naive
// decisions under the assumption that most use cases would rather spend their
// time tweaking the profiles sent to matchmaking such that they are less and less
// likely to choose the same players.
// Note: the example only works with the code within the same release/branch.
/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package evaluator

import (
	"context"
	"flag"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/set"
	redisHelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/ignorelist"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"
	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
	"go.opencensus.io/tag"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	// Logrus structured logging setup
	evLogFields = log.Fields{
		"app":       "openmatch",
		"component": "evaluator",
	}
	evLog = log.WithFields(evLogFields)

	cfg    *viper.Viper
	pool   *redis.Pool
	dryrun bool
)

func initializeApplication() {

	// Parse commandline flags.
	flag.BoolVar(&dryrun, "dryrun", false, "Print eval results, but do not take approval or rejection actions")
	flag.Parse()

	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(EvaluatorLogLines, KeySeverity))

	// Viper config management initialization
	var err error
	cfg, err = config.Read()
	if err != nil {
		evLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Get redis connection pool.
	pool, err = redisHelpers.ConnectionPool(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocEvaluatorViews := DefaultEvaluatorViews
	ocEvaluatorViews = append(ocEvaluatorViews, config.CfgVarCountView) // config loader view.

	// Waiting on https://github.com/opencensus-integrations/redigo/pull/1
	// ocEvaluatorViews = append(ocEvaluatorViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	evLog.WithFields(log.Fields{"viewscount": len(ocEvaluatorViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocEvaluatorViews)

}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	initializeApplication()

	// TODO: implement robust cancellation logic.
	ctx, cancel := context.WithCancel(context.Background())
	_ = cancel

	// Get connection to redis
	redisConn := pool.Get()
	defer redisConn.Close()

	start := time.Now()
	checkProposals := true
	pollSleep := cfg.GetInt("evaluator.pollIntervalMs")
	interval := cfg.GetInt64("evaluator.intervalMs")

	// Logging for the proposal queue
	pqLog := evLog.WithFields(log.Fields{"proposalQueue": cfg.GetString("queues.proposals.name")})

	// main loop; run evaluator when proposals are in the proposals queue
	//
	// We are ready to evaluate if EITHER of these conditions are met (logical OR):
	//  - all MMFs are complete
	//  - timeout is reached
	// AND this condition is met:
	//  - at least one MMF has been started since the last we evaluated.
	//
	// Tuning how frequently the evaluator runs is a complex topic and
	// probably only of interest to users running large-scale production
	// workloads with many concurrently running matchmaking functions,
	// which have some overlap in the matchmaking player pools. Suffice to
	// say that under load, this switch should almost always trigger the
	// timeout interval code path.  The concurrentMMFs check to see how
	// many are still running is meant as a deadman's switch to prevent
	// waiting to run the evaluator when all your MMFs are already
	// finished.

	for {
		cycleStart := time.Now()

		// Get number of running MMFs
		// 'nil' return from redigo indicates that no new MMFs have even been started
		// since the last time the evaluator ran.  We don't want to evaluate in that case.
		r, err := redisHelpers.Retrieve(ctx, pool, "concurrentMMFs")
		if err != nil {
			if err.Error() == "redigo: nil returned" {
				// No MMFs have run since we last evaluated; reset timer and loop
				start = time.Now()
				time.Sleep(time.Duration(pollSleep) * time.Millisecond)

				if cfg.IsSet("debug") && cfg.GetBool("debug") == true {
					evLog.Debug("Number of concurrentMMFs is nil")
				}
			}
			continue
		}

		numRunning, err := strconv.Atoi(r)
		if err != nil {
			evLog.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Issue retrieving number of currently running MMFs")
		}

		switch {
		// dividing by 1e6 looks gnarly but is actually the canonical implementation
		// https://github.com/golang/go/issues/5491
		case time.Since(start).Nanoseconds()/1e6 >= interval:
			evLog.WithFields(log.Fields{
				"intervalMs": interval,
			}).Info("Maximum evaluator interval exceeded")
			checkProposals = true

			// Opencensus tagging
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalTrigger, "interval_exceeded"))

		case numRunning <= 0: // Can be less than zero in some less common cases.
			evLog.Info("All MMFs complete")
			checkProposals = true
			numRunning = 0

			// Opencensus tagging
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalTrigger, "mmfs_completed"))
		}

		if checkProposals {
			// Make sure there are proposals in the queue. No need to run the
			// evaluator if there are none.
			checkProposals = false
			pqLog.Info("checking statestorage for match object proposals")
			results, err := redisHelpers.Count(context.Background(), pool, cfg.GetString("queues.proposals.name"))
			switch {
			case err != nil:
				pqLog.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("couldn't retrieve the length of the proposal queue from statestorage")
			case results == 0:
				pqLog.Warn("no proposals in the queue to evaluate")
			default:
				pqLog.WithFields(log.Fields{
					"numProposals": results,
				}).Info("proposals available, evaluating")
				go evaluator(ctx)
			}

			// Delete the counter.  This causes subsequent queries to return
			// 'nil' until it is re-created again on first write.  We exploit
			// this to short-circuit and not ever run evaluation logic if no MMF
			// has written to the counter since last evaluation.
			err = redisHelpers.Delete(context.Background(), pool, "concurrentMMFs")
			if err != nil {
				evLog.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("error deleting concurrent MMF counter")
			}
			start = time.Now()
		}

		// A sleep here is not critical but just a useful safety valve in case
		// things are broken, to keep the main loop from going all-out and spamming the log.
		sleepyTime := 1000
		evLog.WithFields(log.Fields{"sleepms": sleepyTime, "elapsed": time.Since(cycleStart)}).Info("Evaluation complete. Sleeping...")
		time.Sleep(time.Duration(sleepyTime) * time.Millisecond)
	} // End main for loop
}

//////////////////////////////////
// TODO: Clean this up, it's just transplanted from a different part of the
// codebase until the evaluator gets a usability/customizability overhaul.
// TODO: This needs all the stats in the evaluator_stats.go implemented.
//////////////////////////////////

// evaluator evaluates proposals looking for conflicts: 'overloaded' players
// that are in more than one proposal. It then chooses which proposals to
// approve and which to reject.
// This is just an example; it is no where near efficient and rejects multiple
// matches that could be approved.  Don't go to production without a solid
// strategy for conflict resolution and a rigourous implementation!
func evaluator(ctx context.Context) {
	evLog.Info("Starting example MMF proposal evaluation")

	// TODO: write some code to allow the context to be safely cancelled
	//ctx, cancel := context.WithCancel(ctx)
	//defer cancel()

	// Get connection to redis
	redisConn := pool.Get()
	defer redisConn.Close()

	start := time.Now()

	// Formats of vars:
	//  proposedMatchIds  []string,
	//  overloadedPlayers map[string][]int
	//  overloadedMatches map[int][]int
	//  approvedMatches   map[int]bool
	proposedMatchIds, overloadedPlayers, overloadedMatches, approvedMatches, err := stub(cfg, pool)

	// Make a slice of overloaded players from the map.
	overloadedPlayerList := make([]string, 0, len(overloadedPlayers))
	for k := range overloadedPlayers {
		overloadedPlayerList = append(overloadedPlayerList, k)
	}

	// Make a slice of overloaded matches from the set.
	overloadedMatchList := make([]int, 0, len(overloadedMatches))
	for k := range overloadedMatches {
		overloadedMatchList = append(overloadedMatchList, k)
	}

	// Make a slice of approved matches from the set.
	approvedMatchList := make([]int, 0, len(approvedMatches))
	for k := range approvedMatches {
		approvedMatchList = append(approvedMatchList, k)
	}

	// Print everything out for debugging/troubleshooting
	if cfg.IsSet("debug") && cfg.GetBool("debug") == true {
		evLog.Info("overloadedPlayers")
		pretty.PrettyPrint(overloadedPlayers)
		evLog.Info("overloadePlayerList")
		pretty.PrettyPrint(overloadedPlayerList)
		evLog.Info("overloadedMatchList")
		pretty.PrettyPrint(overloadedMatchList)
		evLog.Info("approvedMatchList")
		pretty.PrettyPrint(approvedMatchList)
	}

	// Choose matches to approve from among the overloaded ones.
	approved, rejected, err := chooseMatches(overloadedMatchList)
	approvedMatchList = append(approvedMatchList, approved...)

	// Arrays used to look for rejected players
	approvedPlayers := make([]string, 0)
	potentiallyRejectedPlayers := make([]string, 0)

	// run redis commands to approve matches
	for _, proposalIndex := range approvedMatchList {
		// The match object was already written by the MMF, just change the
		// name to what the Backend API (apisrv.go) is looking for.
		proposedID := proposedMatchIds[proposalIndex]
		// Incoming proposal keys look like this:
		// proposal.80e43fa085844eebbf53fc736150ef96.testprofile
		// format:
		// "proposal".unique_matchobject_id.profile_name
		values := strings.Split(proposedID, ".")
		moID, proID := values[1], values[2]
		backendID := moID + "." + proID

		// TODO: would be more efficient to gather this while looking for
		// overlaps.  This is just a test to make sure this works.
		a, err := getProposedPlayers(pool, proposedID)
		if err != nil {
			evLog.Error("Failed to get approved players ", err)
		}
		approvedPlayers = append(approvedPlayers, a...)

		// Acrtually approve the match
		evLog.Infof("approving proposal number %v, ID: %v -> %v\n", proposalIndex, proposedID, backendID)
		if !dryrun {
			evLog.Debugf(" RENAME %v %v", proposedID, backendID)
			_, err = redisConn.Do("RENAME", proposedID, backendID)
			if err != nil {
				// RENAME only fails if the source key doesn't exist
				evLog.Error("Failure to move proposal to approved backendID! ", err)
			}
		}

	}

	// Dump out some info about rejected proposals
	//	rejectedMO := &pb.MatchObject{
	//		Error: "Proposed match rejected due to player conflict",
	//	}

	for _, proposalIndex := range rejected {
		proposedID := proposedMatchIds[proposalIndex]
		// Incoming proposal keys look like this:
		// proposal.80e43fa085844eebbf53fc736150ef96.testprofile
		// format:
		// "proposal".unique_matchobject_id.profile_name
		values := strings.Split(proposedID, ".")
		moID, proID := values[1], values[2]
		backendID := moID + "." + proID

		// TODO: would be more efficient to gather this while looking for
		// overlaps.  This is just a test to make sure this works.
		pr, err := getProposedPlayers(pool, proposedID)
		if err != nil {
			evLog.Error("Failed to get rejected players ", err)
		}

		potentiallyRejectedPlayers = append(potentiallyRejectedPlayers, pr...)
		evLog.Infof("rejecting proposal number %v, ID: %v", proposalIndex, proposedID)
		if !dryrun {
			// Set error message
			_, err = redisConn.Do("HSET", proposedID, "error", "Proposed match rejected due to player conflict")
			if err != nil {
				evLog.Error("Failure to set rejected error! ", err)
			}

			// Remove any rosters
			_, err = redisConn.Do("HSET", proposedID, "rosters", "")
			if err != nil {
				evLog.Error("Failure to delete rejected rosters! ", err)
			}

			// TODO: dry this with the approved flow
			evLog.Debug("RENAME ", proposedID)
			_, err = redisConn.Do("RENAME", proposedID, backendID)
			if err != nil {
				evLog.Error("Failure to delete rejected proposal! ", err)
			}
			// Example of an action you can take on rejection - notify the backend client
			// and tell them to query again.  You could also kick off another MMF run (but be sure
			// to have some condition after which you quit retrying, and to correctly increment/decrement
			// the concurrentMMF counter in state storage if you do this).
			//rejectedMO.Id = backendID
			//err = redispb.MarshalToRedis(ctx, pool, rejectedMO, cfg.GetInt("redis.expirations.matchobject"))
			//if err != nil {
			//	evLog.Error("Failure to notify backend of rejected proposal! ", err)
			//}

		}
	}

	// Remove players that are (only) in the rejected games from the ignorelist
	// overloadedPlayers map[string][]int
	rejectedPlayers := set.Difference(potentiallyRejectedPlayers, approvedPlayers)
	if cfg.IsSet("debug") && cfg.GetBool("debug") {
		evLog.Info("approvedPlayers")
		pretty.PrettyPrint(approvedPlayers)
		evLog.Info("potentiallyRejectedPlayers")
		pretty.PrettyPrint(potentiallyRejectedPlayers)
		evLog.Info("rejectedPlayers")
		pretty.PrettyPrint(rejectedPlayers)
	}

	if len(rejectedPlayers) > 0 {
		evLog.Info("Removing rejectedPlayers from 'proposed' ignorelist")
		if !dryrun {
			err = ignorelist.Remove(redisConn, "proposed", rejectedPlayers)
			if err != nil {
				evLog.Error("Failed to remove rejected players from ignorelist", err)
			}
		}
	} else {
		evLog.Warning("No rejected players to re-queue!")
	}

	evLog.Printf("---- Finished in %v seconds.", time.Since(start).Seconds())
}

// stub is the name of this function because it's just a functioning test, not
// a recommended approach! Be sure to use robust evaluation logic for
// production matchmakers!
func stub(cfg *viper.Viper, pool *redis.Pool) ([]string, map[string][]int, map[int][]int, map[int]bool, error) {
	//Init Logger
	evLog := log.New()
	evLog.Println("Initializing example MMF proposal evaluator")

	// Get redis conneciton
	redisConn := pool.Get()
	defer redisConn.Close()

	// Put some config vars into other vars for readability
	proposalq := cfg.GetString("queues.proposals.name")

	numProposals, err := redis.Int(redisConn.Do("SCARD", proposalq))
	evLog.Debug("SCARD ", proposalq, " returned ", numProposals)
	cmd := "SPOP"
	if dryrun {
		// SRANDMEMBER leaves the proposal in the queue (SPOP does not)
		cmd = "SRANDMEMBER"
	}
	evLog.Println(cmd, proposalq, numProposals)
	proposals, err := redis.Strings(redisConn.Do(cmd, proposalq, numProposals))
	if err != nil {
		evLog.Println(err)
	}
	evLog.Debug("proposals = ", proposals)

	// This is a far cry from effecient but we expect a pretty small set of players under consideration
	// at any given time
	// Map that implements a set https://golang.org/doc/effective_go.html#maps
	overloadedPlayers := make(map[string][]int)
	overloadedMatches := make(map[int][]int)
	approvedMatches := make(map[int]bool)
	allPlayers := make(map[string]int)

	// Loop through each proposal, and look for 'overloaded' players (players in multiple proposals)
	for index, propKey := range proposals {
		approvedMatches[index] = true // This proposal is approved until proven otherwise
		playerList, err := getProposedPlayers(pool, propKey)
		if err != nil {
			evLog.Println(err)
		}

		for _, pID := range playerList {
			if allPlayers[pID] != 0 {
				// Seen this player at least once before; gather the indicies of all the match
				// proposals with this player
				overloadedPlayers[pID] = append(overloadedPlayers[pID], index)
				overloadedMatches[index] = []int{}
				delete(approvedMatches, index)
				if len(overloadedPlayers[pID]) == 1 {
					adjustedIndex := allPlayers[pID] - 1
					overloadedPlayers[pID] = append(overloadedPlayers[pID], adjustedIndex)
					overloadedMatches[adjustedIndex] = []int{}
					delete(approvedMatches, adjustedIndex)
				}
			} else {
				// First time seeing this player.  Track which match proposal had them in case we see
				// them again
				// TODO: Fix this terrible indexing hack: default int value is 0, but so is the
				// lowest propsal index.  Since we need to use interpret 0 as
				// 'unset' in this context, add one to index, and remove it if/when we put this
				// player  in the overloadedPlayers map.
				adjustedIndex := index + 1
				allPlayers[pID] = adjustedIndex
			}
		}
	}
	return proposals, overloadedPlayers, overloadedMatches, approvedMatches, err
}

// chooseMatches looks through all match proposals that ard overloaded (that
// is, have a player that is also in another proposed match) and chooses those
// to approve and those to reject.
// TODO: this needs a complete overhaul in a 'real' graph search
func chooseMatches(overloaded []int) ([]int, []int, error) {
	// Super naive - take one overloaded match and approved it, reject all others.
	evLog.Debug("overloaded = ", overloaded)
	evLog.Debug("len(overloaded) = ", len(overloaded))
	if len(overloaded) > 0 {
		evLog.Debug("overloaded[0:0] = ", overloaded[0:0])
		evLog.Debug("overloaded[1:] = ", overloaded[1:])
		return overloaded[0:1], overloaded[1:], nil
	}
	return []int{}, overloaded, nil
}

// getProposedPlayers is a function that may be moved to an API call in the future.
func getProposedPlayers(pool *redis.Pool, propKey string) ([]string, error) {

	// Get the proposal match object from redis
	mo := &pb.MatchObject{Id: propKey}
	evLog.WithFields(log.Fields{"id": propKey}).Info("Getting matchobject with ID ")
	err := redispb.UnmarshalFromRedis(context.Background(), pool, mo)
	if err != nil {
		return nil, err
	}

	// Loop through all rosters, appending players IDs to a list.
	playerList := make([]string, 0)
	for _, r := range mo.Rosters {
		for _, p := range r.Players {
			playerList = append(playerList, p.Id)
		}
	}

	return playerList, err
}
