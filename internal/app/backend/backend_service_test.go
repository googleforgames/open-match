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
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/statestore"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

func TestDoFetchMatchesInChannel(t *testing.T) {
	insecureCfg := viper.New()
	secureCfg := viper.New()
	pub, _, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{})
	if err != nil {
		t.Fatalf("cannot create TLS keys: %s", err)
	}
	secureCfg.Set("api.tls.rootCertificateFile", pub)
	restFuncCfg := &pb.FetchMatchesRequest{
		Config:   &pb.FunctionConfig{Host: "om-test", Port: 54321, Type: pb.FunctionConfig_REST},
		Profiles: []*pb.MatchProfile{{Name: "1"}, {Name: "2"}},
	}
	grpcFuncCfg := &pb.FetchMatchesRequest{
		Config:   &pb.FunctionConfig{Host: "om-test", Port: 54321, Type: pb.FunctionConfig_GRPC},
		Profiles: []*pb.MatchProfile{{Name: "1"}, {Name: "2"}},
	}
	unsupporteFuncCfg := &pb.FetchMatchesRequest{
		Config:   &pb.FunctionConfig{Host: "om-test", Port: 54321, Type: 3},
		Profiles: []*pb.MatchProfile{{Name: "1"}, {Name: "2"}},
	}

	tests := []struct {
		description string
		req         *pb.FetchMatchesRequest
		wantErr     error
		cfg         config.View
	}{
		{
			"trusted certificate is required when requesting a secure http client",
			restFuncCfg,
			status.Error(codes.InvalidArgument, "failed to connect to match function"),
			secureCfg,
		},
		{
			"trusted certificate is required when requesting a secure grpc client",
			grpcFuncCfg,
			status.Error(codes.InvalidArgument, "failed to connect to match function"),
			secureCfg,
		},
		{
			"the mmfResult channel received data successfully under the insecure mode with rest config",
			restFuncCfg,
			nil,
			insecureCfg,
		},
		{
			"the mmfResult channel received data successfully under the insecure mode with grpc config",
			grpcFuncCfg,
			nil,
			insecureCfg,
		},
		{
			"one of the rest/grpc config is required to process the request",
			unsupporteFuncCfg,
			status.Error(codes.InvalidArgument, "provided match function type is not supported"),
			insecureCfg,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cc := rpc.NewClientCache(test.cfg)
			resultChan := make(chan mmfResult, len(test.req.GetProfiles()))
			err := doFetchMatchesReceiveMmfResult(context.Background(), cc, test.req, resultChan)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func TestDoFetchMatchesFilterChannel(t *testing.T) {
	tests := []struct {
		description string
		preAction   func(chan mmfResult, context.CancelFunc)
		wantMatches []*pb.Match
		wantCode    codes.Code
	}{
		{
			description: "test the filter can exit the for loop when context was canceled",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()
			},
			wantMatches: nil,
			wantCode:    codes.Unknown,
		},
		{
			description: "test the filter can return an error when one of the mmfResult contains an error",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1", Tickets: []*pb.Ticket{{Id: "123"}}}}, err: nil}
				mmfChan <- mmfResult{matches: nil, err: status.Error(codes.Unknown, "some error")}
			},
			wantMatches: nil,
			wantCode:    codes.Unknown,
		},
		{
			description: "test the filter can return an error when one of the mmf calls return match with empty tickets",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1"}}, err: nil}
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "2"}}, err: nil}
			},
			wantMatches: []*pb.Match{{MatchId: "1"}, {MatchId: "2"}},
			wantCode:    codes.FailedPrecondition,
		},
		{
			description: "test the filter can return proposals when all mmfResults are valid",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1", Tickets: []*pb.Ticket{{Id: "123"}}}}, err: nil}
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "2", Tickets: []*pb.Ticket{{Id: "321"}}}}, err: nil}
			},
			wantMatches: []*pb.Match{{MatchId: "1", Tickets: []*pb.Ticket{{Id: "123"}}}, {MatchId: "2", Tickets: []*pb.Ticket{{Id: "321"}}}},
			wantCode:    codes.OK,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resultChan := make(chan mmfResult, 2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			test.preAction(resultChan, cancel)

			matches, err := doFetchMatchesValidateProposals(ctx, resultChan, 2)

			for _, match := range matches {
				assert.Contains(t, test.wantMatches, match)
			}
			assert.Equal(t, test.wantCode, status.Convert(err).Code())
		})
	}
}

func TestDoAssignTickets(t *testing.T) {
	fakeProperty := "test-property"
	fakeTickets := []*pb.Ticket{
		{
			Id: "1",
			Properties: structs.Struct{
				fakeProperty: structs.Number(1),
			}.S(),
		},
		{
			Id: "2",
			Properties: structs.Struct{
				fakeProperty: structs.Number(2),
			}.S(),
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
				err := store.FilterTickets(ctx, []*pb.Filter{{Attribute: fakeProperty, Min: 0, Max: 3}}, 10, func(filterTickets []*pb.Ticket) error {
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
			ctx, cancel := context.WithCancel(context.Background())
			cfg := viper.New()
			cfg.Set("ticketIndices", []string{fakeProperty})
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
				store.FilterTickets(ctx, []*pb.Filter{{Attribute: fakeProperty, Min: 0, Max: 2}}, 10, func(filterTickets []*pb.Ticket) error {
					wantFilteredTickets = filterTickets
					return nil
				})
				assert.Nil(t, wantFilteredTickets)
			}
		})
	}
}

// TODOs: add unit tests to doFetchMatchesFilterSkiplistIds and doFetchMatchesAddSkiplistIds
