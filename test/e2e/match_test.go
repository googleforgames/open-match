// +build !e2ecluster

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

package e2e

import (
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/testing/e2e"
	"testing"

	"open-match.dev/open-match/pkg/pb"
)

func TestFetchMatches(t *testing.T) {
	om, closer := e2e.New(t)
	defer closer()

	be := om.MustBackendGRPC()

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
				Host: "om-matchfunction",
				Port: int32(54321),
				Type: pb.FunctionConfig_GRPC,
			},
			[]*pb.MatchProfile{{Name: "some name"}},
			[]*pb.Match{},
			codes.Unavailable,
		},
		{
			"expects empty response since the store is empty",
			om.MustMmfConfigGRPC(),
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

				resp, err := be.FetchMatches(om.Context(), &pb.FetchMatchesRequest{Config: test.fc, Profiles: test.profile})
				assert.Equal(t, test.wantCode, status.Convert(err).Code())

				if err == nil {
					for _, match := range resp.Matches {
						assert.Contains(t, test.wantMatch, &pb.Match{
							MatchProfile:  match.GetMatchProfile(),
							MatchFunction: match.GetMatchFunction(),
							Tickets:       match.GetTickets(),
							Rosters:       match.GetRosters(),
						})
					}
				}
			})
		}
	})
}
