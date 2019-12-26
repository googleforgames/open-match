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
	"fmt"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
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
	cfg, closer := createRedis(t)
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
}

func TestTicketLifecycle(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
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
	cfg, closer := createRedis(t)
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
		var results []*pb.Ticket
		pool := &pb.Pool{
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{DoubleArg: "testindex1", Min: 0, Max: 10},
				{DoubleArg: "testindex2", Min: 0, Max: 10},
			},
		}
		service.FilterTickets(ctx, pool, 100, func(tickets []*pb.Ticket) error {
			results = tickets
			return nil
		})
		assert.Equal(expectLen, len(results))
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

func TestTicketIndexing(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	for i := 0; i < 10; i++ {
		id := fmt.Sprintf("ticket.no.%d", i)

		ticket := &pb.Ticket{
			Id: id,
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"testindex1": float64(i),
					"testindex2": 0.5,
				},
			},
			Assignment: &pb.Assignment{
				Connection: "test-tbd",
			},
		}

		err := service.CreateTicket(ctx, ticket)
		assert.Nil(err)

		err = service.IndexTicket(ctx, ticket)
		assert.Nil(err)
	}

	// Remove one ticket, to test that it doesn't fall over.
	err := service.DeleteTicket(ctx, "ticket.no.5")
	assert.Nil(err)

	// Remove ticket from index, should not show up.
	err = service.DeindexTicket(ctx, "ticket.no.6")
	assert.Nil(err)

	found := []string{}

	pool := &pb.Pool{
		DoubleRangeFilters: []*pb.DoubleRangeFilter{
			{
				DoubleArg: "testindex1",
				Min:       2.5,
				Max:       8.5,
			},
			{
				DoubleArg: "testindex2",
				Min:       0.49,
				Max:       0.51,
			},
		},
	}

	err = service.FilterTickets(ctx, pool, 2, func(tickets []*pb.Ticket) error {
		assert.True(len(tickets) <= 2)
		for _, ticket := range tickets {
			found = append(found, ticket.Id)
		}
		return nil
	})
	assert.Nil(err)

	expected := []string{
		"ticket.no.3",
		"ticket.no.4",
		"ticket.no.7",
		"ticket.no.8",
	}
	assert.ElementsMatch(expected, found)
}

func TestGetAssignmentBeforeSet(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
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

func TestUpdateAssignmentFatal(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	var assignmentResp *pb.Assignment

	// Now create a ticket in the state store service
	err := service.CreateTicket(ctx, &pb.Ticket{
		Id:         "1",
		Assignment: &pb.Assignment{Connection: "2"},
	})
	assert.Nil(err)

	// Try to update the assignmets with the ticket created and some non-existed tickets
	err = service.UpdateAssignments(ctx, []string{"1", "2", "3"}, &pb.Assignment{Connection: "localhost"})
	// UpdateAssignment failed because the ticket does not exists
	assert.Equal(codes.NotFound, status.Convert(err).Code())
	assert.Nil(assignmentResp)
}

func TestGetAssignmentNormal(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
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

func TestUpdateAssignmentNormal(t *testing.T) {
	// Create State Store
	assert := assert.New(t)
	cfg, closer := createRedis(t)
	defer closer()
	service := New(cfg)
	assert.NotNil(service)
	defer service.Close()
	ctx := utilTesting.NewContext(t)

	// Create a ticket without assignment
	err := service.CreateTicket(ctx, &pb.Ticket{
		Id: "1",
	})
	assert.Nil(err)
	// Create a ticket already with an assignment
	err = service.CreateTicket(ctx, &pb.Ticket{
		Id:         "3",
		Assignment: &pb.Assignment{Connection: "4"},
	})
	assert.Nil(err)

	fakeAssignment := &pb.Assignment{Connection: "Halo"}
	err = service.UpdateAssignments(ctx, []string{"1", "3"}, fakeAssignment)
	assert.Nil(err)
	// Verify the transaction behavior of the UpdateAssignment.
	ticket, err := service.GetTicket(ctx, "1")
	assert.Equal(fakeAssignment.Connection, ticket.Assignment.Connection)
	assert.Nil(err)
	// Verify the transaction behavior of the UpdateAssignment.
	ticket, err = service.GetTicket(ctx, "3")
	assert.Equal(fakeAssignment.Connection, ticket.Assignment.Connection)
	assert.Nil(err)
}

func TestConnect(t *testing.T) {
	assert := assert.New(t)
	cfg, closer := createRedis(t)
	defer closer()
	store := New(cfg)
	defer store.Close()
	ctx := utilTesting.NewContext(t)

	is, ok := store.(*instrumentedService)
	assert.True(ok)
	rb, ok := is.s.(*redisBackend)
	assert.True(ok)

	ctx, cancel := context.WithCancel(ctx)
	cancel()
	conn, err := rb.connect(ctx)
	assert.NotNil(err)
	assert.Nil(conn)
}

func createRedis(t *testing.T) (config.View, func()) {
	cfg := viper.New()
	mredis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("cannot create redis %s", err)
	}

	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 1000)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.healthCheckTimeout", 100*time.Millisecond)
	cfg.Set("redis.pool.maxActive", 1000)
	cfg.Set("redis.expiration", 42000)
	cfg.Set("storage.ignoreListTTL", "200ms")
	cfg.Set("backoff.initialInterval", 100*time.Millisecond)
	cfg.Set("backoff.randFactor", 0.5)
	cfg.Set("backoff.multiplier", 0.5)
	cfg.Set("backoff.maxInterval", 300*time.Millisecond)
	cfg.Set("backoff.maxElapsedTime", 100*time.Millisecond)
	cfg.Set(telemetry.ConfigNameEnableMetrics, true)

	return cfg, func() { mredis.Close() }
}
