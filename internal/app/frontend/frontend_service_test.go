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

	"github.com/golang/protobuf/ptypes"
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
				require.NoError(t, err)
				require.Equal(t, test.ticket.SearchFields.DoubleArgs["test-arg"], res.SearchFields.DoubleArgs["test-arg"])
			}
		})
	}
}

func TestCreateBackfill(t *testing.T) {
	cfg := viper.New()
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()
	ctx := utilTesting.NewContext(t)
	fs := frontendService{cfg, store}
	var testCases = []struct {
		description     string
		request         *pb.CreateBackfillRequest
		result          *pb.Backfill
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "nil request check",
			request:         nil,
			expectedCode:    codes.InvalidArgument,
			expectedMessage: "request is nil",
		},
		{
			description:     "nil backfill - error is returned",
			request:         &pb.CreateBackfillRequest{Backfill: nil},
			expectedCode:    codes.InvalidArgument,
			expectedMessage: ".backfill is required",
		},
		{
			description:     "createTime should not exist in input",
			request:         &pb.CreateBackfillRequest{Backfill: &pb.Backfill{CreateTime: ptypes.TimestampNow()}},
			expectedCode:    codes.InvalidArgument,
			expectedMessage: "backfills cannot be created with create time set",
		},
		{
			description:     "empty Backfill, no errors",
			request:         &pb.CreateBackfillRequest{Backfill: &pb.Backfill{}},
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
		{
			description: "normal backfill",
			request: &pb.CreateBackfillRequest{
				Backfill: &pb.Backfill{
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							"search": "me",
						}}}},
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			res, err := fs.CreateBackfill(ctx, tc.request)
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode.String(), status.Convert(err).Code().String())
				require.Contains(t, status.Convert(err).Message(), tc.expectedMessage)
			}
		})
	}

	// expect error with canceled context
	store, closer = statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()
	fs = frontendService{cfg, store}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err := fs.CreateBackfill(ctx, &pb.CreateBackfillRequest{Backfill: &pb.Backfill{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}})
	require.NotNil(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Nil(t, res)
}

