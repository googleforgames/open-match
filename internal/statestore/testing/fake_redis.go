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
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/consts"
	"open-match.dev/open-match/internal/statestore"
)

// New creates a new in memory Redis instance for testing.
func New(t *testing.T, cfg config.Mutable) func() {
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to create miniredis, %v", err)
	}
	cfg.Set(consts.RedisHostName, mredis.Host())
	cfg.Set(consts.RedisPort, mredis.Port())
	cfg.Set(consts.RedisConnMaxIdle, PoolMaxIdle)
	cfg.Set(consts.RedisConnMaxActive, PoolMaxActive)
	cfg.Set(consts.RedisConnIdleTimeout, PoolIdleTimeout)
	cfg.Set(consts.RedisConnHealthCheckTimeout, PoolHealthCheckTimeout)
	cfg.Set(consts.RedisIgnoreListTimeToLive, IgnoreListTTL)
	cfg.Set(consts.BackoffInitInterval, InitialInterval)
	cfg.Set(consts.BackoffRandFactor, RandFactor)
	cfg.Set(consts.BackoffMultiplier, Multiplier)
	cfg.Set(consts.BackoffMaxInterval, MaxInterval)
	cfg.Set(consts.BackoffMaxElapsedTime, MaxElapsedTime)

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
