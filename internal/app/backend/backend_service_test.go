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
	"io"
	"testing"

	"open-match.dev/open-match/internal/app/frontend"

	"github.com/rs/xid"
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

func TestAssignTicketsEmptyRequest(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	assignTicketsLoop(tc, &pb.AssignTicketsRequest{}, func(_ *pb.AssignTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
	assignTicketsLoop(tc, &pb.AssignTicketsRequest{Assignment: &pb.Assignment{}}, func(_ *pb.AssignTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
	assignTicketsLoop(tc, &pb.AssignTicketsRequest{TicketId: []string{xid.New().String(), xid.New().String()}}, func(_ *pb.AssignTicketsResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})

	req := &pb.AssignTicketsRequest{
		TicketId: []string{xid.New().String(), xid.New().String()},
		Assignment: &pb.Assignment{
			Connection: "localhost",
		},
	}
	assignTicketsLoop(tc, req, func(resp *pb.AssignTicketsResponse, err error) {
		assert.Equal(codes.NotFound, status.Convert(err).Code())
	})
}

func TestAssignTicketsNormalRequest(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	fe := pb.NewFrontendClient(tc.MustGRPC())

	ctResp, err := fe.CreateTicket(tc.Context(), &pb.CreateTicketRequest{Ticket: &pb.Ticket{}})
	assert.Nil(err)

	req := &pb.AssignTicketsRequest{
		TicketId: []string{ctResp.Ticket.Id},
		Assignment: &pb.Assignment{
			Connection: "localhost",
		},
	}
	assignTicketsLoop(tc, req, func(resp *pb.AssignTicketsResponse, err error) {
		assert.Equal(&pb.AssignTicketsResponse{}, resp)
	})

	gtResp, err := fe.GetTicket(tc.Context(), &pb.GetTicketRequest{TicketId: ctResp.Ticket.Id})
	assert.Nil(err)
	assert.Equal("localhost", gtResp.Assignment.Connection)
}

func TestFetchMatchesEmptyRequest(t *testing.T) {
	assert := assert.New(t)
	tc := createBackendForTest(t)
	defer tc.Close()

	fetchMatchesLoop(t, tc, &pb.FetchMatchesRequest{}, func(_ *pb.FetchMatchesResponse, err error) {
		assert.Equal(codes.InvalidArgument, status.Convert(err).Code())
	})
}

// TODO: Add FetchMatchesNormalTest when Hostname getter used to initialize mmf service is in
// https://github.com/GoogleCloudPlatform/open-match/pull/473
func fetchMatchesLoop(t *testing.T, tc *rpcTesting.TestContext, req *pb.FetchMatchesRequest, handleResponse func(*pb.FetchMatchesResponse, error)) {
	c := pb.NewBackendClient(tc.MustGRPC())
	stream, err := c.FetchMatches(tc.Context(), req)
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

func assignTicketsLoop(tc *rpcTesting.TestContext, req *pb.AssignTicketsRequest, handleResponse func(*pb.AssignTicketsResponse, error)) {
	c := pb.NewBackendClient(tc.MustGRPC())
	resp, err := c.AssignTickets(tc.Context(), req)
	handleResponse(resp, err)
	if err != nil {
		return
	}
}

func createBackendForTest(t *testing.T) *rpcTesting.TestContext {
	var closerFunc func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		_, closer := statestoreTesting.New(t, cfg)
		closerFunc = closer

		cfg.Set("storage.page.size", 10)
		BindService(p, cfg)
		frontend.BindService(p, cfg)
		mmlogic.BindService(p, cfg)
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closerFunc)
	return tc
}
