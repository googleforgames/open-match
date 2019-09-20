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

	"open-match.dev/open-match/internal/config"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.tickets",
	})
)

// Ticket generates a ticket based on the config for scale testing
func Ticket(cfg config.View) *pb.Ticket {
	characters := cfg.GetStringSlice("testConfig.characters")
	regions := cfg.GetStringSlice("testConfig.regions")
	min := cfg.GetFloat64("testConfig.minRating")
	max := cfg.GetFloat64("testConfig.maxRating")
	latencyMap := latency(regions)
	ticket := &pb.Ticket{
		Properties: structs.Struct{
			"mmr.rating": structs.Number(normalDist(40, min, max, 20)),
			// TODO: Use string attribute value for the character attribute.
			characters[rand.Intn(len(characters))]: structs.Number(float64(time.Now().Unix())),
		}.S(),
	}

	for _, r := range regions {
		ticket.Properties.Fields[r] = structs.Number(latencyMap[r])
	}

	return ticket
}

// latency generates a latency mapping of each region to a latency value. It picks
// one region with latency between 0ms to 100ms and sets latencies to all other regions
// to a value between 100ms to 300ms.
func latency(regions []string) map[string]float64 {
	latencies := make(map[string]float64)
	for _, r := range regions {
		latencies[r] = normalDist(175, 100, 300, 75)
	}

	latencies[regions[rand.Intn(len(regions))]] = normalDist(25, 0, 100, 75)
	return latencies
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
