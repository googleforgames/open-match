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

// Package testcases contains lists of ticket filtering test cases.
package testcases

import (
	"fmt"
	"math"
	"time"

	"github.com/golang/protobuf/ptypes"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	"open-match.dev/open-match/pkg/pb"
)

// TestCase defines a single filtering test case to run.
type TestCase struct {
	Name   string
	Ticket *pb.Ticket
	Pool   *pb.Pool
}

// IncludedTestCases returns a list of test cases where using the given filter,
// the ticket is included in the result.
func IncludedTestCases() []TestCase {
	now := time.Now()
	return []TestCase{
		{
			"no filters or fields",
			&pb.Ticket{},
			&pb.Pool{},
		},

		simpleDoubleRange("simpleInRange", 5, 0, 10),
		simpleDoubleRange("exactMatch", 5, 5, 5),
		simpleDoubleRange("infinityMax", math.Inf(1), 0, math.Inf(1)),
		simpleDoubleRange("infinityMin", math.Inf(-1), math.Inf(-1), 0),

		{
			"String equals simple positive",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					StringArgs: map[string]string{
						"field": "value",
					},
				},
			},
			&pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field",
						Value:     "value",
					},
				},
			},
		},

		{
			"TagPresent simple positive",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					Tags: []string{
						"mytag",
					},
				},
			},
			&pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "mytag",
					},
				},
			},
		},

		{
			"TagPresent multiple all present",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					Tags: []string{
						"A", "B", "C",
					},
				},
			},
			&pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "A",
					},
					{
						Tag: "C",
					},
					{
						Tag: "B",
					},
				},
			},
		},

		multipleFilters(true, true, true),

		{
			"CreatedBefore simple positive",
			&pb.Ticket{},
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"CreatedAfter simple positive",
			&pb.Ticket{},
			&pb.Pool{
				CreatedAfter: timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"Between CreatedBefore and CreatedAfter positive",
			&pb.Ticket{},
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 1)),
				CreatedAfter:  timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"No time search criteria positive",
			&pb.Ticket{},
			&pb.Pool{},
		},
	}
}

// ExcludedTestCases returns a list of test cases where using the given filter,
// the ticket is NOT included in the result.
func ExcludedTestCases() []TestCase {
	now := time.Now()
	return []TestCase{
		{
			"DoubleRange no SearchFields",
			&pb.Ticket{},
			&pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{
					{
						DoubleArg: "field",
						Min:       math.Inf(-1),
						Max:       math.Inf(1),
					},
				},
			},
		},
		{
			"StringEquals no SearchFields",
			&pb.Ticket{},
			&pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field",
						Value:     "value",
					},
				},
			},
		},
		{
			"TagPresent no SearchFields",
			&pb.Ticket{},
			&pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "value",
					},
				},
			},
		},

		{
			"double range missing field",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					DoubleArgs: map[string]float64{
						"otherfield": 0,
					},
				},
			},
			&pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{
					{
						DoubleArg: "field",
						Min:       math.Inf(-1),
						Max:       math.Inf(1),
					},
				},
			},
		},

		simpleDoubleRange("valueTooLow", -1, 0, 10),
		simpleDoubleRange("valueTooHigh", 11, 0, 10),
		simpleDoubleRange("minIsNan", 5, math.NaN(), 10),
		simpleDoubleRange("maxIsNan", 5, 0, math.NaN()),
		simpleDoubleRange("minMaxAreNan", 5, math.NaN(), math.NaN()),
		simpleDoubleRange("valueIsNan", math.NaN(), 0, 10),
		simpleDoubleRange("valueIsNanInfRange", math.NaN(), math.Inf(-1), math.Inf(1)),
		simpleDoubleRange("allAreNan", math.NaN(), math.NaN(), math.NaN()),

		{
			"String equals simple negative", // and case sensitivity
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					StringArgs: map[string]string{
						"field": "value",
					},
				},
			},
			&pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field",
						Value:     "VALUE",
					},
				},
			},
		},

		{
			"String equals missing field",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					StringArgs: map[string]string{
						"otherfield": "othervalue",
					},
				},
			},
			&pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "field",
						Value:     "value",
					},
				},
			},
		},

		{
			"TagPresent simple negative", // and case sensitivity
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					Tags: []string{
						"MYTAG",
					},
				},
			},
			&pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "mytag",
					},
				},
			},
		},

		{
			"TagPresent multiple with one missing",
			&pb.Ticket{
				SearchFields: &pb.SearchFields{
					Tags: []string{
						"A", "B", "C",
					},
				},
			},
			&pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "A",
					},
					{
						Tag: "D",
					},
					{
						Tag: "C",
					},
				},
			},
		},

		{
			"CreatedBefore simple negative",
			&pb.Ticket{},
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"CreatedAfter simple negative",
			&pb.Ticket{},
			&pb.Pool{
				CreatedAfter: timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"Created before time range negative",
			&pb.Ticket{},
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 2)),
				CreatedAfter:  timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"Created after time range negative",
			&pb.Ticket{},
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * -1)),
				CreatedAfter:  timestamp(now.Add(time.Hour * -2)),
			},
		},

		multipleFilters(false, true, true),
		multipleFilters(true, false, true),
		multipleFilters(true, true, false),
	}
}

func simpleDoubleRange(name string, value, min, max float64) TestCase {
	return TestCase{
		"double range " + name,
		&pb.Ticket{
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"field": value,
				},
			},
		},
		&pb.Pool{
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{
					DoubleArg: "field",
					Min:       min,
					Max:       max,
				},
			},
		},
	}
}

func multipleFilters(doubleRange, stringEquals, tagPresent bool) TestCase {
	a := float64(0)
	if !doubleRange {
		a = 10
	}

	b := "hi"
	if !stringEquals {
		b = "bye"
	}

	c := "yo"
	if !tagPresent {
		c = "cya"
	}

	return TestCase{
		fmt.Sprintf("multiplefilters: %v, %v, %v", doubleRange, stringEquals, tagPresent),
		&pb.Ticket{
			SearchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{
					"a": a,
				},
				StringArgs: map[string]string{
					"b": b,
				},
				Tags: []string{c},
			},
		},
		&pb.Pool{
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{
					DoubleArg: "a",
					Min:       -1,
					Max:       1,
				},
			},
			StringEqualsFilters: []*pb.StringEqualsFilter{
				{
					StringArg: "b",
					Value:     "hi",
				},
			},
			TagPresentFilters: []*pb.TagPresentFilter{
				{
					Tag: "yo",
				},
			},
		},
	}
}

func timestamp(t time.Time) *tspb.Timestamp {
	tsp, err := ptypes.TimestampProto(t)
	if err != nil {
		panic(err)
	}

	return tsp
}
