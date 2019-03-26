// The evaluator is, generalized, a weighted graph problem
// https://www.google.co.jp/search?q=computer+science+weighted+graph&oq=computer+science+weighted+graph
// However, it's up to the developer to decide what values in their matchmaking
// decision process are the weights as well as what to prioritize (make as many
// groups as possible is a common goal).  The default evaluator makes naive
// decisions under the assumption that most use cases would rather spend their
// time tweaking the profiles sent to matchmaking such that they are less and less
// likely to choose the same players.

/*
Copyright 2018 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"flag"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/set"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/ignorelist"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"
	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	lgr    *log.Logger
	cfg    *viper.Viper
	pool   *redis.Pool
	dryrun bool
)

func main() {
	flag.BoolVar(&dryrun, "dryrun", false, "Print eval results, but do not take approval or rejection actions")
	flag.Parse()

	//Init Logger
	lgr = log.New()
	lgr.Info("Initializing example MMF proposal evaluator")

	// Read config
	// This golang example uses the Open Match config module.  If you are
	// writing an evaluator in another language, most libraries/modules that
	// can parse a YAML file will work.
	lgr.Println("Initializing config...")
	cfg, err := config.Read()
	if err != nil {
		panic(nil)
	}

	// Connect to redis
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz // redis pool docs: https://godoc.org/github.com/gomodule/redigo/redis#Pool
	pool, err := redishelpers.ConnectionPool(cfg)
	redisConn := pool.Get()
	defer redisConn.Close()

	// TODO: write some code to allow the context to be safely cancelled
	/*
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
	*/

	start := time.Now()

	//proposedMatchIds  []string,
	//overloadedPlayers map[string][]int
	//overloadedMatches map[int][]int
	//approvedMatches   map[int]bool
	proposedMatchIds, overloadedPlayers, overloadedMatches, approvedMatches, err := stub(cfg, pool)
	overloadedPlayerList, overloadedMatchList, approvedMatchList := generateLists(overloadedPlayers, overloadedMatches, approvedMatches)

	lgr.Info("overloadedPlayers")
	pretty.PrettyPrint(overloadedPlayers)
	lgr.Info("overloadePlayerList")
	pretty.PrettyPrint(overloadedPlayerList)
	lgr.Info("overloadedMatchList")
	pretty.PrettyPrint(overloadedMatchList)
	lgr.Info("approvedMatchList")
	pretty.PrettyPrint(approvedMatchList)

	// Choose matches to approve from among the overloaded ones.
	approved, rejected, err := chooseMatches(overloadedMatchList)
	approvedMatchList = append(approvedMatchList, approved...)
	approvedPlayers := []string{}
	potentiallyRejectedPlayers := []string{}

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
		if len(values) != 3 {
			err = errors.New("proposal key didn't have three parts")
			lgr.Error(err)
			return
		}
		moID, proID := values[1], values[2]
		backendID := moID + "." + proID

		// TODO: would be more efficient to gather this while looking for
		// overlaps.  This is just a test to make sure this works.
		a, err := getProposedPlayers(pool, proposedID)
		if err != nil {
			lgr.Error("Failed to get approved players ", err)
		}
		approvedPlayers = append(approvedPlayers, a...)

		// Acrtually approve the match
		lgr.Infof("approving proposal number %v, ID: %v -> %v\n", proposalIndex, proposedID, backendID)
		if !dryrun {
			lgr.Infof(" RENAME %v %v", proposedID, backendID)
			_, err = redisConn.Do("RENAME", proposedID, backendID)
			if err != nil {
				// RENAME only fails if the source key doesn't exist
				lgr.Error("Failure to move proposal to approved backendID! ", err)
			}
		}

	}

	// Dump out some info about rejected proposals
	for _, proposalIndex := range rejected {
		proposedID := proposedMatchIds[proposalIndex]

		// TODO: would be more efficient to gather this while looking for
		// overlaps.  This is just a test to make sure this works.
		pr, err := getProposedPlayers(pool, proposedID)
		if err != nil {
			lgr.Error("Failed to get rejected players ", err)
		}

		potentiallyRejectedPlayers = append(potentiallyRejectedPlayers, pr...)
		lgr.Infof("rejecting proposal number %v, ID: %v", proposalIndex, proposedID)
		if !dryrun {
			lgr.Infof(" DEL %v", proposedID)
			_, err = redisConn.Do("DEL", proposedID)
			if err != nil {
				lgr.Error("Failure to delete rejected proposal! ", err)
			}
		}
	}

	// Remove players that are (only) in the rejected games from the ignorelist
	// overloadedPlayers map[string][]int
	rejectedPlayers := set.Difference(potentiallyRejectedPlayers, approvedPlayers)
	lgr.Info("approvedPlayers")
	pretty.PrettyPrint(approvedPlayers)
	lgr.Info("potentiallyRejectedPlayers")
	pretty.PrettyPrint(potentiallyRejectedPlayers)
	lgr.Info("rejectedPlayers")
	pretty.PrettyPrint(rejectedPlayers)

	if len(rejectedPlayers) > 0 {
		lgr.Info("Removing rejectedPlayers from 'proposed' ignorelist")
		if !dryrun {
			err = ignorelist.Remove(redisConn, "proposed", rejectedPlayers)
			if err != nil {
				lgr.Error("Failed to remove rejected players from ignorelist", err)
			}
		}
	} else {
		lgr.Warning("No rejected players to re-queue!")
	}

	lgr.Printf("0 Finished in %v seconds.", time.Since(start).Seconds())
}

