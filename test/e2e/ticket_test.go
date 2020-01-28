// +build !e2ecluster

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

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestAssignTickets(t *testing.T) {
	om, closer := e2e.New(t)
	defer closer()
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	ctx := om.Context()

	ctResp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	var tt = []struct {
		description    string
		ticketIds      []string
		assignment     *pb.Assignment
		wantAssignment *pb.Assignment
		wantCode       codes.Code
	}{
		{
			"expects invalid argument code since request is empty",
			nil,
			nil,
			nil,
			codes.InvalidArgument,
		},
		{
			"expects invalid argument code since assignment is nil",
			[]string{"1"},
			nil,
			nil,
			codes.InvalidArgument,
		},
		{
			"expects not found code since ticket id does not exist in the statestore",
			[]string{"2"},
			&pb.Assignment{Connection: "localhost"},
			nil,
			codes.NotFound,
		},
		{
			"expects not found code since ticket id 'unknown id' does not exist in the statestore",
			[]string{ctResp.GetTicket().GetId(), "unknown id"},
			&pb.Assignment{Connection: "localhost"},
			nil,
			codes.NotFound,
		},
		{
			"expects ok code",
			[]string{ctResp.GetTicket().GetId()},
			&pb.Assignment{Connection: "localhost"},
			&pb.Assignment{Connection: "localhost"},
			codes.OK,
		},
	}

	t.Run("TestAssignTickets", func(t *testing.T) {
		for _, test := range tt {
			test := test
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				ctx := om.Context()
				_, err := be.AssignTickets(ctx, &pb.AssignTicketsRequest{TicketIds: test.ticketIds, Assignment: test.assignment})
				assert.Equal(t, test.wantCode, status.Convert(err).Code())

				// If assign ticket succeeds, validate the assignment
				if err == nil {
					for _, id := range test.ticketIds {
						gtResp, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: id})
						assert.Nil(t, err)
						// grpc will write something to the reserved fields of this protobuf object, so we have to do comparisons fields by fields.
						assert.Equal(t, test.wantAssignment.GetConnection(), gtResp.GetAssignment().GetConnection())
					}
				}
			})
		}
	})
}

// TestTicketLifeCycle tests creating, getting and deleting a ticket using Frontend service.
func TestTicketLifeCycle(t *testing.T) {
	assert := assert.New(t)

	om, closer := e2e.New(t)
	defer closer()
	fe := om.MustFrontendGRPC()
	assert.NotNil(fe)
	ctx := om.Context()

	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-property": 1,
			},
		},
		Assignment: &pb.Assignment{
			Connection: "test-tbd",
		},
	}

	// Create a ticket, validate that it got an id and set its id in the expected ticket.
	resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: ticket})
	assert.NotNil(resp)
	assert.Nil(err)
	want := resp.Ticket
	assert.NotNil(want)
	assert.NotNil(want.GetId())
	ticket.Id = want.GetId()
	validateTicket(t, resp.GetTicket(), ticket)

	// Fetch the ticket and validate that it is identical to the expected ticket.
	gotTicket, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: ticket.GetId()})
	assert.NotNil(gotTicket)
	assert.Nil(err)
	validateTicket(t, gotTicket, ticket)

	// Delete the ticket and validate that it was actually deleted.
	_, err = fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticket.GetId()})
	assert.Nil(err)
	validateDelete(ctx, t, fe, ticket.GetId())
}

// validateTicket validates that the fetched ticket is identical to the expected ticket.
func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
	assert.Equal(t, got.GetId(), want.GetId())
	assert.Equal(t, got.SearchFields.DoubleArgs["test-property"], want.SearchFields.DoubleArgs["test-property"])
	assert.Equal(t, got.GetAssignment().GetConnection(), want.GetAssignment().GetConnection())
}

// validateDelete validates that the ticket is actually deleted from the state storage.
// Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
func validateDelete(ctx context.Context, t *testing.T, fe pb.FrontendServiceClient, id string) {
	start := time.Now()
	for {
		if time.Since(start) > 5*time.Second {
			break
		}

		// Attempt to fetch the ticket every 100ms
		_, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: id})
		if err != nil {
			// Only failure to fetch with NotFound should be considered as success.
			assert.Equal(t, status.Code(err), codes.NotFound)
			return
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.Failf(t, "ticket %v not deleted after 5 seconds", id)
}
