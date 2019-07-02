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

	miniredis "github.com/alicebob/miniredis/v2"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
)

// New creates a new in memory Redis instance for testing.
func New(t *testing.T, cfg config.Mutable) func() {
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to create miniredis, %v", err)
	}
	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 10)
	cfg.Set("redis.pool.maxActive", 10)
	cfg.Set("redis.pool.idleTimeout", "10s")
	cfg.Set("redis.pool.healthCheckTimeout", "100ms")
	cfg.Set("redis.ignoreLists.ttl", "10000ms")
	cfg.Set("backoff.initialInterval", 30*time.Millisecond)
	cfg.Set("backoff.randFactor", 0.5)
	cfg.Set("backoff.multiplier", 0.5)
	cfg.Set("backoff.maxInterval", 300*time.Millisecond)
	cfg.Set("backoff.maxElapsedTime", 1000*time.Millisecond)

	return func() {
		mredis.Close()
	}
}

// NewStoreServiceForTesting creates a new statestore service for testing
func NewStoreServiceForTesting(t *testing.T, cfg config.Mutable) (statestore.Service, func()) {
	closer := New(t, cfg)
	s := statestore.New(cfg)

	return s, closer
}
