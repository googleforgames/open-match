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

	tickets := []*pb.Ticket{
		{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 100}},
				},
			},
		},
		{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level":  {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"attack": {Kind: &structpb.Value_NumberValue{NumberValue: 50}},
				},
			},
		},
		{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"speed": {Kind: &structpb.Value_NumberValue{NumberValue: 522}},
				},
			},
		},
		{
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"mana":  {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				},
			},
		},
	}

	// 1. Create a few tickets with delicate designs and hand crafted properties
	for _, ticket := range tickets {
		ctResp, err := fe.CreateTicket(om.Context(), &pb.CreateTicketRequest{Ticket: ticket})
		assert.Nil(t, err)
		assert.NotNil(t, ctResp)
	}

	_, err := be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{
		Config: mmfCfg,
		Profile: []*pb.MatchProfile{
			{
				Name: "test-profile",
			},
		},
	})
	assert.Nil(t, err)
}
