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

package minimatch

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"open-match.dev/open-match/internal/pb"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
)

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