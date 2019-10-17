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

package main

import (
	"open-match.dev/open-match/pkg/pb"
)

var (
	// emptyRosterSpot is the string that represents an empty slot on a Roster.
	emptyRosterSpot         = "EMPTY_ROSTER_SPOT"
	minRating       float64 = 0
	maxRating       float64 = 100
	ticketsPerMatch         = 4
)

// generateProfiles generates test profiles for the matchmaker101 tutorial.
func generateProfiles() []*pb.MatchProfile {
	return []*pb.MatchProfile{
		{
			Name: "basic_mmr_profile",
			Pools: []*pb.Pool{
				{
					Name: "default",
					DoubleRangeFilters: []*pb.DoubleRangeFilter{
						{
							DoubleArg: "attribute.mmr",
							Min:       minRating,
							Max:       maxRating,
						},
					},
				},
			},
			Rosters: []*pb.Roster{
				makeRosterSlots("default", ticketsPerMatch),
			},
		},
	}
}

// makeRosterSlots generates a roster with the specified name and with the
// specified number of empty roster slots.
func makeRosterSlots(name string, count int) *pb.Roster {
	roster := &pb.Roster{
		Name: name,
	}

	for i := 0; i <= count; i++ {
		roster.TicketIds = append(roster.TicketIds, emptyRosterSpot)
	}

	return roster
}
