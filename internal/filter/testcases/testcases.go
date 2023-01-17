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

	"google.golang.org/protobuf/types/known/timestamppb"
	"open-match.dev/open-match/pkg/pb"
)

// TestCase defines a single filtering test case to run.
type TestCase struct {
	Name         string
	SearchFields *pb.SearchFields
	Pool         *pb.Pool
}

// IncludedTestCases returns a list of test cases where using the given filter,
// the ticket is included in the result.
func IncludedTestCases() []TestCase {
	now := time.Now()
	return []TestCase{
		{
			"no filters or fields",
			nil,
			&pb.Pool{},
		},

		simpleDoubleRange("simpleInRange", 5, 0, 10, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("simpleInRange", 5, 0, 10, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("simpleInRange", 5, 0, 10, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("simpleInRange", 5, 0, 10, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("exactMatch", 5, 5, 5, pb.DoubleRangeFilter_NONE),

		simpleDoubleRange("infinityMax", math.Inf(1), 0, math.Inf(1), pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("infinityMax", math.Inf(1), 0, math.Inf(1), pb.DoubleRangeFilter_MIN),

		simpleDoubleRange("infinityMin", math.Inf(-1), math.Inf(-1), 0, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("infinityMin", math.Inf(-1), math.Inf(-1), 0, pb.DoubleRangeFilter_MAX),

		simpleDoubleRange("excludeNone", 0, 0, 1, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("excludeNone", 1, 0, 1, pb.DoubleRangeFilter_NONE),

		simpleDoubleRange("excludeMin", 1, 0, 1, pb.DoubleRangeFilter_MIN),

		simpleDoubleRange("excludeMax", 0, 0, 1, pb.DoubleRangeFilter_MAX),

		simpleDoubleRange("excludeBoth", 2, 0, 3, pb.DoubleRangeFilter_BOTH),
		simpleDoubleRange("excludeBoth", 1, 0, 3, pb.DoubleRangeFilter_BOTH),

		{
			"String equals simple positive",
			&pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
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
			&pb.SearchFields{
				Tags: []string{
					"mytag",
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
			&pb.SearchFields{
				Tags: []string{
					"A", "B", "C",
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
			nil,
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"CreatedAfter simple positive",
			nil,
			&pb.Pool{
				CreatedAfter: timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"Between CreatedBefore and CreatedAfter positive",
			nil,
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 1)),
				CreatedAfter:  timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"No time search criteria positive",
			nil,
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
			nil,
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
			nil,
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
			nil,
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
			&pb.SearchFields{
				DoubleArgs: map[string]float64{
					"otherfield": 0,
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

		simpleDoubleRange("exactMatch", 5, 5, 5, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("exactMatch", 5, 5, 5, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("exactMatch", 5, 5, 5, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("valueTooLow", -1, 0, 10, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("valueTooLow", -1, 0, 10, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("valueTooLow", -1, 0, 10, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("valueTooLow", -1, 0, 10, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("valueTooHigh", 11, 0, 10, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("valueTooHigh", 11, 0, 10, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("valueTooHigh", 11, 0, 10, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("valueTooHigh", 11, 0, 10, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("minIsNan", 5, math.NaN(), 10, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("minIsNan", 5, math.NaN(), 10, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("minIsNan", 5, math.NaN(), 10, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("minIsNan", 5, math.NaN(), 10, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("maxIsNan", 5, 0, math.NaN(), pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("maxIsNan", 5, 0, math.NaN(), pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("maxIsNan", 5, 0, math.NaN(), pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("maxIsNan", 5, 0, math.NaN(), pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("minMaxAreNan", 5, math.NaN(), math.NaN(), pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("minMaxAreNan", 5, math.NaN(), math.NaN(), pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("minMaxAreNan", 5, math.NaN(), math.NaN(), pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("minMaxAreNan", 5, math.NaN(), math.NaN(), pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("valueIsNan", math.NaN(), 0, 10, pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("valueIsNan", math.NaN(), 0, 10, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("valueIsNan", math.NaN(), 0, 10, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("valueIsNan", math.NaN(), 0, 10, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("valueIsNanInfRange", math.NaN(), math.Inf(-1), math.Inf(1), pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("valueIsNanInfRange", math.NaN(), math.Inf(-1), math.Inf(1), pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("valueIsNanInfRange", math.NaN(), math.Inf(-1), math.Inf(1), pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("valueIsNanInfRange", math.NaN(), math.Inf(-1), math.Inf(1), pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("infinityMax", math.Inf(1), 0, math.Inf(1), pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("infinityMax", math.Inf(1), 0, math.Inf(1), pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("infinityMin", math.Inf(-1), math.Inf(-1), 0, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("infinityMin", math.Inf(-1), math.Inf(-1), 0, pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("allAreNan", math.NaN(), math.NaN(), math.NaN(), pb.DoubleRangeFilter_NONE),
		simpleDoubleRange("allAreNan", math.NaN(), math.NaN(), math.NaN(), pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("allAreNan", math.NaN(), math.NaN(), math.NaN(), pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("allAreNan", math.NaN(), math.NaN(), math.NaN(), pb.DoubleRangeFilter_BOTH),

		simpleDoubleRange("valueIsMax", 1, 0, 1, pb.DoubleRangeFilter_MAX),
		simpleDoubleRange("valueIsMin", 0, 0, 1, pb.DoubleRangeFilter_MIN),
		simpleDoubleRange("excludeBoth", 0, 0, 1, pb.DoubleRangeFilter_BOTH),
		simpleDoubleRange("excludeBoth", 1, 0, 1, pb.DoubleRangeFilter_BOTH),

		{
			"String equals simple negative", // and case sensitivity
			&pb.SearchFields{
				StringArgs: map[string]string{
					"field": "value",
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
			&pb.SearchFields{
				StringArgs: map[string]string{
					"otherfield": "othervalue",
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
			&pb.SearchFields{
				Tags: []string{
					"MYTAG",
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
			&pb.SearchFields{
				Tags: []string{
					"A", "B", "C",
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
			nil,
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * -1)),
			},
		},
		{
			"CreatedAfter simple negative",
			nil,
			&pb.Pool{
				CreatedAfter: timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"Created before time range negative",
			nil,
			&pb.Pool{
				CreatedBefore: timestamp(now.Add(time.Hour * 2)),
				CreatedAfter:  timestamp(now.Add(time.Hour * 1)),
			},
		},
		{
			"Created after time range negative",
			nil,
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

func simpleDoubleRange(name string, value, min, max float64, exclude pb.DoubleRangeFilter_Exclude) TestCase {
	return TestCase{
		"double range " + name,
		&pb.SearchFields{
			DoubleArgs: map[string]float64{
				"field": value,
			},
		},
		&pb.Pool{
			DoubleRangeFilters: []*pb.DoubleRangeFilter{
				{
					DoubleArg: "field",
					Min:       min,
					Max:       max,
					Exclude:   exclude,
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
		&pb.SearchFields{
			DoubleArgs: map[string]float64{
				"a": a,
			},
			StringArgs: map[string]string{
				"b": b,
			},
			Tags: []string{c},
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

func timestamp(t time.Time) *timestamppb.Timestamp {
	tsp := timestamppb.New(t)
	if err := tsp.CheckValid(); err != nil {
		panic(err)
	}

	return tsp
}
