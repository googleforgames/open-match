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
	"fmt"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

// multifilterProfiles generates a multiple profiles, each containing a single Pool
// that specifies multiple filters to pick a partitioned player population. Note
// that across all the profiles returned, the entire population is covered and given
// the overlapping nature of filters, multiple profiles returned by this method may
// match to the same set of players.
func multifilterProfiles(cfg config.View) []*pb.MatchProfile {
	regions := cfg.GetStringSlice("testConfig.regions")
	ratingFilters := makeRangeFilters(&rangeConfig{
		name:         "Rating",
		min:          cfg.GetInt("testConfig.minRating"),
		max:          cfg.GetInt("testConfig.maxRating"),
		rangeSize:    cfg.GetInt("testConfig.multifilter.rangeSize"),
		rangeOverlap: cfg.GetInt("testConfig.multifilter.rangeOverlap"),
	})

	latencyFilters := makeRangeFilters(&rangeConfig{
		name:         "Latency",
		min:          0,
		max:          100,
		rangeSize:    70,
		rangeOverlap: 0,
	})

	var profiles []*pb.MatchProfile
	for _, region := range regions {
		for _, latency := range latencyFilters {
			for _, rating := range ratingFilters {
				poolName := fmt.Sprintf("%s_%s_%s", region, rating.name, latency.name)
				p := &pb.Pool{
					Name: poolName,
					FloatRangeFilters: []*pb.FloatRangeFilter{
						{
							Attribute: "mmr.rating",
							Min:       float64(rating.min),
							Max:       float64(rating.max),
						},
						{
							Attribute: region,
							Min:       float64(latency.min),
							Max:       float64(latency.max),
						},
					},
				}

				prof := &pb.MatchProfile{
					Name:    fmt.Sprintf("Profile_%s", poolName),
					Pools:   []*pb.Pool{p},
					Rosters: []*pb.Roster{makeRosterSlots(p.GetName(), cfg.GetInt("testConfig.ticketsPerMatch"))},
				}

				profiles = append(profiles, prof)
			}
		}
	}

	return profiles
}
