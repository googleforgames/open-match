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
package pool

import (
	"github.com/rs/xid"
	"open-match.dev/open-match/internal/pb"
	mmfHarness "open-match.dev/open-match/pkg/harness/golang"
)

// MakeMatches is where your custom matchmaking logic lives.
// This is the core match making function that will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func MakeMatches(params *mmfHarness.MatchFunctionParams) []*pb.Match {
	var result []*pb.Match
	for pool, tickets := range params.PoolNameToTickets {
		roster := &pb.Roster{Name: pool}
		for _, ticket := range tickets {
			roster.TicketId = append(roster.TicketId, ticket.Id)
		}

		result = append(result, &pb.Match{
			MatchId:       xid.New().String(),
			MatchProfile:  params.ProfileName,
			MatchFunction: "pool-based-match",
			Ticket:        tickets,
			Roster:        []*pb.Roster{roster},
		})
	}

	return result
}
