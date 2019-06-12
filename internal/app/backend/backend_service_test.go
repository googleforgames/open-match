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

	"open-match.dev/open-match/examples/functions/golang/pool"
	"open-match.dev/open-match/internal/app/frontend"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	mmfHarness "open-match.dev/open-match/pkg/harness/golang"

	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
)

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

func TestMmfHTTPConfig(t *testing.T) {
	assert := assert.New(t)
	beTc := createBackendForTest(t)
	mmfTc := createMatchFunctionForTest(t, beTc)
	defer beTc.Close()
	defer mmfTc.Close()

	mmfClient, mmfURL := mmfTc.MustHTTP()

	matches, err := matchesFromHTTPMMF(beTc.Context(), &pb.MatchProfile{}, mmfClient, mmfURL)
	assert.Equal(0, len(matches))
	assert.Nil(err)
}

func TestMmfGRPCConfig(t *testing.T) {
	assert := assert.New(t)
	beTc := createBackendForTest(t)
	mmfTc := createMatchFunctionForTest(t, beTc)
	defer beTc.Close()
	defer mmfTc.Close()

	mmfConn := mmfTc.MustGRPC()

	matches, err := matchesFromGRPCMMF(beTc.Context(), &pb.MatchProfile{}, pb.NewMatchFunctionClient(mmfConn))
	assert.Equal(0, len(matches))
	assert.Nil(err)
}

func TestFetchMatches(t *testing.T) {
	assert := assert.New(t)
	beTc := createBackendForTest(t)
	defer beTc.Close()

	be := pb.NewBackendClient(beTc.MustGRPC())

	var tt = []struct {
		req    *pb.FetchMatchesRequest
		resp   *pb.FetchMatchesResponse
		action func(*pb.FetchMatchesRequest) (pb.Backend_FetchMatchesClient, error)
		code   codes.Code
	}{
		{
			&pb.FetchMatchesRequest{},
			nil,
			func(req *pb.FetchMatchesRequest) (pb.Backend_FetchMatchesClient, error) {
				return be.FetchMatches(beTc.Context(), req)
			},
			codes.InvalidArgument,
		},
	}

	for _, test := range tt {
		stream, err := test.action(test.req)
		if err != nil {
			t.Fatalf("error querying tickets, %v", err)
		}
		for {
			_, err := stream.Recv()
			if err == io.EOF {
				break
			}

			assert.Equal(test.code, status.Convert(err).Code())
			if err != nil {
				return
			}
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

// Create a mmf service using a started test server.
// Inject the port config of mmlogic using that the passed in test server
func createMatchFunctionForTest(t *testing.T, c *rpcTesting.TestContext) *rpcTesting.TestContext {
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()

		// The below configuration is used by GRPC harness to create an mmlogic client to query tickets.
		cfg.Set("api.mmlogic.hostname", c.GetHostname())
		cfg.Set("api.mmlogic.grpcport", c.GetGRPCPort())
		cfg.Set("api.mmlogic.httpport", c.GetHTTPPort())

		if err := mmfHarness.BindService(p, cfg, &mmfHarness.FunctionSettings{
			Func: pool.MakeMatches,
		}); err != nil {
			t.Error(err)
		}
	})
	return tc
}
