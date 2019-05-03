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
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestRedisConnection(t *testing.T) {
	assert := assert.New(t)
	rb, closer := createRedis(t)
	defer closer()
	assert.NotNil(rb)
	assert.Nil(closer())
}

func TestRedisGetSet(t *testing.T) {
	assert := assert.New(t)
	rb, closer := createRedis(t)

	assert.NotNil(rb)

	assert.Nil(closer())
}

func createRedis(t *testing.T) (Service, func() error) {
	cfg := viper.New()
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("cannot create redis %s", err)
	}

	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 1000)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.maxActive", 1000)

	rs, err := newRedis(cfg)
	if err != nil {
		t.Fatalf("cannot connect to fake redis %s", err)
	}
	return rs, func() error {
		rbCloseErr := rs.Close()
		mredis.Close()
		return rbCloseErr
	}
}
