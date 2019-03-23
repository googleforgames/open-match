/*
 Note:
 This is a harness for rigging up custom matchmaking logic to a gRPC API
 endpoint, so that other services can make a simple gRPC request to kick off
 the logic.  The harness is optional, but tries to handle many of the
 boilerplate steps that matchmaking functions are required to execute (outlined
 in the <REPO_ROOT>/README.md file).  If you don't want to use the MMLogic API
 and would rather your MMF talk directly to Redis to do something else, feel
 free to not use this harness at all and instead write your own mmfRun()
 function with the specified function signature.  Errors are populated through
 the backend API back to the backend API client */
package apisrv

import (
	"context"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"

	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/spf13/viper"

	"go.opencensus.io/stats"
)

var (
	// Logrus structured logging setup
	mmfLogFields = log.Fields{
		"app":       "openmatch",
		"component": "function_mmlogic_harness",
	}
	mmfLog = log.WithFields(mmfLogFields)
)

// Step 2 - Talk to Redis.
// This example uses the MM Logic API in OM to read/write to/from redis.

// mmfRun is used to kick off the served MMF.
func mmfRun(ctx context.Context, fnArgs *api.Arguments, cfg *viper.Viper, mmlogic api.MmLogicClient) error {

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Get a cancel-able context
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Prepare to measure how long this takes
	var start time.Time
	runtime := time.Now()
	defer stats.Record(ctx, FnLatencySecs.M(time.Since(runtime).Seconds()))

	// Step 3 - Read the profile written to the Backend API
	profile, err := mmlogic.GetProfile(ctx, &api.MatchObject{Id: fnArgs.Request.ProfileId})
	if err != nil {
		mmfLog.Error(err)
		stats.Record(ctx, FnFailures.M(1))
		return err
	}
	if cfg.IsSet("debug") && cfg.GetBool("debug") {
		mmfLog.Debug("Profile: ", profile)
	}

	// Step 4 - Select the player data from Redis that we want for our matchmaking logic.
	playerPools := make([]*api.PlayerPool, len(profile.Pools))
	numPlayers := 0
	for index, emptyPool := range profile.Pools {
		poolLog := mmfLog.WithFields(log.Fields{"poolName": emptyPool.Name})
		playerPools[index] = proto.Clone(emptyPool).(*api.PlayerPool)
		playerPools[index].Roster = &api.Roster{Players: []*api.Player{}}
		poolLog.Info("Retrieving pool")
		// Taking out pool name for cardinality when procedurally generating
		// names. If you aren't procedurally generating names, having this
		// attached will let you get a lot of interesting metrics aggregated by
		// pool.
		//poolCtx, _ := tag.New(ctx, tag.Insert(KeyPoolName, "aggregate"))
		poolCtx := ctx

		// DEBUG: Print how long the filtering takes
		if cfg.IsSet("debug") && cfg.GetBool("debug") {
			start = time.Now()
		}

		// Pool filter results are streamed in chunks as they can be too large
		// to send in one grpc message.  Loop to get them all.
		stream, err := mmlogic.GetPlayerPool(ctx, emptyPool)
		if err != nil {
			poolLog.Error(err)
			stats.Record(ctx, FnFailures.M(1))
			return err
		}
		for {
			partialResults, err := stream.Recv()
			if err == io.EOF {
				// Break when all results are received
				break
			}
			if err != nil {
				poolLog.Error(err)
				stats.Record(ctx, FnFailures.M(1))
				return err
			}

			// Update stats with the latest results
			emptyPool.Stats = partialResults.Stats

			// Put players into the Pool's Roster with the attributes we
			// already retrieved through our filter queries..
			if partialResults.Roster != nil && len(partialResults.Roster.Players) > 0 {
				for _, player := range partialResults.Roster.Players {
					playerPools[index].Roster.Players = append(playerPools[index].Roster.Players, proto.Clone(player).(*api.Player))

					// Count players retrieved, used to short circuit logic if no players matched the filters.
					numPlayers++
				}
			}

			if cfg.IsSet("debug") && cfg.GetBool("debug") {
				poolLog.WithFields(log.Fields{
					"elapsed": time.Now().Sub(start).Seconds(),
					"count":   len(playerPools[index].Roster.Players),
				}).Debug("retrieval stats")
			}
		}
		if emptyPool.Stats != nil {
			// Emit metrics for MMFs to Open Census.
			stats.Record(poolCtx, FnPoolPlayersRetreivedTotal.M(emptyPool.Stats.Count))
			stats.Record(poolCtx, FnPoolElapsedSeconds.M(emptyPool.Stats.Elapsed))
			log.Infof(" Pool latency: %0.2v", emptyPool.Stats.Elapsed)
		}
	}

	// Generate a MatchObject message to write to state storage with the
	// results in it.  By default, assume we weren't successful (set ID to
	// ResultID, which skips the evaluator when outputting errors) until proven
	// otherwise.
	mo := &api.MatchObject{
		Id:         fnArgs.Request.ResultId,
		Properties: profile.Properties,
		Pools:      profile.Pools,
	}

	// Return error when there are no players in the pools
	if numPlayers == 0 {
		if cfg.IsSet("debug") && cfg.GetBool("debug") {
			mmfLog.Info("All player pools are empty, writing directly to ResultID to skip the evaluator")
		}
		// Writing to the error key skips the evaluator
		mo.Error = "insufficient players"
		stats.Record(ctx, FnFailures.M(1))
	} else {

		////////////////////////////////////////////////////////////////
		// Step 5 - Run custom matchmaking logic to try to find a match
		// This is in the file mmf.go.  Customize with your own match logic!
		results, rosters, err := makeMatches(ctx, profile.Properties, profile.Rosters, playerPools)
		////////////////////////////////////////////////////////////////

		if err != nil {
			mmfLog.WithFields(log.Fields{
				"error": err.Error(),
				"id":    fnArgs.Request.ResultId,
			}).Error("MMF returned an unrecoverable error")
			stats.Record(ctx, FnFailures.M(1))
			mo.Error = err.Error()
		} else {
			// No error!
			// Prepare the proposal match object.
			mo.Id = fnArgs.Request.ProposalId
			mo.Properties = results
			mo.Rosters = rosters
		}

	}

	// DEBUG
	if cfg.IsSet("debug") && cfg.GetBool("debug") {
		mmfLog.Debug("Output MatchObject:", mo)
	}

	// Step 6 - Write the outcome of the matchmaking logic back to state storage.
	// Step 7 - Remove the selected players from consideration by other MMFs.
	// CreateProposal does both of these for you, and some other tasks as well.
	success, err := mmlogic.CreateProposal(ctx, mo)
	if err != nil {
		mmfLog.Error(err)
		stats.Record(ctx, FnFailures.M(1))
	} else {
		mmfLog.WithFields(log.Fields{"id": fnArgs.Request.ProposalId, "success": success.Success}).Info("MMF write to state storage")
	}

	// Step 8 - Export stats about this run.
	stats.Record(ctx, FnRequests.M(1))
	return nil
}
