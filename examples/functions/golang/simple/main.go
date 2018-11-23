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
	"strings"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/playerq"
	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
	intersect "github.com/juliangruber/go-intersect"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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

	fmt.Println("MMF request inserted at ", timestamp)
	fmt.Println("Looking for profile in key", profileKey)
	fmt.Println("Placing results in MatchObjectID", proposalKey)

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

	filters := make(map[string]map[string]int64)
	pools.ForEach(func(_, pool gjson.Result) bool {
		// Loop through each pool.
		pfilters := gjson.Get(pool.String(), "filters")
		pfilters.ForEach(func(_, filter gjson.Result) bool {
			// Make a map entry for this filter.
			searchKey := gjson.Get(filter.String(), "attribute").String()
			filters[searchKey] = map[string]int64{"min": 0, "max": 9999999999}

			// Parse the min and max values.  JSON format is "min-max"
			min := gjson.Get(filter.String(), "minv")
			if min.Bool() {
				filters[searchKey]["min"] = int64(min.Int())
			}
			max := gjson.Get(filter.String(), "maxv")
			if max.Bool() {
				filters[searchKey]["max"] = int64(max.Int())
			}
			fmt.Printf("%v\n", searchKey)
			fmt.Printf("%v\n", filters[searchKey])

			return true // keep iterating
		})
		return true // keep iterating
	})
	os.Exit(0)

	if len(filters) < 1 {
		fmt.Printf("No filters in the default pool for the profile, %v\n", len(filters))
		fmt.Println("SET", errorKey, `{"error": "insufficient_filters"}`)
		redisConn.Do("SET", errorKey, `{"error": "insufficient_filters"}`)
		return
	}

	//init 2d array
	stuff := make([][]string, len(filters))
	for i := range stuff {
		stuff[i] = make([]string, 0)
	}

	i := 0
	for key, value := range filters {
		// TODO: this needs a lot of time and effort on building sane values for how many IDs to pull at once.
		// TODO: this should also be run concurrently per index we're filtering on
		fmt.Printf("key = %+v\n", key)
		fmt.Printf("value = %+v\n", value)
		results, err := redis.Strings(redisConn.Do("ZRANGEBYSCORE", key, value["min"], value["max"], "LIMIT", "0", "10000"))
		if err != nil {
			panic(err)
		}

		// Store off these results in the 2d array used to calculate intersections below.
		stuff[i] = append(stuff[i], results...)
		i++
	}

	fmt.Println("overlap")
	overlap := stuff[0]
	for i := range stuff {
		if i > 0 {
			set := fmt.Sprint(intersect.Hash(overlap, stuff[i]))
			overlap = strings.Split(set[1:len(set)-1], " ")
		}
	}
	pretty.PrettyPrint(overlap)

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
