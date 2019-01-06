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
	"reflect"
	"strings"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/gogo/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// Logrus structured logging setup
var (
	sLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
	}
	sLog = log.WithFields(sLogFields)
)

// MarshalToRedis marshals a protobuf message to a redis hash.
// The protobuf message in question must have an 'id' field.
// If a positive integer TTL is provided, it will also be set.
func MarshalToRedis(ctx context.Context, pool *redis.Pool, pb proto.Message, ttl int) error {

	// We want to serialize to redis as JSON, not the typical protobuf string
	// serializer, so start by marshalling to json.
	this := jsonpb.Marshaler{}
	jsonMsg, err := this.MarshalToString(pb)
	if err != nil {
		sLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
			"protobuf":  pb,
		}).Error("failure marshaling protobuf message to JSON")
		return err
	}

	// Get redis key
	keyResult := gjson.Get(jsonMsg, "id")

	// Return error if the provided protobuf message doesn't have an ID field
	if !keyResult.Exists() {
		err = errors.New("cannot unmarshal protobuf messages without an id field")
		sLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to retrieve from redis")
		return err
	}
	key := keyResult.String()

	// Prepare redis command.
	cmd := "HSET"
	resultLog := sLog.WithFields(log.Fields{
		"key": key,
		"cmd": cmd,
	})

	// Get the Redis connection.
	redisConn, err := pool.GetContext(context.Background())
	defer redisConn.Close()
	if err != nil {
		sLog.WithFields(log.Fields{
			"error":     err.Error(),
			"component": "statestorage",
		}).Error("failed to connect to redis")
		return err
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
		value := ""
		//value, err = strconv.Unquote(gjson.Get(jsonMsg, field).String())
		value = gjson.Get(jsonMsg, field).String()
		if err != nil {
			resultLog.Error("Issue with Unquoting string", err)
		}
		if field != "id" {
			// This isn't the ID field, so write it to the redis hash.
			redisConn.Send(cmd, key, field, value)
			if err != nil {
				resultLog.WithFields(log.Fields{
					"error":     err.Error(),
					"component": "statestorage",
					"field":     field,
				}).Error("State storage error")
				return err
			}
			resultLog.WithFields(log.Fields{
				"component": "statestorage",
				"field":     field,
				"value":     value,
			}).Info("State storage operation")

		}
	}
	if ttl > 0 {
		redisConn.Send("EXPIRE", key, ttl)
		resultLog.WithFields(log.Fields{
			"component": "statestorage",
			"ttl":       ttl,
		}).Info("State storage expiration set")
	} else {
		resultLog.WithFields(log.Fields{
			"component": "statestorage",
			"ttl":       ttl,
		}).Debug("State storage expiration not set")
	}

	_, err = redisConn.Do("EXEC")
	return err
}
