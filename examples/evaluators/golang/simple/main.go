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
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gobs/pretty"
	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
)

func main() {

	//Init Logger
	lgr := log.New(os.Stdout, "MMFEval: ", log.LstdFlags)
	lgr.Println("Initializing example MMF proposal evaluator")

	// Read config
	lgr.Println("Initializing config...")
	cfg, err := readConfig("matchmaker_config", map[string]interface{}{
		"REDIS_SENTINEL_SERVICE_HOST": "redis-sentinel",
		"REDIS_SENTINEL_SERVICE_PORT": "6379",
		"auth": map[string]string{
			// Read from k8s secret eventually
			// Probably doesn't need a map, just here for reference
			"password": "12fa",
		},
	})
	if err != nil {
		panic(nil)
	}

	// Connect to redis
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz // redis pool docs: https://godoc.org/github.com/gomodule/redigo/redis#Pool
	redisURL := "redis://" + cfg.GetString("REDIS_SENTINEL_SERVICE_HOST") + ":" + cfg.GetString("REDIS_SENTINEL_SERVICE_PORT")
	lgr.Println("Connecting to redis at", redisURL)
	pool := redis.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 60 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}
	redisConn := pool.Get()
	defer redisConn.Close()

	// TODO: write some code to allow the context to be safely cancelled
	/*
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
	*/

	start := time.Now()

	proposedMatchIds, overloadedPlayers, overloadedMatches, approvedMatches, err := stub(cfg, redisConn)
	overloadedPlayerList, overloadedMatchList, approvedMatchList := generateLists(overloadedPlayers, overloadedMatches, approvedMatches)

	fmt.Println("overloadedPlayers")
	pretty.PrettyPrint(overloadedPlayers)
	fmt.Println("overloadedPlayerList")
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
		values := strings.Split(proposedID, ".")
		timestamp, moID, proID := values[0], values[1], values[2]
		proposalID := "proposal." + timestamp + "." + moID + "." + proID
		backendID := moID + "." + proID
		fmt.Printf("approving proposal #%+v:%+v\n", proposalIndex, moID)
		fmt.Println("RENAME", proposalID, backendID)
		_, err = redisConn.Do("RENAME", proposalID, backendID)
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

func readConfig(filename string, defaults map[string]interface{}) (*viper.Viper, error) {
	/*
	   Examples of redis-related env vars as written by k8s
	   REDIS_SENTINEL_PORT_6379_TCP=tcp://10.55.253.195:6379
	   REDIS_SENTINEL_PORT=tcp://10.55.253.195:6379
	   REDIS_SENTINEL_PORT_6379_TCP_ADDR=10.55.253.195
	   REDIS_SENTINEL_SERVICE_PORT=6379
	   REDIS_SENTINEL_PORT_6379_TCP_PORT=6379
	   REDIS_SENTINEL_PORT_6379_TCP_PROTO=tcp
	   REDIS_SENTINEL_SERVICE_HOST=10.55.253.195
	*/
	v := viper.New()
	for key, value := range defaults {
		v.SetDefault(key, value)
	}
	v.SetConfigName(filename)
	v.SetConfigType("json")
	v.AddConfigPath(".")
	v.AutomaticEnv()

	// Optional read from config if it exists
	err := v.ReadInConfig()
	if err != nil {
		//lgr.Printf("error when reading config: %v\n", err)
		//lgr.Println("continuing...")
		err = nil
	}
	return v, err
}

func stub(cfg *viper.Viper, redisConn redis.Conn) ([]string, map[string][]int, map[int][]int, map[int]bool, error) {
	//Init Logger
	lgr := log.New(os.Stdout, "MMFEvalStub: ", log.LstdFlags)
	lgr.Println("Initializing example MMF proposal evaluator")

	// Put some config vars into other vars for readability
	proposalq := cfg.GetString("queues.proposals.name")

	lgr.Println("SCARD", proposalq)
	numProposals, err := redis.Int(redisConn.Do("SCARD", proposalq))
	lgr.Println("SPOP", proposalq, numProposals)
	propKeys, err := redis.Strings(redisConn.Do("SPOP", proposalq, numProposals))
	if err != nil {
		lgr.Println(err)
	}
	fmt.Printf("propKeys = %+v\n", propKeys)

	// Convert []string to []interface{} so it can be passed as variadic input
	// https://golang.org/doc/faq#convert_slice_of_interface
	rosterKeys := propToRoster(propKeys)
	lgr.Printf("MGET %+v", rosterKeys)
	rosters, err := redis.Strings(redisConn.Do("MGET", rosterKeys...))
	pretty.PrettyPrint(rosters)

	// This is a far cry from effecient but we expect a pretty small set of players under consideration
	// at any given time
	// Map that implements a set https://golang.org/doc/effective_go.html#maps
	overloadedPlayers := make(map[string][]int)
	overloadedMatches := make(map[int][]int)
	approvedMatches := make(map[int]bool)
	allPlayers := make(map[string]int)
	for index := range rosters {
		approvedMatches[index] = true
	}
	for index, playerList := range rosters {
		for _, pID := range strings.Split(playerList, " ") {
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
	return propKeys, overloadedPlayers, overloadedMatches, approvedMatches, err
}

func propToRoster(in []string) []interface{} {
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
