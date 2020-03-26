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

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/matchfunction/mmf"
)

func TestServiceHealth(t *testing.T) {
	om := e2e.New(t)
	if err := om.HealthCheck(); err != nil {
		t.Errorf("cluster health checks failed, %s", err)
	}
}

func TestGetClients(t *testing.T) {
	om := e2e.New(t)

	if c := om.MustFrontendGRPC(); c == nil {
		t.Error("cannot get frontendService client")
	}

	if c := om.MustBackendGRPC(); c == nil {
		t.Error("cannot get backendService client")
	}

	if c := om.MustQueryServiceGRPC(); c == nil {
		t.Error("cannot get queryService client")
	}
}

func TestGameMatchWorkFlow(t *testing.T) {
	/*
		This end to end test does the following things step by step
		1. Create a few tickets with delicate designs and hand crafted search fields
		2. Call backend.FetchMatches and verify it returns expected matches.
		3. Call backend.FetchMatches within storage.ignoreListTTL seconds and expects it return a match with duplicate tickets in step 2.
		4. Wait for storage.ignoreListTTL seconds and call backend.FetchMatches the third time, expect the same result as step 2.
		5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
		6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
		7. Call frontend.DeleteTicket to delete the tickets returned in step 6.
		8. Call backend.FetchMatches and verify the response does not contain the tickets got deleted.
	*/

	om := e2e.New(t)
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	mmfCfg := om.MustMmfConfigGRPC()
	ctx := om.Context()

	ticket1 := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR:   1,
				e2e.DoubleArgLevel: 1,
			},
		},
	}

	ticket2 := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR:   5,
				e2e.DoubleArgLevel: 5,
			},
		},
	}

	ticket3 := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR:   10,
				e2e.DoubleArgLevel: 10,
			},
		},
	}

	ticket4 := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR:   15,
				e2e.DoubleArgLevel: 15,
			},
		},
	}

	ticket5 := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR:   20,
				e2e.DoubleArgLevel: 20,
			},
		},
	}

	tickets := []*pb.Ticket{ticket1, ticket2, ticket3, ticket4, ticket5}

	var err error
	// 1. Create a few tickets with delicate designs and hand crafted search fields
	for i := 0; i < len(tickets); i++ {
		var ticket *pb.Ticket
		ticket, err = fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: tickets[i]}, grpc.WaitForReady(true))
		require.Nil(t, err)
		require.NotNil(t, ticket)
		// Assign Open Match ids back to the input tickets
		*tickets[i] = *ticket
	}

	fmReq := &pb.FetchMatchesRequest{
		Config: mmfCfg,
		Profile: &pb.MatchProfile{
			Name: "test-profile",
			Pools: []*pb.Pool{
				{
					Name:               "ticket12",
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: e2e.DoubleArgMMR, Min: 0, Max: 6}, {DoubleArg: e2e.DoubleArgLevel, Min: 0, Max: 100}},
				},
				{
					Name:               "ticket23",
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: e2e.DoubleArgMMR, Min: 3, Max: 10}, {DoubleArg: e2e.DoubleArgLevel, Min: 0, Max: 100}},
				},
				{
					Name:               "ticket5",
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: e2e.DoubleArgMMR, Min: 0, Max: 100}, {DoubleArg: e2e.DoubleArgLevel, Min: 17, Max: 25}},
				},
				{
					Name:               "ticket234",
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: e2e.DoubleArgMMR, Min: 3, Max: 17}, {DoubleArg: e2e.DoubleArgLevel, Min: 3, Max: 17}},
				},
			},
		},
	}

	// 2. Call backend.FetchMatches and expects two matches with the following tickets
	matches := []*pb.Match{MustMakeMatch(ticket2, ticket3, ticket4), MustMakeMatch(ticket5)}
	validateFetchMatchesResponse(ctx, t, matches, be, fmReq)

	// 3. Call backend.FetchMatches within storage.ignoreListTTL seconds and expects it return a match with ticket1 .
	matches = []*pb.Match{MustMakeMatch(ticket1)}
	validateFetchMatchesResponse(ctx, t, matches, be, fmReq)

	// 4. Wait for storage.ignoreListTTL seconds and call backend.FetchMatches the third time, expect the same result as step 2.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	matches = []*pb.Match{MustMakeMatch(ticket2, ticket3, ticket4), MustMakeMatch(ticket5)}
	validateFetchMatchesResponse(ctx, t, matches, be, fmReq)

	// 5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
	var gotAtResp *pb.AssignTicketsResponse

	for _, m := range matches {
		tids := []string{}
		for _, ticket := range m.Tickets {
			tids = append(tids, ticket.GetId())
		}
		req := &pb.AssignTicketsRequest{
			Assignments: []*pb.AssignmentGroup{
				{
					TicketIds:  tids,
					Assignment: &pb.Assignment{Connection: "agones-1"},
				},
			},
		}

		gotAtResp, err = be.AssignTickets(ctx, req, grpc.WaitForReady(true))
		require.Nil(t, err)
		require.NotNil(t, gotAtResp)
	}

	// 6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	matches = []*pb.Match{MustMakeMatch(ticket1)}
	validateFetchMatchesResponse(ctx, t, matches, be, fmReq)

	// 7. Call frontend.DeleteTicket to delete the tickets returned in step 6.
	_, err = fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticket1.GetId()}, grpc.WaitForReady(true))
	require.Nil(t, err)

	// 8. Call backend.FetchMatches and verify the response does not contain the tickets got deleted.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	matches = []*pb.Match{}
	validateFetchMatchesResponse(ctx, t, matches, be, fmReq)
}

func validateFetchMatchesResponse(ctx context.Context, t *testing.T, expectedMatches []*pb.Match, be pb.BackendServiceClient, fmReq *pb.FetchMatchesRequest) {
	stream, err := be.FetchMatches(ctx, fmReq, grpc.WaitForReady(true))
	require.Nil(t, err)
	matches := make([]*pb.Match, 0)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		require.Nil(t, err)
		matches = append(matches, resp.GetMatch())
	}

	require.ElementsMatch(t, expectedMatches, matches)
}

func MustMakeMatch(tickets ...*pb.Ticket) *pb.Match {
	m, err := mmf.MakeMatch("test-profile", tickets...)
	if err != nil {
		panic(err)
	}
	return m
}
