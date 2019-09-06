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
	"open-match.dev/open-match/pkg/gopb"
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

	ticket1 := &gopb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	ticket2 := &gopb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
			},
		},
	}

	ticket3 := &gopb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
			},
		},
	}

	ticket4 := &gopb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
			},
		},
	}

	ticket5 := &gopb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
			},
		},
	}

	tickets := []*gopb.Ticket{ticket1, ticket2, ticket3, ticket4, ticket5}

	var err error
	// 1. Create a few tickets with delicate designs and hand crafted properties
	for i := 0; i < len(tickets); i++ {
		var ctResp *gopb.CreateTicketResponse
		ctResp, err = fe.CreateTicket(ctx, &gopb.CreateTicketRequest{Ticket: tickets[i]})
		assert.Nil(t, err)
		assert.NotNil(t, ctResp)
		// Assign Open Match ids back to the input tickets
		*tickets[i] = *ctResp.GetTicket()
	}

	fmReq := &gopb.FetchMatchesRequest{
		Config: mmfCfg,
		Profiles: []*gopb.MatchProfile{
			{
				Name: "test-profile",
				Pools: []*gopb.Pool{
					{
						Name:    "ticket12",
						Filters: []*gopb.Filter{{Attribute: "level", Min: 0, Max: 6}, {Attribute: "defense", Min: 0, Max: 100}},
					},
					{
						Name:    "ticket23",
						Filters: []*gopb.Filter{{Attribute: "level", Min: 3, Max: 10}, {Attribute: "defense", Min: 0, Max: 100}},
					},
					{
						Name:    "ticket5",
						Filters: []*gopb.Filter{{Attribute: "level", Min: 0, Max: 100}, {Attribute: "defense", Min: 17, Max: 25}},
					},
					{
						Name:    "ticket234",
						Filters: []*gopb.Filter{{Attribute: "level", Min: 3, Max: 17}, {Attribute: "defense", Min: 3, Max: 17}},
					},
				},
			},
		},
	}

	// 2. Call backend.FetchMatches and expects two matches with the following tickets
	var wantTickets = [][]*gopb.Ticket{{ticket2, ticket3, ticket4}, {ticket5}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 3. Call backend.FetchMatches within redis.ignoreLists.ttl seconds and expects it return a match with ticket1 .
	wantTickets = [][]*gopb.Ticket{{ticket1}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 4. Wait for redis.ignoreLists.ttl seconds and call backend.FetchMatches the third time, expect the same result as step 2.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*gopb.Ticket{{ticket2, ticket3, ticket4}, {ticket5}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
	var gotAtResp *gopb.AssignTicketsResponse
	for _, tickets := range wantTickets {
		tids := []string{}
		for _, ticket := range tickets {
			tids = append(tids, ticket.GetId())
		}
		gotAtResp, err = be.AssignTickets(ctx, &gopb.AssignTicketsRequest{TicketIds: tids, Assignment: &gopb.Assignment{Connection: "agones-1"}})
		assert.Nil(t, err)
		assert.NotNil(t, gotAtResp)
	}

	// 6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*gopb.Ticket{{ticket1}}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)

	// 7. Call frontend.DeleteTicket to delete the tickets returned in step 6.
	var gotDtResp *gopb.DeleteTicketResponse
	gotDtResp, err = fe.DeleteTicket(ctx, &gopb.DeleteTicketRequest{TicketId: ticket1.GetId()})
	assert.Nil(t, err)
	assert.NotNil(t, gotDtResp)

	// 8. Call backend.FetchMatches and verify the response does not contain the tickets got deleted.
	time.Sleep(statestoreTesting.IgnoreListTTL)
	wantTickets = [][]*gopb.Ticket{}
	validateFetchMatchesResponse(ctx, t, wantTickets, be, fmReq)
}

func validateFetchMatchesResponse(ctx context.Context, t *testing.T, wantTickets [][]*gopb.Ticket, be gopb.BackendClient, fmReq *gopb.FetchMatchesRequest) {
	stream, err := be.FetchMatches(ctx, fmReq)
	assert.Nil(t, err)
	matches := make([]*gopb.Match, 0)
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
