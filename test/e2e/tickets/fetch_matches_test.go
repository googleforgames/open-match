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

package tickets

import (
	"fmt"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestGameMatchWorkFlow(t *testing.T) {
	/*
		This end to end test does the following things step by step
		1. Create a few tickets with delicate designs and hand crafted properties
		2. Call backend.FetchMatches and verify it returns expected matches.
		3. Call backend.FetchMatches within redis.ignoreLists.ttl seconds and expect empty response.
		4. Wait for redis.ignoreLists.ttl seconds and call backend.FetchMatches the third time, expect the same result as step 2.
		5. Call backend.AssignTickets to assign DGSs for the tickets in FetchMatches' response
		6. Call backend.FetchMatches and verify it no longer returns tickets got assigned in the previous step.
		7. Call frontend.DeleteTicket to delete part of the tickets that got assigned.
		8. Call backend.AssignTickets to the tickets got deleted and expect an error.
	*/

	om, closer := e2e.New(t)
	defer closer()
	fe := om.MustFrontendGRPC()
	be := om.MustBackendGRPC()
	mmfCfg := om.MustMmfConfigGRPC()

	ticket1 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
			},
		},
	}

	ticket2 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
			},
		},
	}

	ticket3 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
			},
		},
	}

	ticket4 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 15}},
			},
		},
	}

	ticket5 := &pb.Ticket{
		Properties: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
				"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 20}},
			},
		},
	}

	tickets := []*pb.Ticket{ticket1, ticket2, ticket3, ticket4, ticket5}

	// 1. Create a few tickets with delicate designs and hand crafted properties
	for _, ticket := range tickets {
		ctResp, err := fe.CreateTicket(om.Context(), &pb.CreateTicketRequest{Ticket: ticket})
		assert.Nil(t, err)
		assert.NotNil(t, ctResp)
	}

	fmResp, err := be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{
		Config: mmfCfg,
		Profile: []*pb.MatchProfile{
			{
				Name: "test-profile",
				Pool: []*pb.Pool{
					{
						Name:   "ticket12",
						Filter: []*pb.Filter{{Attribute: "level", Min: 0, Max: 6}, {Attribute: "defense", Min: 0, Max: 100}},
					},
					{
						Name:   "ticket23",
						Filter: []*pb.Filter{{Attribute: "level", Min: 3, Max: 15}, {Attribute: "defense", Min: 0, Max: 100}},
					},
					{
						Name:   "ticket5",
						Filter: []*pb.Filter{{Attribute: "level", Min: 0, Max: 100}, {Attribute: "defense", Min: 17, Max: 25}},
					},
					{
						Name:   "ticket234",
						Filter: []*pb.Filter{{Attribute: "level", Min: 3, Max: 17}, {Attribute: "defense", Min: 3, Max: 17}},
					},
				},
			},
		},
	})
	assert.Nil(t, err)
	fmt.Println(fmResp)
	assert.NotNil(t, fmResp)
}
