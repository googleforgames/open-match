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
	"testing"

	structpb "github.com/golang/protobuf/ptypes/struct"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/pkg/gopb"
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
				"ri$foo": 1,
			},
		},
		{
			description: "range no index",
			indices:     []string{},
			ticket: structs.Struct{
				"foo": structs.Number(1),
			}.S(),
			expectedValues: map[string]float64{},
		},
		{
			description:    "range no value",
			indices:        []string{"foo"},
			ticket:         structs.Struct{}.S(),
			expectedValues: map[string]float64{},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("ticketIndices", test.indices)

			actual := extractIndexedFields(cfg, &gopb.Ticket{Properties: test.ticket})

			assert.Equal(t, test.expectedValues, actual.values)
		})
	}
}

func TestExtractIndexFilters(t *testing.T) {
	tests := []struct {
		description string
		pool        *gopb.Pool
		expected    []indexFilter
	}{
		{
			description: "empty",
			pool:        &gopb.Pool{},
			expected:    []indexFilter{},
		},
		{
			description: "range",
			pool: &gopb.Pool{
				Filters: []*gopb.Filter{
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
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			actual := extractIndexFilters(test.pool)

			assert.Equal(t, test.expected, actual)
		})
	}
}

func TestExtractDeindexFilters(t *testing.T) {
	tests := []struct {
		description string
		indices     []string
		expected    []string
	}{
		{
			description: "range",
			indices:     []string{"foo"},
			expected:    []string{"ri$foo"},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			cfg := viper.New()
			cfg.Set("ticketIndices", test.indices)

			actual := extractDeindexFilters(cfg)

			assert.Equal(t, test.expected, actual)
		})
	}
}
