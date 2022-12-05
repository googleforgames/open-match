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

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/filter/testcases"
	"open-match.dev/open-match/pkg/pb"
)

func TestNoPool(t *testing.T) {
	om := newOM(t)

	{
		stream, err := om.Query().QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: nil})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		require.Nil(t, resp)
	}

	{
		stream, err := om.Query().QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: nil})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		require.Nil(t, resp)
	}
}

func TestNoTickets(t *testing.T) {
	om := newOM(t)

	{
		stream, err := om.Query().QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{
		stream, err := om.Query().QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}
}

func TestPaging(t *testing.T) {
	om := newOM(t)

	pageSize := 10 // TODO: read from config
	if pageSize < 1 {
		require.Fail(t, "invalid page size")
	}

	totalTickets := pageSize*5 + 1
	expectedIds := map[string]struct{}{}

	for i := 0; i < totalTickets; i++ {
		resp, err := om.Frontend().CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		require.NotNil(t, resp)
		require.Nil(t, err)

		expectedIds[resp.Id] = struct{}{}
	}

	stream, err := om.Query().QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
	require.Nil(t, err)

	foundIds := map[string]struct{}{}

	for i := 0; i < 5; i++ {
		var resp *pb.QueryTicketsResponse
		resp, err = stream.Recv()
		require.Nil(t, err)
		require.Equal(t, pageSize, len(resp.Tickets))

		for _, ticket := range resp.Tickets {
			foundIds[ticket.Id] = struct{}{}
		}
	}

	resp, err := stream.Recv()
	require.Nil(t, err)
	require.Equal(t, len(resp.Tickets), 1)
	foundIds[resp.Tickets[0].Id] = struct{}{}

	require.Equal(t, expectedIds, foundIds)

	resp, err = stream.Recv()
	require.Error(t, err)
	require.Equal(t, err.Error(), io.EOF.Error())
	require.Nil(t, resp)
}

func TestTicketFound(t *testing.T) {
	for _, tc := range testcases.IncludedTestCases() {
		tc := tc
		t.Run("QueryTickets_"+tc.Name, func(t *testing.T) {
			if !returnedByQuery(t, tc) {
				require.Fail(t, "Expected to find ticket in pool but didn't.")
			}
		})
		t.Run("QueryTicketIds_"+tc.Name, func(t *testing.T) {
			if !returnedByQueryID(t, tc) {
				require.Fail(t, "Expected to find id in pool but didn't.")
			}
		})
	}
}

func TestTicketNotFound(t *testing.T) {
	for _, tc := range testcases.ExcludedTestCases() {
		tc := tc
		t.Run("QueryTickets_"+tc.Name, func(t *testing.T) {
			if returnedByQuery(t, tc) {
				require.Fail(t, "Expected to not find ticket in pool but did.")
			}
		})
		t.Run("QueryTicketIds_"+tc.Name, func(t *testing.T) {
			if returnedByQueryID(t, tc) {
				require.Fail(t, "Expected to not find id in pool but did.")
			}
		})
	}
}

func returnedByQuery(t *testing.T, tc testcases.TestCase) (found bool) {
	om := newOM(t)

	{
		ticket := pb.Ticket{
			SearchFields: tc.SearchFields,
		}
		resp, err := om.Frontend().CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: &ticket})

		require.NotNil(t, resp)
		require.Nil(t, err)
	}

	stream, err := om.Query().QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: tc.Pool})
	require.Nil(t, err)

	tickets := []*pb.Ticket{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.Nil(t, err)

		tickets = append(tickets, resp.Tickets...)
	}

	if len(tickets) > 1 {
		require.Fail(t, "More than one ticket found")
	}

	return len(tickets) == 1
}

func returnedByQueryID(t *testing.T, tc testcases.TestCase) (found bool) {
	om := newOM(t)

	{
		ticket := pb.Ticket{
			SearchFields: tc.SearchFields,
		}
		resp, err := om.Frontend().CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: &ticket})

		require.NotNil(t, resp)
		require.Nil(t, err)
	}

	stream, err := om.Query().QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: tc.Pool})
	require.Nil(t, err)

	ids := []string{}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.Nil(t, err)

		ids = append(ids, resp.GetIds()...)
	}

	if len(ids) > 1 {
		require.Fail(t, "More than one ticket found")
	}

	return len(ids) == 1
}
