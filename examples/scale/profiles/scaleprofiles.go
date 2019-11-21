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

	"open-match.dev/open-match/pkg/pb"
)

// The current setup generates 4000 profiles in total.
// Please add/remove indices from the slices below to modify the number of profiles,
var (
	regions = []string{
		"region.europe0",
		"region.europe1",
		"region.europe2",
		"region.europe3",
		"region.europe4",
		"region.europe5",
		"region.europe6",
		"region.europe7",
		"region.europe8",
		"region.europe9",
	}
	platforms = []string{
		"platform.pc",
		"platform.xbox",
		"platform.ps",
		"platform.nintendo",
		"platform.any",
	}
	playlists = []string{
		"mmr.playlist1",
		"mmr.playlist2",
		"mmr.playlist3",
		"mmr.playlist4",
		"mmr.playlist5",
		"mmr.playlist6",
		"mmr.playlist7",
		"mmr.playlist8",
	}
	gameSizeMap = map[string]int{
		"mmr.playlist1": 4,
	}
)

func scaleProfiles() []*pb.MatchProfile {
	mmrRanges := makeRangeFilters(&rangeConfig{
		name:         "mmr",
		min:          0,
		max:          100,
		rangeSize:    10,
		rangeOverlap: 0,
	})

	var profiles []*pb.MatchProfile
	for _, region := range regions {
		for _, platform := range platforms {
			for _, playlist := range playlists {
				for _, mmrRange := range mmrRanges {
					poolName := fmt.Sprintf("%s_%s_%s_%v_%v", region, platform, playlist, mmrRange.min, mmrRange.max)
					p := &pb.Pool{
						Name: poolName,
						DoubleRangeFilters: []*pb.DoubleRangeFilter{
							{
								DoubleArg: region,
								Min:       0,
								Max:       math.MaxFloat64,
							},
							{
								DoubleArg: platform,
								Min:       0,
								Max:       math.MaxFloat64,
							},
							{
								DoubleArg: playlist,
								Min:       float64(mmrRange.min),
								Max:       float64(mmrRange.max),
							},
						},
					}
					prof := &pb.MatchProfile{
						Name:    fmt.Sprintf("Profile_%s", poolName),
						Pools:   []*pb.Pool{p},
						Rosters: []*pb.Roster{makeRosterSlots(p.GetName(), gameSizeMap[playlist])},
					}

					profiles = append(profiles, prof)
				}
			}
		}
	}

	return profiles
}
