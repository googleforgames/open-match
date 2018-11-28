// Package playerq is a player queue specific redis implementation and will be removed in a future version.
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
package playerindicies

import (
	"context"

	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Logrus structured logging setup
var (
	piLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
		"caller":    "statestorage/redis/playerindices/playerindicies.go",
	}
	piLog = log.WithFields(piLogFields)
)

// Indexing is fairly limited right now.  It is all done using Sorted Sets in
// Redis, which require integer 'scores' in order to index players.
//
// Here are the guidelines if you want
// to index a player attribute when the player's request comes in the Frontend
// API so that you can filter on that attribute in the Profiles you pass to the
// Backend API.
//  - Indexed fields should always be a key with an integer value, so use dictionaries/maps/hashes instad of lists/arrays for your data.
//  - When indexing fields in a player's JSON object, the key to index should be configured using dot notation.
//  - If you're trying to index a flag, just use the epoch timestamp of the request as the value unles you have a compelling reason to do otherwise.
// For Example, if you want to index the following:
//  - Player's ping to us-east
//  - Bool flag denoting player's choice to play CTF mode
//  - Bool flag denoting player's choice to play TeamDM mode
//  - Bool flag denoting player's choice to play SunsetValley map
//  - Players' matchmaking ranking
// DON'T structure your JSON like this:
//   player {
//     "pings": {"us-east": 70, "eu-central": 120 },
//     "maps":  ["sunsetvalley", "bigskymountain"] ,
//     "modes": "ctf"
//   }
// But instead, use dictionaries with key/value pairs instead of lists:
//   player {
//     "pings": {"us-east": 70, "eu-central": 120 },
//     "maps":  {"sunsetvalley": 1234567890, "bigskymountain": 1234567890 } ,
//     "modes": {"ctf": 1234567890}
//   }
// Then, configure your list of indices for OM to look like this:
//   "indices": [
//     "pings.us-east",
//     "modes.ctf",
//     "modes.teamdm",
//     "maps.sunsetvalley",
//     "mmr.rating",
//   ]
// For now, OM reads your config/matchmaker_json.config file for a list of
// indicies, and stores those in Redis.  You can update the list of indices in
// Redis at run-time if you need to add or remove an index. Changes will
// affect all indexed players that come in the Frontend API from
// that point on.

// Index a player's JSON object representation in state storage.
func Index(rPool *redis.Pool, playerId string) error {

	redisConn := rPool.Get()
	defer redisConn.Close()

	iLog := piLog.WithFields(log.Fields{"playerId": playerId})

	indices, err := redis.Strings(redisConn.Do("SMEMBERS", "indicies"))
	if err != nil {
		iLog.Error("Failure to get list of indicies")
		iLog.Error(err)
		return err
	}

	//TODO impelent the other side of this
	player := &om_messages.Player{Id: playerId}
	err = redispb.UnmarshalFromRedis(context.Background(), rPool, player)

	// Loop through all attributes we want to index.
	for _, attribute := range indices {
		value := gjson.Get(player.Properties, attribute)
		if value.Exists() {
			_, err := redisConn.Do("ZADD", attribute, playerId, value)
			if err != nil {
				iLog.WithFields(log.Fields{
					"attribute": attribute,
					"value":     value,
				}).Error("Unable to index player attribute")
				iLog.Fatal(err)
			}
		} else {
			iLog.WithFields(log.Fields{"attribute": attribute}).Debug("No such attribute to index!")
		}

	}
	return err
}

// Deindex a player without deleting there JSON object representation from
// state storage.  Unindexing is done in two stages: first the player is added
// to an ignore list, which 'atomically' removes them from consideration. A
// Goroutine is then kicked off to 'lazily' remove them from any field indicies
// that contain them.
func Deindex(redisConn redis.Conn, playerID string) error {

	//TODO: remove deindexing from delete and call this instead

	results, err := Retrieve(redisConn, playerID)
	if err != nil {
		log.Println("couldn't retreive player properties for ", playerID)
	}

	redisConn.Send("MULTI")

	// Remove playerID from indices
	for iName := range results {
		log.WithFields(log.Fields{
			"field": iName,
			"key":   playerID}).Debug("Un-indexing field")
		redisConn.Send("ZREM", iName, playerID)
	}
	_, err = redisConn.Do("EXEC")
	check(err, "")
	return err

}
