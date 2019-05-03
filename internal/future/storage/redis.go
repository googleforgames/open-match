// Copyright 2019 Google LLC
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

package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "storage.redis",
	})
)

type redisBackend struct {
	redisPool *redis.Pool
}

func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

func (rb *redisBackend) GetDeprecatedRedisPool() *redis.Pool {
	return rb.redisPool
}

func (rb *redisBackend) Put(ctx context.Context, key string, values map[string]string) (string, error) {
	return rb.tx(ctx, "HSET", key, values)
}

func (rb *redisBackend) Get(ctx context.Context, key string) (string, error) {
	return rb.tx(ctx, "GET", key, map[string]string{})
}

func (rb *redisBackend) tx(ctx context.Context, command string, key string, values map[string]string) (string, error) {
	redisConn, err := rb.getConnection(ctx)
	if err != nil {
		return "", err
	}
	defer redisConn.Close()
	redisLogger.WithFields(logrus.Fields{
		"command": command,
		"key":     key,
		"values":  values,
	}).Debug("Redis transaction")

	cmdArgs := make([]interface{}, 0)
	cmdArgs = append(cmdArgs, key)
	for field, value := range values {
		cmdArgs = append(cmdArgs, field, value)
	}

	return redis.String(redisConn.Do(command, cmdArgs...))
}

func (rb *redisBackend) getConnection(ctx context.Context) (redis.Conn, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.New(codes.Unavailable, err.Error()).Err()
	}
	return redisConn, nil
}

// NewRedis creates a storage.Service backed by Redis database.
func newRedis(cfg config.View) (Service, error) {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"
	maskedURL := redisURL
	if cfg.IsSet("redis.password") && cfg.GetString("redis.password") != "" {
		redisURL += cfg.GetString("redis.user") + ":" + cfg.GetString("redis.password") + "@"
		maskedURL += cfg.GetString("redis.user") + ":******@"
	}
	redisURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")
	maskedURL += cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")

	redisLogger.WithField("redisURL", maskedURL).Debug("Attempting to connect to Redis")
	pool := &redis.Pool{
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
		redisLogger.WithFields(logrus.Fields{
			"error": err.Error(),
			"query": "SELECT 0"}).Error("state storage connection error")
		return nil, fmt.Errorf("cannot connect to Redis at %s, %s", maskedURL, err)
	}

	redisLogger.Info("Connected to Redis")
	return &redisBackend{
		redisPool: pool,
	}, nil
}
