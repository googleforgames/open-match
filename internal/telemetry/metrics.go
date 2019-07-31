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
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Gauge creates a gauge metric.
func Gauge(name string, description string, tags ...tag.Key) *stats.Int64Measure {
	s := stats.Int64(name, description, "1")
	gaugeView(s, tags...)
	return s
}

// SetGauge sets the value of the metric
func SetGauge(ctx context.Context, s *stats.Int64Measure, n int, tags ...tag.Mutator) {
	mCtx, err := tag.New(ctx, tags...)
	if err != nil {
		return
	}
	stats.Record(mCtx, s.M(int64(n)))
}

// Counter creates a counter metric.
func Counter(name string, description string, tags ...tag.Key) *stats.Int64Measure {
	s := stats.Int64(name, "Count of "+description+".", "1")
	counterView(s, tags...)
	return s
}

// IncrementCounter +1's the counter metric.
func IncrementCounter(ctx context.Context, s *stats.Int64Measure, tags ...tag.Mutator) {
	IncrementCounterN(ctx, s, 1, tags...)
}

// IncrementCounterN increases a metric by n.
func IncrementCounterN(ctx context.Context, s *stats.Int64Measure, n int, tags ...tag.Mutator) {
	mCtx, err := tag.New(ctx, tags...)
	if err != nil {
		return
	}
	stats.Record(mCtx, s.M(int64(n)))
}

// gaugeView converts the measurement into a view for a gauge metric.
func gaugeView(s *stats.Int64Measure, tags ...tag.Key) *view.View {
	v := &view.View{
		Name:        s.Name(),
		Measure:     s,
		Description: s.Description(),
		Aggregation: view.LastValue(),
		TagKeys:     tags,
	}
	err := view.Register(v)
	if err != nil {
		logger.WithError(err).Infof("cannot register view for metric: %s, it will not be reported", s.Name())
	}
	return v
}

// counterView converts the measurement into a view for a counter.
func counterView(s *stats.Int64Measure, tags ...tag.Key) *view.View {
	v := &view.View{
		Name:        s.Name(),
		Measure:     s,
		Description: s.Description(),
		Aggregation: view.Count(),
		TagKeys:     tags,
	}
	err := view.Register(v)
	if err != nil {
		logger.WithError(err).Infof("cannot register view for metric: %s, it will not be reported", s.Name())
	}
	return v
}
