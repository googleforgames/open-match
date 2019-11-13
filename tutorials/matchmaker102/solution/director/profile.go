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
	"fmt"

	"open-match.dev/open-match/pkg/pb"
)

// generateProfiles generates test profiles for the matchmaker101 tutorial.
func generateProfiles() []*pb.MatchProfile {
	var profiles []*pb.MatchProfile
	modes := []string{"mode.demo", "mode.ctf", "mode.battleroyale"}
	roles := []string{"role.dps", "role.support", "role.tank"}
	for _, mode := range modes {
		var pools []*pb.Pool
		for _, role := range roles {
			pools = append(pools, &pb.Pool{
				Name: fmt.Sprintf("pool_%s_%s", mode, role),
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: mode,
					},
				},
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "attributes.role",
						Value:     role,
					},
				},
			})
		}

		profiles = append(profiles, &pb.MatchProfile{
			Name:  "profile_" + mode,
			Pools: pools,
		})
	}

	return profiles
}
