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

package synchronizer

import (
	"context"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/examples"
	simple "open-match.dev/open-match/examples/evaluator/golang/simple/evaluate"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	"open-match.dev/open-match/pkg/gopb"
	evalHarness "open-match.dev/open-match/pkg/harness/evaluator/golang"
	"open-match.dev/open-match/pkg/structs"
)

// TODO: refactor this test to have it moved to e2e pkg
func TestEvaluator(t *testing.T) {
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		assert.Nil(t, evalHarness.BindService(p, cfg, simple.Evaluate))
	})
	defer tc.Close()

	cfg := viper.New()
	cfg.Set("api.evaluator.hostname", "127.0.0.1")
	cfg.Set("api.evaluator.grpcport", tc.GetGRPCPort())
	cfg.Set("api.evaluator.httpport", tc.GetHTTPPort())

	ticket1 := &gopb.Ticket{Id: "1"}
	ticket2 := &gopb.Ticket{Id: "2"}
	ticket3 := &gopb.Ticket{Id: "3"}

	ticket12Score1 := &gopb.Match{
		MatchId: "12",
		Tickets: []*gopb.Ticket{ticket1, ticket2},
		Properties: structs.Struct{
			examples.MatchScore: structs.Number(1),
		}.S(),
	}

	ticket12Score10 := &gopb.Match{
		MatchId: "21",
		Tickets: []*gopb.Ticket{ticket2, ticket1},
		Properties: structs.Struct{
			examples.MatchScore: structs.Number(10),
		}.S(),
	}

	ticket123Score5 := &gopb.Match{
		MatchId: "123",
		Tickets: []*gopb.Ticket{ticket1, ticket2, ticket3},
		Properties: structs.Struct{
			examples.MatchScore: structs.Number(5),
		}.S(),
	}

	ticket3Score50 := &gopb.Match{
		MatchId: "3",
		Tickets: []*gopb.Ticket{ticket3},
		Properties: structs.Struct{
			examples.MatchScore: structs.Number(50),
		}.S(),
	}

	tests := []struct {
		description string
		testMatches []*gopb.Match
		wantMatches []*gopb.Match
	}{
		{
			description: "test empty request returns empty response",
			testMatches: []*gopb.Match{},
			wantMatches: []*gopb.Match{},
		},
		{
			description: "test input matches output when receiving one match",
			testMatches: []*gopb.Match{ticket12Score1},
			wantMatches: []*gopb.Match{ticket12Score1},
		},
		{
			description: "test deduplicates and expect the one with higher score",
			testMatches: []*gopb.Match{ticket12Score1, ticket12Score10},
			wantMatches: []*gopb.Match{ticket12Score10},
		},
		{
			description: "test first returns matches with higher score",
			testMatches: []*gopb.Match{ticket123Score5, ticket12Score10},
			wantMatches: []*gopb.Match{ticket12Score10},
		},
		{
			description: "test evaluator returns two matches with the highest score",
			testMatches: []*gopb.Match{ticket12Score1, ticket12Score10, ticket123Score5, ticket3Score50},
			wantMatches: []*gopb.Match{ticket12Score10, ticket3Score50},
		},
	}

	ec := &evaluatorClient{cfg: cfg}
	assert.Nil(t, ec.initialize())

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			gotMatches, err := ec.httpEvaluate(context.Background(), test.testMatches)

			assert.Nil(t, err)
			assert.Equal(t, len(test.wantMatches), len(gotMatches))
			for _, match := range gotMatches {
				if !contains(test.testMatches, match) {
					t.Errorf("matches: %v does not contains match %v", test.testMatches, match)
				}
			}
		})
	}

}

func contains(matches []*gopb.Match, match *gopb.Match) bool {
	for _, m := range matches {
		if proto.Equal(m, match) {
			return true
		}
	}
	return false
}
