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

package tickets

import (
	"math/rand"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

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
		"mmr.playlist9",
		"mmr.playlist10",
		"mmr.playlist11",
		"mmr.playlist12",
		"mmr.playlist13",
		"mmr.playlist14",
		"mmr.playlist15",
	}
)

// Ticket generates a ticket for profile scale testing
func Ticket() *pb.Ticket {
	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: make(map[string]float64),
		},
	}

	addRegionsAttributes(ticket.SearchFields)
	addPlatformAttributes(ticket.SearchFields)
	addPlaylistAttributes(ticket.SearchFields)

	return ticket
}

func addRegionsAttributes(fields *pb.SearchFields) {
	// Each ticket can have 1-3 regions. Pick a random number of regions between 1 and 3
	regionCount := rand.Intn(3) + 1

	// Pick a random indices for playlists.
	regionIndex := randomInRange(len(regions)-1, 0, regionCount)

	// Add an attribute for each picked region
	for r := range regionIndex {
		fields.DoubleArgs[regions[r]] = float64(time.Now().Unix())
	}
}

func addPlatformAttributes(fields *pb.SearchFields) {
	fields.DoubleArgs[platforms[rand.Intn(len(platforms))]] = float64(time.Now().Unix())
}

func addPlaylistAttributes(fields *pb.SearchFields) {
	// Each ticket can have 1-3 playlists. Pick a random number of playlists between 1 and 3
	plCount := rand.Intn(3) + 1

	// Pick a random indices for playlists.
	plIndex := randomInRange(len(playlists)-1, 0, plCount)

	// For each of the picked playlist, add an attribute with a mmr from a normal distribution
	for pl := range plIndex {
		fields.DoubleArgs[playlists[pl]] = normalDist(40, 0, 100, 20)
	}
}

func randomInRange(max int, min int, count int) []int {
	if count <= 0 {
		return []int{}
	}

	var exists = make(map[int]bool)
	for len(exists) < count {
		rnum := rand.Intn(max-min+1) + min
		exists[rnum] = true
	}

	var result []int
	for k := range exists {
		result = append(result, k)
	}

	return result
}

// normalDist generates a random integer in a normal distribution
func normalDist(avg float64, min float64, max float64, stdev float64) float64 {
	sample := (rand.NormFloat64() * stdev) + avg
	switch {
	case sample > max:
		sample = max
	case sample < min:
		sample = min
	}

	return sample
}
