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

package frontend

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
)

func TestServerBinding(t *testing.T) {
	bs := func(p *rpc.Params) {
		p.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterFrontendServer(s, &frontendService{})
		}, pb.RegisterFrontendHandlerFromEndpoint)
	}

	rpcTesting.TestServerBinding(t, bs)
}

// validateTicket validates that the fetched ticket is identical to the expected ticket.
func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
	assert.Equal(t, got.Id, want.Id)
	assert.Equal(t, got.Properties, want.Properties)
	assert.Equal(t, got.Assignment.Connection, want.Assignment.Connection)
	assert.Equal(t, got.Assignment.Properties, want.Assignment.Properties)
	assert.Equal(t, got.Assignment.Error, want.Assignment.Error)
}

// validateDelete validates that the ticket is actually deleted from the state storage.
// Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
func validateDelete(t *testing.T, fe *frontendService, id string) {
	start := time.Now()
	for {
		if time.Since(start) > 5*time.Second {
			break
		}

		// Attempt to fetch the ticket every 100ms
		_, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: id})
		if err != nil {
			// Only failure to fetch with NotFound should be considered as success.
			assert.Equal(t, status.Code(err), codes.NotFound)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Failf(t, "ticket %v not deleted after 5 seconds", id)
}

// TestFrontendService tests creating, getting and deleting a ticket using Frontend service.
func TestFrontendService(t *testing.T) {
	assert := assert.New(t)
	cfg := createStore(t)
	fe, err := newFrontend(cfg)
	assert.Nil(err)
	assert.NotNil(fe)

	ticket := &pb.Ticket{
		Properties: "test-property",
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Create a ticket, validate that it got an id and set its id in the expected ticket.
	resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
	assert.NotNil(resp)
	assert.Nil(err)
	want := resp.Ticket
	assert.NotNil(want)
	assert.NotNil(want.Id)
	ticket.Id = want.Id
	validateTicket(t, resp.Ticket, ticket)

	// Fetch the ticket and validate that it is identical to the expected ticket.
	gotTicket, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: ticket.Id})
	assert.NotNil(gotTicket)
	assert.Nil(err)
	validateTicket(t, gotTicket, ticket)

	// Delete the ticket and validate that it was actually deleted.
	_, err = fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticket.Id})
	assert.Nil(err)
	validateDelete(t, fe, ticket.Id)
}

func createStore(t *testing.T) config.View {
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
	cfg.Set("redis.expiration", 42000)
	cfg.Set("playerIndices", []string{"testindex1", "testindex2"})

	return cfg
}
