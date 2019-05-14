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
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
)

func TestStatestoreSetup(t *testing.T) {
	assert := assert.New(t)
	cfg := createBackend(t)
	service, err := New(cfg)
	assert.Nil(err)
	assert.NotNil(service)
	defer service.Close()
}

func TestTicketLifecycle(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg := createBackend(t)
	service, err := New(cfg)
	assert.Nil(err)
	assert.NotNil(service)
	defer service.Close()

	// Initialize test data
	id := xid.New().String()
	ticket := &pb.Ticket{
		Id:         id,
		Properties: "test-property",
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Validate that GetTicket fails for a Ticket that does not exist.
	_, err = service.GetTicket(context.Background(), id)
	assert.NotNil(err)

	// Validate nonexisting Ticket deletion
	err = service.DeleteTicket(context.Background(), id)
	assert.Nil(err)

	// Validate nonexisting Ticket deindexing
	err = service.DeindexTicket(context.Background(), id, []string{"test1", "test2"})
	assert.Nil(err)

	// Validate Ticket creation
	err = service.CreateTicket(context.Background(), ticket)
	assert.Nil(err)

	// Validate Ticket retrival
	result, err := service.GetTicket(context.Background(), ticket.Id)
	assert.Nil(err)
	assert.NotNil(result)
	assert.Equal(ticket.Id, result.Id)
	assert.Equal(ticket.Properties, result.Properties)
	assert.Equal(ticket.Assignment.Connection, result.Assignment.Connection)

	// Validate Ticket deletion
	err = service.DeleteTicket(context.Background(), id)
	assert.Nil(err)

	_, err = service.GetTicket(context.Background(), id)
	assert.NotNil(err)
}

func TestTicketIndexing(t *testing.T) {
	// TODO: The change to filter tickets is currently in progress.
	// Testing Ticket indexing will be added along with this implementation.
}

func createBackend(t *testing.T) config.View {
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

	return cfg
}
