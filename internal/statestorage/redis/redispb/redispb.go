// Package redispb marshals and unmarshals protobuf messages for redis state storage.
//  More details about the protobuf messages used in Open Match can be found in the api/protobuf-spec/om_messages.proto file.
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

Ignorelists are modeled in redis as sorted sets.  Participant IDs are the
elements, and the values are the epoch timestamp in seconds of when the element
was added to the list.
*/
package redispb

import (
	"context"
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
	rpLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
		"caller":    "internal/statestorage/redis/redispb/redispb.go",
	}
	rpLog = log.WithFields(rpLogFields)
)

// MarshalToRedis marshals a protobuf message to a redis hash.
func MarshalToRedis(pb proto.Message, pool *redis.Pool) (err error) {

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
		return err
	}

	// Get redis key and command ready.
	key := gjson.Get(jsonMsg, "id").String()
	cmd := "HSET"
	mrLog := rpLog.WithFields(log.Fields{
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
	}
	redisConn.Send("MULTI")

	// Write all non-id fields from the protobuf message to state storage.
	// Use reflection to get the field names from the protobuf message.
	pbInfo := reflect.ValueOf(pb).Elem()
	for i := 0; i < pbInfo.NumField(); i++ {
		field := strings.ToLower(pbInfo.Type().Field(i).Name)
		value := gjson.Get(jsonMsg, field)
		if field != "id" {
			// This isn't the ID field, so write it to the redis hash.
			redisConn.Send(cmd, key, field, value)
			if err != nil {
				mrLog.WithFields(log.Fields{
					"error":     err.Error(),
					"component": "statestorage",
					"field":     field,
				}).Error("State storage error")
				return
			}
			mrLog.WithFields(log.Fields{
				"component": "statestorage",
				"field":     field,
				"value":     value,
			}).Info("State storage operation")

		}
	}
	_, err = redisConn.Do("EXEC")
	return
}
func UnmarshalFromRedis(redisConn redis.Conn, pb proto.Message) error {
	return nil
}
