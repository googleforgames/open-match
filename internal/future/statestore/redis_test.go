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
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/future/pb"
	testUtil "open-match.dev/open-match/internal/future/testing"
)

func TestRedisConnection(t *testing.T) {
	assert := assert.New(t)
	rb, closer := createRedis(t)
	defer closer()
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

func TestFilterTickets(t *testing.T) {
	assert := assert.New(t)
	cfg := viper.New()
	cfg.Set("storage.page.size", 1000)

	rs, rsCloser := createRedis(t)
	defer rsCloser()

	// Inject test data into the fake redis server
	ctx := context.Background()
	rb, ok := rs.(*redisBackend)
	assert.True(ok)
	redisConn, err := rb.redisPool.GetContext(ctx)
	assert.Nil(err)

	redisConn.Do("ZADD", "level",
		1, "alice",
		10, "bob",
		20, "charlie",
		30, "donald",
		40, "eddy",
	)
	redisConn.Do("ZADD", "attack",
		1, "alice",
		10, "bob",
		20, "charlie",
		30, "donald",
		40, "eddy",
	)
	redisConn.Do("ZADD", "defense",
		1, "alice",
		10, "bob",
		20, "charlie",
		30, "donald",
		40, "eddy",
	)

	ticketsData, err := rs.FilterTickets(ctx, []*pb.Filter{
		testUtil.NewPbFilter("level", 0, 15),
		testUtil.NewPbFilter("attack", 5, 25),
	})

	assert.Nil(err)
	assert.Equal(map[string]map[string]int64{"bob": map[string]int64{"attack": 10, "level": 10}}, ticketsData)
}
