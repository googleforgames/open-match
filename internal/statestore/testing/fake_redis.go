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

	"fmt"
	"github.com/alicebob/miniredis"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"
)

// New creates a new in memory Redis instance for testing.
func New(t *testing.T, cfg config.Mutable) (statestore.Service, func()) {
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to create miniredis, %v", err)
	}
	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 10)
	cfg.Set("redis.pool.maxActive", 10)
	cfg.Set("redis.pool.idleTimeout", "10s")

	redisURL := fmt.Sprintf("redis://%s:%s", mredis.Host(), mredis.Port())

	redis, err := statestore.NewRedis(cfg, redisURL, redisURL)
	if err != nil {
		t.Fatalf("error connecting to fake redis, %s, %v", redisURL, err)
	}
	return redis, func() {
		mredis.Close()
	}
}
