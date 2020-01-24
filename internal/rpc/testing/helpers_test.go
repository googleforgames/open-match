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

package testing

import (
	"strings"
	"testing"

	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	shellTesting "open-match.dev/open-match/internal/testing"
)

// TestMustServerParamsForTesting verifies that a server can stand up in (insecure or TLS) mode.
func TestMustServe(t *testing.T) {
	runMustServeTest(t, MustServe)
}

// TestMustServerParamsForTesting verifies that a server can stand up in insecure mode.
func TestMustServeInsecure(t *testing.T) {
	runMustServeTest(t, MustServeInsecure)
}

// TestMustServerParamsForTesting verifies that a server can stand up in TLS mode.
func TestMustServeTLS(t *testing.T) {
	runMustServeTest(t, MustServeTLS)
}

func runMustServeTest(t *testing.T, mustServeFunc func(*testing.T, func(*rpc.ServerParams)) *TestContext) {
	assert := assert.New(t)
	ff := &shellTesting.FakeFrontend{}
	tc := mustServeFunc(t, func(spf *rpc.ServerParams) {
		spf.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterFrontendServiceServer(s, ff)
		}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	})
	defer tc.Close()

	conn := tc.MustGRPC()
	c := pb.NewFrontendServiceClient(conn)
	resp, err := c.CreateTicket(tc.Context(), &pb.CreateTicketRequest{})
	assert.Nil(err)
	assert.NotNil(resp)

	hc, endpoint := tc.MustHTTP()
	hResp, err := hc.Post(endpoint+"/v1/frontendservice/tickets", "application/json", strings.NewReader("{}"))
	assert.Nil(err)
	if hResp != nil {
		assert.Equal(200, hResp.StatusCode)
	}
}
