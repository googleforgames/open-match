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

// Package pool provides a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package main

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
		"pool1": []*pb.Ticket{{Id: "1"}, {Id: "2"}, {Id: "3"}},
		"pool2": []*pb.Ticket{{Id: "4"}},
		"pool3": []*pb.Ticket{{Id: "5"}, {Id: "6"}, {Id: "7"}},
	}

	p := &mmfHarness.MatchFunctionParams{
		Logger:            &logrus.Entry{},
		ProfileName:       "test-profile",
		Rosters:           []*pb.Roster{},
		PoolNameToTickets: poolNameToTickets,
	}

	matches := MakeMatches(p)
	assert.Equal(len(matches), 3)

	for _, match := range matches {
		assert.Equal(2, len(match.Ticket))
		assert.Equal("a-simple-1v1-matchfunction", match.MatchFunction)
		assert.Equal(p.ProfileName, match.MatchProfile)
		assert.Nil(match.Roster)
	}
}
