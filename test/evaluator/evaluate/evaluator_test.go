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

package evaluate

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/testing/customize/evaluator"
	"open-match.dev/open-match/pkg/pb"
)

func mustAny(m proto.Message) *any.Any {
	result, err := ptypes.MarshalAny(m)
	if err != nil {
		panic(err)
	}
	return result
}

func TestEvaluate(t *testing.T) {
	ticket1 := &pb.Ticket{Id: "1"}
	ticket2 := &pb.Ticket{Id: "2"}
	ticket3 := &pb.Ticket{Id: "3"}

	ticket12Score1 := &pb.Match{
		Tickets: []*pb.Ticket{ticket1, ticket2},
		Extensions: map[string]*any.Any{
			"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
				Score: 1,
			}),
		},
	}

	ticket12Score10 := &pb.Match{
		Tickets: []*pb.Ticket{ticket2, ticket1},
		Extensions: map[string]*any.Any{
			"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
				Score: 10,
			}),
		},
	}

	ticket123Score5 := &pb.Match{
		Tickets: []*pb.Ticket{ticket1, ticket2, ticket3},
		Extensions: map[string]*any.Any{
			"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
				Score: 5,
			}),
		},
	}

	ticket3Score50 := &pb.Match{
		Tickets: []*pb.Ticket{ticket3},
		Extensions: map[string]*any.Any{
			"evaluation_input": mustAny(&pb.DefaultEvaluationCriteria{
				Score: 50,
			}),
		},
	}

	tests := []struct {
		description string
		testMatches []*pb.Match
		wantMatches []*pb.Match
	}{
		{
			description: "test empty request returns empty response",
			testMatches: []*pb.Match{},
			wantMatches: []*pb.Match{},
		},
		{
			description: "test input matches output when receiving one match",
			testMatches: []*pb.Match{ticket12Score1},
			wantMatches: []*pb.Match{ticket12Score1},
		},
		{
			description: "test deduplicates and expect the one with higher score",
			testMatches: []*pb.Match{ticket12Score1, ticket12Score10},
			wantMatches: []*pb.Match{ticket12Score10},
		},
		{
			description: "test first returns matches with higher score",
			testMatches: []*pb.Match{ticket123Score5, ticket12Score10},
			wantMatches: []*pb.Match{ticket12Score10},
		},
		{
			description: "test evaluator returns two matches with the highest score",
			testMatches: []*pb.Match{ticket12Score1, ticket12Score10, ticket123Score5, ticket3Score50},
			wantMatches: []*pb.Match{ticket12Score10, ticket3Score50},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			gotMatches, err := Evaluate(&evaluator.Params{Matches: test.testMatches})

			assert.Nil(t, err)
			assert.Equal(t, len(test.wantMatches), len(gotMatches))
			for _, match := range gotMatches {
				assert.Contains(t, test.wantMatches, match)
			}
		})
	}
}
