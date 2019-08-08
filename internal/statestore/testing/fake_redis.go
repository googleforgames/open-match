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

	miniredis "github.com/alicebob/miniredis/v2"
	"open-match.dev/open-match/internal"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
)

// New creates a new in memory Redis instance for testing.
func New(t *testing.T, cfg config.Mutable) func() {
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to create miniredis, %v", err)
	}
	cfg.Set(internal.RedisHostName, mredis.Host())
	cfg.Set(internal.RedisPort, mredis.Port())
	cfg.Set(internal.RedisConnMaxIdle, PoolMaxIdle)
	cfg.Set(internal.RedisConnMaxActive, PoolMaxActive)
	cfg.Set(internal.RedisConnIdleTimeout, PoolIdleTimeout)
	cfg.Set(internal.RedisConnHealthCheckTimeout, PoolHealthCheckTimeout)
	cfg.Set(internal.RedisIgnoreListTimeToLive, IgnoreListTTL)
	cfg.Set(internal.BackoffInitInterval, InitialInterval)
	cfg.Set(internal.BackoffRandFactor, RandFactor)
	cfg.Set(internal.BackoffMultiplier, Multiplier)
	cfg.Set(internal.BackoffMaxInterval, MaxInterval)
	cfg.Set(internal.BackoffMaxElapsedTime, MaxElapsedTime)

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
