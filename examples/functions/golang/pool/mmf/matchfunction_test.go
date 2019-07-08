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

	"open-match.dev/open-match/examples"
	"open-match.dev/open-match/pkg/pb"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/structs"
)

func TestMakeMatches(t *testing.T) {
	assert := assert.New(t)

	tickets := []*pb.Ticket{
		{
			Id: "1",
			Properties: structs.Struct{
				"level":   structs.Number(10),
				"defense": structs.Number(100),
			}.S(),
		},
		{
			Id: "2",
			Properties: structs.Struct{
				"level":  structs.Number(10),
				"attack": structs.Number(50),
			}.S(),
		},
		{
			Id: "3",
			Properties: structs.Struct{
				"level": structs.Number(10),
				"speed": structs.Number(522),
			}.S(),
		}, {
			Id: "4",
			Properties: structs.Struct{
				"level": structs.Number(10),
				"mana":  structs.Number(1),
			}.S(),
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

	matches, err := MakeMatches(p)
	assert.Nil(err)
	assert.Equal(len(matches), 2)

	actual := []*pb.Match{}
	for _, match := range matches {
		actual = append(actual, &pb.Match{
			MatchProfile:  match.MatchProfile,
			MatchFunction: match.MatchFunction,
			Tickets:       match.Tickets,
			Rosters:       match.Rosters,
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
			Tickets:       tickets,
			Rosters:       []*pb.Roster{{Name: poolName, TicketIds: tids}},
			Properties: structs.Struct{
				examples.MatchScore: structs.Number(scoreCalculator(tickets)),
			}.S(),
		}
	}

	for poolName, tickets := range poolNameToTickets {
		assert.Contains(actual, matchGen(poolName, tickets))
	}
}
