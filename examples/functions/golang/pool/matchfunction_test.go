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

package pool

import (
	"testing"

	"open-match.dev/open-match/internal/pb"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	mmfHarness "open-match.dev/open-match/pkg/harness/golang"
)

func TestMakeMatches(t *testing.T) {
	assert := assert.New(t)

	poolNameToTickets := map[string][]*pb.Ticket{
		"pool1": []*pb.Ticket{{Id: "1"}, {Id: "2"}},
		"pool2": []*pb.Ticket{{Id: "3"}, {Id: "4"}},
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
		})
	}

	assert.Contains(actual, &pb.Match{
		MatchProfile:  p.ProfileName,
		MatchFunction: matchName,
		Ticket:        []*pb.Ticket{{Id: "1"}, {Id: "2"}},
		Roster:        []*pb.Roster{&pb.Roster{Name: "pool1", TicketId: []string{"1", "2"}}},
	})
	assert.Contains(actual, &pb.Match{
		MatchProfile:  p.ProfileName,
		MatchFunction: matchName,
		Ticket:        []*pb.Ticket{{Id: "3"}, {Id: "4"}},
		Roster:        []*pb.Roster{&pb.Roster{Name: "pool2", TicketId: []string{"3", "4"}}},
	})
}
