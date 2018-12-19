// Package redispb marshals and unmarshals Open Match Frontend protobuf
// messages ('Players' or groups) for redis state storage.
//  More details about the protobuf messages used in Open Match can be found in
//  the api/protobuf-spec/om_messages.proto file.
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

All of this can probably be done more succinctly with some more interface and
reflection, this is a hack but works for now.
*/
package redispb

import (
	"context"
	"errors"
	"fmt"
	"time"

	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// UnmarshalPlayerFromRedis unmarshals a Player from a redis hash.
// This can probably be deprecated if we work on getting the above generic enough.
// The problem is that protobuf message reflection is pretty messy.
func UnmarshalPlayerFromRedis(ctx context.Context, pool *redis.Pool, player *om_messages.Player) error {

	// Get the Redis connection.
	redisConn, err := pool.GetContext(context.Background())
	defer redisConn.Close()
	if err != nil {
		rpLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to connect to redis")
		return err
	}

	// Prepare redis command.
	cmd := "HGETALL"
	key := player.Id
	resultLog := rpLog.WithFields(log.Fields{
		"component": "statestorage",
		"cmd":       cmd,
		"key":       key,
	})
	playerMap, err := redis.StringMap(redisConn.Do(cmd, key))

	// Put values from redis into the Player message
	player.Properties = playerMap["properties"]
	player.Pool = playerMap["pool"]
	player.Assignment = playerMap["assignment"]
	player.Status = playerMap["status"]
	player.Error = playerMap["error"]

	// TODO: Room for improvement here.
	attrsJSON := fmt.Sprintf("{\"attributes\": %v}", playerMap["attributes"])
	err = jsonpb.UnmarshalString(attrsJSON, player)
	if err != nil {
		resultLog.Error("failure on attributes")
		resultLog.Error(playerMap["attributes"])
		log.Error(err)
	}
	rpLog.Debug("Final player:")
	rpLog.Debug(player)
	return err

}

// Watcher makes a channel and returns it immediately.  It also launches an
// asynchronous goroutine that watches a redis key and returns updates to
// that key on the channel.
//
// The pattern for this function is from 'Go Concurrency Patterns', it is a function
// that wraps a closure goroutine, and returns a channel.
// reference: https://talks.golang.org/2012/concurrency.slide#25
func PlayerWatcher(ctx context.Context, pool *redis.Pool, pb om_messages.Player) <-chan om_messages.Player {

	watchChan := make(chan om_messages.Player)
	results := om_messages.Player{Id: pb.Id}

	go func() {
		// var declaration
		var err = errors.New("haven't queried Redis yet")

		// Loop, querying redis until this key has a value
		for err != nil {
			select {
			case <-ctx.Done():
				// Cleanup
				close(watchChan)
				return
			default:
				results = om_messages.Player{Id: pb.Id}
				err = UnmarshalPlayerFromRedis(ctx, pool, &results)
				if err != nil {
					rpLog.Debug("No new results")
					time.Sleep(2 * time.Second) // TODO: exp bo + jitter
				}
			}
		}
		// Return value retreived from Redis asynchonously and tell calling function we're done
		rpLog.Debug("state storage watched player record update detected")
		watchChan <- results
	}()

	return watchChan
}
