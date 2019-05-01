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

// Package redispb marshals and unmarshals Open Match Backend protobuf messages
// ('MatchObject') for redis state storage.
//  More details about the protobuf messages used in Open Match can be found in
//  the api/protobuf-spec/om_messages.proto file.
package redispb

import (
	"context"
	"errors"
	"fmt"
	"time"

	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/cenkalti/backoff"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

// Logrus structured logging setup
var (
	moLogFields = logrus.Fields{
		"app":       "openmatch",
		"component": "statestorage",
	}
	moLog = logrus.WithFields(moLogFields)
)

// UnmarshalFromRedis unmarshals a MatchObject from a redis hash.
// This can probably be made generic to work with other pb messages in the future.
// In every case where we don't get an update, we return an error.
func UnmarshalFromRedis(ctx context.Context, pool *redis.Pool, pb *om_messages.MatchObject) error {

	// Get the Redis connection.
	redisConn, err := pool.GetContext(context.Background())
	defer redisConn.Close()
	if err != nil {
		moLog.WithFields(logrus.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to connect to redis")
		return err
	}

	// Prepare redis command.
	cmd := "HGETALL"
	key := pb.Id
	resultLog := moLog.WithFields(logrus.Fields{
		"component": "statestorage",
		"cmd":       cmd,
		"key":       key,
	})
	pbMap, err := redis.StringMap(redisConn.Do(cmd, key))
	if len(pbMap) == 0 {
		return errors.New("matchobject key does not exist")
	}

	// Put values from redis into the MatchObject message
	pb.Error = pbMap["error"]
	pb.Properties = pbMap["properties"]

	// TODO: Room for improvement here.
	if j := pbMap["pools"]; j != "" {
		poolsJSON := fmt.Sprintf("{\"pools\": %v}", j)
		err = jsonpb.UnmarshalString(poolsJSON, pb)
		if err != nil {
			resultLog.Error("failure on pool")
			resultLog.Error(j)
			resultLog.Error(err)
		}
	}

	if j := pbMap["rosters"]; j != "" {
		rostersJSON := fmt.Sprintf("{\"rosters\": %v}", j)
		err = jsonpb.UnmarshalString(rostersJSON, pb)
		if err != nil {
			resultLog.Error("failure on roster")
			resultLog.Error(j)
			resultLog.Error(err)
		}
	}
	moLog.Debug("Final pb:")
	moLog.Debug(pb)
	return err
}

// Watcher makes a channel and returns it immediately.  It also launches an
// asynchronous goroutine that watches a redis key and returns updates to
// that key on the channel.
//
// The pattern for this function is from 'Go Concurrency Patterns', it is a function
// that wraps a closure goroutine, and returns a channel.
// reference: https://talks.golang.org/2012/concurrency.slide#25
//
// NOTE: runs until cancelled, timed out or result is found in Redis.
func Watcher(bo backoff.BackOffContext, pool *redis.Pool, pb om_messages.MatchObject) <-chan om_messages.MatchObject {

	watchChan := make(chan om_messages.MatchObject)
	results := om_messages.MatchObject{Id: pb.Id}

	go func() {
		defer close(watchChan)

		// Loop, querying redis until this key has a value
		for {
			results = om_messages.MatchObject{Id: pb.Id}
			err := UnmarshalFromRedis(bo.Context(), pool, &results)
			if err == nil {
				// Return value retreived from Redis asynchonously and tell calling function we're done
				moLog.Debug("state storage watched record update detected")
				watchChan <- results
				return
			}

			if d := bo.NextBackOff(); d != backoff.Stop {
				moLog.Debug("No new results, backing off")
				time.Sleep(d)
			} else {
				moLog.Debug("No new results after all backoff attempts")
				return
			}
		}
	}()

	return watchChan
}
