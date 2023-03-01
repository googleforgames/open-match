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
	"fmt"
	"io"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/pkg/pb"
)

// TestAssignTickets covers assigning multiple tickets, using two different
// assignment groups.
func TestAssignTickets(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)
	t3, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
			{
				TicketIds:  []string{t2.Id, t3.Id},
				Assignment: &pb.Assignment{Connection: "b"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	expected := &pb.AssignTicketsResponse{}
	require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))

	get, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, err)
	require.Equal(t, "a", get.Assignment.Connection)

	get, err = om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t2.Id})
	require.Nil(t, err)
	require.Equal(t, "b", get.Assignment.Connection)

	get, err = om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t3.Id})
	require.Nil(t, err)
	require.Equal(t, "b", get.Assignment.Connection)
}

// TestAssignTicketsEmpty covers calls to assign when empty TicketIds
func TestAssignTicketsEmpty(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, resp)
	require.Equal(t, codes.InvalidArgument.String(), status.Convert(err).Code().String())
	require.Equal(t, "AssignmentGroupTicketIds is empty", status.Convert(err).Message())

}

// TestAssignTicketsInvalidArgument covers various invalid calls to assign
// tickets.
func TestAssignTicketsInvalidArgument(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	ctResp, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

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
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call",
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
			"Ticket id " + ctResp.Id + " is assigned multiple times in one assign tickets call",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			_, err := om.Backend().AssignTickets(ctx, tt.req)
			require.Equal(t, codes.InvalidArgument.String(), status.Convert(err).Code().String())
			require.Equal(t, tt.msg, status.Convert(err).Message())
		})
	}
}

// TestAssignTicketsMissingTicket covers that when a ticket was deleted before
// being assigned, the assign tickets calls succeeds, however it returns a
// notice that the ticket was missing.
func TestAssignTicketsMissingTicket(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t2, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	t3, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	_, err = om.Frontend().DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: t2.Id})
	require.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id, t2.Id, t3.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	expected := &pb.AssignTicketsResponse{
		Failures: []*pb.AssignmentFailure{
			{
				TicketId: t2.Id,
				Cause:    pb.AssignmentFailure_TICKET_NOT_FOUND,
			},
		},
	}
	require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))
}

func TestTicketDelete(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	_, err = om.Frontend().DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: t1.Id})
	require.Nil(t, err)

	resp, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, resp)
	require.Equal(t, "Ticket id: "+t1.Id+" not found", status.Convert(err).Message())
	require.Equal(t, codes.NotFound, status.Convert(err).Code())
}

// TestEmptyReleaseTicketsRequest covers that it is valid to not have any ticket
// ids when releasing tickets.  (though it's not really doing anything...)
func TestEmptyReleaseTicketsRequest(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	resp, err := om.Backend().ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
		TicketIds: nil,
	})

	require.Nil(t, err)
	expected := &pb.ReleaseTicketsResponse{}
	require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))
}

// TestReleaseTickets covers that tickets returned from matches are no longer
// returned by query tickets, but will return after being released.
func TestReleaseTickets(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	var ticket *pb.Ticket

	{ // Create ticket
		var err error
		ticket, err = om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		require.Nil(t, err)
		require.NotEmpty(t, ticket.Id)
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	var matchReturnedAt time.Time

	{ // Ticket returned from match
		om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{ticket},
			}
			return nil
		})
		om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
			m := <-in
			_, ok := <-in
			require.False(t, ok)
			matchReturnedAt = time.Now()
			out <- m.MatchId
			return nil
		})

		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config: om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{Name: "pool"},
				},
			},
		})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Match.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Match.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Ticket NOT present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Return ticket
		resp, err := om.Backend().ReleaseTickets(ctx, &pb.ReleaseTicketsRequest{
			TicketIds: []string{ticket.Id},
		})

		require.Nil(t, err)
		expected := &pb.ReleaseTicketsResponse{}
		require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	// Ensure that the release timeout did NOT have enough time to affect this
	// test.
	require.True(t, time.Since(matchReturnedAt) < pendingReleaseTimeout, "%s", time.Since(matchReturnedAt))
}

// TestReleaseAllTickets covers that tickets are released and returned by query
// after calling ReleaseAllTickets.  Does test available fetch matches, not
// after fetch matches, as that's covered by TestReleaseTickets.
func TestReleaseAllTickets(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	var ticket *pb.Ticket

	{ // Create ticket
		var err error
		ticket, err = om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		require.Nil(t, err)
		require.NotEmpty(t, ticket.Id)
	}

	var matchReturnedAt time.Time

	{ // Ticket returned from match
		om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{ticket},
			}
			return nil
		})
		om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
			m := <-in
			_, ok := <-in
			require.False(t, ok)
			matchReturnedAt = time.Now()
			out <- m.MatchId
			return nil
		})

		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config: om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{Name: "pool"},
				},
			},
		})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Match.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Match.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Return ticket
		resp, err := om.Backend().ReleaseAllTickets(ctx, &pb.ReleaseAllTicketsRequest{})

		require.Nil(t, err)
		expected := &pb.ReleaseAllTicketsResponse{}
		require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	// Ensure that the release timeout did NOT have enough time to affect this
	// test.
	require.True(t, time.Since(matchReturnedAt) < pendingReleaseTimeout, "%s", time.Since(matchReturnedAt))
}

