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
	"github.com/rs/xid"
	harness "open-match.dev/open-match/internal/harness/golang"
	"open-match.dev/open-match/internal/pb"
)

// MakeMatches is an example mmf function used for testing purpose and will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func MakeMatches(params *harness.MatchFunctionParams) []*pb.Match {
	var result []*pb.Match
	for pool, tickets := range params.PoolNameToTickets {
		roster := &pb.Roster{Name: pool}
		for _, ticket := range tickets {
			roster.TicketId = append(roster.TicketId, ticket.Id)
		}

		result = append(result, &pb.Match{
			MatchId:       xid.New().String(),
			MatchProfile:  params.ProfileName,
			MatchFunction: params.MmfName,
			Ticket:        tickets,
			Roster:        []*pb.Roster{roster},
		})
	}

	return result
}
