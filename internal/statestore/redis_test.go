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
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/rs/xid"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/telemetry"
	internalTesting "open-match.dev/open-match/internal/testing"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestStatestoreSetup(t *testing.T) {
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
}

func TestTicketLifecycle(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()

	ctx := utilTesting.NewContext(t)

	// Initialize test data
	id := xid.New().String()
	ticket := &pb.Ticket{
		Id: id,
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"testindex1": 42,
			},
		},
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Validate that GetTicket fails for a Ticket that does not exist.
	_, err := service.GetTicket(ctx, id)
	assert.NotNil(err)
	assert.Equal(status.Code(err), codes.NotFound)

	// Validate nonexisting Ticket deletion
	err = service.DeleteTicket(ctx, id)
	assert.Nil(err)

	// Validate nonexisting Ticket deindexing
	err = service.DeindexTicket(ctx, id)
	assert.Nil(err)

	// Validate Ticket creation
	err = service.CreateTicket(ctx, ticket)
	assert.Nil(err)

	// Validate Ticket retrival
	result, err := service.GetTicket(ctx, ticket.Id)
	assert.Nil(err)
	assert.NotNil(result)
	assert.Equal(ticket.Id, result.Id)
	assert.Equal(ticket.SearchFields.DoubleArgs["testindex1"], result.SearchFields.DoubleArgs["testindex1"])
	assert.Equal(ticket.Assignment.Connection, result.Assignment.Connection)

	// Validate Ticket deletion
	err = service.DeleteTicket(ctx, id)
	assert.Nil(err)

	_, err = service.GetTicket(ctx, id)
	assert.NotNil(err)
}

func TestIgnoreLists(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets := internalTesting.GenerateFloatRangeTickets(
		internalTesting.Property{Name: "testindex1", Min: 0, Max: 10, Interval: 2},
		internalTesting.Property{Name: "testindex2", Min: 0, Max: 10, Interval: 2},
	)

	ticketIds := []string{}
	for _, ticket := range tickets {
		assert.Nil(service.CreateTicket(ctx, ticket))
		assert.Nil(service.IndexTicket(ctx, ticket))
		ticketIds = append(ticketIds, ticket.GetId())
	}

	verifyTickets := func(service Service, expectLen int) {
		ids, err := service.GetIndexedIDSet(ctx)
		assert.Nil(err)
		assert.Equal(expectLen, len(ids))
	}

	// Verify all tickets are created and returned
	verifyTickets(service, len(tickets))

	// Add the first three tickets to the ignore list and verify changes are reflected in the result
	assert.Nil(service.AddTicketsToIgnoreList(ctx, ticketIds[:3]))
	verifyTickets(service, len(tickets)-3)

	// Sleep until the ignore list expired and verify we still have all the tickets
	time.Sleep(cfg.GetDuration("storage.ignoreListTTL"))
	verifyTickets(service, len(tickets))
}

func TestDeleteTicketsFromIgnoreList(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	tickets := internalTesting.GenerateFloatRangeTickets(
		internalTesting.Property{Name: "testindex1", Min: 0, Max: 10, Interval: 2},
		internalTesting.Property{Name: "testindex2", Min: 0, Max: 10, Interval: 2},
	)

	ticketIds := []string{}
	for _, ticket := range tickets {
		assert.Nil(service.CreateTicket(ctx, ticket))
		assert.Nil(service.IndexTicket(ctx, ticket))
		ticketIds = append(ticketIds, ticket.GetId())
	}

	verifyTickets := func(service Service, expectLen int) {
		ids, err := service.GetIndexedIDSet(ctx)
		assert.Nil(err)
		assert.Equal(expectLen, len(ids))
	}

	// Verify all tickets are created and returned
	verifyTickets(service, len(tickets))

	// Add the first three tickets to the ignore list and verify changes are reflected in the result
	assert.Nil(service.AddTicketsToIgnoreList(ctx, ticketIds[:3]))
	verifyTickets(service, len(tickets)-3)

	assert.Nil(service.DeleteTicketsFromIgnoreList(ctx, ticketIds[:3]))
	verifyTickets(service, len(tickets))
}

