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
	"time"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"open-match.dev/open-match/internal/config"
)

// NewRedisForTesting create a new in-memory miniredis for testing
func NewRedisForTesting(cfg *viper.Viper) (config.View, func(), error) {
	mredis, err := miniredis.Run()
	if err != nil {
		return nil, nil, err
	}

	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 1000)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.maxActive", 1000)
	cfg.Set("redis.expiration", 42000)
	cfg.Set("playerIndices", []string{"testindex1", "testindex2"})

	return cfg, func() { mredis.Close() }, err
}
