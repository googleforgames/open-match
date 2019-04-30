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
	"fmt"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

var (
	// Logrus structured logging setup
	piLogFields = logrus.Fields{
		"app":       "openmatch",
		"component": "statestorage",
	}
	piLog = logrus.WithFields(piLogFields)

	// OM Internal metadata indices
	MetaIndices = []string{
		"OM_METADATA.created",
		"OM_METADATA.accessed",
	}
)

// Indexing  is all done using Sorted Sets in Redis, which require integer
// 'scores' for each attribute in order to index players.
//
// Here are the guidelines if you want to index a player attribute in your
// Properties JSON blob when the  player's request comes in the Frontend  API
// so that you can filter on that  attribute in the Profiles you pass to the
// Backend API.
//  - Fields you want to index in your JSON should always be a key with an
//    integer value, so use dictionaries/maps/hashes instad of lists/arrays for
//    your data. (see examples below)
//  - When indexing fields in a player's JSON object, the key to index should
//    be compatible with dot notation. This means no keys with meta characters
//    (anything escaped by regexp.QuoteMeta() shouldn't be used in your key
//    name!)
//  - If you're trying to index a flag, just use the epoch timestamp of the
//    request as the value unless you have a compelling reason to do otherwise.
//
// For example, if you want to index the following:
//    - Player's ping value to us-east
//    - Bool flag 'true' denoting player's choice to play CTF mode
//    - Bool flag 'true' denoting player's choice to play TeamDM mode
//    - Bool flag 'true' denoting player's choice to play SunsetValley map
//    - Players' matchmaking ranking value
//
// DON'T structure your JSON like this:
//   player {
//     "pings": {"us-east": 70, "eu-central": 120 },
//     "maps":  ["sunsetvalley", "bigskymountain"] ,
//     "modes": "ctf"
//   }
// Instead, use dictionaries with key/value pairs instead of lists (use epoch
// timestamp as the value if your attribute should act as a boolean flag):
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
//
// For now, OM reads your 'config/matchmaker_config.(json|yaml)' file for a
// list of indices.
// In a full deployment, it is expected that you don't manage the config file
// directly, but instead put the contents of that file into a Kubernetes
// ConfigMap.  Kubernetes will write those contents to a file inside your
// running container for you.
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
func Create(ctx context.Context, rPool *redis.Pool, cfg config.View, player om_messages.Player) error {

	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	iLog := piLog.WithFields(logrus.Fields{"playerId": player.Id})

	// Get the indices from configuration
	indices, err := Retrieve(cfg)
	// Get metadata indicies
	indices = append(indices, MetaIndices...)
	if err != nil {
		iLog.Error(err.Error())
		return err
	}

	// Start putting this player into the indices in Redis.
	redisConn.Send("MULTI")
	// Loop through all attributes we want to index.
	for _, attribute := range indices {
		// Default value for all attributes if missing or malformed is the current epoch timestamp.
		value := time.Now().Unix()

		// If this is a user-defined index, look for it in the input player properties JSON
		if !strings.HasPrefix(attribute, "OM_METADATA") {
			// NOTE: This gjson call has issues with JSON keys containing meta characters (dot, slash, etc).
			//   The regexp.QuoteMeta below gets around those issues, but won't pick up dot-notation keys!
			//   End result is that you shouldn't use meta characters in your JSON property keys!
			//v := gjson.Get(player.Properties, regexp.QuoteMeta(attribute))
			v := gjson.Get(player.Properties, attribute)

			// If this attribute wasn't provided in the JSON, continue to the
			// next attribute to index.
			if !v.Exists() {
				iLog.WithFields(logrus.Fields{"attribute": attribute, "value": v.Raw}).Debug("Couldn't find index in JSON: ", player.Properties)
				continue
			} else if -9223372036854775808 <= v.Int() && v.Int() <= 9223372036854775807 {
				// value contains a valid 64-bit integer
				value = v.Int()
			} else {
				iLog.WithFields(logrus.Fields{"attribute": attribute}).Debug("No valid value for attribute, not indexing")
			}

		}
		// Index the attribute by value.
		iLog.Debug(fmt.Sprintf("%v %v %v %v", "ZADD", attribute, player.Id, value))
		redisConn.Send("ZADD", attribute, value, player.Id)
	}

	// Run pipelined Redis commands.
	_, err = redisConn.Do("EXEC")

	return err
}

