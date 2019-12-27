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
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.profiles",
	})

	// Greedy config type is used to pick profiles that pull in all players.
	Greedy = "greedy"
	// MultiFilter config type is used to pick profiles that have a Pool that uses multiple filters.
	MultiFilter = "multifilter"
	// MultiPool config type is used to pick profiles that have multiple Pools with multiple filters each.
	MultiPool = "multipool"
	// ScaleProfiles
	ScaleProfiles = "scaleprofiles"
	// emptyRosterSpot is the string that represents an empty slot on a Roster.
	emptyRosterSpot = "EMPTY_ROSTER_SPOT"
)

// Generate generates test profiles for scale demo
func Generate(cfg config.View) []*pb.MatchProfile {
	profile := cfg.GetString("testConfig.profile")
	switch profile {
	case Greedy:
		return greedyProfiles(cfg)
	case MultiFilter:
		return multifilterProfiles(cfg)
	case MultiPool:
		return multipoolProfiles(cfg)
	case ScaleProfiles:
		return scaleProfiles()
	}

	logger.Warningf("Unexpected profile name %s, not returning any profiles", profile)
	return nil
}
