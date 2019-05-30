// Copyright 2018 Google LLC
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

package set

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type stringOperation func([]string, []string) []string

func TestStringOperations(t *testing.T) {
	assert := assert.New(t)

	a1 := []string{"a", "b", "c", "d", "e"}
	a2 := []string{"b", "c", "f", "g"}
	i := []string{"b", "c"}
	u := []string{"a", "b", "c", "d", "e", "f", "g"}
	d := []string{"a", "d", "e"}

	var setTests = []struct {
		in1      []string
		in2      []string
		expected []string
		op       stringOperation
	}{
		{a1, a1, []string{}, Difference},
		{a1, a2, d, Difference},
		{a1, nil, a1, Difference},
		{nil, a2, []string{}, Difference},
		{a1, a2, u, Union},
		{nil, a2, a2, Union},
		{a1, nil, a1, Union},
		{a1, a2, i, Intersection},
		{a1, nil, nil, Intersection},
		{nil, a2, nil, Intersection},
	}

	for i, tt := range setTests {
		t.Run(fmt.Sprintf("%#v-%d", tt.op, i), func(t *testing.T) {
			actual := tt.op(tt.in1, tt.in2)
			sort.Strings(tt.expected)
			sort.Strings(actual)
			assert.EqualValues(tt.expected, actual)
		})
	}
}