func TestGetAssignmentBeforeSet(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	var assignmentResp *pb.Assignment

	err := service.GetAssignments(ctx, "id", func(assignment *pb.Assignment) error {
		assignmentResp = assignment
		return nil
	})
	// GetAssignment failed because the ticket does not exists
	assert.Equal(status.Convert(err).Code(), codes.NotFound)
	assert.Nil(assignmentResp)
}

func TestGetAssignmentNormal(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t, true, "")
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "1",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	assert.Nil(err)

	var assignmentResp *pb.Assignment
	ctx, cancel := context.WithCancel(ctx)
	callbackCount := 0
	returnedErr := errors.New("some errors")

	err = service.GetAssignments(ctx, "1", func(assignment *pb.Assignment) error {
		assignmentResp = assignment

		if callbackCount == 5 {
			cancel()
			return returnedErr
		} else if callbackCount > 0 {
			// Test the assignment returned was successfully passed in to the callback function
			assert.Equal(assignmentResp.Connection, "2")
		}

		callbackCount++
		return nil
	})

	// Test GetAssignments was retried for 5 times and returned with expected error
	assert.Equal(5, callbackCount)
	assert.Equal(returnedErr, err)
}

func TestConnect(t *testing.T) {
	testConnect(t, false, "")
	testConnect(t, false, "redispassword")
	testConnect(t, true, "")
	testConnect(t, true, "redispassword")
}

func testConnect(t *testing.T, withSentinel bool, withPassword string) {
	assert := assert.New(t)
	cfg, closer := createRedis(t, withSentinel, withPassword)
	defer closer()
	store := New(cfg)
	defer store.Close()
	ctx := utilTesting.NewContext(t)

	is, ok := store.(*instrumentedService)
	assert.True(ok)
	rb, ok := is.s.(*redisBackend)
	assert.True(ok)

	conn, err := rb.connect(ctx)
	assert.NotNil(conn)
	assert.Nil(err)

	rply, err := redis.String(conn.Do("PING"))
	assert.Nil(err)
	assert.Equal("PONG", rply)
}

func createRedis(t *testing.T, withSentinel bool, withPassword string) (config.View, func()) {
	cfg := viper.New()
	closerFuncs := []func(){}
	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}
	closerFuncs = append(closerFuncs, mredis.Close)

	cfg.Set("redis.pool.maxIdle", 5)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.healthCheckTimeout", 100*time.Millisecond)
	cfg.Set("redis.pool.maxActive", 5)
	cfg.Set("storage.ignoreListTTL", "200ms")
	cfg.Set("backoff.initialInterval", 100*time.Millisecond)
	cfg.Set("backoff.randFactor", 0.5)
	cfg.Set("backoff.multiplier", 0.5)
	cfg.Set("backoff.maxInterval", 300*time.Millisecond)
	cfg.Set("backoff.maxElapsedTime", 100*time.Millisecond)
	cfg.Set(telemetry.ConfigNameEnableMetrics, true)

	if withSentinel {
		s := minisentinel.NewSentinel(mredis)
		err = s.StartAddr("localhost:0")
		if err != nil {
			t.Fatalf("failed to start minisentinel, %v", err)
		}

		closerFuncs = append(closerFuncs, s.Close)
		cfg.Set("redis.sentinelHostname", s.Host())
		cfg.Set("redis.sentinelPort", s.Port())
		cfg.Set("redis.sentinelMaster", s.MasterInfo().Name)
		cfg.Set("redis.sentinelEnabled", true)
		// TODO: enable sentinel auth test cases when the library support it.
		cfg.Set("redis.sentinelUsePassword", false)
	} else {
		cfg.Set("redis.hostname", mredis.Host())
		cfg.Set("redis.port", mredis.Port())
	}

	if len(withPassword) > 0 {
		mredis.RequireAuth(withPassword)
		tmpFile, err := ioutil.TempFile("", "password")
		if err != nil {
			t.Fatal("failed to create temp file for password")
		}
		if _, err := tmpFile.WriteString(withPassword); err != nil {
			t.Fatal("failed to write pw to temp file")
		}

		closerFuncs = append(closerFuncs, func() { os.Remove(tmpFile.Name()) })
		cfg.Set("redis.usePassword", true)
		cfg.Set("redis.passwordPath", tmpFile.Name())
	}

	return cfg, func() {
		for _, closer := range closerFuncs {
			closer()
		}
	}
}
