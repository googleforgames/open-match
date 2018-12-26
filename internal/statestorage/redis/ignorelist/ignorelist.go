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
	"context"
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

// Move moves a list of players from one ignorelist to another.
// TODO: Make cancellable with context
func Move(ctx context.Context, pool *redis.Pool, playerIDs []string, src string, dest string) error {

	// Get redis connection
	redisConn := pool.Get()
	defer redisConn.Close()

	// Setup default logging
	ilLog.WithFields(log.Fields{
		"src":        src,
		"dest":       dest,
		"numPlayers": len(playerIDs),
	}).Debug("moving players to a different ignorelist")

	redisConn.Send("MULTI")
	SendAdd(redisConn, dest, playerIDs)
	SendRemove(redisConn, src, playerIDs)
	_, err := redisConn.Do("EXEC")
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
// 'cfg' is a viper.Sub sub-tree of the config file with just the parameters for
// this ignorelist.
func Retrieve(redisConn redis.Conn, cfg *viper.Viper, il string) ([]string, error) {

	// String var init
	cmd := "ZRANGEBYSCORE"

	ilName := il
	if cfg.IsSet("name") && cfg.GetString("name") != "" {
		ilName = cfg.GetString("name")
	}

	// Timestamp var init
	now := time.Now().Unix()
	var minTS, maxTS int64

	// Default to all players on the list
	maxTS = now
	minTS = 0

	// Ignorelists are sorted sets with a 'score' that is the epoch timestamp
	// (in seconds) of when a player was added.
	// examples: Assume current timestamp is 1000001000
	//  duration is 800 and offset is 0:
	// Ignore all players in the list with timestamps between 1000000200 - 1000001000
	//  duration is 0 and offset is 500:
	// Ignore players in the list with timestamps between 0 - 1000000500
	if cfg.IsSet("offset") && cfg.GetInt64("offset") > 0 {
		maxTS = now - cfg.GetInt64("offset")
	}
	if cfg.IsSet("duration") && cfg.GetInt64("duration") > 0 {
		minTS = maxTS - cfg.GetInt64("duration")
	}

	ilLog.WithFields(log.Fields{
		"query": cmd,
		"key":   ilName,
		"minv":  minTS,
		"maxv":  maxTS,
	}).Debug("state storage transaction operation")

	results, err := redis.Strings(redisConn.Do(cmd, ilName, minTS, maxTS))

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
