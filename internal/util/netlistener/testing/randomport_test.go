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

package testing

import (
	"fmt"
	"testing"
)

const (
	numIterations = 1000
)

func TestMustListen(t *testing.T) {
	for i := 0; i < numIterations; i++ {
		testName := fmt.Sprintf("[%d] MustListen", i)
		t.Run(testName, func(t *testing.T) {
			t.Parallel()

			lh := MustListen()
			defer lh.Close()
			if lh.Number() <= 0 {
				t.Errorf("Expected %d > 0, port is out of range.", lh.Number())
			}
		})
	}
}
