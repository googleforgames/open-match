// Package redisHelpers is a package for wrapping redis functionality.
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
package redisHelpers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// Logrus structured logging setup
// Note: nearly every log in this package under normal operation should be at
//  the Debug logging level as it can get very spammy.
var (
	rhLogFields = log.Fields{
		"app":       "openmatch",
		"component": "redishelpers",
	}
	rhLog = log.WithFields(rhLogFields)
)

// ConnectionPool reads the configuration and attempts to instantiate a redis connection
// pool based on the configured hostname and port.
func ConnectionPool(cfg *viper.Viper) *redis.Pool {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"
	if cfg.IsSet("redis.user") && cfg.GetString("redis.user") != "" &&
		cfg.IsSet("redis.password") && cfg.GetString("redis.password") != "" {
		redisURL += cfg.GetString("redis.user") + ":" + cfg.GetString("redis.password") + "@"
	}
	redisURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")

	rhLog.WithFields(log.Fields{"redisURL": redisURL}).Debug("Attempting to connect to Redis")
	pool := redis.Pool{
		MaxIdle:     cfg.GetInt("redis.pool.maxIdle"),
		MaxActive:   cfg.GetInt("redis.pool.maxActive"),
		IdleTimeout: cfg.GetDuration("redis.pool.idleTimeout") * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}

	// Sanity check that connection works before passing it back.  Redigo
	// always returns a valid connection, and will just fail on the first
	// query: https://godoc.org/github.com/gomodule/redigo/redis#Pool.Get
	redisConn := pool.Get()
	defer redisConn.Close()
	_, err := redisConn.Do("SELECT", "0")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": "SELECT 0"}).Error("state storage connection error")
		return nil
	}

	rhLog.Info("Connected to Redis")
	return &pool
}

// Watcher makes a channel and returns it immediately.  It also launches an
// asynchronous goroutine that watches a redis key and returns the value of
// that key once it exists on the channel.
//
// The pattern for this function is from 'Go Concurrency Patterns', it is a function
// that wraps a closure goroutine, and returns a channel.
// reference: https://talks.golang.org/2012/concurrency.slide#25
func Watcher(ctx context.Context, pool *redis.Pool, key string) <-chan string {
	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})
	rhLog.Debug("Watching key in statestorage for changes")

	watchChan := make(chan string)

	go func() {
		// var declaration
		var results string
		var err = errors.New("haven't queried Redis yet")

		// Loop, querying redis until this key has a value
		for err != nil {
			select {
			case <-ctx.Done():
				// Cleanup
				close(watchChan)
				return
			default:
				results, err = Retrieve(ctx, pool, key)
				if err != nil {
					time.Sleep(5 * time.Second) // TODO: exp bo + jitter
				}
			}
		}
		// Return value retreived from Redis asynchonously and tell calling function we're done
		rhLog.Debug("state storage watched record update detected")
		watchChan <- results
		close(watchChan)
	}()

	return watchChan
}

// Create is a concurrent-safe, context-aware redis SET of the input key to the
// input value
func Create(ctx context.Context, pool *redis.Pool, key string, values map[string]string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "HSET"

	cLog := rhLog.WithFields(log.Fields{
		"query":  cmd,
		"key":    key,
		"values": values,
	})

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		cLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return "", err
	}

	// Build command arguments.
	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, key)
	for field, value := range values {
		cmdArgs = append(cmdArgs, field, value)
	}

	// Run redis query and return
	cLog.Debug("state storage operation")
	return redis.String(redisConn.Do(cmd, cmdArgs...))
}

// Retrieve is a concurrent-safe, context-aware redis GET on the input key
func Retrieve(ctx context.Context, pool *redis.Pool, key string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "GET"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("state storage connection error")
		return "", err
	}

	// Run redis query and return
	return redis.String(redisConn.Do(cmd, key))
}

// RetrieveField is a concurrent-safe, context-aware redis HGET on the input field of the input key
func RetrieveField(ctx context.Context, pool *redis.Pool, key string, field string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "HGET"

	cLog := rhLog.WithFields(log.Fields{
		"query": cmd,
		"key":   key,
		"field": field,
	})

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		cLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return "", err
	}

	// Run redis query and return
	cLog.Debug("state storage operation")
	return redis.String(redisConn.Do(cmd, key, field))
}

