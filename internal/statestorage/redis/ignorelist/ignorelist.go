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
	"fmt"
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	}).Debug("state storage operation")

	// Run the Redis command.
	_, err := redisConn.Do(cmd, cmdArgs...)
	return err
}

// SendAdd is identical to Add only does a redigo 'Send' as part of a MULTI command.
func SendAdd(redisConn redis.Conn, ignorelistID string, playerIDs []string) {

	// Logrus logging
	cmd := "ZADD"

	cmdArgs := buildElementValueList(ignorelistID, playerIDs)

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"args":  cmdArgs,
	}).Debug("state storage transaction operation")

	// Run the Redis command.
	redisConn.Send(cmd, cmdArgs...)
}

// Add is an alias for Create() in this implementation
func Add(redisConn redis.Conn, ignorelistID string, playerIDs []string) error {
	return Create(redisConn, ignorelistID, playerIDs)
}

// Remove deletes playerIDs from an ignorelist.
func Remove(redisConn redis.Conn, ignorelistID string, playerIDs []string) error {

	// Logrus logging
	cmd := "ZREM"

	cmdArgs := buildElementValueList(ignorelistID, playerIDs)

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"args":  cmdArgs,
	}).Debug("state storage operation")

	// Run the Redis command.
	_, err := redisConn.Do(cmd, cmdArgs...)
	return err
}

// SendRemove is identical to Remove only does a redigo 'Send' as part of a MULTI command.
func SendRemove(redisConn redis.Conn, ignorelistID string, playerIDs []string) {

	// Logrus logging
	cmd := "ZREM"

	cmdArgs := buildElementValueList(ignorelistID, playerIDs)

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"args":  cmdArgs,
	}).Debug("state storage transaction operation")

	// Run the Redis command.
	redisConn.Send(cmd, cmdArgs...)
}

// Retrieve returns a list of playerIDs in the ignorelist
func Retrieve(redisConn redis.Conn, cfg *viper.Viper, il string) ([]string, error) {

	// String var init
	cmd := "ZRANGEBYSCORE"
	expire := fmt.Sprintf("ignoreLists.%v.expireAfter", il)
	offset := fmt.Sprintf("ignoreLists.%v.ignoreAfter", il)

	// Timestamp var init
	now := time.Now().Unix()
	minTS := int64(0)
	maxTS := now

	// Ignorelists are sorted sets with a 'score' that is the epoch timestamp
	// (in seconds) of when a player was added. Ignorelist entries are are valid
	// if the timestamp is (now - 'ignoreAfter'), and are expired/invalid if the
	// timestamp is (now - 'expireAfter'), so use those as min/max scores in the
	// sorted set query.
	if cfg.IsSet(expire) {
		minTS = now - cfg.GetInt64(expire)
	}
	if cfg.IsSet(offset) {
		maxTS = maxTS - cfg.GetInt64(offset)
	}

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"minv":  minTS,
		"maxv":  maxTS,
	}).Debug("state storage transaction operation")

	results, err := redis.Strings(redisConn.Do(cmd, il, maxTS, minTS))

	return results, err
}

func retrieve(redisConn redis.Conn, ignorelistID string, from int64, until int64) ([]string, error) {
	cmd := "ZRANGEBYSCORE"

	// ignorelists are sorted sets with scores set to epoch timestamps (in seconds)
	results, err := redis.Strings(redisConn.Do(cmd, ignorelistID, from, until))
	return results, err
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
