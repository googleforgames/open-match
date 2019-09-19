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

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/assert"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestGameMatchWorkFlow(t *testing.T) {
	/*
		This end to end test does the following things step by step
		1. Create a few tickets with delicate designs and hand crafted properties
		2. Call backend.FetchMatches and verify it returns expected matches.
		3. Call backend.FetchMatches within redis.ignoreLists.ttl seconds and expects it return a match with duplicate tickets in step 2.
		4. Wait for redis.ignoreLists.ttl seconds and call backend.FetchMatches the third time, expect the same result as step 2.
		5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
		6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
		7. Call frontend.DeleteTicket to delete the tickets returned in step 6.
		8. Call backend.FetchMatches and verify the response does not contain the tickets got deleted.
	*/

	om, closer := e2e.New(t)
	defer closer()
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	mmfCfg := om.MustMmfConfigGRPC()
	ctx := om.Context()

	ticket1 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				e2e.AttributeMMR:   {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				e2e.AttributeLevel: {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	ticket2 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				e2e.AttributeMMR:   {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				e2e.AttributeLevel: {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
			},
		},
	}

	ticket3 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				e2e.AttributeMMR:   {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
				e2e.AttributeLevel: {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
			},
		},
	}

	ticket4 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				e2e.AttributeMMR:   {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
				e2e.AttributeLevel: {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
			},
		},
	}

	ticket5 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				e2e.AttributeMMR:   {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
				e2e.AttributeLevel: {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
			},
		},
	}

	tickets := []*pb.Ticket{ticket1, ticket2, ticket3, ticket4, ticket5}

	var err error
	// 1. Create a few tickets with delicate designs and hand crafted properties
	for i := 0; i < len(tickets); i++ {
		var ctResp *pb.CreateTicketResponse
		ctResp, err = fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: tickets[i]})
		assert.Nil(t, err)
		assert.NotNil(t, ctResp)
		// Assign Open Match ids back to the input tickets
		*tickets[i] = *ctResp.GetTicket()
	}

	fmReq := &pb.FetchMatchesRequest{
		Config: mmfCfg,
		Profiles: []*pb.MatchProfile{
			{
				Name: "test-profile",
				Pools: []*pb.Pool{
					{
						Name:              "ticket12",
						FloatRangeFilters: []*pb.FloatRangeFilter{{Attribute: e2e.AttributeMMR, Min: 0, Max: 6}, {Attribute: e2e.AttributeLevel, Min: 0, Max: 100}},
					},
					{
						Name:              "ticket23",
						FloatRangeFilters: []*pb.FloatRangeFilter{{Attribute: e2e.AttributeMMR, Min: 3, Max: 10}, {Attribute: e2e.AttributeLevel, Min: 0, Max: 100}},
					},
					{
						Name:              "ticket5",
						FloatRangeFilters: []*pb.FloatRangeFilter{{Attribute: e2e.AttributeMMR, Min: 0, Max: 100}, {Attribute: e2e.AttributeLevel, Min: 17, Max: 25}},
					},
					{
						Name:              "ticket234",
						FloatRangeFilters: []*pb.FloatRangeFilter{{Attribute: e2e.AttributeMMR, Min: 3, Max: 17}, {Attribute: e2e.AttributeLevel, Min: 3, Max: 17}},
					},
				},
			},
		},
	}

	// 2. Call backend.FetchMatches and expects two matches with the following tickets
	var wantTickets = [][]*pb.Ticket{{ticket2, ticket3, ticket4}, {ticket5}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 3. Call backend.FetchMatches within redis.ignoreLists.ttl seconds and expects it return a match with ticket1 .
	wantTickets = [][]*pb.Ticket{{ticket1}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 4. Wait for redis.ignoreLists.ttl seconds and call backend.FetchMatches the third time, expect the same result as step 2.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*pb.Ticket{{ticket2, ticket3, ticket4}, {ticket5}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
	var gotAtResp *pb.AssignTicketsResponse
	for _, tickets := range wantTickets {
		tids := []string{}
		for _, ticket := range tickets {
			tids = append(tids, ticket.GetId())
		}
		gotAtResp, err = be.AssignTickets(ctx, &pb.AssignTicketsRequest{TicketIds: tids, Assignment: &pb.Assignment{Connection: "agones-1"}})
		assert.Nil(t, err)
		assert.NotNil(t, gotAtResp)
	}

	// 6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*pb.Ticket{{ticket1}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 7. Call frontend.DeleteTicket to delete the tickets returned in step 6.
	var gotDtResp *pb.DeleteTicketResponse
	gotDtResp, err = fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticket1.GetId()})
	assert.Nil(t, err)
	assert.NotNil(t, gotDtResp)

	// 8. Call backend.FetchMatches and verify the response does not contain the tickets got deleted.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*pb.Ticket{}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)
}

func validateFetchMatchesResponse(ctx context.Context, t *testing.T, wantTickets [][]*pb.Ticket, be pb.BackendClient, fmReq *pb.FetchMatchesRequest) {
	stream, err := be.FetchMatches(ctx, fmReq)
	assert.Nil(t, err)
	matches := make([]*pb.Match, 0)
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		assert.Nil(t, err)
		matches = append(matches, resp.GetMatch())
	}

	assert.Equal(t, len(wantTickets), len(matches))
	for _, match := range matches {
		assert.Contains(t, wantTickets, match.GetTickets())
	}
}
