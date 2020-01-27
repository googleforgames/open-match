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
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
	"open-match.dev/open-match/internal/telemetry"
	shellTesting "open-match.dev/open-match/internal/testing"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
	"strings"
	"testing"
	"time"
)

func TestStartStopServer(t *testing.T) {
	assert := assert.New(t)
	grpcLh := MustListen()
	httpLh := MustListen()
	ff := &shellTesting.FakeFrontend{}

	params := NewServerParamsFromListeners(grpcLh, httpLh)
	params.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	s := &Server{}
	defer s.Stop()

	waitForStart, err := s.Start(params)
	assert.Nil(err)
	waitForStart()

	conn, err := grpc.Dial(fmt.Sprintf(":%d", grpcLh.Number()), grpc.WithInsecure())
	assert.Nil(err)

	endpoint := fmt.Sprintf("http://localhost:%d", httpLh.Number())
	httpClient := &http.Client{
		Timeout: time.Second,
	}

	runGrpcWithProxyTests(t, assert, s.serverWithProxy, conn, httpClient, endpoint)
}

func TestMustServeForever(t *testing.T) {
	assert := assert.New(t)
	grpcLh := MustListen()
	httpLh := MustListen()
	ff := &shellTesting.FakeFrontend{}

	params := NewServerParamsFromListeners(grpcLh, httpLh)
	params.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	serveUntilKilledFunc, stopServingFunc, err := startServingIndefinitely(params)
	assert.Nil(err)
	go func() {
		// Wait for 500ms before killing the server.
		// It really doesn't matter if it actually comes up.
		// We just care that the server can respect an unexpected shutdown quickly after starting.
		time.Sleep(time.Millisecond * 500)
		stopServingFunc()
	}()
	serveUntilKilledFunc()
	// This test will intentionally deadlock if the stop function is not respected.
}

func runGrpcWithProxyTests(t *testing.T, assert *assert.Assertions, s grpcServerWithProxy, conn *grpc.ClientConn, httpClient *http.Client, endpoint string) {
	ctx := utilTesting.NewContext(t)
	feClient := pb.NewFrontendServiceClient(conn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	assert.Nil(err)
	assert.NotNil(grpcResp)

	httpReq, err := http.NewRequest(http.MethodPost, endpoint+"/v1/frontendservice/tickets", strings.NewReader("{}"))
	assert.Nil(err)
	assert.NotNil(httpReq)
	httpResp, err := httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("{}", string(body))

	httpReq, err = http.NewRequest(http.MethodGet, endpoint+telemetry.HealthCheckEndpoint, nil)
	assert.Nil(err)

	httpResp, err = httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()
	body, err = ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("ok", string(body))

	s.stop()
}
