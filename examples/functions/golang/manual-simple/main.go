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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/set"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/ignorelist"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gomodule/redigo/redis"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

/*
Here are the things a MMF needs to do:

*Read/write from the Open Match state storage — Open Match ships with Redis as
the default state storage.
*Be packaged in a (Linux) Docker container.
*Read a profile you wrote to state storage using the Backend API.
*Select from the player data you wrote to state storage using the Frontend API.
*Run your custom logic to try to find a match.
*Write the match object it creates to state storage at a specified key.
*Remove the players it selected from consideration by other MMFs.
*Notify the MMForc of completion.
*(Optional & NYI, but recommended) Export stats for metrics collection.
*/
func main() {
	// Read config file.
	cfg, err := config.Read()
	if err != nil {
		panic(err)
	}

	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	redisConn := pool.Get()
	if err != nil {
		panic(err)
	}
	defer redisConn.Close()

	// decrement the number of running MMFs once finished
	defer func() {
		fmt.Println("DECR concurrentMMFs")
		_, err = redisConn.Do("DECR", "concurrentMMFs")
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Environment vars set by the MMForc
	jobName := os.Getenv("PROFILE")
	timestamp := os.Getenv("MMF_TIMESTAMP")
	proposalKey := os.Getenv("MMF_PROPOSAL_ID")
	profileKey := os.Getenv("MMF_PROFILE_ID")
	errorKey := os.Getenv("MMF_ERROR_ID")
	rosterKey := os.Getenv("MMF_ROSTER_ID")
	_ = jobName
	_ = timestamp
	_ = proposalKey
	_ = profileKey
	_ = errorKey
	_ = rosterKey

	fmt.Println("MMF request inserted at ", timestamp)
	fmt.Println("Looking for profile in key", profileKey)
	fmt.Println("Placing results in MatchObjectID", proposalKey)

	// Retrieve profile from Redis.
	// NOTE: This can also be done with a call to the MMLogic API.
	profile, err := redis.StringMap(redisConn.Do("HGETALL", profileKey))
	if err != nil {
		panic(err)
	}
	fmt.Println("=========Profile")
	p, err := json.MarshalIndent(profile, "", "  ")
	fmt.Println(string(p))

	// select players
	const numPlayers = 8
	// ZRANGE is 0-indexed
	pools := gjson.Get(profile["properties"], os.Getenv("JSONKEYS_POOLS"))
	fmt.Println("=========Pools")
	fmt.Printf("pool.String() = %+v\n", pools.String())

	// Parse all the pools.
	// NOTE: When using pool definitions like these that are using the
	//   PlayerPool protobuf message data schema, you can avoid all of this by
	//   using the MMLogic API call to automatically parse the pools, run the
	//   filters, and return the results in one gRPC call per pool.
	//
	// ex: poolRosters["defaultPool"]["mmr.rating"]=[]string{"abc", "def", "ghi"}
	poolRosters := make(map[string]map[string][]string)

	// Loop through each pool.
	pools.ForEach(func(_, pool gjson.Result) bool {
		pName := gjson.Get(pool.String(), "name").String()
		pFilters := gjson.Get(pool.String(), "filters")
		poolRosters[pName] = make(map[string][]string)

		// Loop through each filter for this pool
		pFilters.ForEach(func(_, filter gjson.Result) bool {
			// Note: This only works when running only one filter on each attribute!
			searchKey := gjson.Get(filter.String(), "attribute").String()
			min := int64(0)
			max := int64(time.Now().Unix())
			poolRosters[pName][searchKey] = make([]string, 0)

			// Parse the min and max values.
			if minv := gjson.Get(filter.String(), "minv"); minv.Bool() {
				min = int64(minv.Int())
			}
			if maxv := gjson.Get(filter.String(), "maxv"); maxv.Bool() {
				max = int64(maxv.Int())
			}
			fmt.Printf("%v: %v: [%v-%v]\n", pName, searchKey, min, max)

			// NOTE: This only pulls the first 50000 matches for a given index!
			//   This is an example, and probably shouldn't be used outside of
			//   testing without some performance tuning based on the size of
			//   your indexes. In prodution, this could be run concurrently on
			//   multiple parts of the index, and combined.
			// NOTE: It is recommended you also send back some stats about this
			//   query along with your MMF, which can be useful when your backend
			//   API client is deciding which profiles to send. This example does
			//   not return stats, but when using the MMLogic API, this is done
			//   for you.
			poolRosters[pName][searchKey], err = redis.Strings(
				redisConn.Do("ZRANGEBYSCORE", searchKey, min, max, "LIMIT", "0", "50000"))
			if err != nil {
				panic(err)
			}

			return true // keep iterating
		})

		return true // keep iterating
	})

	// Get ignored players.
	combinedIgnoreList := make([]string, 0)
	// Loop through all ignorelists configured in the config file.
	for il := range cfg.GetStringMap("ignoreLists") {
		ilCfg := config.Sub(cfg, fmt.Sprintf("ignoreLists.%v", il))
		thisIl, err := ignorelist.Retrieve(redisConn, ilCfg, il)
		if err != nil {
			panic(err)
		}

		// Join this ignorelist to the others we've retrieved
		combinedIgnoreList = set.Union(combinedIgnoreList, thisIl)
	}

	// Cycle through all filters for each pool, and calculate the overlap
	// (players that match all filters)
	overlaps := make(map[string][]string)
	// Loop through pools
	for pName, p := range poolRosters {
		fmt.Println(pName)

		// Var init
		overlaps[pName] = make([]string, 0)
		first := true // Flag used to initialize the overlap on the first iteration.

		// Loop through rosters that matched each filter
		for fName, roster := range p {
			if first {
				first = false
				overlaps[pName] = roster
			}
			// Calculate overlap
			overlaps[pName] = set.Intersection(overlaps[pName], roster)

			// Print out for visibility/debugging
			fmt.Printf("  filtering: %-20v | participants remaining: %-5v\n", fName, len(overlaps[pName]))
		}

		// Remove players on ignorelists
		overlaps[pName] = set.Difference(overlaps[pName], combinedIgnoreList)
		fmt.Printf("  removing: %-21v | participants remaining: %-5v\n", "(ignorelists)", len(overlaps[pName]))
	}

	// Loop through each roster in the profile and fill in players.
	rosters := gjson.Get(profile["properties"], os.Getenv("JSONKEYS_ROSTERS"))
	fmt.Println("=========Rosters")
	fmt.Printf("rosters.String() = %+v\n", rosters.String())

	// Parse all the rosters in the profile, adding players if we can.
	// NOTE: This is using roster definitions that follow the Roster protobuf
	// message data schema.
	profileRosters := make(map[string][]string)
	//proposedRosters := make([]string, 0)
	mo := &pb.MatchObject{}
	mo.Rosters = make([]*pb.Roster, 0)

	// List of all player IDs on all proposed rosters, used to add players to
	// the ignore list.
	// NOTE: when using the MMLogic API, writing your final proposal to state
	// storage will automatically add players to the ignorelist, so you don't
	// need to track them separately and add them to the ignore list yourself.
	playerList := make([]string, 0)

	rosters.ForEach(func(_, roster gjson.Result) bool {
		rName := gjson.Get(roster.String(), "name").String()
		fmt.Println(rName)
		rPlayers := gjson.Get(roster.String(), "players")
		profileRosters[rName] = make([]string, 0)
		pbRoster := pb.Roster{Name: rName, Players: []*pb.Player{}}

		rPlayers.ForEach(func(_, player gjson.Result) bool {
			// TODO: This is  where you would put your own custom matchmaking
			// logic.  MMFs have full access to the state storage in Redis, so
			// you can choose some participants from the pool according to your
			// favored strategy. You have complete freedom to read the
			// participant's records from Redis and make decisions accordingly.
			//
			// This example just chooses the players in the order they were
			// returned from state storage.

			//fmt.Printf("  %v\n", player.String()) //DEBUG
			proposedPlayer := player.String()
			// Get the name of the pool that the profile wanted this player pulled from.
			desiredPool := gjson.Get(player.String(), "pool").String()

			if _, ok := overlaps[desiredPool]; ok {
				// There are players that match all the desired filters.
				if len(overlaps[desiredPool]) > 0 {
					// Propose the next player returned from state storage for this
					// slot in the match rosters.

					// Functionally, a pop from the overlap array into the proposed slot.
					playerID := ""
					playerID, overlaps[desiredPool] = overlaps[desiredPool][0], overlaps[desiredPool][1:]

					proposedPlayer, err = sjson.Set(proposedPlayer, "id", playerID)
					if err != nil {
						panic(err)
					}
					profileRosters[rName] = append(profileRosters[rName], proposedPlayer)
					fmt.Printf("  proposing: %v\n", proposedPlayer)

					pbRoster.Players = append(pbRoster.Players, &pb.Player{Id: playerID, Pool: desiredPool})
					playerList = append(playerList, playerID)

				} else {
					// Not enough players, exit.
					fmt.Println("Not enough players in the pool to fill all player slots in requested roster", rName)
					fmt.Printf("%+v\n", roster.String())
					fmt.Println("HSET", errorKey, "error", "insufficient_players")
					redisConn.Do("HSET", errorKey, "error", "insufficient_players")
					os.Exit(0)
				}

			}
			return true
		})

		//proposedRoster, err := sjson.Set(roster.String(), "players", profileRosters[rName])
		mo.Rosters = append(mo.Rosters, &pbRoster)
		//fmt.Sprintf("[%v]", strings.Join(profileRosters[rName], ",")))
		//if err != nil {
		//	panic(err)
		//}
		//proposedRosters = append(proposedRosters, proposedRoster)

		return true
	})

	// Write back the match object to state storage so the evaluator can look at it, and update the ignorelist.
	// NOTE: the MMLogic API CreateProposal automates most of this for you, as
	//  long as you send it properly formatted data (i.e. data that fits the schema of
	//  the protobuf messages)
	// Add proposed players to the ignorelist so other MMFs won't consider them.
	if len(playerList) > 0 {
		fmt.Printf("Adding %d players to ignorelist\n", len(playerList))
		err = ignorelist.Add(redisConn, "proposed", playerList)
		if err != nil {
			fmt.Println("Unable to add proposed players to the ignorelist")
			panic(err)
		}
	}

	// Write the match object that will be sent back to the DGS
	jmarshaler := jsonpb.Marshaler{}
	moJSON, err := jmarshaler.MarshalToString(mo)
	proposedRosters := gjson.Get(moJSON, "rosters")

	fmt.Println("===========Proposal")
	// Set the properties field.
	// This is a filthy hack due to the way sjson escapes & quotes values it inserts.
	// Better in most cases than trying to marshal the JSON into giant multi-dimensional
	// interface maps only to dump it back out to a string after.
	// Note: this hack isn't necessary for most users, who just use this same
	// data directly from the protobuf message 'rosters' field, or write custom
	// rosters directly to the JSON properties when choosing players.  This is here
	// for backwards compatibility with backends that haven't been updated to take
	// advantage of the new rosters field in the MatchObject protobuf message introduced
	// in 0.2.0.
	profile["properties"], err = sjson.Set(profile["properties"], os.Getenv("JSONKEYS_ROSTERS"), proposedRosters.String())
	profile["properties"] = strings.Replace(profile["properties"], "\\", "", -1)
	profile["properties"] = strings.Replace(profile["properties"], "]\"", "]", -1)
	profile["properties"] = strings.Replace(profile["properties"], "\"[", "[", -1)
	if err != nil {
		fmt.Println("problem with sjson")
		fmt.Println(err)
	}
	fmt.Printf("Proposed ID: %v | Properties: %v", proposalKey, profile["properties"])

	// Write the roster that will be sent to the evaluator.  This needs to be written to the
	// "rosters" key of the match object, in the protobuf format for an array of
	// rosters protobuf messages.  You can write this output by hand (not recommended)
	// or use the MMLogic API call CreateProposal will a filled out MatchObject protobuf message
	// and let it do the work for you.
	profile["rosters"] = proposedRosters.String()

	fmt.Println("===========Redis")
	// Start writing proposed results to Redis.
	redisConn.Send("MULTI")
	for key, value := range profile {
		if key != "id" {
			fmt.Println("HSET", proposalKey, key, value)
			redisConn.Send("HSET", proposalKey, key, value)
		}
	}

	//Finally, write the propsal key to trigger the evaluation of these results
	fmt.Println("SADD", cfg.GetString("queues.proposals.name"), proposalKey)
	redisConn.Send("SADD", cfg.GetString("queues.proposals.name"), proposalKey)
	_, err = redisConn.Do("EXEC")
	if err != nil {
		panic(err)
	}

}
