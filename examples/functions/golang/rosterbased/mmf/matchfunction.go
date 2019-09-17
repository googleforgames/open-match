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
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "roster-based-matchfunction"
	emptyRosterSpot = "EMPTY_ROSTER_SPOT"
	logger    = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "mmf.rosterbased",
	})
)

// MakeMatches is where your custom matchmaking logic lives.
func MakeMatches(p *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	// This simple match function does the following things
	// 1. Deduplicates the tickets from the pools into a single list.
	// 2. Groups players into 1v1 matches.
	start := time.Now()

	wantTickets, err := wantPoolTickets(p.Rosters)
	if err != nil {
		return nil, err
	}

	if !sufficientPlayers(p.PoolNameToTickets, wantTickets) {
		return nil, status.Error(codes.InvalidArgument, "Tickets map does not contain sufficient tickets to make a match")
	}

	var matches []*pb.Match
	matchNum := 0
	poolTickets := p.PoolNameToTickets
	t := time.Now().Format("2006-01-02T15:04:05.00")
	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		// Populate 5 players from each pool in a match. If either pool runs out of tickets, exit.
		for poolName, pool := range poolTickets {
			if len(pool) < 5 {
				logger.Infof("Insufficient tickets in Pool %v. Cannot make match.", poolName)
				insufficientTickets = true
				break
			}

			matchTickets = append(matchTickets, pool[0:5]...)
			poolTickets[poolName] = pool[5:]
		}

		if insufficientTickets {
			break
		}

		// We have ranged over all pools, sufficient tickets for the match - create one.
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", p.ProfileName, t, matchNum),
			MatchProfile:  p.ProfileName,
			MatchFunction: matchName,
			Tickets:       matchTickets,
		})
	}

	logger.Infof("function execution took %v, returned %v matches", time.Since(start), len(matches))
	return matches, nil
}

func sufficientPlayers(haveTickets map[string][]*pb.Ticket, wantTickets map[string]int) bool {
	for name, tickets := range haveTickets {
		if val, ok := wantTickets[name]; ok {
			// Pool in the profile found in the roster, validate that we have sufficient tickets. 
			if val > len(tickets) {
				// Tickets returned for this Pool are fewer than the empty slots for this Pool in the Roster.
				return false
			}

			// Pool is present in Roster and sufficient tickets present.
			delete(wantTickets, name)
		}
	}

	if len(wantTickets) != 0 {
		// There are some rosters that did not have any Pool in the profile.
		return false
	}

	return true
}

// wantPoolTickets returns a map of Pool name to number of empty ticket slots for that Pool.
func wantPoolTickets(rosters []*pb.Roster) (map [string]int, error) {
	wantTickets := make(map[string]int)
	for _, r := range rosters {
		if _, ok := wantTickets[r.Name]; ok {
			// We do not expect multiple Roster Pools to have the same name.
			logger.Errorf("multiple rosters with same name not supported by this mmf")
			return nil, status.Error(codes.InvalidArgument, "multiple rosters with same name not supported by this mmf")
		}
	
		wantTickets[r.Name] = 0
		for _, slot := range r.TicketIds {
			if slot == "emptyRosterSpot" {
				wantTickets[r.Name] = wantTickets[r.Name] + 1
			}
		}
	}
	
	return wantTickets, nil
}