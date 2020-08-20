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
	"log"
	"math/rand"

	"open-match.dev/open-match/pkg/pb"
)

// Ticket generates a Ticket with a mode search field that has one of the
// randomly selected modes.
func makeTicket() *pb.Ticket {
	ticket := &pb.Ticket{
		SearchFields: &pb.SearchFields{
			DoubleArgs: map[string]float64{
				"mmr": getSkill(),
			},
		},
	}

	return ticket
}

// enterQueueTime returns the time at which the Ticket entered matchmaking queue. To simulate
// variability, we pick any random second interval in the past 5 seconds as the players queue
// entry time.
func getSkill() float64 {
	skill := rand.NormFloat64()*1.5 + 5
	if skill < 0.0 {
		skill = 0.0
		log.Println("Skill level for ticket is ", skill)
		return skill
	}
	if skill > 5.0 {
		skill = 5.0
		log.Println("Skill level for ticket is ", skill)
		return skill
	}
	log.Println("Skill level for ticket is ", skill)
	return skill
}
