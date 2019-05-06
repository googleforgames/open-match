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

package redispb

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestorage/redis/playerindices"
)

// Logrus structured logging setup
var (
	pLogFields = logrus.Fields{
		"app":       "openmatch",
		"component": "statestorage",
	}
	pLog = logrus.WithFields(pLogFields)
)

// UnmarshalPlayerFromRedis unmarshals a Player from a redis hash.
// This can probably be deprecated if we work on getting the above generic enough.
// The problem is that protobuf message reflection is pretty messy.
func UnmarshalPlayerFromRedis(ctx context.Context, pool *redis.Pool, player *pb.Player) error {

	// Get the Redis connection.
	redisConn, err := pool.GetContext(context.Background())
	defer redisConn.Close()
	if err != nil {
		pLog.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to connect to redis")
		return err
	}

	// Prepare redis command.
	cmd := "HGETALL"
	key := player.Id
	resultLog := pLog.WithFields(logrus.Fields{
		"component": "statestorage",
		"cmd":       cmd,
		"key":       key,
	})

	// Run redis command
	playerMap, err := redis.StringMap(redisConn.Do(cmd, key))

	// Put values from redis into the Player message
	player.Properties = playerMap["properties"]
	player.Pool = playerMap["pool"]
	player.Assignment = playerMap["assignment"]
	player.Status = playerMap["status"]
	player.Error = playerMap["error"]

	// TODO: Room for improvement here.
	if a := playerMap["attributes"]; a != "" {
		attrsJSON := fmt.Sprintf("{\"attributes\": %v}", a)
		err = jsonpb.UnmarshalString(attrsJSON, player)
		if err != nil {
			resultLog.Error("failure on attributes")
			resultLog.Error(a)
		}
	}

	resultLog.Debug("state storage operation: player unmarshalled")
	return err

}

// PlayerWatcher makes a channel and returns it immediately.  It also launches an
// asynchronous goroutine that watches a redis key and returns updates to
// that key on the channel.
//
// The pattern for this function is from 'Go Concurrency Patterns', it is a function
// that wraps a closure goroutine, and returns a channel.
// reference: https://talks.golang.org/2012/concurrency.slide#25
//
// NOTE: this function will never stop querying Redis during normal operation! You need to
//  disconnect the client from the frontend API (which closes the context) once
//  you've received the results you were waiting for to stop doing work!
func PlayerWatcher(bo backoff.BackOffContext, pool *redis.Pool, player pb.Player) <-chan pb.Player {

	pwLog := pLog.WithFields(logrus.Fields{"playerId": player.Id})

	// Establish channel to return results on.
	watchChan := make(chan pb.Player, 1)

	go func() {
		defer close(watchChan)

		// var declaration
		var prevResults = ""

		// Loop, querying redis until this key has a value or the Redis query fails
		for {
			// Update the player's 'accessed' timestamp to denote they haven't disappeared
			err := playerindices.Touch(bo.Context(), pool, player.Id)
			if err != nil {
				// Not fatal, but this error should be addressed.  This could
				// cause the player to expire while still actively connected!
				pwLog.WithFields(logrus.Fields{"error": err.Error()}).Error("Unable to update accessed metadata timestamp")
			}

			// Get player from redis.
			results := pb.Player{Id: player.Id}
			err = UnmarshalPlayerFromRedis(bo.Context(), pool, &results)
			if err != nil {
				// Return error and quit.
				pwLog.Debug("State storage error:", err.Error())
				results.Error = err.Error()
				watchChan <- results
				return
			}

			// Check for new results and send them. Store a copy of the
			// latest version in string form so we can compare it easily in
			// future loops.
			//
			// If we decide to watch other message fields for updates,
			// they will need to be added here.
			//
			// This can be made much cleaner if protobuffer reflection improves.
			curResults := fmt.Sprintf("%v%v%v", results.Assignment, results.Status, results.Error)
			if prevResults == curResults {
				pwLog.Debug("No new results, backing off")
			} else {
				// Return value retreived from Redis
				pwLog.Debug("state storage watched player record changed")
				watchedFields := pb.Player{
					// Return only the watched fields to minimize traffic
					Id:         results.Id,
					Assignment: results.Assignment,
					Status:     results.Status,
					Error:      results.Error,
				}
				watchChan <- watchedFields
				prevResults = curResults

				// Reset backoff & timeout counters
				bo.Reset()
			}

			if d := bo.NextBackOff(); d != backoff.Stop {
				time.Sleep(d)
			} else {
				return
			}
		}
	}()

	return watchChan
}
