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

package statestore

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/pkg/pb"
)

func TestExtractIndexedFields(t *testing.T) {
	tests := []struct {
		description    string
		searchFields   *pb.SearchFields
		expectedValues map[string]float64
	}{
		{
			description: "range",
			searchFields: &pb.SearchFields{
				DoubleArgs: map[string]float64{"foo": 1},
			},
			expectedValues: map[string]float64{
				"allTickets": 0,
				"ri$foo":     1,
			},
		},
		{
			description: "tag",
			searchFields: &pb.SearchFields{
				Tags: []string{"foo"},
			},
			expectedValues: map[string]float64{
				"allTickets": 0,
				"ti$foo":     0,
			},
		},
		{
			description: "string",
			searchFields: &pb.SearchFields{
				StringArgs: map[string]string{
					"foo": "bar",
				},
			},
			expectedValues: map[string]float64{
				"allTickets":  0,
				"si$foo$vbar": 0,
			},
		},
		{
			description: "no value",
			expectedValues: map[string]float64{
				"allTickets": 0,
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			actual := extractIndexedFields(&pb.Ticket{SearchFields: test.searchFields})

			assert.Equal(t, test.expectedValues, actual)
		})
	}
}

func TestExtractIndexFilters(t *testing.T) {
	tests := []struct {
		description string
		pool        *pb.Pool
		expected    []*indexFilter
	}{
		{
			description: "empty",
			pool:        &pb.Pool{},
			expected: []*indexFilter{
				{
					name: "allTickets",
					min:  math.Inf(-1),
					max:  math.Inf(1),
				},
			},
		},
		{
			description: "range",
			pool: &pb.Pool{
				DoubleRangeFilters: []*pb.DoubleRangeFilter{
					{
						DoubleArg: "foo",
						Min:       -1,
						Max:       1,
					},
				},
			},
			expected: []*indexFilter{
				{
					name: "ri$foo",
					min:  -1,
					max:  1,
				},
			},
		},
		{
			description: "tag",
			pool: &pb.Pool{
				TagPresentFilters: []*pb.TagPresentFilter{
					{
						Tag: "foo",
					},
				},
			},
			expected: []*indexFilter{
				{
					name: "ti$foo",
					min:  0,
					max:  0,
				},
			},
		},
		{
			description: "string equals",
			pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						StringArg: "foo",
						Value:     "bar",
					},
				},
			},
			expected: []*indexFilter{
				{
					name: "si$foo$vbar",
					min:  0,
					max:  0,
				},
			},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.description, func(t *testing.T) {
			actual := extractIndexFilters(test.pool)

			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestNameCollision(t *testing.T) {
	names := []string{
		rangeIndexName("foo"),
		tagIndexName("foo"),
		stringIndexName("foo", "bar"),
		indexCacheName("foo"),
		stringIndexName("$v", ""),
		stringIndexName("", "$v"),
	}
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			assert.NotEqual(t, names[i], names[j])
		}
	}
}
