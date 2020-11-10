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

package frontend

import (
	"context"
	"errors"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestDoCreateTickets(t *testing.T) {
	cfg := viper.New()

	tests := []struct {
		description string
		preAction   func(cancel context.CancelFunc)
		ticket      *pb.Ticket
		wantCode    codes.Code
	}{
		{
			description: "expect error with canceled context",
			preAction:   func(cancel context.CancelFunc) { cancel() },
			ticket: &pb.Ticket{
				SearchFields: &pb.SearchFields{
					DoubleArgs: map[string]float64{
						"test-arg": 1,
					},
				},
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect normal return with default context",
			preAction:   func(_ context.CancelFunc) {},
			ticket: &pb.Ticket{
				SearchFields: &pb.SearchFields{
					DoubleArgs: map[string]float64{
						"test-arg": 1,
					},
				},
			},
			wantCode: codes.OK,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
			defer closer()

			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			test.preAction(cancel)

			res, err := doCreateTicket(ctx, &pb.CreateTicketRequest{Ticket: test.ticket}, store)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())
			if err == nil {
				matched, err := regexp.MatchString(`[0-9a-v]{20}`, res.GetId())
				require.True(t, matched)
				require.Nil(t, err)
				require.Equal(t, test.ticket.SearchFields.DoubleArgs["test-arg"], res.SearchFields.DoubleArgs["test-arg"])
			}
		})
	}
}

func TestDoCreateBackfill(t *testing.T) {
	cfg := viper.New()

	tests := []struct {
		description string
		preAction   func(cancel context.CancelFunc)
		backfill    *pb.Backfill
		wantCode    codes.Code
	}{
		{
			description: "expect error with canceled context",
			preAction:   func(cancel context.CancelFunc) { cancel() },
			backfill: &pb.Backfill{
				SearchFields: &pb.SearchFields{
					DoubleArgs: map[string]float64{
						"test-arg": 1,
					},
				},
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect normal return with default context",
			preAction:   func(_ context.CancelFunc) {},
			backfill: &pb.Backfill{
				SearchFields: &pb.SearchFields{
					DoubleArgs: map[string]float64{
						"test-arg": 1,
					},
				},
			},
			wantCode: codes.OK,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
			defer closer()

			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			test.preAction(cancel)

			res, err := doCreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: test.backfill}, store)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())
			if err == nil {
				matched, err := regexp.MatchString(`[0-9a-v]{20}`, res.GetId())
				require.True(t, matched)
				require.Nil(t, err)
				require.Equal(t, test.backfill.SearchFields.DoubleArgs["test-arg"], res.SearchFields.DoubleArgs["test-arg"])
			}
		})
	}
}

func TestDoWatchAssignments(t *testing.T) {
	testTicket := &pb.Ticket{
		Id: "test-id",
	}

	senderGenerator := func(tmp []*pb.Assignment, stopCount int) func(*pb.Assignment) error {
		return func(assignment *pb.Assignment) error {
			tmp = append(tmp, assignment)
			if len(tmp) == stopCount {
				return errors.New("some error")
			}
			return nil
		}
	}

	tests := []struct {
		description     string
		preAction       func(context.Context, *testing.T, statestore.Service, []*pb.Assignment, *sync.WaitGroup)
		wantCode        codes.Code
		wantAssignments []*pb.Assignment
	}{
		{
			description:     "expect error because ticket id does not exist",
			preAction:       func(_ context.Context, _ *testing.T, _ statestore.Service, _ []*pb.Assignment, _ *sync.WaitGroup) {},
			wantCode:        codes.NotFound,
			wantAssignments: []*pb.Assignment{},
		},
		{
			description: "expect two assignment reads from preAction writes and fail in grpc aborted code",
			preAction: func(ctx context.Context, t *testing.T, store statestore.Service, wantAssignments []*pb.Assignment, wg *sync.WaitGroup) {
				require.Nil(t, store.CreateTicket(ctx, testTicket))

				go func(wg *sync.WaitGroup) {
					for i := 0; i < len(wantAssignments); i++ {
						time.Sleep(50 * time.Millisecond)
						_, _, err := store.UpdateAssignments(ctx, &pb.AssignTicketsRequest{
							Assignments: []*pb.AssignmentGroup{
								{
									TicketIds:  []string{testTicket.GetId()},
									Assignment: wantAssignments[i],
								},
							},
						})
						require.Nil(t, err)
						wg.Done()
					}
				}(wg)
			},
			wantCode:        codes.Aborted,
			wantAssignments: []*pb.Assignment{{Connection: "1"}, {Connection: "2"}},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(len(test.wantAssignments))
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			ctx := utilTesting.NewContext(t)

			gotAssignments := []*pb.Assignment{}

			test.preAction(ctx, t, store, test.wantAssignments, &wg)
			err := doWatchAssignments(ctx, testTicket.GetId(), senderGenerator(gotAssignments, len(test.wantAssignments)), store)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())

			wg.Wait()
			for i := 0; i < len(gotAssignments); i++ {
				require.Equal(t, gotAssignments[i], test.wantAssignments[i])
			}
		})
	}
}

