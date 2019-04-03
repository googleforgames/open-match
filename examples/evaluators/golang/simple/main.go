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
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	pb "github.com/GoogleCloudPlatform/open-match/internal/pb"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"
	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
)

func main() {

	//Init Logger
	lgr := log.New(os.Stdout, "MMFEval: ", log.LstdFlags)
	lgr.Println("Initializing example MMF proposal evaluator")

	// Read config
	lgr.Println("Initializing config...")
	cfg, err := config.Read()
	if err != nil {
		panic(nil)
	}

	// Connect to redis
	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		lgr.Fatal(err)
	}
	defer pool.Close()

	redisConn := pool.Get()
	defer redisConn.Close()

	// TODO: write some code to allow the context to be safely cancelled
	/*
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
	*/

	start := time.Now()

	proposedMatchIds, overloadedPlayers, overloadedMatches, approvedMatches, err := stub(cfg, pool)
	overloadedPlayerList, overloadedMatchList, approvedMatchList := generateLists(overloadedPlayers, overloadedMatches, approvedMatches)

	fmt.Println("overloadedPlayers")
	pretty.PrettyPrint(overloadedPlayers)
	fmt.Println("overloadePlayerList")
	pretty.PrettyPrint(overloadedPlayerList)
	fmt.Println("overloadedMatchList")
	pretty.PrettyPrint(overloadedMatchList)
	fmt.Println("approvedMatchList")
	pretty.PrettyPrint(approvedMatchList)

	approved, rejected, err := chooseMatches(overloadedMatchList)
	approvedMatchList = append(approvedMatchList, approved...)

	// run redis commands to approve matches
	for _, proposalIndex := range approvedMatchList {
		// The match object was already written by the MMF, just change the
		// name to what the Backend API (apisrv.go) is looking for.
		proposedID := proposedMatchIds[proposalIndex]
		// Incoming proposal keys look like this:
		// proposal.1542600048.80e43fa085844eebbf53fc736150ef96.testprofile
		// format:
		// "proposal".timestamp.unique_matchobject_id.profile_name
		values := strings.Split(proposedID, ".")
		moID, proID := values[2], values[3]
		backendID := moID + "." + proID
		fmt.Printf("approving proposal #%+v:%+v\n", proposalIndex, moID)
		fmt.Println("RENAME", proposedID, backendID)
		_, err = redisConn.Do("RENAME", proposedID, backendID)
		if err != nil {
			// RENAME only fails if the source key doesn't exist
			fmt.Printf("err = %+v\n", err)
		}
	}

	//TODO: Need to requeue for another job run here.
	for _, proposalIndex := range rejected {
		fmt.Println("rejecting ", proposalIndex)
		proposedID := proposedMatchIds[proposalIndex]
		fmt.Printf("proposedID = %+v\n", proposedID)
		values := strings.Split(proposedID, ".")
		fmt.Printf("values = %+v\n", values)
		timestamp, moID, proID := values[0], values[1], values[2]
		fmt.Printf("timestamp = %+v\n", timestamp)
		fmt.Printf("moID = %+v\n", moID)
		fmt.Printf("proID = %+v\n", proID)
	}

	lgr.Printf("0 Finished in %v seconds.", time.Since(start).Seconds())

}

// chooseMatches looks through all match proposals that ard overloaded (that
// is, have a player that is also in another proposed match) and chooses those
// to approve and those to reject.
// TODO: this needs a complete overhaul in a 'real' graph search
func chooseMatches(overloaded []int) ([]int, []int, error) {
	// Super naive - take one overloaded match and approved it, reject all others.
	fmt.Printf("overloaded = %+v\n", overloaded)
	fmt.Printf("len(overloaded) = %+v\n", len(overloaded))
	if len(overloaded) > 0 {
		fmt.Printf("overloaded[0:2] = %+v\n", overloaded[0:0])
		fmt.Printf("overloaded[1:] = %+v\n", overloaded[1:])
		return overloaded[0:1], overloaded[1:], nil
	}
	return []int{}, overloaded, nil
}

func stub(cfg config.View, pool *redis.Pool) ([]string, map[string][]int, map[int][]int, map[int]bool, error) {
	//Init Logger
	lgr := log.New(os.Stdout, "MMFEvalStub: ", log.LstdFlags)
	lgr.Println("Initializing example MMF proposal evaluator")

	// Get redis conneciton
	redisConn := pool.Get()
	defer redisConn.Close()

	// Put some config vars into other vars for readability
	proposalq := cfg.GetString("queues.proposals.name")

	lgr.Println("SCARD", proposalq)
	numProposals, err := redis.Int(redisConn.Do("SCARD", proposalq))
	lgr.Println("SPOP", proposalq, numProposals)
	proposals, err := redis.Strings(redisConn.Do("SPOP", proposalq, numProposals))
	if err != nil {
		lgr.Println(err)
	}
	fmt.Printf("proposals = %+v\n", proposals)

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
	mo := &pb.MatchObject{Id: propKey}
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

func propToRoster(in []string) []interface{} {
	// Convert []string to []interface{} so it can be passed as variadic input
	// https://golang.org/doc/faq#convert_slice_of_interface
	out := make([]interface{}, len(in))
	for i, v := range in {
		values := strings.Split(v, ".")
		timestamp, moID, proID := values[0], values[1], values[2]

		out[i] = "roster." + timestamp + "." + moID + "." + proID
	}
	return out
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
