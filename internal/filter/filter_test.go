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

package filter

import (
	"testing"

	"open-match.dev/open-match/internal/filter/testcases"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

// Only test Filter and InFilters for general use cases, not specific scenearios
// on a filter/property interaction.  Filter and InFilters defer to InFilter for
// all specific Filter and Ticket Property interaction.

func TestFilter(t *testing.T) {
	tickets := []*pb.Ticket{
		{
			Id: "good",
			Properties: structs.Struct{
				"field1": structs.Number(5),
				"field2": structs.Number(5),
			}.S(),
		},

		{
			Id: "first_bad",
			Properties: structs.Struct{
				"field1": structs.Number(-5),
				"field2": structs.Number(5),
			}.S(),
		},

		{
			Id: "second_bad",
			Properties: structs.Struct{
				"field1": structs.Number(5),
				"field2": structs.Number(-5),
			}.S(),
		},

		{
			Id: "both_bad",
			Properties: structs.Struct{
				"field1": structs.Number(-5),
				"field2": structs.Number(-5),
			}.S(),
		},
	}

	filters := []*pb.Filter{
		{
			Attribute: "field1",
			Min:       0,
			Max:       10,
		},
		{
			Attribute: "field2",
			Min:       0,
			Max:       10,
		},
	}

	result := Filter(tickets, filters)

	if len(result) != 1 {
		t.Fatalf("Expected exactly 1 ticket to pass filters, got %d", len(result))
	}

	if result[0].Id != "good" {
		t.Fatalf("Expected ticket id good, got %s", result[0].Id)
	}
}

func TestInFilters(t *testing.T) {
	ticket := &pb.Ticket{
		Id: "good",
	}

	filters := []*pb.Filter{}

	passed := InFilters(ticket, filters)

	if !passed {
		t.Fatal("No filters should allow all tickets")
	}
}

func TestInFilter(t *testing.T) {
	for _, tt := range testcases.IncludedTestCases() {
		t.Run(tt.Name, func(t *testing.T) {
			if !InFilter(tt.Ticket, tt.Filter) {
				t.Error("Ticket should be included in filter.")
			}
		})
	}

	for _, tt := range testcases.ExcludedTestCases() {
		t.Run(tt.Name, func(t *testing.T) {
			if InFilter(tt.Ticket, tt.Filter) {
				t.Error("Ticket should be excluded from filter.")
			}
		})
	}
}
