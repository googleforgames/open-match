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

// generateProfiles generates test profiles for the default evaluator tutorial.
func generateProfiles() []*pb.MatchProfile {
	var profiles []*pb.MatchProfile
	// Create custom skill range for tickets related to range of random skill level generated in ticket.go
	for skill := 0; skill <= 5; skill++ {
		profiles = append(profiles, &pb.MatchProfile{
			Name: "skill_based_profile",
			Pools: []*pb.Pool{
				{
					Name: "pool_skill" + skill,
					DoubleRangeFilters: []*pb.DoubleRangeFilter{
						{
							// Add DoubleArg name specified in ticket

							// Add Min and Max arguments to specify range for profile
						},
					},
				},
			},
		},
		)
	}

	return profiles
}
