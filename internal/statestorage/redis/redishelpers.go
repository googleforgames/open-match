// redisHelpers is a package for wrapping redis functionality.
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
		"caller":    "statestorage/redis/redishelpers.go",
	}
	rhLog = log.WithFields(rhLogFields)
)

// ConnectionPool reads the configuration and attempts to instantiate a redis connection
// pool based on the configured hostname and port.
// TODO: needs to be reworked to use redis sentinel when we're ready to support it.
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
		rhLog.Debug("Statestorage watched record update detected")
		watchChan <- results
		close(watchChan)
	}()

	return watchChan
}

// Create is a concurrent-safe, context-aware redis SET of the input key to the
// input value
func Create(ctx context.Context, pool *redis.Pool, key string, value string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "SET"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	return redis.String(redisConn.Do("SET", key, value))
}

// Retrieve is a concurrent-safe, context-aware redis GET on the input key
func Retrieve(ctx context.Context, pool *redis.Pool, key string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "GET"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	return redis.String(redisConn.Do("GET", key))
}

// Update is a concurrent-safe, context-aware redis SADD of the input value to the
// input key's set. (Yes, it is an imperfect mapping, likely rework this at some point)
func Update(ctx context.Context, pool *redis.Pool, key string, value string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "SADD"
	rhLog.WithFields(log.Fields{"query": cmd, "value": value}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd,
			"value": value}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	_, err = redisConn.Do("SADD", key, value)
	return "", err
}

// Delete is a concurrent-safe, context-aware redis DEL on the input key
func Delete(ctx context.Context, pool *redis.Pool, key string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "DEL"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	_, err = redisConn.Do("DEL", key)
	return "", err
}

// Count is a concurrent-safe, context-aware redis SCARD on the input key
func Count(ctx context.Context, pool *redis.Pool, key string) (int, error) {
	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "SCARD"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return 0, err
	}

	// Run redis query and return
	return redis.Int(redisConn.Do("SCARD", key))
}

// Increment increments a redis value at key.
func Increment(ctx context.Context, pool *redis.Pool, key string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "INCR"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	return redis.String(redisConn.Do("INCR", key))
}

// Decrement decrements a redis value at key.
func Decrement(ctx context.Context, pool *redis.Pool, key string) (string, error) {

	// Add the key as a field to all logs for the execution of this function.
	rhLog = rhLog.WithFields(log.Fields{"key": key})

	cmd := "DECR"
	rhLog.WithFields(log.Fields{"query": cmd}).Debug("Statestorage operation")

	// Get a connection to redis
	redisConn, err := pool.GetContext(ctx)
	defer redisConn.Close()

	// Encountered an issue getting a connection from the pool.
	if err != nil {
		rhLog.WithFields(log.Fields{
			"error": err.Error(),
			"query": cmd}).Error("Statestorage connection error")
		return "", err
	}

	// Run redis query and return
	return redis.String(redisConn.Do("DECR", key))
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
