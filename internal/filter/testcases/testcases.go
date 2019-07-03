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
	"math"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

// TestCase defines a single filtering test case to run.
type TestCase struct {
	Name   string
	Ticket *pb.Ticket
	Filter *pb.Filter
}

// IncludedTestCases returns a list of test cases where using the given filter,
// the ticket is included in the result.
func IncludedTestCases() []TestCase {
	return []TestCase{
		simpleRangeTest("simpleInRange", 5, 0, 10),
		simpleRangeTest("exactMatch", 5, 5, 5),
		simpleRangeTest("infinityMax", math.Inf(1), 0, math.Inf(1)),
		simpleRangeTest("infinityMin", math.Inf(-1), math.Inf(-1), 0),
	}
}

// ExcludedTestCases returns a list of test cases where using the given filter,
// the ticket is NOT included in the result.
func ExcludedTestCases() []TestCase {
	return []TestCase{
		{
			"missingField",
			&pb.Ticket{
				// Using struct literals without helper for degenerate cases only.
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
			"missingFieldsMap",
			&pb.Ticket{
				// Using struct literals without helper for degenerate cases only.
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
			"missingProperties",
			&pb.Ticket{
				// Using struct literals without helper for degenerate cases only.
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
}

func simpleRangeTest(name string, value, min, max float64) TestCase {
	return TestCase{
		name,
		&pb.Ticket{
			Properties: structs.Struct{
				"field": structs.Number(value),
			}.S(),
		},
		&pb.Filter{
			Attribute: "field",
			Min:       min,
			Max:       max,
		},
	}
}
