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
	"math"
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/internal/pb"
)

// Only test Filter and InFilters for general use cases, not specific scenearios
// on a filter/property interaction.  Filter and InFilters defer to InFilter for
// all specific Filter and Ticket Property interaction.

func TestFilter(t *testing.T) {
	tickets := []*pb.Ticket{
		&pb.Ticket{
			Id: "good",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"field1": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
					"field2": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				},
			},
		},

		&pb.Ticket{
			Id: "first_bad",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"field1": {Kind: &structpb.Value_NumberValue{NumberValue: -5}},
					"field2": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
				},
			},
		},

		&pb.Ticket{
			Id: "second_bad",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"field1": {Kind: &structpb.Value_NumberValue{NumberValue: 5}},
					"field2": {Kind: &structpb.Value_NumberValue{NumberValue: -5}},
				},
			},
		},

		&pb.Ticket{
			Id: "both_bad",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"field1": {Kind: &structpb.Value_NumberValue{NumberValue: -5}},
					"field2": {Kind: &structpb.Value_NumberValue{NumberValue: -5}},
				},
			},
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
	for _, tt := range includedTestCases {
		t.Run(tt.ticket.Id, func(t *testing.T) {
			if !InFilter(tt.ticket, tt.filter) {
				t.Error("Ticket should be included in filter.")
			}
		})
	}

	for _, tt := range excludedTestCases {
		t.Run(tt.ticket.Id, func(t *testing.T) {
			if InFilter(tt.ticket, tt.filter) {
				t.Error("Ticket should be excluded from filter.")
			}
		})
	}
}

type testCase struct {
	ticket *pb.Ticket
	filter *pb.Filter
}

var includedTestCases = []testCase{
	simpleRangeTest("simpleInRange", 5, 0, 10),
	simpleRangeTest("exactMatch", 5, 5, 5),
	simpleRangeTest("infinityMax", math.Inf(1), 0, math.Inf(1)),
	simpleRangeTest("infinityMin", math.Inf(-1), math.Inf(-1), 0),
}

var excludedTestCases = []testCase{
	{
		&pb.Ticket{
			Id: "missingField",
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{},
			},
		},
		&pb.Filter{
			Attribute: "field",
			Min:       5,
			Max:       5,
		},
	},

	{
		&pb.Ticket{
			Id: "missingFieldsMap",
			Properties: &structpb.Struct{
				Fields: nil,
			},
		},
		&pb.Filter{
			Attribute: "field",
			Min:       5,
			Max:       5,
		},
	},

	{
		&pb.Ticket{
			Id:         "missingProperties",
			Properties: nil,
		},
		&pb.Filter{
			Attribute: "field",
			Min:       5,
			Max:       5,
		},
	},

	simpleRangeTest("valueTooLow", -1, 0, 10),
	simpleRangeTest("valueTooHigh", 11, 0, 10),
	simpleRangeTest("minIsNan", 5, math.NaN(), 10),
	simpleRangeTest("maxIsNan", 5, 0, math.NaN()),
	simpleRangeTest("valueIsNan", math.NaN(), 0, 10),
}

func simpleRangeTest(name string, value, min, max float64) testCase {
	return testCase{
		&pb.Ticket{
			Id: name,
			Properties: &structpb.Struct{
				Fields: map[string]*structpb.Value{
					"field": {Kind: &structpb.Value_NumberValue{NumberValue: value}},
				},
			},
		},
		&pb.Filter{
			Attribute: "field",
			Min:       min,
			Max:       max,
		},
	}
}