// Delete a player's indices without deleting their JSON object representation from
// state storage.
// Note: In Open Match, it is best practice to 'lazily' remove indices
// by running this as a goroutine.
// TODO: make this quit cleanly if the context is cancelled.
func Delete(ctx context.Context, rPool *redis.Pool, cfg config.View, playerID string) error {

	diLog := piLog.WithFields(logrus.Fields{"playerID": playerID})

	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	// Get the list of indices to delete
	indices, err := Retrieve(cfg)
	// Look for previously configured indices
	indices = append(indices, RetrievePrevious(cfg)...)
	if err != nil {
		diLog.Error(err.Error())
		return err
	}

	// Remove playerID from indices
	redisConn.Send("MULTI")
	for _, attribute := range indices {
		diLog.WithFields(logrus.Fields{"attribute": attribute}).Debug("De-indexing")
		redisConn.Send("ZREM", attribute, playerID)
	}
	_, err = redisConn.Do("EXEC")
	return err

}

// DeleteMeta removes a player's internal Open Match metadata indices, and should only be used
// after deleting their JSON object representation from state storage.
// Note: In Open Match, it is best practice to 'lazily' remove indices
// by running this as a goroutine.
// TODO: make this quit cleanly if the context is cancelled.
func DeleteMeta(ctx context.Context, rPool *redis.Pool, playerID string) {

	dmLog := piLog.WithFields(logrus.Fields{"playerID": playerID})

	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	// Remove playerID from metaindices
	redisConn.Send("MULTI")
	for _, attribute := range MetaIndices {
		dmLog.WithFields(logrus.Fields{"attribute": attribute}).Debug("De-indexing from metadata")
		redisConn.Send("ZREM", attribute, playerID)
	}
	_, err := redisConn.Do("EXEC")
	if err != nil {
		dmLog.WithFields(logrus.Fields{"error": err.Error}).Error("Error de-indexing from metadata")
	}
}

// Touch is analogous to the Unix touch command.  It updates the accessed time of the player
// in the OM_METADATA.accessed index to the current epoch timestamp.
func Touch(ctx context.Context, rPool *redis.Pool, playerID string) error {
	// Connect to redis
	redisConn := rPool.Get()
	defer redisConn.Close()

	_, err := redisConn.Do("ZADD", "OM_METADATA.accessed", time.Now().Unix(), playerID)
	return err
}

// Retrieve pulls the player indices from configuration
func Retrieve(cfg config.View) (indices []string, err error) {
	// In addition to the user-defined indices from the config file, Open Match
	// forces the following indicies to exist for all players.  'created' is
	// used to calculate how long a player has been waiting for a match,
	// 'accessed' is used to determine when a player needs to be expired out of
	// state storage.
	indices = append(indices, []string{}...)

	if cfg.IsSet("playerIndices") {
		indices = append(indices, cfg.GetStringSlice("playerIndices")...)
	} else {
		err = errors.New("Failure to get list of indices")
		return nil, err
	}

	return
}

// RetrievePrevious attempts to handle an edge case when the user has removed an
// index from the list of player indices but players still exist who are
// indexed using the (now no longer used) index. The user should put the
// index they are no longer using into this config parameter so that
// deleting players with previous indexes doesn't result in a Redis memory
// leak.  In a future version, Open Match should track previous indices
// itself and handle this for the user.
func RetrievePrevious(cfg config.View) []string {
	if cfg.IsSet("previousPlayerIndices") {
		return cfg.GetStringSlice("previousPlayerIndices")
	}
	return nil
}
