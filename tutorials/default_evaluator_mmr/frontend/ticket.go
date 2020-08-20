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
	// Uncomment if following the tutorials
	// "time"
	// "math/rand"

	"open-match.dev/open-match/pkg/pb"
)

// Ticket generates a Ticket with a mode search field that has one of the
// randomly selected modes.
func makeTicket() *pb.Ticket {
	ticket := &pb.Ticket{
		DoubleArgs: map[string]float64{
				[Insert DoubleArgs Name]: getSkill(),
			},
	}

	return ticket
}

func getSkill() float64 {
	// Implement your logic to return a random skill level (float64)
	return 0.0
}
