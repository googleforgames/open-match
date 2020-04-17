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

package testing

import (
	"testing"
	"time"

	"github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
)

// New creates a new in memory Redis instance with Sentinel for testing.
func New(t *testing.T, cfg config.Mutable) func() {
	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}

	s := minisentinel.NewSentinel(mredis)
	err = s.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start minisentinel, %v", err)
	}

	cfg.Set("redis.sentinelEnabled", true)
	cfg.Set("redis.sentinelHostname", s.Host())
	cfg.Set("redis.sentinelPort", s.Port())
	cfg.Set("redis.sentinelMaster", s.MasterInfo().Name)
	cfg.Set("redis.pool.maxIdle", PoolMaxIdle)
	cfg.Set("redis.pool.maxActive", PoolMaxActive)
	cfg.Set("redis.pool.idleTimeout", PoolIdleTimeout)
	cfg.Set("redis.pool.healthCheckTimeout", PoolHealthCheckTimeout)
	cfg.Set("storage.ignoreListTTL", IgnoreListTTL)
	cfg.Set("backoff.initialInterval", InitialInterval)
	cfg.Set("backoff.randFactor", RandFactor)
	cfg.Set("backoff.multiplier", Multiplier)
	cfg.Set("backoff.maxInterval", MaxInterval)
	cfg.Set("backoff.maxElapsedTime", MaxElapsedTime)

	return func() {
		s.Close()
		mredis.Close()
	}
}

// NewStoreServiceForTesting creates a new statestore service for testing
func NewStoreServiceForTesting(t *testing.T, cfg config.Mutable) (statestore.Service, func()) {
	closer := New(t, cfg)
	s := statestore.New(cfg)

	return s, closer
}

const (
	// PoolMaxIdle is the number of maximum idle redis connections in a redis pool
	PoolMaxIdle = 5
	// PoolMaxActive is the number of maximum active redis connections in a redis pool
	PoolMaxActive = 5
	// PoolIdleTimeout is the idle duration allowance of a redis connection a a redis pool
	PoolIdleTimeout = 10 * time.Second
	// PoolHealthCheckTimeout is the read/write timeout of a healthcheck HTTP request
	PoolHealthCheckTimeout = 100 * time.Millisecond
	// IgnoreListTTL is the time to live duration of Open Match ignore list settings
	IgnoreListTTL = 500 * time.Millisecond
	// InitialInterval is the initial backoff time of a backoff strategy
	InitialInterval = 30 * time.Millisecond
	// RandFactor is the randomization factor of a backoff strategy
	RandFactor = .5
	// Multiplier is the multuplier used in a exponential backoff strategy
	Multiplier = .5
	// MaxInterval is the maximum retry interval of a backoff strategy
	MaxInterval = 300 * time.Millisecond
	// MaxElapsedTime is the maximum total retry time of a backoff stragegy
	MaxElapsedTime = 1000 * time.Millisecond
)