// chooseMatches looks through all match proposals that ard overloaded (that
// is, have a player that is also in another proposed match) and chooses those
// to approve and those to reject.
// TODO: this needs a complete overhaul in a 'real' graph search
func chooseMatches(overloaded []int) ([]int, []int, error) {
	// Super naive - take one overloaded match and approved it, reject all others.
	lgr.Infof("overloaded = %+v\n", overloaded)
	lgr.Infof("len(overloaded) = %+v\n", len(overloaded))
	if len(overloaded) > 0 {
		lgr.Infof("overloaded[0:0] = %+v\n", overloaded[0:0])
		lgr.Infof("overloaded[1:] = %+v\n", overloaded[1:])
		return overloaded[0:1], overloaded[1:], nil
	}
	return []int{}, overloaded, nil
}

// stub is the name of this function because it's just a functioning test, not
// a recommended approach! Be sure to use robust evaluation logic for
// production matchmakers!
func stub(cfg *viper.Viper, pool *redis.Pool) ([]string, map[string][]int, map[int][]int, map[int]bool, error) {
	//Init Logger
	lgr := log.New()
	lgr.Println("Initializing example MMF proposal evaluator")

	// Get redis conneciton
	redisConn := pool.Get()
	defer redisConn.Close()

	// Put some config vars into other vars for readability
	proposalq := cfg.GetString("queues.proposals.name")

	numProposals, err := redis.Int(redisConn.Do("SCARD", proposalq))
	lgr.Info("SCARD ", proposalq, " returned ", numProposals)
	cmd := "SPOP"
	if dryrun {
		cmd = "SRANDMEMBER"
	}
	lgr.Println(cmd, proposalq, numProposals)
	proposals, err := redis.Strings(redisConn.Do(cmd, proposalq, numProposals))
	if err != nil {
		lgr.Println(err)
	}
	lgr.Infof("proposals = %+v\n", proposals)

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
			lgr.Println(err)
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
				// Terrible indexing hack: default int value is 0, but so is the
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

// getProposedPlayers is a function that may be moved to an API call in the future.
func getProposedPlayers(pool *redis.Pool, propKey string) ([]string, error) {

	// Get the proposal match object from redis
	mo := &om_messages.MatchObject{Id: propKey}
	lgr.Info("Getting matchobject with ID ", propKey)
	err := redispb.UnmarshalFromRedis(context.Background(), pool, mo)
	if err != nil {
		return nil, err
	}

	// Loop through all rosters, appending players IDs to a list.
	playerList := []string{}
	for _, r := range mo.Rosters {
		for _, p := range r.Players {
			playerList = append(playerList, p.Id)
		}
	}

	return playerList, err
}

func generateLists(overloadedPlayers map[string][]int, overloadedMatches map[int][]int, approvedMatches map[int]bool) ([]string, []int, []int) {
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
	return overloadedPlayerList, overloadedMatchList, approvedMatchList
}
