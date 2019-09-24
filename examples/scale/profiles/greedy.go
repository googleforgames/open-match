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

package profiles

import (
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

// greedyProfiles generates a single profile that has only one Pool that has a single
// filter which covers the entire range of player ranks, thereby pulling in the entire
// player population during each profile execution.
func greedyProfiles(cfg config.View) []*pb.MatchProfile {
	return []*pb.MatchProfile{
		{
			Name: "greedy",
			Pools: []*pb.Pool{
				{
					Name: "all",
					FloatRangeFilters: []*pb.FloatRangeFilter{
						{
							Attribute: e2e.AttributeMMR,
							Min:       float64(cfg.GetInt("testConfig.minRating")),
							Max:       float64(cfg.GetInt("testConfig.maxRating")),
						},
					},
				},
			},
			Rosters: []*pb.Roster{
				makeRosterSlots("all", cfg.GetInt("testConfig.ticketsPerMatch")),
			},
		},
	}
}
