// Package ignorelist is an ignore list specific redis implementation and will be removed in a future version.
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
package ignorelist

import (
	"strconv"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
)

// Logrus structured logging setup
var (
	ilLogFields = log.Fields{
		"app":       "openmatch",
		"component": "statestorage",
		"caller":    "statestorage/redis/ignorelist/ignorelist.go",
	}
	ilLog = log.WithFields(ilLogFields)
)

// Create generates an ignorelist in redis by ZADD'ing elements with a score of
// the current time in seconds since the epoch
func Create(redisConn redis.Conn, ignorelistID string, playerIDs []string) error {

	// Logrus logging
	cmd := "ZADD"

	cmdArgs := buildElementValueList(ignorelistID, playerIDs)

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"args":  cmdArgs,
	}).Debug("Statestorage operation")

	// Run the Redis command.
	_, err := redisConn.Do(cmd, cmdArgs...)
	return err
}

// Update is an alias for Create() in this implementation
func Update(redisConn redis.Conn, ignorelistID string, playerIDs []string) error {
	return Create(redisConn, ignorelistID, playerIDs)
}

// Retrieve returns a list of playerIDs added to the ignorelist, older than the given timestamp.
// To get the full list, just send the current epoch timestamp.
func Retrieve(redisConn redis.Conn, ignorelistID string, olderThanTS int64) ([]string, error) {
	cmd := "ZRANGEBYSCORE"

	results, err := redis.Strings(redisConn.Do(cmd, ignorelistID, "-inf", strconv.Itoa(int(olderThanTS))))
	return results, err
}

// Delete removes players from the sorted set.
func Delete(redisConn redis.Conn, ignorelistID string, playerIDs string) error {

	cmd := "ZREM"

	// Make array of arguments ot the ZREM command.  First element is the key (ignorelist ID)
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, ignorelistID)

	// Add player IDs to the array.
	players := strings.Split(playerIDs, " ")
	for _, pId := range players {
		cmdArgs = append(cmdArgs, pId)
	}

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"args":  cmdArgs,
	}).Debug("Statestorage operation")

	// Run the Redis command.
	_, err := redisConn.Do(cmd, cmdArgs...)
	return err
}

// buildElemmentValueLIst builds an array of strings to send to the redis commad.
// ZADD expects values (in this case we will use epoch timestamps) followed by
// an element (we will use player IDs).
//
// Also, the redis module method requires that all the redis command
// arguments be in a single variadic output, so we prepend the ignorelist ID
// reference: https://stackoverflow.com/a/45279233/3113674
// TODO: fold this into Create if we delete everything else that uses it, no
// need for it to be it's own function in that case
func buildElementValueList(ignorelistID string, playerIDs []string) []interface{} {
	// Build an array of strings to send to the redis commad.  ZADD expects
	// values (in this case we will use epoch timestamps) followed by an
	// element (we will use player IDs).
	// Also, the redis module method requires that all the redis command
	// arguments be in a single variadic output, so prepend the ignorelist ID
	// reference: https://stackoverflow.com/a/45279233/3113674
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, ignorelistID)
	now := time.Now().Unix()
	for _, pId := range playerIDs {
		cmdArgs = append(cmdArgs, strconv.Itoa(int(now)), pId)
	}
	return cmdArgs
}
