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
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/player"
	"github.com/GoogleCloudPlatform/open-match/test/cmd/clientloadgen/redis/playerq"

	"github.com/gomodule/redigo/redis"
	"github.com/spf13/viper"
)

var interactive = flag.Bool("i", false, "toggle interactive mode")
var genPlayers = flag.Int("n", 20, "number of players to generate at each iteration in non-intercative mode")
var sleep = flag.Int("s", 5, "number of seconds to sleep after each iteration in non-interactive mode")

func main() {
	flag.Parse()

	conf, err := readConfig("", map[string]interface{}{
		"REDIS_SERVICE_HOST": "127.0.0.1",
		"REDIS_SERVICE_PORT": "6379",
	})
	check(err, "QUIT")

	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz
	redisURL := "redis://" + conf.GetString("REDIS_SERVICE_HOST") + ":" + conf.GetString("REDIS_SERVICE_PORT")

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

	for {
		var numPlayers int
		if *interactive {
			fmt.Println(">> Input number of players to generate")

			var t string
			fmt.Scanln(&t)
			numPlayers, err = strconv.Atoi(t)
			if err != nil {
				panic(err)
			}
		} else {
			numPlayers = *genPlayers
		}
		if numPlayers <= 0 {
			panic("num players must be positive integer")
		}

		start := time.Now()

		for i := 1; i <= numPlayers; i++ {
			_, err = queuePlayer(redisConn)
			check(err, "")
		}

		elapsed := time.Since(start)
		check(err, "")
		fmt.Printf("Redis queries and Xid generation took %s\n", elapsed)

		if !*interactive {
			fmt.Println("Sleeping")
			time.Sleep(time.Duration(*sleep) * time.Second)
		}
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

func readConfig(filename string, defaults map[string]interface{}) (*viper.Viper, error) {
	/*
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
	v.AddConfigPath(".")
	v.AutomaticEnv()

	// Optional read from config if it exists
	err := v.ReadInConfig()
	if err != nil {
		//fmt.Printf("error when reading config: %v\n", err)
		//fmt.Println("continuing...")
		err = nil
	}
	return v, err
}
