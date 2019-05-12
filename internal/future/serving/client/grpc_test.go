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

package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/serving/rpc"
	shellTesting "open-match.dev/open-match/internal/future/testing"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
)

func TestInsecureGRPCFromConfig(t *testing.T) {
	assert := assert.New(t)

	// Create netlisteners on random ports used for rpv serving
	grpcLh := netlistenerTesting.MustListen()
	httpLh := netlistenerTesting.MustListen()
	rpcParams := rpc.NewParamsFromListeners(grpcLh, httpLh)

	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServer(s, ff)
	}, pb.RegisterFrontendHandlerFromEndpoint)
	s := &rpc.Server{}
	defer s.Stop()
	waitForStart, err := s.Start(rpcParams)
	assert.Nil(err)
	waitForStart()

	grpcConn, err := GRPCFromParams(&Params{
		Hostname: "localhost",
		Port:     grpcLh.Number(),
	})
	assert.Nil(err)
	assert.NotNil(grpcConn)

	ctx := context.Background()
	feClient := pb.NewFrontendClient(grpcConn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	assert.Nil(err)
	assert.NotNil(grpcResp)
}
