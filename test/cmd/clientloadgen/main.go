// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package main is the clientloadgen application.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/player"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/redis/playerq"

	"github.com/gomodule/redigo/redis"
)

func main() {
	cfg, err := config.Read()
	check(err, "QUIT")

	pool, err := redishelpers.ConnectionPool(cfg)

	redisConn := pool.Get()
	defer redisConn.Close()

	// Make a new player generator
	player.New()

	const numPlayers = 20
	fmt.Println("Starting client api stub...")

	for {
		start := time.Now()

		for i := 1; i <= numPlayers; i++ {
			_, err = queuePlayer(redisConn)
			check(err, "")
		}

		elapsed := time.Since(start)
		check(err, "")
		fmt.Printf("Redis queries and Xid generation took %s\n", elapsed)
		fmt.Println("Sleeping")
		time.Sleep(5 * time.Second)
	}
}

// Queue a player; dump all their matchmaking constraints into a JSON string.
func queuePlayer(redisConn redis.Conn) (playerID string, err error) {
	playerID, playerData, debug := player.Generate()
	_ = debug // TODO.  For now you could copy this into playerdata before creating player if you want it available in redis
	pdJSON, _ := json.Marshal(playerData)
	err = playerq.Create(redisConn, playerID, string(pdJSON))
	check(err, "")
	// This assumes you have at least ping data for this one region, comment out if you don't need this output
	fmt.Printf("Generated player %v in %v\n\tPing to %v: %3dms\n", playerID, debug["city"], "gcp.europe-west2", playerData["region.europe-west2"])
	return
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
