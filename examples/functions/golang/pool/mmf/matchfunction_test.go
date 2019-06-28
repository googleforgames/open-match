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

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
)

func TestMakeMatches(t *testing.T) {
	assert := assert.New(t)

	tickets := []*pb.Ticket{
		{
			Id: "1",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level":   {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"defense": {Kind: &structpb.Value_NumberValue{NumberValue: 100}},
				},
			},
		},
		{
			Id: "2",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level":  {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"attack": {Kind: &structpb.Value_NumberValue{NumberValue: 50}},
				},
			},
		},
		{
			Id: "3",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"speed": {Kind: &structpb.Value_NumberValue{NumberValue: 522}},
				},
			},
		}, {
			Id: "4",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"level": {Kind: &structpb.Value_NumberValue{NumberValue: 10}},
					"mana":  {Kind: &structpb.Value_NumberValue{NumberValue: 1}},
				},
			},
		},
	}

	poolNameToTickets := map[string][]*pb.Ticket{
		"pool1": tickets[:2],
		"pool2": tickets[2:],
	}

	p := &mmfHarness.MatchFunctionParams{
		Logger:            &logrus.Entry{},
		ProfileName:       "test-profile",
		Rosters:           []*pb.Roster{},
		PoolNameToTickets: poolNameToTickets,
	}

	matches := MakeMatches(p)
	assert.Equal(len(matches), 2)

	actual := []*pb.Match{}
	for _, match := range matches {
		actual = append(actual, &pb.Match{
			MatchProfile:  match.MatchProfile,
			MatchFunction: match.MatchFunction,
			Ticket:        match.Ticket,
			Roster:        match.Roster,
			Properties:    match.Properties,
		})
	}

	matchGen := func(poolName string, tickets []*pb.Ticket) *pb.Match {
		tids := []string{}
		for _, ticket := range tickets {
			tids = append(tids, ticket.GetId())
		}

		return &pb.Match{
			MatchProfile:  p.ProfileName,
			MatchFunction: matchName,
			Ticket:        tickets,
			Roster:        []*pb.Roster{{Name: poolName, TicketId: tids}},
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					matchScore: {Kind: &structpb.Value_NumberValue{NumberValue: scoreCalculator(tickets)}},
				},
			},
		}
	}

	for poolName, tickets := range poolNameToTickets {
		assert.Contains(actual, matchGen(poolName, tickets))
	}
}
