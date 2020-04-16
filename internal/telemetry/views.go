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

// Ticket views
var (
	TotalBytesPerTicketView = measureToDefaultView(
		TotalBytesPerTicket,
		"openmatch.dev/total_bytes_per_ticket",
		"Total bytes per ticket",
		DefaultBytesDistribution,
	)

	SearchFieldsPerTicketView = measureToDefaultView(
		SearchFieldsPerTicket,
		"openmatch.dev/searchfields_per_ticket",
		"SearchFields per ticket",
		DefaultCountDistribution,
	)

	TicketsAssignedView = measureToDefaultView(
		TicketsAssigned,
		"openmatch.dev/tickets_assigned",
		"Number of tickets assigned per request",
		view.Sum(),
	)

	TicketsReleasedView = measureToDefaultView(
		TicketsReleased,
		"openmatch.dev/tickets_released",
		"Number of tickets released per request",
		view.Sum(),
	)
)

// Match views
var (
	TotalMatches = measureToDefaultView(
		TotalBytesPerMatch,
		"openmatch.dev/total_matches",
		"Total number of matches",
		view.Count(),
	)

	TotalBytesPerMatchView = measureToDefaultView(
		TotalBytesPerMatch,
		"openmatch.dev/total_bytes_per_match",
		"Total bytes per match",
		DefaultBytesDistribution,
	)

	TicketsPerMatchView = measureToDefaultView(
		TicketsPerMatch,
		"openmatch.dev/tickets_per_match",
		"Tickets per ticket",
		DefaultCountDistribution,
	)
)

// Query views
var (
	TicketsPerQueryView = measureToDefaultView(
		TicketsPerQuery,
		"openmatch.dev/query/tickets_per_query",
		"Tickets per query",
		DefaultCountDistribution,
	)

	QueryCacheTotalItemsView = measureToDefaultView(
		QueryCacheTotalItems,
		"openmatch.dev/query/total_cache_items",
		"Total number of tickets query service cached",
		view.LastValue(),
	)

	QueryCacheDeltaItemsView = measureToDefaultView(
		QueryCacheDeltaItems,
		"openmatch.dev/query/delta_cache_items",
		"Number of tickets cache delta items",
		view.LastValue(),
	)

	QueryCacheUpdateView = measureToDefaultView(
		QueryCacheWaitingQueries,
		"openmatch.dev/query/cache_updates",
		"Number of query cache updates in total",
		view.Count(),
	)

	QueryCacheWaitingQueriesView = measureToDefaultView(
		QueryCacheWaitingQueries,
		"openmatch.dev/query/waiting_requests",
		"Number of waiting requests in total",
		DefaultCountDistribution,
	)

	QueryCacheUpdateLatencyView = measureToDefaultView(
		QueryCacheUpdateLatency,
		"openmatch.dev/query/update_latency",
		"Time elapsed of each query cache update",
		DefaultMillisecondsDistribution,
	)
)

// Evaluator views
var (
	MatchesPerEvaluateRequestView = measureToDefaultView(
		MatchesPerEvaluateRequest,
		"openmatch.dev/evaluator/matches_per_request",
		"Number of matches sent to the evaluator per request",
		DefaultCountDistribution,
	)

	MatchesPerEvaluateResponseView = measureToDefaultView(
		MatchesPerEvaluateResponse,
		"openmatch.dev/evaluator/matches_per_response",
		"Number of matches sent to the evaluator per response",
		DefaultCountDistribution,
	)

	CollidedMatchesPerEvaluateView = measureToDefaultView(
		CollidedMatchesPerEvaluate,
		"openmatch.dev/evaluator/collided_matches_per_call",
		"Number of collided matches per default evaluator call",
		view.Sum(),
	)
)

// Synchronizer views
var (
	SynchronizerIterationLatencyView = measureToDefaultView(
		SynchronizerIterationLatency,
		"openmatch.dev/synchronizer/iteration_latency",
		"Time elapsed of each synchronizer iteration",
		DefaultMillisecondsDistribution,
	)

	SynchronizerRegistrationWaitTimeView = measureToDefaultView(
		SynchronizerRegistrationWaitTime,
		"openmatch.dev/synchronizer/registration_wait_time",
		"Time elapsed of registration wait time",
		DefaultMillisecondsDistribution,
	)

	SynchronizerRegistrationMMFDoneTimeView = measureToDefaultView(
		SynchronizerRegistrationMMFDoneTime,
		"openmatch.dev/synchronizer/registration_mmf_done_time",
		"Time elapsed wasted in registration window with done MMFs",
		DefaultMillisecondsDistribution,
	)
)

// Health views
var (
	ProbeReadinessView = measureToDefaultView(
		ProbeReadiness,
		"openmatch.dev/health/readiness",
		"Count of readiness probes",
		view.Count(),
	)
)

// DefaultViews are provided by this package by default is prometheus agent is enabled.
var DefaultViews = []*view.View{
	TotalBytesPerTicketView,
	SearchFieldsPerTicketView,
	TicketsAssignedView,
	TicketsReleasedView,
	TotalMatches,
	TotalBytesPerMatchView,
	TicketsPerMatchView,
	TicketsPerQueryView,
	QueryCacheTotalItemsView,
	QueryCacheDeltaItemsView,
	QueryCacheUpdateView,
	QueryCacheWaitingQueriesView,
	QueryCacheUpdateLatencyView,
	MatchesPerEvaluateRequestView,
	MatchesPerEvaluateResponseView,
	CollidedMatchesPerEvaluateView,
	SynchronizerIterationLatencyView,
	SynchronizerRegistrationWaitTimeView,
	SynchronizerRegistrationMMFDoneTimeView,
	ProbeReadinessView,
}

/////////////
// TODO: Deprecate usage of the original code
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
