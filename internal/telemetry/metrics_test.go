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

package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opencensus.io/stats"
	utilTesting "open-match.dev/open-match/internal/util/testing"
)

func TestIncrementCounter(t *testing.T) {
	ctx := utilTesting.NewContext(t)
	c := Counter("telemetry/fake_metric", "fake")
	IncrementCounter(ctx, c)
	IncrementCounter(ctx, c)
}

func TestDoubleMetric(t *testing.T) {
	assert := assert.New(t)
	c := Counter("telemetry/fake_metric", "fake")
	c2 := Counter("telemetry/fake_metric", "fake")
	assert.Equal(c, c2)
}

func TestDoubleRegisterView(t *testing.T) {
	assert := assert.New(t)
	mFakeCounter := stats.Int64("telemetry/fake_metric", "Fake", "1")
	v := counterView(mFakeCounter)
	v2 := counterView(mFakeCounter)
	assert.Equal(v, v2)
}
