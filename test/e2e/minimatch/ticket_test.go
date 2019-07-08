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

package minimatch

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	internalTesting "open-match.dev/open-match/internal/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

func TestAssignTickets(t *testing.T) {
	evalTc := createEvaluatorForTest(t)
	defer evalTc.Close()
	tc := createMinimatchForTest(t, evalTc)
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
				_, err := be.AssignTickets(tc.Context(), &pb.AssignTicketsRequest{TicketIds: test.ticketIds, Assignment: test.assignment})
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

	evalTc := createEvaluatorForTest(t)
	defer evalTc.Close()
	tc := createMinimatchForTest(t, evalTc)
	defer tc.Close()
	fe := pb.NewFrontendClient(tc.MustGRPC())
	assert.NotNil(fe)

	ticket := &pb.Ticket{
		Properties: structs.Struct{
			"test-property": structs.Number(1),
		}.S(),
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
	tests := []struct {
		description   string
		pool          *pb.Pool
		preAction     func(fe pb.FrontendClient, t *testing.T)
		wantCode      codes.Code
		wantTickets   []*pb.Ticket
		wantPageCount int
	}{
		{
			description:   "expects invalid argument code since pool is empty",
			preAction:     func(_ pb.FrontendClient, _ *testing.T) {},
			pool:          nil,
			wantCode:      codes.InvalidArgument,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since the store is empty",
			preAction:   func(_ pb.FrontendClient, _ *testing.T) {},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: "ok",
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since all tickets in the store are filtered out",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.SkillAttribute,
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with 5 tickets with e2e.Map1Attribute=2 and e2e.Map2Attribute in range of [0,10)",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.Map1Attribute,
					Min:       1,
					Max:       3,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateTickets(
				internalTesting.Property{Name: e2e.Map1Attribute, Min: 2, Max: 3, Interval: 2},
				internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 1,
		},
		{
			// Test inclusive filters and paging works as expected
			description: "expects response with 15 tickets with e2e.Map1Attribute=2,4,6 and e2e.Map2Attribute=[0,10)",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},

			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.Map1Attribute,
					Min:       2,
					Max:       6,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateTickets(
				internalTesting.Property{Name: e2e.Map1Attribute, Min: 2, Max: 7, Interval: 2},
				internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 2,
		},
	}

	t.Run("TestQueryTickets", func(t *testing.T) {
		for _, test := range tests {
			test := test
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()
				evalTc := createEvaluatorForTest(t)
				defer evalTc.Close()
				tc := createMinimatchForTest(t, evalTc)
				defer tc.Close()

				mml := pb.NewMmLogicClient(tc.MustGRPC())
				fe := pb.NewFrontendClient(tc.MustGRPC())
				pageCounts := 0

				test.preAction(fe, t)

				stream, err := mml.QueryTickets(tc.Context(), &pb.QueryTicketsRequest{Pool: test.pool})
				assert.Nil(t, err)

				var actualTickets []*pb.Ticket

				for {
					resp, err := stream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						assert.Equal(t, test.wantCode, status.Convert(err).Code())
						break
					}

					actualTickets = append(actualTickets, resp.Tickets...)
					pageCounts++
				}

				require.Equal(t, len(test.wantTickets), len(actualTickets))
				// Test fields by fields because of the randomness of the ticket ids...
				// TODO: this makes testing overcomplicated. Should figure out a way to avoid the randomness
				// This for loop also relies on the fact that redis range query and the ticket generator both returns tickets in sorted order.
				// If this fact changes, we might need an ugly nested for loop to do the validness checks.
				for i := 0; i < len(actualTickets); i++ {
					assert.Equal(t, test.wantTickets[i].GetAssignment(), actualTickets[i].GetAssignment())
					assert.Equal(t, test.wantTickets[i].GetProperties(), actualTickets[i].GetProperties())
				}
				assert.Equal(t, test.wantPageCount, pageCounts)
			})
		}
	})
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