func TestDoDeleteTicket(t *testing.T) {
	fakeTicket := &pb.Ticket{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}

	tests := []struct {
		description string
		preAction   func(context.Context, context.CancelFunc, statestore.Service)
		wantCode    codes.Code
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect ok code since delete ticket does not care about if ticket exists or not",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			wantCode:    codes.OK,
		},
		{
			description: "expect ok code",
			preAction: func(ctx context.Context, _ context.CancelFunc, store statestore.Service) {
				store.CreateTicket(ctx, fakeTicket)
				store.IndexTicket(ctx, fakeTicket)
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)

			err := doDeleteTicket(ctx, fakeTicket.GetId(), store)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())
		})
	}
}

func TestDoGetTicket(t *testing.T) {
	fakeTicket := &pb.Ticket{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}

	tests := []struct {
		description string
		preAction   func(context.Context, context.CancelFunc, statestore.Service)
		wantTicket  *pb.Ticket
		wantCode    codes.Code
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect not found code since ticket does not exist",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			wantCode:    codes.NotFound,
		},
		{
			description: "expect ok code with output ticket equivalent to fakeTicket",
			preAction: func(ctx context.Context, _ context.CancelFunc, store statestore.Service) {
				store.CreateTicket(ctx, fakeTicket)
				store.IndexTicket(ctx, fakeTicket)
			},
			wantCode:   codes.OK,
			wantTicket: fakeTicket,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)

			ticket, err := store.GetTicket(ctx, fakeTicket.GetId())
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())

			if err == nil {
				require.Equal(t, test.wantTicket.GetId(), ticket.GetId())
				require.Equal(t, test.wantTicket.SearchFields.DoubleArgs, ticket.SearchFields.DoubleArgs)
			}
		})
	}
}

func TestDoGetBackfill(t *testing.T) {
	fakeBackfill := &pb.Backfill{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}

	tests := []struct {
		description string
		preAction   func(context.Context, context.CancelFunc, statestore.Service)
		wantTicket  *pb.Backfill
		wantCode    codes.Code
	}{
		{
			description: "expect unavailable code since context is canceled before being called",
			preAction: func(_ context.Context, cancel context.CancelFunc, _ statestore.Service) {
				cancel()
			},
			wantCode: codes.Unavailable,
		},
		{
			description: "expect not found code since ticket does not exist",
			preAction:   func(_ context.Context, _ context.CancelFunc, _ statestore.Service) {},
			wantCode:    codes.NotFound,
		},
		{
			description: "expect ok code with output ticket equivalent to fakeBackfill",
			preAction: func(ctx context.Context, _ context.CancelFunc, store statestore.Service) {
				store.CreateBackfill(ctx, fakeBackfill, []string{})
			},
			wantCode:   codes.OK,
			wantTicket: fakeBackfill,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(utilTesting.NewContext(t))
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)

			backfill, l, err := store.GetBackfill(ctx, fakeBackfill.GetId())
			require.Empty(t, l)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())

			if err == nil {
				require.Equal(t, test.wantTicket.GetId(), backfill.GetId())
				require.Equal(t, test.wantTicket.SearchFields.DoubleArgs, backfill.SearchFields.DoubleArgs)
			}
		})
	}
}
