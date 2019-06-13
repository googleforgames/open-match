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
	"io"
	"sync"
	"testing"
	"time"

	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/config"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"

	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
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
		shouldErr   error
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
			"the mmfResult channel received data succesfully under the insecure mode with rest config",
			restFuncCfg,
			nil,
			insecureCfg,
		},
		{
			"the mmfResult channel received data succesfully under the insecure mode with grpc config",
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
			if !cmp.Equal(test.shouldErr, err) {
				t.Errorf("Expected an error: %s, but was %s\n", test.shouldErr, err)
			}
		})
	}
}

func TestDoFetchMatchesSendResponse(t *testing.T) {
	// TODO: Add more test
}

func TestDoFetchMatchesFilterChannel(t *testing.T) {
	tests := []struct {
		description   string
		preAction     func(chan mmfResult, context.CancelFunc)
		shouldMatches []*pb.Match
		shouldErr     bool
	}{
		{
			description: "test the filter can exit the for loop when context was cancelled",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				go func() {
					time.Sleep(100 * time.Millisecond)
					cancel()
				}()
			},
			shouldMatches: nil,
			shouldErr:     true,
		},
		{
			description: "test the filter can return an error when one of the mmfResult contains an error",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{&pb.Match{MatchId: "1"}}, err: nil}
				mmfChan <- mmfResult{matches: nil, err: errors.New("some error")}
			},
			shouldMatches: nil,
			shouldErr:     true,
		},
		{
			description: "test the filter can return proposals when all mmfResults are valid",
			preAction: func(mmfChan chan mmfResult, cancel context.CancelFunc) {
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "1"}}, err: nil}
				mmfChan <- mmfResult{matches: []*pb.Match{{MatchId: "2"}}, err: nil}
			},
			shouldMatches: []*pb.Match{{MatchId: "1"}, {MatchId: "2"}},
			shouldErr:     false,
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
				assert.Contains(t, test.shouldMatches, match)
			}
			assert.Equal(t, test.shouldErr, err != nil)
			if test.shouldErr != (err != nil) {
				t.Errorf("expect error: %v, but was %s", test.shouldErr, err.Error())
			}
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

func TestAssignTickets(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	fe := pb.NewFrontendClient(tc.MustGRPC())
	be := pb.NewBackendClient(tc.MustGRPC())

	ctResp, err := fe.CreateTicket(tc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(err)

	var tt = []struct {
		req  *pb.AssignTicketsRequest
		resp *pb.AssignTicketsResponse
		code codes.Code
	}{
		{
			&pb.AssignTicketsRequest{},
			nil,
			codes.InvalidArgument,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{"1"},
			},
			nil,
			codes.InvalidArgument,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{"2"},
				Assignment: &pb.Assignment{
					Connection: "localhost",
				},
			},
			nil,
			codes.NotFound,
		},
		{
			&pb.AssignTicketsRequest{
				TicketId: []string{ctResp.Ticket.Id},
				Assignment: &pb.Assignment{
					Connection: "localhost",
				},
			},
			&pb.AssignTicketsResponse{},
			codes.OK,
		},
	}

	for _, test := range tt {
		resp, err := be.AssignTickets(tc.Context(), test.req)
		assert.Equal(test.resp, resp)
		if err != nil {
			assert.Equal(test.code, status.Convert(err).Code())
		} else {
			gtResp, err := fe.GetTicket(tc.Context(), &pb.GetTicketRequest{TicketId: ctResp.Ticket.Id})
			assert.Nil(err)
			assert.Equal(test.req.Assignment.Connection, gtResp.Assignment.Connection)
		}
	}
}

func TestFetchMatches(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	be := pb.NewBackendClient(tc.MustGRPC())

	var tt = []struct {
		req  *pb.FetchMatchesRequest
		resp *pb.FetchMatchesResponse
		code codes.Code
	}{
		{
			&pb.FetchMatchesRequest{},
			nil,
			codes.InvalidArgument,
		},
	}

	for _, test := range tt {
		fetchMatchesLoop(t, tc, be, test.req, func(_ *pb.FetchMatchesResponse, err error) {
			assert.Equal(test.code, status.Convert(err).Code())
		})
	}
}

// TODO: Add FetchMatchesNormalTest when Hostname getter used to initialize mmf service is in
// https://github.com/GoogleCloudPlatform/open-match/pull/473
func fetchMatchesLoop(t *testing.T, tc *rpcTesting.TestContext, be pb.BackendClient, req *pb.FetchMatchesRequest, handleResponse func(*pb.FetchMatchesResponse, error)) {
	stream, err := be.FetchMatches(tc.Context(), req)
	if err != nil {
		t.Fatalf("error querying tickets, %v", err)
	}
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		handleResponse(resp, err)
		if err != nil {
			return
		}
	}
}

func createBackendForTest(t *testing.T) *rpcTesting.TestContext {
	var closerFunc func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closerFunc = statestoreTesting.New(t, cfg)

		cfg.Set("storage.page.size", 10)
		BindService(p, cfg)
		frontend.BindService(p, cfg)
		mmlogic.BindService(p, cfg)
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closerFunc)
	return tc
}
