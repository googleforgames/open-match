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

package expbo

import (
	"math"
	"testing"
	"time"

	"github.com/cenkalti/backoff"
)

func TestUnmarshalExponentialBackOff(t *testing.T) {
	s := "[0.25 30] *1.5 ~0.33 <300"

	b := backoff.NewExponentialBackOff()
	err := UnmarshalExponentialBackOff(s, b)
	if err != nil {
		t.Fatalf(`error umarshaling "%s": %+v`, s, err)
	}

	if b.InitialInterval != 250*time.Millisecond {
		t.Error("unexpected InitialInterval value:", b.InitialInterval)
	}
	if b.MaxInterval != 30*time.Second {
		t.Error("unexpected MaxInterval value:", b.MaxInterval)
	}
	if math.Abs(b.Multiplier-1.5) > 1e-8 {
		t.Error("unexpected Multiplier value:", b.Multiplier)
	}
	if math.Abs(b.RandomizationFactor-0.33) > 1e-8 {
		t.Error("unexpected RandomizationFactor value:", b.RandomizationFactor)
	}
	if b.MaxElapsedTime != 5*time.Minute {
		t.Error("unexpected MaxElapsedTime value:", b.MaxElapsedTime)
	}
}
