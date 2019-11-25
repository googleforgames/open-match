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

var (
	// HistogramBounds defines a unified bucket boundaries for all histogram typed time metrics in Open Match
	HistogramBounds = []float64{0, 50, 100, 200, 400, 800, 1600, 3200, 6400, 12800, 25600, 51200}
)

// Gauge creates a gauge metric to be recorded with dimensionless unit.
func Gauge(name string, description string, tags ...tag.Key) *stats.Int64Measure {
	s := stats.Int64(name, description, "1")
	gaugeView(s, tags...)
	return s
}

// SetGauge sets the value of the metric
func SetGauge(ctx context.Context, s *stats.Int64Measure, n int64, tags ...tag.Mutator) {
	mCtx, err := tag.New(ctx, tags...)
	if err != nil {
		return
	}
	stats.Record(mCtx, s.M(n))
}

// Counter creates a counter metric to be recorded with dimensionless unit.
func Counter(name string, description string, tags ...tag.Key) *stats.Int64Measure {
	s := stats.Int64(name, "Count of "+description+".", stats.UnitDimensionless)
	counterView(s, tags...)
	return s
}

// HistogramWithBounds creates a prometheus histogram metric to be recorded with specified bounds and metric type.
func HistogramWithBounds(name string, description string, unit string, bounds []float64, tags ...tag.Key) *stats.Int64Measure {
	s := stats.Int64(name, description, unit)
	histogramView(s, bounds, tags...)
	return s
}

// RecordUnitMeasurement records a data point using the input metric by one unit with given tags.
func RecordUnitMeasurement(ctx context.Context, s *stats.Int64Measure, tags ...tag.Mutator) {
	RecordNUnitMeasurement(ctx, s, 1, tags...)
}

// RecordNUnitMeasurement records a data point using the input metric by N units with given tags.
func RecordNUnitMeasurement(ctx context.Context, s *stats.Int64Measure, n int64, tags ...tag.Mutator) {
	if err := stats.RecordWithTags(ctx, tags, s.M(n)); err != nil {
		logger.WithError(err).Infof("cannot record stat with tags %#v", tags)
		return
	}
}

// histogramView converts the measurement into a view for a histogram metric.
func histogramView(s *stats.Int64Measure, bounds []float64, tags ...tag.Key) *view.View {
	v := &view.View{
		Name:        s.Name(),
		Measure:     s,
		Description: s.Description(),
		Aggregation: view.Distribution(bounds...),
		TagKeys:     tags,
	}
	err := view.Register(v)
	if err != nil {
		logger.WithError(err).Infof("cannot register view for metric: %s, it will not be reported", s.Name())
	}
	return v
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
