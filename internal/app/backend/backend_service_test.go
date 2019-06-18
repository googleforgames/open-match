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
	"errors"
	"sync"
	"testing"
	"time"

	structpb "github.com/golang/protobuf/ptypes/struct"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/statestore"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestDoFetchMatchesInChannel(t *testing.T) {
	insecureCfg := viper.New()
	secureCfg := viper.New()
	secureCfg.Set("tls.enabled", true)
	restFuncCfg := &pb.FetchMatchesRequest{
		Config:  &pb.FunctionConfig{Name: "test", Type: &pb.FunctionConfig_Rest{Rest: &pb.RestFunctionConfig{Host: "om-test", Port: int32(54321)}}},
		Profile: []*pb.MatchProfile{{Name: "1"}, {Name: "2"}},
	}
	grpcFuncCfg := &pb.FetchMatchesRequest{
		Config:  &pb.FunctionConfig{Name: "test", Type: &pb.FunctionConfig_Grpc{Grpc: &pb.GrpcFunctionConfig{Host: "om-test", Port: int32(54321)}}},
		Profile: []*pb.MatchProfile{{Name: "1"}, {Name: "2"}},
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
			nil,
			status.Error(codes.InvalidArgument, "provided match function type is not supported"),
			insecureCfg,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resultChan := make(chan mmfResult, len(test.req.GetProfile()))
			err := doFetchMatchesInChannel(context.Background(), test.cfg, &sync.Map{}, test.req, resultChan)
			assert.Equal(t, test.wantErr, err)
		})
	}
}

func TestDoFetchMatchesSendResponse(t *testing.T) {
	// The following test suites generate sender functions that will increment test.count for each call
	// to make sure the send response loop can exit gracefully under different circumstances.
	totalProposals := 10
	failAtProposals := 5
	fakeProposals := make([]*pb.Match, totalProposals)

	tests := []struct {
		description     string
		count           int
		senderGenerator func(cancel context.CancelFunc, p *int) func(*pb.Match) error
		wantErr         bool
		wantCount       int
	}{
		{
			description: "expect test.count to be 10 without intervening the context",
			senderGenerator: func(cancel context.CancelFunc, p *int) func(*pb.Match) error {
				return func(matches *pb.Match) error {
					*p++
					return nil
				}
			},
			wantErr:   false,
			wantCount: totalProposals,
		},
		{
			description: "expect doFetchMatchesSendResponse returns with an error because of sender failures",
			senderGenerator: func(cancel context.CancelFunc, p *int) func(*pb.Match) error {
				return func(matches *pb.Match) error {
					if *p == failAtProposals {
						return errors.New("some err")
					}
					*p++
					return nil
				}
			},
			wantErr:   true,
			wantCount: failAtProposals,
		},
		{
			description: "expect an context error as context is canceled halfway",
			senderGenerator: func(cancel context.CancelFunc, p *int) func(*pb.Match) error {
				return func(matches *pb.Match) error {
					*p++
					if *p == failAtProposals {
						cancel()
					}
					return nil
				}
			},
			wantErr:   true,
			wantCount: failAtProposals,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			err := doFetchMatchesSendResponse(ctx, fakeProposals, test.senderGenerator(cancel, &test.count))

			assert.Equal(t, test.wantCount, test.count)
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}

func TestDoFetchMatchesFilterChannel(t *testing.T) {
	tests := []struct {
		description string
		preAction   func(chan mmfResult, context.CancelFunc)
		wantMatches []*pb.Match
		wantErr     bool
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
			wantErr:     true,
		},
		{
			description: "test the filter can return an error when one of the mmfResult contains an error",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1"}}, err: nil}
				mmfChan <- mmfResult{matches: nil, err: errors.New("some error")}
			},
			wantMatches: nil,
			wantErr:     true,
		},
		{
			description: "test the filter can return proposals when all mmfResults are valid",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1"}}, err: nil}
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "2"}}, err: nil}
			},
			wantMatches: []*pb.Match{{MatchId: "1"}, {MatchId: "2"}},
			wantErr:     false,
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			resultChan := make(chan mmfResult, 2)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			test.preAction(resultChan, cancel)

			matches, err := doFetchMatchesFilterChannel(ctx, resultChan, 2)

			for _, match := range matches {
				assert.Contains(t, test.wantMatches, match)
			}
			assert.Equal(t, test.wantErr, err != nil)
		})
	}
}

func TestGetHTTPClient(t *testing.T) {
	assert := assert.New(t)
	cache := &sync.Map{}
	client, url, err := getHTTPClient(viper.New(), cache, &pb.FunctionConfig_Rest{Rest: &pb.RestFunctionConfig{Host: "om-test", Port: int32(50321)}})
	assert.Nil(err)
	assert.NotNil(client)
	assert.NotNil(url)
	cachedClient, url, err := getHTTPClient(viper.New(), cache, &pb.FunctionConfig_Rest{Rest: &pb.RestFunctionConfig{Host: "om-test", Port: int32(50321)}})
	assert.Nil(err)
	assert.NotNil(client)
	assert.NotNil(url)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedClient)
}

func TestGetGRPCClient(t *testing.T) {
	assert := assert.New(t)
	cache := &sync.Map{}
	client, err := getGRPCClient(viper.New(), cache, &pb.FunctionConfig_Grpc{Grpc: &pb.GrpcFunctionConfig{Host: "om-test", Port: int32(50321)}})
	assert.Nil(err)
	assert.NotNil(client)
	cachedClient, err := getGRPCClient(viper.New(), cache, &pb.FunctionConfig_Grpc{Grpc: &pb.GrpcFunctionConfig{Host: "om-test", Port: int32(50321)}})
	assert.Nil(err)
	assert.NotNil(client)

	// Test caching by comparing pointer value
	assert.EqualValues(client, cachedClient)
}

func TestDoAssignTickets(t *testing.T) {
	fakeTickets := []*pb.Ticket{
		{
			Id: "1",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				},
			},
		},
		{
			Id: "2",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"test-property": {Kind: &structpb.Value_NumberValue{NumberValue: 2}},
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
				TicketId:   []string{"1"},
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
				TicketId: []string{"1", "2"},
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
			},
			req: &pb.AssignTicketsRequest{
				TicketId: []string{"1", "2"},
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
			store, closer := statestoreTesting.NewStoreServiceForTesting(t, viper.New())
			defer closer()

			test.preAction(ctx, cancel, store)
			err := doAssignTickets(ctx, test.req, store)

			assert.Equal(t, test.wantCode, status.Convert(err).Code())

			if err == nil {
				for _, id := range test.req.GetTicketId() {
					ticket, err := store.GetTicket(ctx, id)
					assert.Nil(t, err)
					assert.Equal(t, test.wantAssignment, ticket.GetAssignment())
				}
			}
		})
	}
}
