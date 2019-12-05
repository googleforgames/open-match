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
	"time"

	"open-match.dev/open-match/pkg/pb"
)

// Ticket generates a Ticket with a mode search field that has one of the
// randomly selected modes.
func makeTicket() *pb.Ticket {
	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			// Tags can support multiple values but for simplicity, the demo function
			// assumes only single mode selection per Ticket.
			Tags: gameModes(),
			DoubleArgs: map[string]float64{
				"time.enterqueue": enterQueueTime(),
			},
		},
	}

	return ticket
}

// gameModes() selects up to two unique game modes for this Ticket.
func gameModes() []string {
	modes := []string{"mode.demo", "mode.ctf", "mode.battleroyale", "mode.2v2"}
	count := rand.Intn(2) + 1
	selected := make(map[string]bool)
	for i := 0; i < count; i++ {
		mode := modes[rand.Intn(len(modes))]
		if !selected[mode] {
			selected[mode] = true
		}
	}

	var result []string
	for k := range selected {
		result = append(result, k)
	}

	return result
}

// enterQueueTime returns the time at which the Ticket entered matchmaking queue. To simulate
// variability, we pick any random second interval in the past 5 seconds as the players queue
// entry time.
func enterQueueTime() float64 {
	return float64(time.Now().Add(-time.Duration(rand.Intn(5)) * time.Second).UnixNano())
}
