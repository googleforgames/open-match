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

package mmf

import (
	"testing"

	"open-match.dev/open-match/pkg/pb"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	internalMmf "open-match.dev/open-match/internal/testing/mmf"
)

func TestMakeMatches(t *testing.T) {
	assert := assert.New(t)

	tickets := []*pb.Ticket{
		{
			Id: "1",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"level":   10,
					"defense": 100,
				},
			},
		},
		{
			Id: "2",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"level":   10,
					"defense": 50,
				},
			},
		},
		{
			Id: "3",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"level":   10,
					"defense": 522,
				},
			},
		}, {
			Id: "4",
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"level": 10,
					"mana":  1,
				},
			},
		},
	}

	poolNameToTickets := map[string][]*pb.Ticket{
		"pool1": tickets[:2],
		"pool2": tickets[2:],
	}

	p := &internalMmf.MatchFunctionParams{
		Logger:            &logrus.Entry{},
		ProfileName:       "test-profile",
		PoolNameToTickets: poolNameToTickets,
	}

	matches, err := MakeMatches(p)
	assert.Nil(err)
	assert.Equal(len(matches), 2)

	actual := []*pb.Match{}
	for _, match := range matches {
		actual = append(actual, &pb.Match{
			MatchProfile:  match.MatchProfile,
			MatchFunction: match.MatchFunction,
			Tickets:       match.Tickets,
			Extensions:    match.Extensions,
		})
	}

	matchGen := func(tickets []*pb.Ticket) *pb.Match {
		tids := []string{}
		for _, ticket := range tickets {
			tids = append(tids, ticket.GetId())
		}

		evaluationInput, err := ptypes.MarshalAny(&pb.DefaultEvaluationCriteria{
			Score: scoreCalculator(tickets),
		})
		if err != nil {
			t.Fatal(err)
		}

		return &pb.Match{
			MatchProfile:  p.ProfileName,
			MatchFunction: matchName,
			Tickets:       tickets,
			Extensions: map[string]*any.Any{
				"evaluation_input": evaluationInput,
			},
		}
	}

	for _, tickets := range poolNameToTickets {
		assert.Contains(actual, matchGen(tickets))
	}
}
