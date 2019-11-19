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

	"open-match.dev/open-match/pkg/pb"
)

type rangeFilter struct {
	name string
	min  int
	max  int
}

type rangeConfig struct {
	name         string
	min          int
	max          int
	rangeSize    int
	rangeOverlap int
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

// makeRangeFilters generates multiple filters over a given range based on
// the size of the range and the overlap specified for the filters.
func makeRangeFilters(config *rangeConfig) []*rangeFilter {
	var filters []*rangeFilter
	r := config.min
	for r <= config.max {
		max := r + config.rangeSize
		if max > config.max {
			max = config.max
		}

		filters = append(filters, &rangeFilter{
			name: fmt.Sprintf("%s_%dto%d", config.name, r, max),
			min:  r,
			max:  max,
		})

		r = r + 1 + (config.rangeSize - config.rangeOverlap)
	}

	return filters
}
