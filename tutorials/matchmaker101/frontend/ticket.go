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
	"math/rand"

	"open-match.dev/open-match/internal/testing/e2e"
	"open-match.dev/open-match/pkg/pb"
)

var (
	// Tickets will be generated with a rating that follows a normal distribution
	// using the below parameters.
	minRating float64 = 0
	maxRating float64 = 100
	avgRating float64 = 40
	stdDev    float64 = 20
)

// Ticket generates a Ticket with data using the package configuration.
func makeTicket() *pb.Ticket {
	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				e2e.DoubleArgMMR: normalDist(avgRating, minRating, maxRating, stdDev),
			},
		},
	}

	return ticket
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
