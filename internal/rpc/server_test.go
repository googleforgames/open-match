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

package rpc

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"open-match.dev/open-match/internal/telemetry"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
	shellTesting "open-match.dev/open-match/testing"
)

func TestStartStopServer(t *testing.T) {
	require := require.New(t)
	grpcL := MustListen()
	httpL := MustListen()
	ff := &shellTesting.FakeFrontend{}

	params := NewServerParamsFromListeners(grpcL, httpL)
	params.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	s := &Server{}
	defer s.Stop()

	err := s.Start(params)
	require.Nil(err)

	conn, err := grpc.Dial(fmt.Sprintf(":%s", MustGetPortNumber(grpcL)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(err)

	endpoint := fmt.Sprintf("http://localhost:%s", MustGetPortNumber(httpL))
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	runGrpcWithProxyTests(t, require, s.serverWithProxy, conn, httpClient, endpoint)
}

func runGrpcWithProxyTests(t *testing.T, require *require.Assertions, s grpcServerWithProxy, conn *grpc.ClientConn, httpClient *http.Client, endpoint string) {
	ctx := utilTesting.NewContext(t)
	feClient := pb.NewFrontendServiceClient(conn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	require.Nil(err)
	require.NotNil(grpcResp)

	httpReq, err := http.NewRequest(http.MethodPost, endpoint+"/v1/frontendservice/tickets", strings.NewReader("{}"))
	require.Nil(err)
	require.NotNil(httpReq)
	httpResp, err := httpClient.Do(httpReq)
	require.Nil(err)
	require.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(httpResp.Body)
	require.Nil(err)
	require.Equal(200, httpResp.StatusCode)
	require.Equal("{}", string(body))

	httpReq, err = http.NewRequest(http.MethodGet, endpoint+telemetry.HealthCheckEndpoint, nil)
	require.Nil(err)

	httpResp, err = httpClient.Do(httpReq)
	require.Nil(err)
	require.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()
	body, err = ioutil.ReadAll(httpResp.Body)
	require.Nil(err)
	require.Equal(200, httpResp.StatusCode)
	require.Equal("ok", string(body))

	s.stop()
}
