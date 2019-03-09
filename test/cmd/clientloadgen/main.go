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
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/player"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/redis/playerq"

	"github.com/gomodule/redigo/redis"
)

var (
	numPlayers int
	sleep      int
)

func main() {
	// Parse command line flags
	flag.IntVar(&numPlayers, "numplayers", 20, "Number of players to generate per cycle")
	flag.IntVar(&sleep, "sleep", 1000, "Number of ms to sleep between cycles")
	flag.Parse()

	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz
	redisURL := "redis://" + os.Getenv("REDIS_SERVICE_HOST") + ":" + os.Getenv("REDIS_SERVICE_PORT")

	// Redis connection
	pool := redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}

	redisConn := pool.Get()
	defer redisConn.Close()

	// Make a new player generator
	player.New()

	fmt.Println("Starting client api stub...")

	// Loop forever, queuing players
	for {
		start := time.Now()

		for i := 1; i <= numPlayers; i++ {
			_, err := queuePlayer(redisConn)
			if err != nil {
				log.Println(err)
			}

		}

		elapsed := time.Since(start)
		fmt.Printf("Generated %v players in %s\n", numPlayers, elapsed)
		fmt.Printf(" Sleeping for %vms\n", sleep)
		time.Sleep(time.Duration(sleep) * time.Millisecond)
	}
}

// Queue a player; dump all their matchmaking constraints into a JSON string.
func queuePlayer(redisConn redis.Conn) (playerID string, err error) {
	playerID, playerData, debug := player.Generate()
	_ = debug // TODO.  For now you could copy this into playerdata before creating player if you want it available in redis
	pdJSON, _ := json.Marshal(playerData)
	err = playerq.Create(redisConn, playerID, string(pdJSON))
	if err != nil {
		log.Fatal(err)
	}
	// This assumes you have at least ping data for this one region, comment out if you don't need this output
	//fmt.Printf("Generated player %v in %v\n\tPing to %v: %3dms\n", playerID, debug["city"], "gcp.europe-west2", playerData["region.europe-west2"])
	return
}
