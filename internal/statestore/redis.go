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

	"github.com/cenkalti/backoff"
	rs "github.com/go-redsync/redsync/v4"
	rsredigo "github.com/go-redsync/redsync/v4/redis/redigo"
	"github.com/golang/protobuf/proto"
	"github.com/gomodule/redigo/redis"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

const (
	allTickets        = "allTickets"
	proposedTicketIDs = "proposed_ticket_ids"

	allBackfills = "allBackfills"
)

var (
	redisLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "statestore.redis",
	})

	// this field is used to create new mutexes
	redsync *rs.Redsync
)

// NewMutex returns a new distributed mutex with given name
func (rb *redisBackend) NewMutex(key string) RedisLocker {
	//TODO: make expiry duration configurable
	m := redsync.NewMutex(fmt.Sprintf("lock/%s", key), rs.WithExpiry(5*time.Minute))
	return redisBackend{mutex: m}
}

//Lock locks r. In case it returns an error on failure, you may retry to acquire the lock by calling this method again.
func (rb redisBackend) Lock(ctx context.Context) error {
	return rb.mutex.LockContext(ctx)
}

// Unlock unlocks r and returns the status of unlock.
func (rb redisBackend) Unlock(ctx context.Context) (bool, error) {
	return rb.mutex.UnlockContext(ctx)
}

type redisBackend struct {
	healthCheckPool *redis.Pool
	redisPool       *redis.Pool
	cfg             config.View
	mutex           *rs.Mutex
}

// Close the connection to the database.
func (rb *redisBackend) Close() error {
	return rb.redisPool.Close()
}

// newRedis creates a statestore.Service backed by Redis database.
func newRedis(cfg config.View) Service {
	pool := GetRedisPool(cfg)
	redsync = rs.New(rsredigo.NewPool(pool))
	return &redisBackend{
		healthCheckPool: getHealthCheckPool(cfg),
		redisPool:       pool,
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
			if ctx != nil && ctx.Err() != nil {
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
			if ctx != nil && ctx.Err() != nil {
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
			if ctx != nil && ctx.Err() != nil {
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
			if ctx != nil && ctx.Err() != nil {
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

func handleConnectionClose(conn *redis.Conn) {
	err := (*conn).Close()
	if err != nil {
		redisLogger.WithFields(logrus.Fields{
			"error": err,
		}).Debug("failed to close redis client connection.")
	}
}

func (rb *redisBackend) newConstantBackoffStrategy() backoff.BackOff {
	backoffStrat := backoff.NewConstantBackOff(rb.cfg.GetDuration("backoff.initialInterval"))
	return backoff.BackOff(backoffStrat)
}

// TODO: add cache the backoff object
// nolint: unused
func (rb *redisBackend) newExponentialBackoffStrategy() backoff.BackOff {
	backoffStrat := backoff.NewExponentialBackOff()
	backoffStrat.InitialInterval = rb.cfg.GetDuration("backoff.initialInterval")
	backoffStrat.RandomizationFactor = rb.cfg.GetFloat64("backoff.randFactor")
	backoffStrat.Multiplier = rb.cfg.GetFloat64("backoff.multiplier")
	backoffStrat.MaxInterval = rb.cfg.GetDuration("backoff.maxInterval")
	backoffStrat.MaxElapsedTime = rb.cfg.GetDuration("backoff.maxElapsedTime")
	return backoff.BackOff(backoffStrat)
}

// CreateBackfill creates a new Backfill in the state storage. If the id already exists, it will be overwritten.
func (rb *redisBackend) CreateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIds []string) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "CreateBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := proto.Marshal(backfill)
	if err != nil {
		err = errors.Wrapf(err, "failed to marshal the backfill proto, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	_, err = redisConn.Do("SET", backfill.GetId(), value)
	if err != nil {
		err = errors.Wrapf(err, "failed to set the value for backfill, id: %s", backfill.GetId())
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// GetBackfill gets the Backfill with the specified id from state storage. This method fails if the Backfill does not exist.
func (rb *redisBackend) GetBackfill(ctx context.Context, id string) (backfill *pb.Backfill, ticketIds []string, err error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, nil, status.Errorf(codes.Unavailable, "GetBackfill, id: %s, failed to connect to redis: %v", id, err)
	}
	defer handleConnectionClose(&redisConn)

	value, err := redis.Bytes(redisConn.Do("GET", id))
	if err != nil {
		// Return NotFound if redigo did not find the ticket in storage.
		if err == redis.ErrNil {
			msg := fmt.Sprintf("Ticket id: %s not found", id)
			return nil, nil, status.Error(codes.NotFound, msg)
		}

		err = errors.Wrapf(err, "failed to get the ticket from state storage, id: %s", id)
		return nil, nil, status.Errorf(codes.Internal, "%v", err)
	}

	if value == nil {
		msg := fmt.Sprintf("Ticket id: %s not found", id)
		return nil, nil, status.Error(codes.NotFound, msg)
	}

	backfill = &pb.Backfill{}
	if err = proto.Unmarshal(value, backfill); err != nil {
		err = errors.Wrapf(err, "failed to unmarshal the backfill proto, id: %s", id)
		return nil, nil, status.Errorf(codes.Internal, "%v", err)
	}

	return backfill, nil, nil
}

// DeleteBackfill removes the Backfill with the specified id from state storage. This method succeeds if the Backfill does not exist.
func (rb *redisBackend) DeleteBackfill(ctx context.Context, id string) error {
	return nil
}

// UpdateBackfill updates an existing Backfill with a new data. Caller has to provide a custom updateFunc if this function is called not for the game server.
func (rb *redisBackend) UpdateBackfill(ctx context.Context, backfill *pb.Backfill, ticketIds []string) error {
	return nil
}

// IndexBackfill adds the backfill to the index.
func (rb *redisBackend) IndexBackfill(ctx context.Context, backfill *pb.Backfill) error {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return status.Errorf(codes.Unavailable, "IndexBackfill, id: %s, failed to connect to redis: %v", backfill.GetId(), err)
	}
	defer handleConnectionClose(&redisConn)

	err = redisConn.Send("SADD", allBackfills, backfill.Id)
	if err != nil {
		err = errors.Wrapf(err, "failed to add backfill to all backfills set, id: %s", backfill.Id)
		return status.Errorf(codes.Internal, "%v", err)
	}

	return nil
}

// DeindexBackfill removes specified Backfill from the index. The Backfill continues to exist.
func (rb *redisBackend) DeindexBackfill(ctx context.Context, id string) error {
	return nil
}

func (rb *redisBackend) GetIndexedBackfills(ctx context.Context) (map[string]struct{}, error) {
	redisConn, err := rb.redisPool.GetContext(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "GetIndexedBackfills, failed to connect to redis: %v", err)
	}
	defer handleConnectionClose(&redisConn)

	idsIndexed, err := redis.Strings(redisConn.Do("SMEMBERS", allBackfills))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "error getting all indexed backfill ids %v", err)
	}

	r := make(map[string]struct{}, len(idsIndexed))
	for _, id := range idsIndexed {
		r[id] = struct{}{}
	}

	return r, nil
}
