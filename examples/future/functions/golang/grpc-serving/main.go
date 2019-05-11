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

// Package main a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package main

import (
	"fmt"
	"time"

	"open-match.dev/open-match/internal/future/pb"

	"github.com/sirupsen/logrus"
)

// makeMatches is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
// Input:
//  - logger:
//			A logger used to generate error/debug logs
//  - properties:
//			'Properties' of a MatchObject.
//  - rosters:
//			An array of Rosters. By convention, your input Roster contains players already in
//          the match, and the names of pools to search when trying to fill an empty slot.
//  - filteredTickets:
//			A map that contains mappings from pool name to a list of tickets that satisfied the filters in the pool
func makeMatches(logger *logrus.Entry, profileName string, properties string, rosters []*pb.Roster, poolNameToTickets map[string][]*pb.Ticket) []*pb.Match {
	// This simple match function does the following things
	// 1. Flatten the poolNameToTickets map into an array
	// 2. Fill in the rosters by iterating through the tickets array
	// 3. Create one single match candidate and return it.

	allTickets := make([]*pb.Ticket, 0)
	ticketOffset := 0
	for _, tickets := range poolNameToTickets {
		allTickets = append(allTickets, tickets...)
	}

	// This example does nothing but fills the rosters until they are full.
	for _, roster := range rosters {
		rosterSize := len(roster.TicketId)
		logger.Tracef("Filling roster: %s, roster size: %d", roster.Name, rosterSize)
		for rosterOffset := 0; rosterOffset < rosterSize; rosterOffset++ {
			roster.TicketId[rosterOffset] = allTickets[ticketOffset].Id
			ticketOffset++
		}
	}

	return []*pb.Match{
		&pb.Match{
			MatchId:       fmt.Sprintf("profile-%s-time-%s", profileName, time.Now().Format("00:00")),
			MatchProfile:  profileName,
			MatchFunction: "a-simple-matchfunction",
			Ticket:        allTickets[:ticketOffset:ticketOffset],
			Roster:        rosters,
			Properties:    properties,
		},
	}
}
