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

// Package mmf provides a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"github.com/rs/xid"
	"open-match.dev/open-match/examples"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

var (
	matchName = "pool-based-match"
)

// MakeMatches is where your custom matchmaking logic lives.
// This is the core match making function that will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func MakeMatches(params *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	var result []*pb.Match
	for pool, tickets := range params.PoolNameToTickets {
		if len(tickets) != 0 {
			roster := &pb.Roster{Name: pool}

			for _, ticket := range tickets {
				roster.TicketIds = append(roster.GetTicketIds(), ticket.GetId())
			}

			result = append(result, &pb.Match{
				MatchId:       xid.New().String(),
				MatchProfile:  params.ProfileName,
				MatchFunction: matchName,
				Tickets:       tickets,
				Rosters:       []*pb.Roster{roster},
				Properties: structs.Struct{
					examples.MatchScore: structs.Number(scoreCalculator(tickets)),
				}.S(),
			})
		}
	}

	return result, nil
}

// This match function defines the quality of a match as the sum of the attribute values of all tickets per match
func scoreCalculator(tickets []*pb.Ticket) float64 {
	matchScore := 0.0
	for _, ticket := range tickets {
		for _, v := range ticket.GetProperties().GetFields() {
			matchScore += v.GetNumberValue()
		}
	}
	return matchScore
}
