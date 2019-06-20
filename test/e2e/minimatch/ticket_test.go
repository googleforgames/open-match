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

package minimatch

import (
	"context"
	"io"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"

	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
)

func TestAssignTickets(t *testing.T) {
	tc := createMinimatchForTest(t)
	defer tc.Close()

	fe := pb.NewFrontendClient(tc.MustGRPC())
	be := pb.NewBackendClient(tc.MustGRPC())

	ctResp, err := fe.CreateTicket(tc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
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
				_, err := be.AssignTickets(tc.Context(), &pb.AssignTicketsRequest{TicketId: test.ticketIds, Assignment: test.assignment})
				assert.Equal(t, test.wantCode, status.Convert(err).Code())

				// If assign ticket succeeds, validate the assignment
				if err == nil {
					for _, id := range test.ticketIds {
						gtResp, err := fe.GetTicket(tc.Context(), &pb.GetTicketRequest{TicketId: id})
						assert.Nil(t, err)
						// grpc will write something to the reserved fields of this protobuf object, so we have to do comparisons fields by fields.
						assert.Equal(t, test.wantAssignment.GetConnection(), gtResp.GetAssignment().GetConnection())
						assert.Equal(t, test.wantAssignment.GetProperties(), gtResp.GetAssignment().GetProperties())
						assert.Equal(t, test.wantAssignment.GetError(), gtResp.GetAssignment().GetError())
					}
				}
			})
		}
	})
}

// TestTicketLifeCycle tests creating, getting and deleting a ticket using Frontend service.
func TestTicketLifeCycle(t *testing.T) {
	assert := assert.New(t)

	tc := createMinimatchForTest(t)
	fe := pb.NewFrontendClient(tc.MustGRPC())
	assert.NotNil(fe)

	ticket := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
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
	assert.NotNil(want.GetId())
	ticket.Id = want.GetId()
	validateTicket(t, resp.GetTicket(), ticket)

	// Fetch the ticket and validate that it is identical to the expected ticket.
	gotTicket, err := fe.GetTicket(context.Background(), &pb.GetTicketRequest{TicketId: ticket.GetId()})
	assert.NotNil(gotTicket)
	assert.Nil(err)
	validateTicket(t, gotTicket, ticket)

	// Delete the ticket and validate that it was actually deleted.
	_, err = fe.DeleteTicket(context.Background(), &pb.DeleteTicketRequest{TicketId: ticket.GetId()})
	assert.Nil(err)
	validateDelete(t, fe, ticket.GetId())
}

func TestQueryTickets(t *testing.T) {
	assert := assert.New(t)
	tc := createMinimatchForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc, &pb.QueryTicketsRequest{}, func(_ *pb.QueryTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
}

func TestQueryTicketsForEmptyDatabase(t *testing.T) {
	assert := assert.New(t)
	tc := createMinimatchForTest(t)
	defer tc.Close()

	queryTicketsLoop(t, tc,
		&pb.QueryTicketsRequest{
			Pool: &pb.Pool{
				Filter: []*pb.Filter{{
					Attribute: "ok",
				}},
			},
		},
		func(resp *pb.QueryTicketsResponse, err error) {
			assert.NotNil(resp)
		})
}

func queryTicketsLoop(t *testing.T, tc *rpcTesting.TestContext, req *pb.QueryTicketsRequest, handleResponse func(*pb.QueryTicketsResponse, error)) {
	c := pb.NewMmLogicClient(tc.MustGRPC())
	stream, err := c.QueryTickets(tc.Context(), req)
	if err != nil {
		t.Fatalf("error querying tickets, %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handleResponse(resp, err)
		if err != nil {
			return
		}
	}
}

// validateTicket validates that the fetched ticket is identical to the expected ticket.
func validateTicket(t *testing.T, got *pb.Ticket, want *pb.Ticket) {
	assert.Equal(t, got.GetId(), want.GetId())
	assert.Equal(t, got.GetProperties().GetFields()["test-property"].GetNumberValue(), want.GetProperties().GetFields()["test-property"].GetNumberValue())
	assert.Equal(t, got.GetAssignment().GetConnection(), want.GetAssignment().GetConnection())
	assert.Equal(t, got.GetAssignment().GetProperties(), want.GetAssignment().GetProperties())
	assert.Equal(t, got.GetAssignment().GetError(), want.GetAssignment().GetError())
}

// validateDelete validates that the ticket is actually deleted from the state storage.
// Given that delete is async, this method retries fetch every 100ms up to 5 seconds.
func validateDelete(t *testing.T, fe pb.FrontendClient, id string) {
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
