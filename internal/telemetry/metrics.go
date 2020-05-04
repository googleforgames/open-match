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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// Default histogram distributions
var (
	DefaultBytesDistribution        = view.Distribution(64, 128, 256, 512, 1024, 2048, 4096, 16384, 65536, 262144, 1048576)
	DefaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000)
	DefaultCountDistribution        = view.Distribution(1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536)
)

// MeasureToView convert an Opencensus measure to Prometheus metric
func MeasureToView(s stats.Measure, metricName, metricDesc string, aggregation *view.Aggregation) *view.View {
	v := &view.View{
		Name:        metricName,
		Description: metricDesc,
		Measure:     s,
		Aggregation: aggregation,
	}
	return v
}
