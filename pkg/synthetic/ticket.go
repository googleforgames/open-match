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

// Package synthetic generates fake data for testing.
package synthetic

import (
	"fmt"
	"math"
	"math/rand"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

// Tickets generates artificial tickets with these properties:
// mmr.rating = value 0 to 100, average 50, standard deviation of 15.
// one of the role properties:
//   role.dps = 1 with 40% chance
//   role.support = 1 with 33% chance
//   role.tank = 1 with 27% chance
// mode.demo = 1 to 5 (inclusive) with decreasing percentage each mode.
// region.europe-east1 = latency number from 0ms to ~300ms
// region.europe-west1 = ^
// region.europe-west2 = ^  (all of these latency numbers are really
// region.europe-west3 = ^   bad and don't represent real world data)
// region.europe-west4 = ^
func Tickets(count int) []*pb.Ticket {
	result := make([]*pb.Ticket, count)
	for i := 0; i < count; i++ {
		result[i] = CustomTicket(defaultGenerators...)
	}
	return result
}

var defaultGenerators = []GenFunc{
	Normal("mmr.rating", 50, 15, 0, 100),
	FieldChoice(map[string]int64{
		"role.dps":     40,
		"role.support": 33,
		"role.tank":    27,
	}),
	ValueChoice("mode.demo", []*structpb.Value{
		structs.Number(1),
		structs.Number(2),
		structs.Number(3),
		structs.Number(4),
		structs.Number(5),
	}, []int64{
		50, 30, 13, 5, 2,
	}),
	BadArtificalLatency(
		"region.europe-east1",
		"region.europe-west1",
		"region.europe-west2",
		"region.europe-west3",
		"region.europe-west4",
	),
}

// GenFunc defines a function which generates values for ticket properties.
type GenFunc func(func(string, *structpb.Value))

// CustomTicket creates a ticket using a custom list of generators.  If multiple
// generators try to add a value to the same property name, CustomTicket will
// panic.
func CustomTicket(generators ...GenFunc) *pb.Ticket {
	properties := structs.Struct{}
	add := func(field string, value *structpb.Value) {
		if _, ok := properties[field]; ok {
			panic(fmt.Sprintf("Field %s populated with two differnet values.", field))
		}
		properties[field] = value
	}

	for _, gen := range generators {
		gen(add)
	}

	return &pb.Ticket{Properties: properties.S()}
}

// Normal creates a generator function which sets a field to a value following
// a normal distribution.  The value is clamped by min and max.
func Normal(field string, avg, stdev, min, max float64) GenFunc {
	return func(add func(string, *structpb.Value)) {
		v := rand.NormFloat64()*stdev + avg
		if v > max {
			v = max
		}
		if v < min {
			v = min
		}

		add(field, structs.Number(v))
	}
}

// FieldChoice creates a generator function which uses a defined weights to
// choose a random property field to set, and sets that property to 1.
func FieldChoice(weights map[string]int64) GenFunc {
	options := make([]string, 0, len(weights))
	relativeWeights := make([]int64, 0, len(weights))
	for k, v := range weights {
		options = append(options, k)
		relativeWeights = append(relativeWeights, v)
	}

	choice := weightedChoice(relativeWeights)
	return func(add func(string, *structpb.Value)) {
		add(options[choice()], structs.Number(1))
	}
}

// ValueChoice creates a generator function which sets the field to a random
// value chosen by the weights.
func ValueChoice(field string, values []*structpb.Value, weights []int64) GenFunc {
	if len(values) != len(weights) {
		panic("Values and weights must have same length")
	}
	choice := weightedChoice(weights)
	return func(add func(string, *structpb.Value)) {
		add(field, values[choice()])
	}
}

func weightedChoice(weights []int64) func() int {
	{
		cp := make([]int64, len(weights))
		copy(cp, weights)
		weights = cp
	}
	totalWeight := int64(0)
	for i, v := range weights {
		weights[i] = totalWeight
		totalWeight += v
	}
	return func() int {
		r := rand.Int63n(totalWeight)
		low := 0
		high := len(weights) - 1
		for low < high {
			mid := low + ((high - low) / 2)
			if r >= weights[mid] {
				if r < weights[mid+1] {
					return mid
				}
				low = mid + 1
			} else {
				high = mid - 1
			}
		}
		return low
	}
}

// BadArtificalLatency creates some latency metrics from a list of regions.
// This is a very poor approximation of real world latency distributions.
func BadArtificalLatency(regions ...string) GenFunc {
	locations := make(map[string][2]float64)
	for _, region := range regions {
		locations[region] = [2]float64{rand.Float64(), rand.Float64()}
	}

	return func(add func(string, *structpb.Value)) {
		pos := [2]float64{rand.Float64(), rand.Float64()}
		for region, rPos := range locations {
			dx := pos[0] - rPos[0]
			dy := pos[1] - rPos[1]
			dist := math.Sqrt(dx*dx + dy*dy)
			add(region, structs.Number(dist*200))
		}
	}
}
