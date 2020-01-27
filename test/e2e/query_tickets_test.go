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
				DoubleRangeFilters: []*pb.DoubleRangeFilter{{
					DoubleArg: "ok",
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since all tickets in the store are filtered out",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.DoubleArgMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.DoubleArgLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{{
					DoubleArg: e2e.DoubleArgDefense,
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with 5 tickets with e2e.FloatRangeDoubleArg1=2 and e2e.FloatRangeDoubleArg2 in range of [0,10)",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.DoubleArgMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.DoubleArgLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{{
					DoubleArg: e2e.DoubleArgMMR,
					Min:       1,
					Max:       3,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.DoubleArgMMR, Min: 2, Max: 3, Interval: 2},
				internalTesting.Property{Name: e2e.DoubleArgLevel, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 1,
		},
		{
			// Test inclusive filters and paging works as expected
			description: "expects response with 15 tickets with e2e.FloatRangeDoubleArg1=2,4,6 and e2e.FloatRangeDoubleArg2=[0,10)",
			gotTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.DoubleArgMMR, Min: 0, Max: 10, Interval: 2},
				internalTesting.Property{Name: e2e.DoubleArgLevel, Min: 0, Max: 10, Interval: 2},
			),
			pool: &pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{{
					DoubleArg: e2e.DoubleArgMMR,
					Min:       2,
					Max:       6,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateFloatRangeTickets(
				internalTesting.Property{Name: e2e.DoubleArgMMR, Min: 2, Max: 7, Interval: 2},
				internalTesting.Property{Name: e2e.DoubleArgLevel, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 2,
		},
		{
			description: "expects 1 ticket with tag e2eTesting.ModeDemo",
			gotTickets: []*pb.Ticket{
				{
					SearchFields: &pb.SearchFields{
						Tags: []string{
							e2eTesting.ModeDemo,
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						Tags: []string{
							"Foo",
						},
					},
				},
			},
			pool: &pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{{
					Tag: e2e.ModeDemo,
				}},
			},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{
					SearchFields: &pb.SearchFields{
						Tags: []string{
							e2eTesting.ModeDemo,
						},
					},
				},
			},
			wantPageCount: 1,
		},
		{
			// Test StringEquals works as expected
			description: "expects 1 ticket with property e2eTesting.Role maps to warrior",
			gotTickets: []*pb.Ticket{
				{
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							e2eTesting.Role: "warrior",
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							e2eTesting.Role: "rogue",
						},
					},
				},
			},
			pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: e2e.Role,
						Value:     "warrior",
					},
				},
			},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{

					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							e2eTesting.Role: "warrior",
						},
					},
				},
			},
			wantPageCount: 1,
		},
		{
			// Test all tickets
			description: "expects all 3 tickets when passing in a pool with no filters",
			gotTickets: []*pb.Ticket{
				{
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							e2eTesting.Role: "warrior",
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						Tags: []string{
							e2eTesting.ModeDemo,
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						DoubleArgs: map[string]float64{
							e2eTesting.DoubleArgMMR: 100,
						},
					},
				},
			},
			pool:     &pb.Pool{},
			wantCode: codes.OK,
			wantTickets: []*pb.Ticket{
				{
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							e2eTesting.Role: "warrior",
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						Tags: []string{
							e2eTesting.ModeDemo,
						},
					},
				},
				{
					SearchFields: &pb.SearchFields{
						DoubleArgs: map[string]float64{
							e2eTesting.DoubleArgMMR: 100,
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
				mml := om.MustQueryServiceGRPC()
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
					assert.Equal(t, test.wantTickets[i].GetSearchFields(), actualTickets[i].GetSearchFields())
				}
				assert.Equal(t, test.wantPageCount, pageCounts)
			})
		}
	})
}
