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

package backend

import (
	"context"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestDoAssignTickets(t *testing.T) {
	fakeProperty := "test-property"
	fakeTickets := []*pb.Ticket{
		{
			Id: "1",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					fakeProperty: 1,
				},
			},
		},
		{
			Id: "2",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					fakeProperty: 2,
				},
			},
		},
	}

	tests := []struct {
		description    string
		preAction      func(context.Context, context.CancelFunc, statestore.Service)
		req            *pb.AssignTicketsRequest
		wantCode       codes.Code
		wantAssignment *pb.Assignment
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			req: &pb.AssignTicketsRequest{
				TicketIds:  []string{"1"},
				Assignment: &pb.Assignment{},
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect invalid argument code since assignment is nil",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			req:      &pb.AssignTicketsRequest{},
			wantCode: codes.InvalidArgument,
		},
		{
			description: "expect not found code since ticket does not exist",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			req: &pb.AssignTicketsRequest{
				TicketIds: []string{"1", "2"},
				Assignment: &pb.Assignment{
					Connection: "123",
				},
			},
			wantCode: codes.NotFound,
		},
		{
			description: "expect ok code",
			preAction: func(ctx context.Context, cancel context.CancelFunc, store statestore.Service) {
				for _, fakeTicket := range fakeTickets {
					store.CreateTicket(ctx, fakeTicket)
					store.IndexTicket(ctx, fakeTicket)
				}
				// Make sure tickets are correctly indexed.
				var wantFilteredTickets []*pb.Ticket
				pool := &pb.Pool{
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: fakeProperty, Min: 0, Max: 3}},
				}
				err := store.FilterTickets(ctx, pool, 10, func(filterTickets []*pb.Ticket) error {
					wantFilteredTickets = filterTickets
					return nil
				})
				assert.Nil(t, err)
				assert.Equal(t, len(fakeTickets), len(wantFilteredTickets))
			},
			req: &pb.AssignTicketsRequest{
				TicketIds: []string{"1", "2"},
				Assignment: &pb.Assignment{
					Connection: "123",
				},
			},
			wantCode: codes.OK,
			wantAssignment: &pb.Assignment{
				Connection: "123",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			cfg := viper.New()
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
			defer closer()

			test.preAction(ctx, cancel, store)

			err := doAssignTickets(ctx, test.req, store)

			assert.Equal(t, test.wantCode, status.Convert(err).Code())

			if err == nil {
				for _, id := range test.req.GetTicketIds() {
					ticket, err := store.GetTicket(ctx, id)
					assert.Nil(t, err)
					assert.Equal(t, test.wantAssignment, ticket.GetAssignment())
				}

				// Make sure tickets are deindexed after assignment
				var wantFilteredTickets []*pb.Ticket
				pool := &pb.Pool{
					DoubleRangeFilters: []*pb.DoubleRangeFilter{{DoubleArg: fakeProperty, Min: 0, Max: 2}},
				}
				store.FilterTickets(ctx, pool, 10, func(filterTickets []*pb.Ticket) error {
					wantFilteredTickets = filterTickets
					return nil
				})
				assert.Nil(t, wantFilteredTickets)
			}
		})
	}
}

// TODOs: add unit tests to doFetchMatchesFilterSkiplistIds and doFetchMatchesAddSkiplistIds
