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
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/playerq"
	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
	intersect "github.com/juliangruber/go-intersect"
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
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz
	redisURL := "redis://" + os.Getenv("REDIS_SENTINEL_SERVICE_HOST") + ":" + os.Getenv("REDIS_SENTINEL_SERVICE_PORT")

	//Single redis connection
	fmt.Println("Connecting to Redis at", redisURL)
	redisConn, err := redis.DialURL(redisURL)
	check(err, "QUIT")
	defer redisConn.Close()
	defer func() {
		// decrement the number of running MMFs since this one is finished
		fmt.Println("DECR concurrentMMFs")
		_, err = redisConn.Do("DECR", "concurrentMMFs")
		if err != nil {
			fmt.Println(err)
		}
	}()

	// PROFILE is passed via the k8s downward API through an env set to jobName.
	jobName := os.Getenv("PROFILE")
	timestamp := os.Getenv("OM_TIMESTAMP")
	moID := os.Getenv("OM_MATCHOBJECT_ID")
	profileKey := os.Getenv("OM_PROFILE_ID")

	fmt.Println("MMF request inserted at ", timestamp)
	fmt.Println("Looking for profile in key", profileKey)
	fmt.Println("Placing results in MatchObjectID", moID)

	profile, err := redis.String(redisConn.Do("GET", profileKey))
	if err != nil {
		panic(err)
	}
	fmt.Println("Got profile!")
	fmt.Println(profile)

	// Redis key under which to store results
	resultsKey := "proposal." + jobName
	rosterKey := "roster." + jobName

	// select players
	const numPlayers = 8
	// ZRANGE is 0-indexed
	defaultPool := gjson.Get(profile, "properties.playerPool")
	fmt.Println("defaultPool")
	fmt.Printf("defaultPool.String() = %+v\n", defaultPool.String())

	filters := make(map[string]map[string]int)
	defaultPool.ForEach(func(key, value gjson.Result) bool {
		// Make a map entry for this filter.
		searchKey := key.String()
		filters[searchKey] = map[string]int{"min": 0, "max": 9999999999}

		// Parse the min and max values.  JSON format is "min-max"
		r := strings.Split(value.String(), "-")
		filters[searchKey]["min"], err = strconv.Atoi(r[0])
		if err != nil {
			log.Println(err)
		}
		filters[searchKey]["max"], err = strconv.Atoi(r[1])
		if err != nil {
			log.Println(err)
		}

		return true // keep iterating
	})
	pretty.PrettyPrint(filters)

	if len(filters) < 1 {
		fmt.Printf("No filters in the default pool for the profile, %v\n", len(filters))
		fmt.Println("SET", moID+"."+profileKey, `{"error": "insufficient_filters"}`)
		redisConn.Do("SET", moID+"."+profileKey, `{"error": "insufficient_filters"}`)
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
	rosterProfile := gjson.Get(profile, "properties.roster")
	fmt.Printf("rosterProfile.String() = %+v\n", rosterProfile.String())

	// TODO: get this by parsing the JSON instead of cheating
	rosterSize := int(gjson.Get(profile, "properties.roster.blue").Int() + gjson.Get(profile, "properties.roster.red").Int())
	matchRoster := make([]string, 0)

	if len(overlap) < rosterSize {
		fmt.Printf("Not enough players in the pool to fill %v player slots in requested roster", rosterSize)
		fmt.Printf("rosterProfile.String() = %+v\n", rosterProfile.String())
		fmt.Println("SET", moID+"."+profileKey, `{"error": "insufficient_players"}`)
		redisConn.Do("SET", moID+"."+profileKey, `{"error": "insufficient_players"}`)
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

	profile, err = sjson.Set(profile, "properties.roster", teamRosters)
	if err != nil {
		panic(err)
	}

	// Write the match object that will be sent back to the DGS
	fmt.Println("Proposing the following group  ", resultsKey)
	fmt.Println("SET", resultsKey, profile)
	_, err = redisConn.Do("SET", resultsKey, profile)
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
