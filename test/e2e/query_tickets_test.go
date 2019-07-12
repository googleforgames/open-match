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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	internalTesting "open-match.dev/open-match/internal/testing"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

func TestQueryTickets(t *testing.T) {
	tests := []struct {
		description   string
		pool          *pb.Pool
		preAction     func(fe pb.FrontendClient, t *testing.T)
		wantCode      codes.Code
		wantTickets   []*pb.Ticket
		wantPageCount int
	}{
		{
			description:   "expects invalid argument code since pool is empty",
			preAction:     func(_ pb.FrontendClient, _ *testing.T) {},
			pool:          nil,
			wantCode:      codes.InvalidArgument,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since the store is empty",
			preAction:   func(_ pb.FrontendClient, _ *testing.T) {},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: "ok",
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with no tickets since all tickets in the store are filtered out",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.SkillAttribute,
				}},
			},
			wantCode:      codes.OK,
			wantTickets:   nil,
			wantPageCount: 0,
		},
		{
			description: "expects response with 5 tickets with e2e.Map1Attribute=2 and e2e.Map2Attribute in range of [0,10)",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},
			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.Map1Attribute,
					Min:       1,
					Max:       3,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateTickets(
				internalTesting.Property{Name: e2e.Map1Attribute, Min: 2, Max: 3, Interval: 2},
				internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 1,
		},
		{
			// Test inclusive filters and paging works as expected
			description: "expects response with 15 tickets with e2e.Map1Attribute=2,4,6 and e2e.Map2Attribute=[0,10)",
			preAction: func(fe pb.FrontendClient, t *testing.T) {
				tickets := internalTesting.GenerateTickets(
					internalTesting.Property{Name: e2e.Map1Attribute, Min: 0, Max: 10, Interval: 2},
					internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
				)

				for _, ticket := range tickets {
					resp, err := fe.CreateTicket(context.Background(), &pb.CreateTicketRequest{Ticket: ticket})
					assert.NotNil(t, resp)
					assert.Nil(t, err)
				}
			},

			pool: &pb.Pool{
				Filters: []*pb.Filter{{
					Attribute: e2e.Map1Attribute,
					Min:       2,
					Max:       6,
				}},
			},
			wantCode: codes.OK,
			wantTickets: internalTesting.GenerateTickets(
				internalTesting.Property{Name: e2e.Map1Attribute, Min: 2, Max: 7, Interval: 2},
				internalTesting.Property{Name: e2e.Map2Attribute, Min: 0, Max: 10, Interval: 2},
			),
			wantPageCount: 2,
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

				test.preAction(fe, t)

				stream, err := mml.QueryTickets(om.Context(), &pb.QueryTicketsRequest{Pool: test.pool})
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
