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

package statestore

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.redis",
	})
)

type redisBackend struct {
	healthCheckPool *redis.Pool
	redisPool       *redis.Pool
	cfg             config.View
}

// Close the connection to the database.
func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) Service {
	return &redisBackend{
		healthCheckPool: getHealthCheckPool(cfg),
		redisPool:       GetRedisPool(cfg),
		cfg:             cfg,
	}
}

func getHealthCheckPool(cfg config.View) *redis.Pool {
	var healthCheckURL string
	var maxIdle = 3
	var maxActive = 0
	var healthCheckTimeout = cfg.GetDuration("redis.pool.healthCheckTimeout")

	if cfg.IsSet("redis.sentinelHostname") {
		sentinelAddr := getSentinelAddr(cfg)
		healthCheckURL = redisURLFromAddr(sentinelAddr, cfg, cfg.GetBool("redis.sentinelUsePassword"))
	} else {
		masterAddr := getMasterAddr(cfg)
		healthCheckURL = redisURLFromAddr(masterAddr, cfg, cfg.GetBool("redis.usePassword"))
	}

	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  10 * healthCheckTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(healthCheckURL, redis.DialConnectTimeout(healthCheckTimeout), redis.DialReadTimeout(healthCheckTimeout))
		},
	}
}

// GetRedisPool configures a new pool to connect to redis given the config.
func GetRedisPool(cfg config.View) *redis.Pool {
	var dialFunc func(context.Context) (redis.Conn, error)
	maxIdle := cfg.GetInt("redis.pool.maxIdle")
	maxActive := cfg.GetInt("redis.pool.maxActive")
	idleTimeout := cfg.GetDuration("redis.pool.idleTimeout")

	if cfg.IsSet("redis.sentinelHostname") {
		sentinelPool := getSentinelPool(cfg)
		dialFunc = func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			sentinelConn, err := sentinelPool.GetContext(ctx)
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("failed to connect to redis sentinel")
				return nil, status.Errorf(codes.Unavailable, "%v", err)
			}

			masterInfo, err := redis.Strings(sentinelConn.Do("SENTINEL", "GET-MASTER-ADDR-BY-NAME", cfg.GetString("redis.sentinelMaster")))
			if err != nil {
				redisLogger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("failed to get current master from redis sentinel")
				return nil, status.Errorf(codes.Unavailable, "%v", err)
			}

			masterURL := redisURLFromAddr(fmt.Sprintf("%s:%s", masterInfo[0], masterInfo[1]), cfg, cfg.GetBool("redis.usePassword"))
			return redis.DialURL(masterURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		}
	} else {
		masterAddr := getMasterAddr(cfg)
		masterURL := redisURLFromAddr(masterAddr, cfg, cfg.GetBool("redis.usePassword"))
		dialFunc = func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return redis.DialURL(masterURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		}
	}

	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  idleTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext:  dialFunc,
	}
}

func getSentinelPool(cfg config.View) *redis.Pool {
	maxIdle := cfg.GetInt("redis.pool.maxIdle")
	maxActive := cfg.GetInt("redis.pool.maxActive")
	idleTimeout := cfg.GetDuration("redis.pool.idleTimeout")

	sentinelAddr := getSentinelAddr(cfg)
	sentinelURL := redisURLFromAddr(sentinelAddr, cfg, cfg.GetBool("redis.sentinelUsePassword"))
	return &redis.Pool{
		MaxIdle:      maxIdle,
		MaxActive:    maxActive,
		IdleTimeout:  idleTimeout,
		Wait:         true,
		TestOnBorrow: testOnBorrow,
		DialContext: func(ctx context.Context) (redis.Conn, error) {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			redisLogger.WithField("sentinelAddr", sentinelAddr).Debug("Attempting to connect to Redis Sentinel")
			return redis.DialURL(sentinelURL, redis.DialConnectTimeout(idleTimeout), redis.DialReadTimeout(idleTimeout))
		},
	}
}

// HealthCheck indicates if the database is reachable.
func (rb *redisBackend) HealthCheck(ctx context.Context) error {
	redisConn, err := rb.healthCheckPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	defer handleConnectionClose(&redisConn)

	_, err = redisConn.Do("PING")
	// Encountered an issue getting a connection from the pool.
	if err != nil {
		return status.Errorf(codes.Unavailable, "%v", err)
	}
	return nil
}

func testOnBorrow(c redis.Conn, lastUsed time.Time) error {
	// Assume the connection is valid if it was used in 30 sec.
	if time.Since(lastUsed) < 15*time.Second {
		return nil
	}

	_, err := c.Do("PING")
	return err
}

func getSentinelAddr(cfg config.View) string {
	return fmt.Sprintf("%s:%s", cfg.GetString("redis.sentinelHostname"), cfg.GetString("redis.sentinelPort"))
}

func getMasterAddr(cfg config.View) string {
	return fmt.Sprintf("%s:%s", cfg.GetString("redis.hostname"), cfg.GetString("redis.port"))
}

func redisURLFromAddr(addr string, cfg config.View, usePassword bool) string {
	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz

	// Add redis user and password to connection url if they exist
	redisURL := "redis://"

	if usePassword {
		passwordFile := cfg.GetString("redis.passwordPath")
		redisLogger.Debugf("loading Redis password from file %s", passwordFile)
		passwordData, err := ioutil.ReadFile(passwordFile)
		if err != nil {
			redisLogger.Fatalf("cannot read Redis password from file %s, desc: %s", passwordFile, err.Error())
		}
		redisURL += fmt.Sprintf("%s:%s@", cfg.GetString("redis.user"), string(passwordData))
	}

	return redisURL + addr
}
