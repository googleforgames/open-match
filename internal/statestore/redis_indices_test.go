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

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/pkg/structs"
)

func TestExtractIndexedFields(t *testing.T) {
	tests := []struct {
		description    string
		indices        []string
		ticket         *structpb.Struct
		expectedValues map[string]float64
	}{
		{
			description: "range",
			indices:     []string{"foo"},
			ticket: structs.Struct{
				"foo": structs.Number(1),
			}.S(),
			expectedValues: map[string]float64{
				"allTickets": 0,
				"ri$foo":     1,
			},
		},
		{
			description: "bools",
			indices:     []string{"t", "f"},
			ticket: structs.Struct{
				"t": structs.Bool(true),
				"f": structs.Bool(false),
			}.S(),
			expectedValues: map[string]float64{
				"allTickets": 0,
				"bi$t":       1,
				"bi$f":       0,
			},
		},
		{
			description: "string",
			indices:     []string{"foo"},
			ticket: structs.Struct{
				"foo": structs.String("bar"),
			}.S(),
			expectedValues: map[string]float64{
				"allTickets":  0,
				"si$foo$vbar": 0,
			},
		},
		{
			description: "no index",
			indices:     []string{},
			ticket: structs.Struct{
				"foo": structs.Number(1),
			}.S(),
			expectedValues: map[string]float64{
				"allTickets": 0,
			},
		},
		{
			description: "no value",
			indices:     []string{"foo"},
			ticket:      structs.Struct{}.S(),
			expectedValues: map[string]float64{
				"allTickets": 0,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("ticketIndices", test.indices)

			actual := extractIndexedFields(cfg, &pb.Ticket{Properties: test.ticket})

			assert.Equal(t, test.expectedValues, actual)
		})
	}
}

func TestExtractIndexFilters(t *testing.T) {
	tests := []struct {
		description string
		pool        *pb.Pool
		expected    []indexFilter
	}{
		{
			description: "empty",
			pool:        &pb.Pool{},
			expected: []indexFilter{
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
				FloatRangeFilters: []*pb.FloatRangeFilter{
					{
						Attribute: "foo",
						Min:       -1,
						Max:       1,
					},
				},
			},
			expected: []indexFilter{
				{
					name: "ri$foo",
					min:  -1,
					max:  1,
				},
			},
		},
		{
			description: "bool false",
			pool: &pb.Pool{
				BoolEqualsFilters: []*pb.BoolEqualsFilter{
					{
						Attribute: "foo",
						Value:     false,
					},
				},
			},
			expected: []indexFilter{
				{
					name: "bi$foo",
					min:  -0.5,
					max:  0.5,
				},
			},
		},
		{
			description: "bool true",
			pool: &pb.Pool{
				BoolEqualsFilters: []*pb.BoolEqualsFilter{
					{
						Attribute: "foo",
						Value:     true,
					},
				},
			},
			expected: []indexFilter{
				{
					name: "bi$foo",
					min:  0.5,
					max:  1.5,
				},
			},
		},
		{
			description: "string equals",
			pool: &pb.Pool{
				StringEqualsFilters: []*pb.StringEqualsFilter{
					{
						Attribute: "foo",
						Value:     "bar",
					},
				},
			},
			expected: []indexFilter{
				{
					name: "si$foo$vbar",
					min:  0,
					max:  0,
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := extractIndexFilters(test.pool)

			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestNameCollision(t *testing.T) {
	names := []string{
		rangeIndexName("foo"),
		boolIndexName("foo"),
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
