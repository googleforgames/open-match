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
	"io"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	internalTesting "open-match.dev/open-match/internal/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	e2eTesting "open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestQueryTickets(t *testing.T) {
	tests := []struct {
		description   string
		pool          *pb.Pool
		gotTickets    []*pb.Ticket
		wantCode      codes.Code
		wantTickets   []*pb.Ticket
		wantPageCount int
	}{
		{
			description:   "expects invalid argument code since pool is empty",
			pool:          nil,
			wantCode:      codes.InvalidArgument,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since the store is empty",
			gotTickets:  []*pb.Ticket{},
			pool: &pb.Pool{
				FloatRangeFilters: []*pb.FloatRangeFilter{{
					Attribute: "ok",
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since all tickets in the store are filtered out",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.AttributeMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.AttributeLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				FloatRangeFilters: []*pb.FloatRangeFilter{{
					Attribute: e2e.AttributeDefense,
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with 5 tickets with e2e.FloatRangeAttribute1=2 and e2e.FloatRangeAttribute2 in range of [0,10)",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.AttributeMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.AttributeLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				FloatRangeFilters: []*pb.FloatRangeFilter{{
					Attribute: e2e.AttributeMMR,
					Min:       1,
					Max:       3,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.AttributeMMR, Min: 2, Max: 3, Interval: 2},
				internalTesting.Property{Name: e2e.AttributeLevel, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 1,
		},
		{
			// Test inclusive filters and paging works as expected
			description: "expects response with 15 tickets with e2e.FloatRangeAttribute1=2,4,6 and e2e.FloatRangeAttribute2=[0,10)",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.AttributeMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.AttributeLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				FloatRangeFilters: []*pb.FloatRangeFilter{{
					Attribute: e2e.AttributeMMR,
					Min:       2,
					Max:       6,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.AttributeMMR, Min: 2, Max: 7, Interval: 2},
				internalTesting.Property{Name: e2e.AttributeLevel, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 2,
		},
		{
			// Test BoolEquals can ignore falsely typed values mapped to bool index
			description: "expects 1 ticket with property e2eTesting.ModeDemo maps to true",
			gotTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_BoolValue{BoolValue: true}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_BoolValue{BoolValue: false}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_StringValue{StringValue: "true"}},
						},
					},
				},
			},
			pool: &pb.Pool{
				BoolEqualsFilters: []*pb.BoolEqualsFilter{{
					Attribute: e2e.ModeDemo,
					Value:     true,
				}},
			},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_BoolValue{BoolValue: true}},
						},
					},
				},
			},
			wantPageCount: 1,
		},
		{
			// Test StringEquals works as expected
			description: "expects 1 ticket with property e2eTesting.ModeDemo maps to true",
			gotTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_StringValue{StringValue: "true"}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_StringValue{StringValue: "false"}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_BoolValue{BoolValue: true}},
						},
					},
				},
			},
			pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						Attribute: e2e.RoleCleric,
						Value:     "true",
					},
				},
			},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_StringValue{StringValue: "true"}},
						},
					},
				},
			},
			wantPageCount: 1,
		},
		{
			// Test all tickets
			description: "expects all 3 tickets with no filters",
			gotTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_StringValue{StringValue: "true"}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_BoolValue{BoolValue: true}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.AttributeMMR: {Kind: &structpb.Value_NumberValue{NumberValue: 100}},
						},
					},
				},
			},
			pool:     &pb.Pool{},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.RoleCleric: {Kind: &structpb.Value_StringValue{StringValue: "true"}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.ModeDemo: {Kind: &structpb.Value_BoolValue{BoolValue: true}},
						},
					},
				},
				{
					Properties: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							e2eTesting.AttributeMMR: {Kind: &structpb.Value_NumberValue{NumberValue: 100}},
						},
					},
				},
			},
			wantPageCount: 1,
		},
	}

	t.Run("TestQueryTickets", func(t *testing.T) {
		for _, test := range tests {
			test := test
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()

				om, closer := e2e.New(t)
				defer closer()
				fe := om.MustFrontendGRPC()
				mml := om.MustMmLogicGRPC()
				pageCounts := 0
				ctx := om.Context()

				for _, ticket := range test.gotTickets {
					resp, err := fe.CreateTicket(ctx, &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}

				stream, err := mml.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: test.pool})
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
