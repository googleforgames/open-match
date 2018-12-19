// Package redispb marshals and unmarshals Open Match Backend protobuf messages
// ('MatchObject') for redis state storage.
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
	"reflect"
	"strings"
	"time"

	om_messages "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Logrus structured logging setup
var (
	rpLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
		"caller":    "internal/statestorage/redis/redispb/redispb.go",
	}
	rpLog = log.WithFields(rpLogFields)
)

// MarshalToRedis marshals a protobuf message to a redis hash.
// The protobuf message in question must have an 'id' field.
func MarshalToRedis(ctx context.Context, pool *redis.Pool, pb proto.Message) (err error) {

	// We want to serialize to redis as JSON, not the typical protobuf string
	// serializer, so start by marshalling to json.
	this := jsonpb.Marshaler{}
	jsonMsg, err := this.MarshalToString(pb)
	if err != nil {
		rpLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
			"protobuf":  pb,
		}).Error("failure marshaling protobuf message to JSON")
		return
	}

	// Get redis key
	keyResult := gjson.Get(jsonMsg, "id")

	// Return error if the provided protobuf message doesn't have an ID field
	if !keyResult.Exists() {
		err = errors.New("cannot unmarshal protobuf messages without an id field")
		rpLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to retrieve from redis")
		return
	}
	key := keyResult.String()

	// Prepare redis command.
	cmd := "HSET"
	resultLog := rpLog.WithFields(log.Fields{
		"key": key,
		"cmd": cmd,
	})

	// Get the Redis connection.
	redisConn, err := pool.GetContext(context.Background())
	defer redisConn.Close()
	if err != nil {
		rpLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to connect to redis")
		return
	}
	redisConn.Send("MULTI")

	// Write all non-id fields from the protobuf message to state storage.
	// Use reflection to get the field names from the protobuf message.
	pbInfo := reflect.ValueOf(pb).Elem()
	for i := 0; i < pbInfo.NumField(); i++ {
		// TODO: change this to use the json name field from the struct tags
		//  something like parseTag() in src/encoding/json/tags.go
		//field := strings.ToLower(pbInfo.Type().Field(i).Tag.Get("json"))
		field := strings.ToLower(pbInfo.Type().Field(i).Name)
		value := gjson.Get(jsonMsg, field)
		if field != "id" {
			// This isn't the ID field, so write it to the redis hash.
			redisConn.Send(cmd, key, field, value)
			if err != nil {
				resultLog.WithFields(log.Fields{
					"error":     err.Error(),
					"component": "statestorage",
					"field":     field,
				}).Error("State storage error")
				return
			}
			resultLog.WithFields(log.Fields{
				"component": "statestorage",
				"field":     field,
				"value":     value,
			}).Info("State storage operation")

		}
	}
	_, err = redisConn.Do("EXEC")
	return
}

// UnmarshalFromRedis unmarshals a MatchObject from a redis hash.
// This can probably be made generic to work with other pb messages in the future.
// In every case where we don't get an update, we return an error.
func UnmarshalFromRedis(ctx context.Context, pool *redis.Pool, pb *om_messages.MatchObject) error {

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
	key := pb.Id
	resultLog := rpLog.WithFields(log.Fields{
		"component": "statestorage",
		"cmd":       cmd,
		"key":       key,
	})
	pbMap, err := redis.StringMap(redisConn.Do(cmd, key))

	// Put values from redis into the MatchObject message
	pb.Error = pbMap["error"]
	pb.Properties = pbMap["properties"]

	// TODO: Room for improvement here.
	poolsJSON := fmt.Sprintf("{\"pools\": %v}", pbMap["pools"])
	err = jsonpb.UnmarshalString(poolsJSON, pb)
	if err != nil {
		resultLog.Error("failure on pool")
		resultLog.Error(pbMap["pools"])
		resultLog.Error(err)
	}

	rostersJSON := fmt.Sprintf("{\"rosters\": %v}", pbMap["rosters"])
	err = jsonpb.UnmarshalString(rostersJSON, pb)
	if err != nil {
		resultLog.Error("failure on roster")
		resultLog.Error(pbMap["rosters"])
		log.Error(err)
	}
	rpLog.Debug("Final pb:")
	rpLog.Debug(pb)
	return err
}

// Watcher makes a channel and returns it immediately.  It also launches an
// asynchronous goroutine that watches a redis key and returns updates to
// that key on the channel.
//
// The pattern for this function is from 'Go Concurrency Patterns', it is a function
// that wraps a closure goroutine, and returns a channel.
// reference: https://talks.golang.org/2012/concurrency.slide#25
func Watcher(ctx context.Context, pool *redis.Pool, pb om_messages.MatchObject) <-chan om_messages.MatchObject {

	watchChan := make(chan om_messages.MatchObject)
	results := om_messages.MatchObject{Id: pb.Id}

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
				//results, err = Retrieve(ctx, pool, key)
				results = om_messages.MatchObject{Id: pb.Id}
				err = UnmarshalFromRedis(ctx, pool, &results)
				if err != nil {
					rpLog.Debug("No new results")
					time.Sleep(2 * time.Second) // TODO: exp bo + jitter
				}
			}
		}
		// Return value retreived from Redis asynchonously and tell calling function we're done
		rpLog.Debug("state storage watched record update detected")
		watchChan <- results
	}()

	return watchChan
}
