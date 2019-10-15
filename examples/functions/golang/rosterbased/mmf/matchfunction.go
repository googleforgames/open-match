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
	matchName       = "roster-based-matchfunction"
	emptyRosterSpot = "EMPTY_ROSTER_SPOT"
	logger          = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "mmf.rosterbased",
	})
)

func MakeMatches(p *mmfHarness.MatchFunctionParams) ([]*pb.Match, error) {
	// This roster based match function expects the match profile to have a
	// populated roster specifying the empty slots for each pool name and also
	// have the ticket pools referenced in the roster. It generates matches by
	// populating players from the specified pools into rosters.
	wantTickets, err := wantPoolTickets(p.Rosters)
	if err != nil {
		return nil, err
	}

	var matches []*pb.Match
	count := 0
	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		matchRosters := []*pb.Roster{}

		// Loop through each pool wanted in the rosters and pick the number of
		// wanted players from the respective Pool.
		for poolName, want := range wantTickets {
			have, ok := p.PoolNameToTickets[poolName]
			if !ok {
				// A wanted Pool was not found in the Pools specified in the profile.
				insufficientTickets = true
				break
			}

			if len(have) < want {
				// The Pool in the profile has fewer tickets than what the roster needs.
				insufficientTickets = true
				break
			}

			// Populate the wanted tickets from the Tickets in the corresponding Pool.
			matchTickets = append(matchTickets, have[0:want]...)
			p.PoolNameToTickets[poolName] = have[want:]
			var ids []string
			for _, ticket := range matchTickets {
				ids = append(ids, ticket.Id)
			}

			matchRosters = append(matchRosters, &pb.Roster{
				Name:      poolName,
				TicketIds: ids,
			})
		}

		if insufficientTickets {
			// Ran out of Tickets. Matches cannot be created from the remaining Tickets.
			break
		}

		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.ProfileName, time.Now().Format("2006-01-02T15:04:05.00"), count),
			MatchProfile:  p.ProfileName,
			MatchFunction: matchName,
			Tickets:       matchTickets,
			Rosters:       matchRosters,
		})

		count++
	}

	return matches, nil
}

// wantPoolTickets parses the roster to return a map of the Pool name to the
// number of empty roster slots for that Pool.
func wantPoolTickets(rosters []*pb.Roster) (map[string]int, error) {
	wantTickets := make(map[string]int)
	for _, r := range rosters {
		if _, ok := wantTickets[r.Name]; ok {
			// We do not expect multiple Roster Pools to have the same name.
			logger.Errorf("multiple rosters with same name not supported")
			return nil, status.Error(codes.InvalidArgument, "multiple rosters with same name not supported")
		}

		wantTickets[r.Name] = 0
		for _, slot := range r.TicketIds {
			if slot == emptyRosterSpot {
				wantTickets[r.Name] = wantTickets[r.Name] + 1
			}
		}
	}

	return wantTickets, nil
}