// TestReleaseTickets covers that tickets are released after a time if returned
// by a match but not assigned
func TestTicketReleaseByTimeout(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	var ticket *pb.Ticket

	{ // Create ticket
		var err error
		ticket, err = om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		fmt.Println("------------------- ", err)
		require.Nil(t, err)
		require.NotEmpty(t, ticket.Id)
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Ticket returned from match
		om.SetMMF(func(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
			out <- &pb.Match{
				MatchId: "1",
				Tickets: []*pb.Ticket{ticket},
			}
			return nil
		})
		om.SetEvaluator(func(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
			m := <-in
			_, ok := <-in
			require.False(t, ok)
			out <- m.MatchId
			return nil
		})

		stream, err := om.Backend().FetchMatches(ctx, &pb.FetchMatchesRequest{
			Config: om.MMFConfigGRPC(),
			Profile: &pb.MatchProfile{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{Name: "pool"},
				},
			},
		})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Match.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Match.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Ticket NOT present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{ // Return ticket
		time.Sleep(pendingReleaseTimeout)
	}

	{ // Ticket present in query
		stream, err := om.Query().QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Nil(t, err)
		require.Len(t, resp.Tickets, 1)
		require.Equal(t, ticket.Id, resp.Tickets[0].Id)

		resp, err = stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}
}

// TestCreateTicketErrors covers invalid arguments when calling create ticket.
func TestCreateTicketErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		req  *pb.CreateTicketRequest
		msg  string
	}{
		{
			"missing ticket",
			&pb.CreateTicketRequest{
				Ticket: nil,
			},
			".ticket is required",
		},
		{
			"already has assignment",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					Assignment: &pb.Assignment{},
				},
			},
			"tickets cannot be created with an assignment",
		},
		{
			"already has create time",
			&pb.CreateTicketRequest{
				Ticket: &pb.Ticket{
					CreateTime: timestamppb.Now(),
				},
			},
			"tickets cannot be created with create time set",
		},
	} {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			om := newOM(t)
			ctx := context.Background()

			resp, err := om.Frontend().CreateTicket(ctx, tt.req)
			require.Nil(t, resp)
			s := status.Convert(err)
			require.Equal(t, codes.InvalidArgument.String(), s.Code().String())
			require.Equal(t, s.Message(), tt.msg)
		})
	}
}

// TestAssignedTicketsNotReturnedByQuery covers that when a ticket has been
// assigned, it will no longer be returned by query.
func TestAssignedTicketsNotReturnedByQuery(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	returned := func() bool {
		stream, err := om.Query().QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		_, err = stream.Recv()
		return err != io.EOF
	}

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	require.True(t, returned())

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	expected := &pb.AssignTicketsResponse{}
	require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))

	require.False(t, returned())
}

// TestAssignedTicketDeleteTimeout covers assigned tickets being deleted after
// a timeout.
func TestAssignedTicketDeleteTimeout(t *testing.T) {
	om := newOM(t)
	ctx := context.Background()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.Nil(t, err)

	req := &pb.AssignTicketsRequest{
		Assignments: []*pb.AssignmentGroup{
			{
				TicketIds:  []string{t1.Id},
				Assignment: &pb.Assignment{Connection: "a"},
			},
		},
	}

	resp, err := om.Backend().AssignTickets(ctx, req)
	require.Nil(t, err)
	expected := &pb.AssignTicketsResponse{}
	require.True(t, proto.Equal(expected, resp), fmt.Sprintf("Protobuf messages are not equal\nexpected: %v\nactual: %v", expected, resp))

	get, err := om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, err)
	require.Equal(t, "a", get.Assignment.Connection)

	om.AdvanceTTLTime(assignedDeleteTimeout)

	get, err = om.Frontend().GetTicket(ctx, &pb.GetTicketRequest{TicketId: t1.Id})
	require.Nil(t, get)
	require.Equal(t, codes.NotFound, status.Convert(err).Code())

}

func TestWatchAssignments(t *testing.T) {
	om := newOM(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t1, err := om.Frontend().CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	require.NoError(t, err)

	{
		req := &pb.AssignTicketsRequest{
			Assignments: []*pb.AssignmentGroup{
				{
					TicketIds:  []string{t1.Id},
					Assignment: &pb.Assignment{Connection: "a"},
				},
			},
		}
		resp, err := om.Backend().AssignTickets(ctx, req)
		require.NoError(t, err)
		require.Len(t, resp.Failures, 0)
	}

	{
		stream, err := om.Frontend().WatchAssignments(ctx, &pb.WatchAssignmentsRequest{TicketId: t1.Id})
		require.NoError(t, err)

		var a *pb.Assignment
		for a.GetConnection() == "" {
			resp, err := stream.Recv()
			require.NoError(t, err)

			a = resp.Assignment
		}

		require.Equal(t, "a", a.Connection)
	}
}
