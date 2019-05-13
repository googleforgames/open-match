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
	"fmt"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/serving/rpc"
	shellTesting "open-match.dev/open-match/internal/future/testing"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

func TestSecureGRPCFromConfig(t *testing.T) {
	runGrpcClientTests(t, true)
}

func TestInsecureGRPCFromConfig(t *testing.T) {
	runGrpcClientTests(t, false)
}

func runGrpcClientTests(t *testing.T, useTLS bool) {
	assert := assert.New(t)

	var (
		configPrefix      = "grpcTest"
		clientHostNameKey = "grpcTest.hostname"
		clientGRPCPortKey = "grpcTest.grpcport"
		grpcConn          *grpc.ClientConn
	)

	// Create netlisteners on random ports used for rpv serving
	grpcLh := netlistenerTesting.MustListen()
	httpLh := netlistenerTesting.MustListen()
	rpcParams := rpc.NewParamsFromListeners(grpcLh, httpLh)

	// Fake the config
	cfg := viper.New()
	cfg.Set(clientHostNameKey, "localhost")
	cfg.Set(clientGRPCPortKey, grpcLh.Number())

	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", httpLh.Number())
	allHostnames := []string{grpcAddress, proxyAddress}
	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting(allHostnames)
	assert.Nil(err)

	// TLS setup
	if useTLS {
		rpcParams.SetTLSConfiguration(pub, pub, priv)
	}

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

	// Acquire grpc client
	if useTLS {
		grpcConn, err = SecureGRPCFromConfig(cfg, configPrefix, pub)
	} else {
		grpcConn, err = InsecureGRPCFromConfig(cfg, configPrefix)
	}
	assert.Nil(err)
	assert.NotNil(grpcConn)

	// Confirm the client works as expected
	ctx := context.Background()
	feClient := pb.NewFrontendClient(grpcConn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	assert.Nil(err)
	assert.NotNil(grpcResp)
}