func TestUpdateBackfill(t *testing.T) {
	cfg := viper.New()
	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()
	ctx := utilTesting.NewContext(t)
	fs := frontendService{cfg, store}
	res, err := fs.CreateBackfill(ctx, &pb.CreateBackfillRequest{
		Backfill: &pb.Backfill{
			SearchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"search": "me",
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, res)

	var testCases = []struct {
		description     string
		request         *pb.UpdateBackfillRequest
		result          *pb.Backfill
		expectedCode    codes.Code
		expectedMessage string
	}{
		{
			description:     "nil request check",
			request:         nil,
			expectedCode:    codes.InvalidArgument,
			expectedMessage: "request is nil",
		},
		{
			description:     "nil backfill - error is returned",
			request:         &pb.UpdateBackfillRequest{Backfill: nil},
			expectedCode:    codes.InvalidArgument,
			expectedMessage: ".backfill is required",
		},
		{
			description:     "empty Backfill, error with no backfill ID",
			request:         &pb.UpdateBackfillRequest{Backfill: &pb.Backfill{}},
			expectedCode:    codes.InvalidArgument,
			expectedMessage: "backfill ID should exist",
		},
		{
			description: "normal backfill",
			request: &pb.UpdateBackfillRequest{
				Backfill: &pb.Backfill{
					Id: res.Id,
					SearchFields: &pb.SearchFields{
						StringArgs: map[string]string{
							"search": "me",
						}}}},
			expectedCode:    codes.OK,
			expectedMessage: "",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.description, func(t *testing.T) {
			res, err = fs.UpdateBackfill(ctx, tc.request)
			if tc.expectedCode == codes.OK {
				require.NoError(t, err)
				require.NotNil(t, res)
				require.Equal(t, tc.request.Backfill.SearchFields.DoubleArgs, res.SearchFields.DoubleArgs)
			} else {
				require.Error(t, err)
				require.Equal(t, tc.expectedCode.String(), status.Convert(err).Code().String())
				require.Contains(t, status.Convert(err).Message(), tc.expectedMessage)
			}
		})
	}

	// expect error with canceled context
	store, closer = statestoreTesting.NewStoreServiceForTesting(t, cfg)
	fs = frontendService{cfg, store}
	defer closer()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	res, err = fs.UpdateBackfill(ctx, &pb.UpdateBackfillRequest{Backfill: &pb.Backfill{
		Id: res.Id,
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}})
	require.NotNil(t, err)
	require.Equal(t, codes.Unknown.String(), status.Convert(err).Code().String())
	require.Nil(t, res)
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
						require.NoError(t, err)
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

// TestAcknowledgeBackfillValidation - test input validation only
func TestAcknowledgeBackfillValidation(t *testing.T) {
	cfg := viper.New()
	tests := []struct {
		description     string
		request         *pb.AcknowledgeBackfillRequest
		expectedMessage string
	}{
		{
			description:     "no BackfillId, error is expected",
			request:         &pb.AcknowledgeBackfillRequest{BackfillId: "", Assignment: &pb.Assignment{Connection: "10.0.0.1"}},
			expectedMessage: ".BackfillId is required",
		},
		{
			description:     "no Assignment, error is expected",
			request:         &pb.AcknowledgeBackfillRequest{BackfillId: "1234", Assignment: nil},
			expectedMessage: ".Assignment is required",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			ctx := context.Background()

			store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
			defer closer()
			fs := frontendService{cfg, store}
			bf, err := fs.AcknowledgeBackfill(ctx, test.request)
			require.Equal(t, codes.InvalidArgument.String(), status.Convert(err).Code().String())
			require.Equal(t, test.expectedMessage, status.Convert(err).Message())
			require.Nil(t, bf)
		})
	}
}

// TestAcknowledgeBackfill verifies timestamp part of AcknowledgeBackfill call,
// assignment part tested in a corresponding E2E test.
// Expired backfill can not be acknowledged
func TestAcknowledgeBackfill(t *testing.T) {
	cfg := viper.New()
	ctx := context.Background()

	store, closer := statestoreTesting.NewStoreServiceForTesting(t, cfg)
	defer closer()

	fakeBackfill := &pb.Backfill{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}
	err := store.CreateBackfill(ctx, fakeBackfill, []string{})
	require.NoError(t, err)
	fs := frontendService{cfg, store}

	resp, err := fs.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{BackfillId: fakeBackfill.Id, Assignment: &pb.Assignment{Connection: "10.0.0.1"}})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NotNil(t, resp.Backfill)
	require.NotNil(t, resp.Tickets)

	// Use wrong BackfillID, error is returned
	resp, err = fs.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{BackfillId: "42", Assignment: &pb.Assignment{Connection: "10.0.0.1"}})
	require.Error(t, err)
	require.Nil(t, resp)
	require.Equal(t, "Backfill id: 42 not found", status.Convert(err).Message())

	time.Sleep(cfg.GetDuration("pendingReleaseTimeout"))
	ids, err := store.GetExpiredBackfillIDs(ctx)
	require.NoError(t, err)
	require.Len(t, ids, 1)

	resp, err = fs.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{BackfillId: fakeBackfill.Id, Assignment: &pb.Assignment{Connection: "10.0.0.1"}})
	require.Nil(t, resp)
	require.Error(t, err)
	require.Equal(t, codes.Unavailable.String(), status.Convert(err).Code().String())
	require.Contains(t, status.Convert(err).Message(), "can not acknowledge an expired backfill, id: 1")

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

func TestGetBackfill(t *testing.T) {
	fakeBackfill := &pb.Backfill{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}
	cfg := viper.New()
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
			fs := frontendService{cfg, store}

			test.preAction(ctx, cancel, store)

			backfill, err := fs.GetBackfill(ctx, &pb.GetBackfillRequest{BackfillId: fakeBackfill.GetId()})
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())

			if err == nil {
				require.Equal(t, test.wantTicket.GetId(), backfill.GetId())
				require.Equal(t, test.wantTicket.SearchFields.DoubleArgs, backfill.SearchFields.DoubleArgs)
			}
		})
	}
}

func TestDoDeleteBackfill(t *testing.T) {
	fakeBackfill := &pb.Backfill{
		Id: "1",
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"test-arg": 1,
			},
		},
	}

	store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
	defer closer()
	ctx := context.Background()

	err := store.CreateBackfill(ctx, fakeBackfill, []string{})
	require.NoError(t, err)

	cfg := viper.New()
	fs := frontendService{cfg, store}

	tests := []struct {
		description string
		id          string
		wantCode    codes.Code
	}{
		{
			description: "expect ok code since delete backfill does not care about if backfill exists or not",
			id:          "222",
			wantCode:    codes.OK,
		},
		{
			description: "expect ok code",
			id:          "1",
			wantCode:    codes.OK,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			_, err := fs.DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: fakeBackfill.GetId()})
			require.NoError(t, err)
			require.Equal(t, test.wantCode.String(), status.Convert(err).Code().String())
		})
	}
}
