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

	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	"open-match.dev/open-match/pkg/pb"
)

func TestFetchMatches(t *testing.T) {
	mainTc := createMinimatchForTest(t)
	defer mainTc.Close()
	mmfTc := createMatchFunctionForTest(t, mainTc)
	defer mmfTc.Close()

	be := pb.NewBackendClient(mainTc.MustGRPC())

	var tt = []struct {
		description string
		fc          *pb.FunctionConfig
		profile     []*pb.MatchProfile
		wantMatch   []*pb.Match
		wantCode    codes.Code
	}{
		{
			"expects invalid argument code since request is empty",
			nil,
			nil,
			nil,
			codes.InvalidArgument,
		},
		{
			"expects unavailable code since there is no mmf being hosted with given function config",
			&pb.FunctionConfig{
				Type: &pb.FunctionConfig_Grpc{
					Grpc: &pb.GrpcFunctionConfig{
						Host: mmfTc.GetHostname(),
						Port: int32(54321),
					},
				},
			},
			[]*pb.MatchProfile{{Name: "some name"}},
			[]*pb.Match{},
			codes.Unavailable,
		},
		{
			"expects empty response since the store is empty",
			&pb.FunctionConfig{
				Type: &pb.FunctionConfig_Grpc{
					Grpc: &pb.GrpcFunctionConfig{
						Host: mmfTc.GetHostname(),
						Port: int32(mmfTc.GetGRPCPort()),
					},
				},
			},
			[]*pb.MatchProfile{{Name: "some name"}},
			[]*pb.Match{},
			codes.OK,
		},
	}

	t.Run("TestFetchMatches", func(t *testing.T) {
		for _, test := range tt {
			test := test
			t.Run(test.description, func(t *testing.T) {
				t.Parallel()

				stream, err := be.FetchMatches(mainTc.Context(), &pb.FetchMatchesRequest{Config: test.fc, Profile: test.profile})
				assert.Nil(t, err)

				var gotMatches []*pb.Match

				for {
					resp, err := stream.Recv()
					if err == io.EOF {
						break
					}
					if err != nil {
						assert.Equal(t, test.wantCode, status.Convert(err).Code())
						break
					}

					gotMatches = append(gotMatches, &pb.Match{
						MatchProfile:  resp.GetMatch().GetMatchProfile(),
						MatchFunction: resp.GetMatch().GetMatchFunction(),
						Ticket:        resp.GetMatch().GetTicket(),
						Roster:        resp.GetMatch().GetRoster(),
					})
				}

				for _, match := range gotMatches {
					assert.Contains(t, test.wantMatch, match)
				}
			})
		}
	})
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