// RetrieveAll is a concurrent-safe, context-aware redis HGETALL on the input key
func RetrieveAll(ctx context.Context, pool *redis.Pool, key string) (map[string]string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "HGETALL"

	cLog := rhLog.WithFields(log.Fields{
		"query": cmd,
		"key":   key,
	})

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		cLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return nil, err
	}

	// Run redis query and return
	cLog.Debug("state storage operation")
	return redis.StringMap(redisConn.Do(cmd, key))

}

// Update is a concurrent-safe, context-aware redis SADD of the input value to the
// input key's set. (Yes, it is an imperfect mapping, likely rework this at some point)
func Update(ctx context.Context, pool *redis.Pool, key string, value string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "SADD"
	rhLog.WithFields(log.Fields{"query": cmd, "value": value}).Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd,
			"value": value}).Error("state storage connection error")
		return "", err
	}

	// Run redis query and return
	_, err = redisConn.Do("SADD", key, value)
	return "", err
}

// UpdateMultiFields is a concurrent-safe, context-aware Redis HSET of the input field
// Keys to update and the values to set the field to are passed in the 'kv' map.
// Example usage is to set multiple player's "assignment" field to various game server connection strings.
// "field" := "assignment"
// "kv" := map[string]string{ "player1": "servername:10000", "player2": "otherservername:10002" }
func UpdateMultiFields(ctx context.Context, pool *redis.Pool, kv map[string]string, field string) error {

	// Add the cmd & field to all logs for the execution of this function.
	cmd := "HSET"
	dfLog := rhLog.WithFields(log.Fields{"field": field, "query": cmd})

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		dfLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return err
	}

	// Run redis query and return
	redisConn.Send("MULTI")
	for key, value := range kv {
		dfLog.WithFields(log.Fields{"key": key, "value": value}).Debug("state storage operation")
		redisConn.Send(cmd, key, field, value)
	}
	_, err = redisConn.Do("EXEC")
	return err
}

// Delete is a concurrent-safe, context-aware redis DEL on the input key
func Delete(ctx context.Context, pool *redis.Pool, key string) error {

	// Add the key as a field to all logs for the execution of this function.
	cmd := "DEL"
	dLog := rhLog.WithFields(log.Fields{"key": key, "query": cmd})
	dLog.Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		dLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return err
	}

	// Run redis query and return
	_, err = redisConn.Do(cmd, key)
	return err
}

// DeleteMultiFields is a concurrent-safe, context-aware Redis DEL of the input field
// from the input keys
func DeleteMultiFields(ctx context.Context, pool *redis.Pool, keys []string, field string) error {

	// Add the cmd & field to all logs for the execution of this function.
	cmd := "HDEL"
	dfLog := rhLog.WithFields(log.Fields{"field": field, "query": cmd})

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		dfLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("state storage connection error")
		return err
	}

	// Run redis query and return
	redisConn.Send("MULTI")
	for _, key := range keys {
		dfLog.WithFields(log.Fields{"key": key}).Debug("state storage operation")
		redisConn.Send(cmd, key, field)
	}
	_, err = redisConn.Do("EXEC")
	return err
}

// Count is a concurrent-safe, context-aware redis SCARD on the input key
func Count(ctx context.Context, pool *redis.Pool, key string) (int, error) {
	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "SCARD"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("state storage connection error")
		return 0, err
	}

	// Run redis query and return
	return redis.Int(redisConn.Do("SCARD", key))
}

// Increment increments a redis value at key.
func Increment(ctx context.Context, pool *redis.Pool, key string) (interface{}, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "INCR"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("state storage connection error")
		return "", err
	}

	// Run redis query and return
	return redisConn.Do("INCR", key)
}

// Decrement decrements a redis value at key.
func Decrement(ctx context.Context, pool *redis.Pool, key string) (interface{}, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "DECR"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("state storage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("state storage connection error")
		return "", err
	}

	// Run redis query and return
	return redisConn.Do("DECR", key)
}

// JSONStringToMap converts a JSON blob (which is how we store many things in
// redis) to a golang map so the individual properties can be accessed.
// Useful helper function when debugging.
func JSONStringToMap(result string) map[string]interface{} {
	jsonPD := make(map[string]interface{})
	byt := []byte(result)
	err := json.Unmarshal(byt, &jsonPD)
	if err != nil {
		fmt.Println(err)
	}
	return jsonPD
}
