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
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

/*
Here are the things a MMF needs to do:

*Read/write from the Open Match state storage — Open Match ships with Redis as the default state storage.
*Be packaged in a (Linux) Docker container.
*Read a profile you wrote to state storage using the Backend API.
*Select from the player data you wrote to state storage using the Frontend API.
*Run your custom logic to try to find a match.
*Write the match object it creates to state storage at a specified key.
*Remove the players it selected from consideration by other MMFs.
*(Optional, but recommended) Export stats for metrics collection.
*/
func main() {
	// Read config file.
	cfg := viper.New()
	cfg, err := config.Read()

	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz
	redisURL := "redis://" + os.Getenv("REDIS_SENTINEL_SERVICE_HOST") + ":" + os.Getenv("REDIS_SENTINEL_SERVICE_PORT")
	fmt.Println("Connecting to Redis at", redisURL)
	redisConn, err := redis.DialURL(redisURL)
	check(err, "QUIT")
	defer redisConn.Close()

	// decrement the number of running MMFs once finished
	defer func() {
		fmt.Println("DECR moncurrentMMFs")
		_, err = redisConn.Do("DECR", "concurrentMMFs")
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Environment vars set by the MMForc
	jobName := os.Getenv("PROFILE")
	timestamp := os.Getenv("OM_TIMESTAMP")
	proposalKey := os.Getenv("OM_PROPOSAL_ID")
	profileKey := os.Getenv("OM_PROFILE_ID")
	errorKey := os.Getenv("OM_ERROR_ID")
	rosterKey := os.Getenv("OM_ROSTER_ID")
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
	fmt.Println("Got profile!")
	p, err := json.MarshalIndent(profile, "", "  ")
	fmt.Println(string(p))

	// select players
	const numPlayers = 8
	// ZRANGE is 0-indexed
	pools := gjson.Get(profile["properties"], cfg.GetString("jsonkeys.pools"))
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
	// ex: poolDefinitions["defaultPool"]["mmr.rating"]["min"]=0
	//poolDefinitions := make(map[string]map[string]map[string]int64)

	// Loop through each pool.
	pools.ForEach(func(_, pool gjson.Result) bool {
		pName := gjson.Get(pool.String(), "name").String()
		pFilters := gjson.Get(pool.String(), "filters")
		//poolDefinitions[pName] = make(map[string]map[string]int64)
		poolRosters[pName] = make(map[string][]string)

		// Loop through each filter for this pool
		pFilters.ForEach(func(_, filter gjson.Result) bool {
			// Note: This only works when running only one filter on each attribute!
			searchKey := gjson.Get(filter.String(), "attribute").String()
			//poolDefinitions[pName][searchKey] = map[string]int64{"min": 0, "max": 9999999999}
			min := int64(0)
			max := int64(time.Now().Unix())
			poolRosters[pName][searchKey] = make([]string, 0)

			// Parse the min and max values.  JSON format is "min-max"
			if minv := gjson.Get(filter.String(), "minv"); minv.Bool() {
				//poolDefinitions[pName][searchKey]["min"] = int64(min.Int())
				min = int64(minv.Int())
			}
			if maxv := gjson.Get(filter.String(), "maxv"); maxv.Bool() {
				//poolDefinitions[pName][searchKey]["max"] = int64(max.Int())
				max = int64(maxv.Int())
			}
			//fmt.Printf("%v: %v: %v\n", pName, searchKey, poolDefinitions[pName][searchKey])
			fmt.Printf("%v: %v: [%v-%v]\n", pName, searchKey, min, max)

			// NOTE: This only pulls the first 10000 matches for a given index!
			//   This is an example, and probably shouldn't be used outside of testing without some
			//   performance tuning based on the size of your indexes.
			//   In prodution, this could be run concurrently on multiple parts of the index, and combined.
			// NOTE: It is recommended you also send back some stats about this
			//   query along with your MMF, which can be useful when your backend
			//   API client is deciding which profiles to send. This example does
			//   not implement this, but when using the MMLogic API, this is done
			//   for you.
			poolRosters[pName][searchKey], err = redis.Strings(redisConn.Do("ZRANGEBYSCORE", searchKey, min, max, "WITHSCORES", "LIMIT", "0", "10000"))
			if err != nil {
				panic(err)
			}

			return true // keep iterating
		})

		// Quit if we see a malformed pool.
		//		if len(poolDefinitions[pName]) < 1 {
		//			fmt.Printf("Error: pool %v with no filters in profile, %v\n", pName, profileKey)
		//			fmt.Println("SET", errorKey, `{"error": "insufficient_filters"}`)
		//			redisConn.Do("SET", errorKey, `{"error": "insufficient_filters"}`)
		//			os.Exit(1)
		//		}

		return true // keep iterating
	})

	for pName, p := range poolRosters {
		fmt.Println("overlap", pName)
		_ = p
		for fName, roster := range p {
			fmt.Println("filter name", fName)
			if len(roster) > 10 {
				fmt.Println(roster[:10])
			}
		}
		//overlap := stuff[0]
		//for i := range stuff {
		//	if i > 0 {
		//		set := fmt.Sprint(intersect.Hash(overlap, stuff[i]))
		//		overlap = strings.Split(set[1:len(set)-1], " ")
		//	}
		//}
		//pretty.PrettyPrint(overlap)
	}

	/*
		// TODO: rigourous logic to put players in desired groups
		teamRosters := make(map[string][]string)
		rosterProfile := gjson.Get(profile["properties"], "properties.roster")
		fmt.Printf("rosterProfile.String() = %+v\n", rosterProfile.String())

		// TODO: get this by parsing the JSON instead of cheating
		rosterSize := int(gjson.Get(profile["properties"], "properties.roster.blue").Int() + gjson.Get(profile["properties"], "properties.roster.red").Int())
		matchRoster := make([]string, 0)

		if len(overlap) < rosterSize {
			fmt.Printf("Not enough players in the pool to fill %v player slots in requested roster", rosterSize)
			fmt.Printf("rosterProfile.String() = %+v\n", rosterProfile.String())
			fmt.Println("SET", errorKey, `{"error": "insufficient_players"}`)
			redisConn.Do("SET", errorKey, `{"error": "insufficient_players"}`)
			return
		}
		rosterProfile.ForEach(func(name, size gjson.Result) bool {
			teamKey := name.String()
			teamRosters[teamKey] = make([]string, size.Int())
			for i := 0; i < int(size.Int()); i++ {
				var playerID string
				// Functionally a Pop from the overlap array into playerID
				playerID, overlap = overlap[0], overlap[1:]
				teamRosters[teamKey][i] = playerID
				matchRoster = append(matchRoster, playerID)
			}
			return true
		})

		pretty.PrettyPrint(teamRosters)
		pretty.PrettyPrint(matchRoster)

		profile["properties"], err = sjson.Set(profile["properties"], "properties.roster", teamRosters)
		if err != nil {
			panic(err)
		}

		// Write the match object that will be sent back to the DGS
		fmt.Println("Proposing the following group  ", proposalKey)
		fmt.Println("HSET", proposalKey, "properties", profile)
		_, err = redisConn.Do("HSET", proposalKey, "properties", profile)
		if err != nil {
			panic(err)
		}

		// Write the roster that will be sent to the evaluator
		fmt.Println("Sending the following roster to the evaluator under key ", rosterKey)
		fmt.Println("SET", rosterKey, strings.Join(matchRoster, " "))
		_, err = redisConn.Do("SET", rosterKey, strings.Join(matchRoster, " "))
		if err != nil {
			panic(err)
		}

		//TODO: make this auto-correcting if the player doesn't end up in a group.
		for _, playerID := range matchRoster {
			fmt.Printf("Attempting to remove player %v from indices\n", playerID)
			// TODO: make playerq module available to everything
			err := playerq.Deindex(redisConn, playerID)
			if err != nil {
				panic(err)
			}

		}

		//Finally, write the propsal key to trigger the evaluation of these
		//results
		// TODO: read this from a config ala proposalq := cfg.GetString("queues.proposals.name")
		proposalq := "proposalq"
		fmt.Println("SADD", proposalq, jobName)
		_, err = redisConn.Do("SADD", proposalq, jobName)
		if err != nil {
			panic(err)
		}
		// DEBUG
		results, err := redis.Strings(redisConn.Do("SMEMBERS", proposalq))
		if err != nil {
			panic(err)
		}
		pretty.PrettyPrint(results)
	*/
}

func check(err error, action string) {
	if err != nil {
		if action == "QUIT" {
			log.Fatal(err)
		} else {
			log.Print(err)
		}
	}
}
