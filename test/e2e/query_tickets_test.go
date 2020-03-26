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

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/filter/testcases"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestNoPool(t *testing.T) {
	om := e2e.New(t)

	q := om.MustQueryServiceGRPC()

	{
		stream, err := q.QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: nil})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		require.Nil(t, resp)
	}

	{
		stream, err := q.QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: nil})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, codes.InvalidArgument, status.Convert(err).Code())
		require.Nil(t, resp)
	}
}

func TestNoTickets(t *testing.T) {
	om := e2e.New(t)

	q := om.MustQueryServiceGRPC()

	{
		stream, err := q.QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}

	{
		stream, err := q.QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: &pb.Pool{}})
		require.Nil(t, err)

		resp, err := stream.Recv()
		require.Equal(t, io.EOF, err)
		require.Nil(t, resp)
	}
}

func TestPaging(t *testing.T) {
	om := e2e.New(t)

	pageSize := 10 // TODO: read from config
	if pageSize < 1 {
		require.Fail(t, "invalid page size")
	}

	totalTickets := pageSize*5 + 1
	expectedIds := map[string]struct{}{}

	fe := om.MustFrontendGRPC()
	for i := 0; i < totalTickets; i++ {
		resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
		require.NotNil(t, resp)
		require.Nil(t, err)

		expectedIds[resp.Id] = struct{}{}
	}

	q := om.MustQueryServiceGRPC()
	stream, err := q.QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: &pb.Pool{}})
	require.Nil(t, err)

	foundIds := map[string]struct{}{}

	for i := 0; i < 5; i++ {
		var resp *pb.QueryTicketsResponse
		resp, err = stream.Recv()
		require.Nil(t, err)
		require.Equal(t, len(resp.Tickets), pageSize)

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
	require.Equal(t, err, io.EOF)
	require.Nil(t, resp)
}

func TestTicketFound(t *testing.T) {
	for _, tc := range testcases.IncludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if !returnedByQuery(t, tc) {
				require.Fail(t, "Expected to find ticket in pool but didn't.")
			}
			if !returnedByQueryID(t, tc) {
				require.Fail(t, "Expected to find id in pool but didn't.")
			}
		})
	}
}

func TestTicketNotFound(t *testing.T) {
	for _, tc := range testcases.ExcludedTestCases() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			if returnedByQuery(t, tc) {
				require.Fail(t, "Expected to not find ticket in pool but did.")
			}
			if returnedByQueryID(t, tc) {
				require.Fail(t, "Expected to not find id in pool but did.")
			}
		})
	}
}

func returnedByQuery(t *testing.T, tc testcases.TestCase) (found bool) {
	om := e2e.New(t)

	{
		fe := om.MustFrontendGRPC()
		resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: tc.Ticket})
		require.NotNil(t, resp)
		require.Nil(t, err)
	}

	q := om.MustQueryServiceGRPC()
	stream, err := q.QueryTickets(context.Background(), &pb.QueryTicketsRequest{Pool: tc.Pool})
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
	om := e2e.New(t)

	{
		fe := om.MustFrontendGRPC()
		resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: tc.Ticket})
		require.NotNil(t, resp)
		require.Nil(t, err)
	}

	q := om.MustQueryServiceGRPC()
	stream, err := q.QueryTicketIds(context.Background(), &pb.QueryTicketIdsRequest{Pool: tc.Pool})
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
