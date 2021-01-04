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
	cfg.Set("backfillLockTimeout", "1m")
	cfg.Set("pendingReleaseTimeout", pendingReleaseTimeout)
	cfg.Set("assignedDeleteTimeout", assignedDeleteTimeout)
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
