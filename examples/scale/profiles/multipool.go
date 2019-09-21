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
	"math"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

// multipoolProfiles generates a multiple profiles, each containing a multiple player
// Pools specifying multiple filters each. The profile also requests a roster of a
// configured  number of players per pool to be placed in a match.
func multipoolProfiles(cfg config.View) []*pb.MatchProfile {
	characters := cfg.GetStringSlice("testConfig.characters")
	regions := cfg.GetStringSlice("testConfig.regions")
	ratingFilters := makeRangeFilters(&rangeConfig{
		name:         "Rating",
		min:          cfg.GetInt("testConfig.minRating"),
		max:          cfg.GetInt("testConfig.maxRating"),
		rangeSize:    cfg.GetInt("testConfig.multipool.rangeSize"),
		rangeOverlap: cfg.GetInt("testConfig.multipool.rangeOverlap"),
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
				var pools []*pb.Pool
				var rosters []*pb.Roster
				for _, character := range characters {
					poolName := fmt.Sprintf("%s_%s_%s_%s", region, rating.name, latency.name, character)
					p := &pb.Pool{
						Name: poolName,
						FloatRangeFilters: []*pb.FloatRangeFilter{
							// TODO: Use StringEqualsFilter for the character attribute.
							{
								Attribute: character,
								Min:       0,
								Max:       math.MaxFloat64,
							},
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

					rosters = append(rosters, makeRosterSlots(poolName, cfg.GetInt("testConfig.multipool.characterCount")))
					pools = append(pools, p)
				}

				prof := &pb.MatchProfile{
					Name:    fmt.Sprintf("Profile_%s", fmt.Sprintf("%s_%s_%s", region, rating.name, latency.name)),
					Pools:   pools,
					Rosters: rosters,
				}

				profiles = append(profiles, prof)
			}
		}
	}

	return profiles
}
