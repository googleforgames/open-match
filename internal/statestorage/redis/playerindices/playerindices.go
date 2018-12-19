// Package playerindices indexes player attributes in Redis for faster
// filtering of player pools.
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
package playerindices

import (
	"context"
	"errors"
	"strconv"
	"time"

	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/tidwall/gjson"
)

// Logrus structured logging setup
var (
	piLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
		"caller":    "statestorage/redis/playerindices/playerindices.go",
	}
	piLog = log.WithFields(piLogFields)
)

// Indexing is fairly limited right now.  It is all done using Sorted Sets in
// Redis, which require integer 'scores' in order to index players.
//
// Here are the guidelines if you want to index a player attribute in your
// Properties JSON blob when the  player's request comes in the Frontend  API
// so that you can filter on that  attribute in the Profiles you pass to the
// Backend API.
//  - Fields you want to index in your JSON should always be a key with an integer value, so use dictionaries/maps/hashes instad of lists/arrays for your data. (see examples below)
//  - When indexing fields in a player's JSON object, the key to index should be compatible with dot notation.  (see configured list of indices below)
//  - If you're trying to index a flag, just use the epoch timestamp of the request as the value unless you have a compelling reason to do otherwise.
// For example, if you want to index the following:
//  - Player's ping value to us-east
//  - Bool flag 'true' denoting player's choice to play CTF mode
//  - Bool flag 'true' denoting player's choice to play TeamDM mode
//  - Bool flag 'true' denoting player's choice to play SunsetValley map
//  - Players' matchmaking ranking value
// DON'T structure your JSON like this:
//   player {
//     "pings": {"us-east": 70, "eu-central": 120 },
//     "maps":  ["sunsetvalley", "bigskymountain"] ,
//     "modes": "ctf"
//   }
// But instead, use dictionaries with key/value pairs instead of lists (use epoch timestamp as the value):
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
// For now, OM reads your 'config/matchmaker_config.(json|yaml)' file for a
// list of indices, which it monitors using the golang module Viper
// (https://github.com/spf13/viper).
// In a full deployment, it is expected that you don't manage the config file
// directly, but instead put the contents of that file into a Kubernetes
// ConfigMap.  Kubernetes will write those contents to a file inside your
// running container for you. You can see where and how this is happening by
// looking at the kubernetes deployment resource definitions in the
// 'deployments/k8s/' directory.
//
// https://github.com/GoogleCloudPlatform/open-match/issues/42 discusses more
// about how configs are managed in Open Match.
//
// You can update the list of indices at run-time if you need to add or remove
// an index. Changes will affect all indexed players that come in the Frontend
// API from that point on.
// NOTE: there are potential edge cases here; see Retrieve() for details.

// Create indices for given player attributes in Redis.
// TODO: make this quit and not index the player if the context is cancelled.
func Create(ctx context.Context, rPool *redis.Pool, cfg *viper.Viper, player om_messages.Player) error {

	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	iLog := piLog.WithFields(log.Fields{"playerId": player.Id})

	// Get the indices from viper
	indices, err := Retrieve(cfg)
	if err != nil {
		iLog.Error(err.Error())
		return err
	}

	// Start putting this player into the indices in Redis.
	redisConn.Send("MULTI")
	// Loop through all attributes we want to index.
	for _, attribute := range indices {
		value := gjson.Get(player.Properties, attribute)
		// Check that value contains a valid 64-bit integer
		if value.Exists() {
			if _, err := strconv.ParseInt(value.String(), 10, 64); err == nil {
				redisConn.Send("ZADD", attribute, player.Id, value)
			} else {
				// Value was malformed or missing; use current epoch timestamp instead.
				redisConn.Send("ZADD", attribute, player.Id, time.Now().Unix())
			}
		} else {
			iLog.WithFields(log.Fields{"attribute": attribute}).Debug("No such attribute to index!")
		}
	}
	_, err = redisConn.Do("EXEC")

	return err
}

// Delete a player's indices without deleting their JSON object representation from
// state storage.
// Note: In Open Match, it is best practice to 'lazily' remove indices
// by running this as a goroutine.
// TODO: make this quit cleanly if the context is cancelled.
func Delete(ctx context.Context, rPool *redis.Pool, cfg *viper.Viper, playerID string) error {

	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	//TODO: remove deindexing from delete and call this instead
	diLog := piLog.WithFields(log.Fields{"playerID": playerID})

	// Get the indices from viper
	indices, err := Retrieve(cfg)
	if err != nil {
		piLog.Error(err.Error())
		return err
	}

	// Remove playerID from indices
	redisConn.Send("MULTI")
	for _, attribute := range indices {
		diLog.WithFields(log.Fields{"attribute": attribute}).Debug("De-indexing")
		redisConn.Send("ZREM", attribute, playerID)
	}
	_, err = redisConn.Do("EXEC")
	return err

}

// Retrieve pulls the player indices from the Viper config
func Retrieve(cfg *viper.Viper) (indices []string, err error) {

	if cfg.IsSet("playerIndices") {
		indices = cfg.GetStringSlice("playerIndices")
	} else {
		err = errors.New("Failure to get list of indices")
		return nil, err
	}
	// This code attempts to handle an edge case when the user has removed an
	// index from the list of player indices but players still exist who are
	// indexed using the (now no longer used) index. The user should put the
	// index they are no longer using into this config parameter so that
	// deleting players with previous indexes doesn't result in a Redis memory
	// leak.  In a future version, Open Match should track previous indices
	// itself and handle this for the user.
	if cfg.IsSet("previousPlayerIndices") {
		indices = append(indices, cfg.GetStringSlice("previousPlayerIndices")...)
	}

	return
}
