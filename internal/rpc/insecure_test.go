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
	"net/http"
	"testing"
	"time"

	"open-match.dev/open-match/pkg/pb"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	shellTesting "open-match.dev/open-match/testing"
)

func TestInsecureStartStop(t *testing.T) {
	require := require.New(t)
	grpcL := MustListen()
	httpL := MustListen()
	ff := &shellTesting.FakeFrontend{}

	params := NewServerParamsFromListeners(grpcL, httpL)
	params.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	s := newInsecureServer(grpcL, httpL)
	defer s.stop()
	err := s.start(params)
	require.Nil(err)

	conn, err := grpc.Dial(fmt.Sprintf(":%s", MustGetPortNumber(grpcL)), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(err)
	defer conn.Close()

	endpoint := fmt.Sprintf("http://localhost:%s", MustGetPortNumber(httpL))
	httpClient := &http.Client{
		Timeout: time.Second,
	}
	runGrpcWithProxyTests(t, require, s, conn, httpClient, endpoint)
}
