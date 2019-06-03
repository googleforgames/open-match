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
	"testing"

	"open-match.dev/open-match/internal/pb"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	harness "open-match.dev/open-match/internal/harness/golang"
)

func TestMakeMatches(t *testing.T) {
	assert := assert.New(t)

	poolNameToTickets := map[string][]*pb.Ticket{
		"pool1": []*pb.Ticket{{Id: "1"}, {Id: "2"}},
		"pool2": []*pb.Ticket{{Id: "3"}, {Id: "4"}},
	}

	p := &harness.MatchFunctionParams{
		Logger:            &logrus.Entry{},
		ProfileName:       "test-profile",
		Rosters:           []*pb.Roster{},
		PoolNameToTickets: poolNameToTickets,
		MmfName:           "test-mmf",
	}

	matches := MakeMatches(p)
	assert.Equal(len(matches), 2)

	expected := []*pb.Match{}
	for _, match := range matches {
		expected = append(expected, &pb.Match{
			MatchProfile:  match.MatchProfile,
			MatchFunction: match.MatchFunction,
			Ticket:        match.Ticket,
			Roster:        match.Roster,
		})
	}

	assert.Contains(expected, &pb.Match{
		MatchProfile:  p.ProfileName,
		MatchFunction: p.MmfName,
		Ticket:        []*pb.Ticket{{Id: "1"}, {Id: "2"}},
		Roster:        []*pb.Roster{&pb.Roster{Name: "pool1", TicketId: []string{"1", "2"}}},
	})
	assert.Contains(expected, &pb.Match{
		MatchProfile:  p.ProfileName,
		MatchFunction: p.MmfName,
		Ticket:        []*pb.Ticket{{Id: "3"}, {Id: "4"}},
		Roster:        []*pb.Roster{&pb.Roster{Name: "pool2", TicketId: []string{"3", "4"}}},
	})
}
