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
	"io"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestAssignTickets(t *testing.T) {
	om := e2e.New(t)
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	ctx := om.Context()

	t1, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)
	t2, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
			{
				TicketIds:  []string{t2.Id},
				Assignment: &pb.Assignment{Connection: "b"},
			},
		},
	}

	resp, err := be.AssignTickets(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, &pb.AssignTicketsResponse{}, resp)

	get, err := fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	assert.Nil(t, err)
	assert.Equal(t, "a", get.Assignment.Connection)

	get, err = fe.GetTicket(ctx, &pb.GetTicketRequest{TicketId: t2.Id})
	assert.Nil(t, err)
	assert.Equal(t, "b", get.Assignment.Connection)
}

func TestAssignTicketsInvalidArgument(t *testing.T) {
	om := e2e.New(t)
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	ctx := om.Context()

	ctResp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	for _, tt := range []struct {
		name string
		req  *pb.AssignTicketsRequest
		msg  string
	}{
		{
			"missing assignment",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{},
				},
			},
			"AssignmentGroup.Assignment is required",
		},
		{
			"ticket used twice one group",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{ctResp.Id, ctResp.Id},
						Assignment: &pb.Assignment{},
					},
				},
			},
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call.",
		},
		{
			"ticket used twice two groups",
			&pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds:  []string{ctResp.Id},
						Assignment: &pb.Assignment{Connection: "a"},
					},
					{
						TicketIds:  []string{ctResp.Id},
						Assignment: &pb.Assignment{Connection: "b"},
					},
				},
			},
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call.",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := be.AssignTickets(ctx, tt.req)
			assert.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
			assert.Equal(t, tt.msg, status.Convert(err).Message())
		})
	}
}

func TestAssignTicketsMissingTicket(t *testing.T) {
	om := e2e.New(t)
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	ctx := om.Context()

	t1, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	t2, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	t3, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(t, err)

	_, err = fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: t2.Id})
	assert.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id, t2.Id, t3.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := be.AssignTickets(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, &pb.AssignTicketsResponse{
		Failures: []*pb.AssignmentFailure{
			{
				TicketId: t2.Id,
				Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
			},
		},
	}, resp)
}

// TestTicketLifeCycle tests creating, getting and deleting a ticket using Frontend service.
func TestTicketLifeCycle(t *testing.T) {
	assert := assert.New(t)

	om := e2e.New(t)
	fe := om.MustFrontendGRPC()
	assert.NotNil(fe)
	ctx := om.Context()

	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-property": 1,
			},
		},
	}

	// Create a ticket, validate that it got an id and set its id in the expected ticket.
	createResp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: ticket})
	assert.NotNil(createResp)
	assert.NotNil(createResp.GetId())
	assert.Nil(err)
	ticket.Id = createResp.GetId()
	validateTicket(t, createResp, ticket)

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

func TestEmptyReleaseTicketsRequest(t *testing.T) {
	om := e2e.New(t)

	be := om.MustBackendGRPC()
	ctx := om.Context()

	resp, err := be.ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
		TicketIds: nil,
	})

	assert.Nil(t, err)
	assert.Equal(t, &pb.ReleaseTicketsResponse{}, resp)
}

func TestReleaseTickets(t *testing.T) {
	om := e2e.New(t)

	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	q := om.MustQueryServiceGRPC()
	ctx := om.Context()

	var ticket *pb.Ticket

	{ // Create ticket
		var err error
		ticket, err = fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		assert.Nil(t, err)
		assert.NotEmpty(t, ticket.Id)
	}

	{ // Ticket present in query
		stream, err := q.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		assert.Nil(t, err)

		resp, err := stream.Recv()
		assert.Nil(t, err)
		assert.Equal(t, &pb.QueryTicketsResponse{
			Tickets: []*pb.Ticket{ticket},
		}, resp)

		resp, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, resp)
	}

	{ // Ticket returned from match
		stream, err := be.FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config: om.MustMmfConfigGRPC(),
			Profile: &pb.MatchProfile{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{Name: "pool"},
				},
			},
		})
		assert.Nil(t, err)

		resp, err := stream.Recv()
		assert.Nil(t, err)
		assert.Len(t, resp.Match.Tickets, 1)
		assert.Equal(t, ticket, resp.Match.Tickets[0])

		resp, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, resp)
	}

	{ // Ticket NOT present in query
		stream, err := q.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		assert.Nil(t, err)

		resp, err := stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, resp)
	}

	{ // Return ticket
		resp, err := be.ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
			TicketIds: []string{ticket.Id},
		})

		assert.Nil(t, err)
		assert.Equal(t, &pb.ReleaseTicketsResponse{}, resp)
	}

	{ // Ticket present in query
		stream, err := q.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		assert.Nil(t, err)

		resp, err := stream.Recv()
		assert.Nil(t, err)
		assert.Equal(t, &pb.QueryTicketsResponse{
			Tickets: []*pb.Ticket{ticket},
		}, resp)

		resp, err = stream.Recv()
		assert.Equal(t, io.EOF, err)
		assert.Nil(t, resp)
	}
}

func TestCreateTicketErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		req  *pb.CreateTicketRequest
		code codes.Code
		msg  string
	}{
		{
			"missing ticket",
			&pb.CreateTicketRequest{
				Ticket: nil,
			},
			codes.InvalidArgument,
			".ticket is required",
		},
		{
			"already has assignment",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					Assignment: &pb.Assignment{},
				},
			},
			codes.InvalidArgument,
			"tickets cannot be created with an assignment",
		},
		{
			"already has create time",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					CreateTime: ptypes.TimestampNow(),
				},
			},
			codes.InvalidArgument,
			"tickets cannot be created with create time set",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			om := e2e.New(t)
			fe := om.MustFrontendGRPC()
			ctx := om.Context()

			resp, err := fe.CreateTicket(ctx, tt.req)
			assert.Nil(t, resp)
			s := status.Convert(err)
			assert.Equal(t, tt.code, s.Code())
			assert.Equal(t, s.Message(), tt.msg)
		})
	}
}
