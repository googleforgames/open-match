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

// Package testing provides testing primitives for the codebase.
package testing

import (
	"fmt"

	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

// Property defines the required fields that we need to generate tickets for testing.
type Property struct {
	Name     string
	Min      float64
	Max      float64
	Interval float64
}

// GenerateTickets takes in two property manifests to generate tickets with two fake properties for testing.
func GenerateTickets(manifest1, manifest2 Property) []*pb.Ticket {
	testTickets := make([]*pb.Ticket, 0)

	for i := manifest1.Min; i < manifest1.Max; i += manifest1.Interval {
		for j := manifest2.Min; j < manifest2.Max; j += manifest2.Interval {
			testTickets = append(testTickets, &pb.Ticket{
				Id: fmt.Sprintf("%s%f-%s%f", manifest1.Name, i, manifest2.Name, j),
				Properties: structs.Struct{
					manifest1.Name: structs.Number(i),
					manifest2.Name: structs.Number(j),
				}.S(),
			})
		}
	}

	return testTickets
}
