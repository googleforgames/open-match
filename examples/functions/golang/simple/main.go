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

	goHarness "open-match.dev/open-match/internal/harness/golang"
	"open-match.dev/open-match/internal/pb"
)

func main() {
	// Invoke the harness to setup a GRPC service that handles requests to run the
	// match function. The harness itself queries open match for player pools for
	// the specified request and passes the pools to the match function to generate
	// proposals.
	goHarness.RunMatchFunction(&goHarness.FunctionSettings{
		FunctionName: "simple-matchfunction",
		Func:         makeMatches,
	})
}

// makeMatches is where your custom matchmaking logic lives.
func makeMatches(view *goHarness.MatchFunctionParams) []*pb.Match {
	// This simple match function does the following things
	// 1. Deduplicates the tickets from the pools into a single list.
	// 2. Groups players into 1v1 matches.

	tickets := make(map[string]*pb.Ticket)

	for _, pool := range view.PoolNameToTickets {
		for _, ticket := range pool {
			tickets[ticket.Id] = ticket
		}
	}

	var matches []*pb.Match

	t := time.Now().Format("2006-01-02T15:04:05.00")

	thisMatch := make([]*pb.Ticket, 0, 2)
	matchNum := 0

	for _, ticket := range tickets {
		thisMatch = append(thisMatch, ticket)

		if len(thisMatch) >= 2 {
			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", view.ProfileName, t, matchNum),
				MatchProfile:  view.ProfileName,
				MatchFunction: "a-simple-matchfunction",
				Ticket:        thisMatch,
			})

			thisMatch = make([]*pb.Ticket, 0, 2)
			matchNum++
		}
	}

	return matches
}
