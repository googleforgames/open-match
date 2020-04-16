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

// Ticket metrics
var (
	TotalBytesPerTicket   = stats.Int64("openmatch.dev/total_bytes_per_ticket", "Total bytes per ticket", stats.UnitBytes)
	SearchFieldsPerTicket = stats.Int64("openmatch.dev/searchfields_per_ticket", "Searchfields per ticket", stats.UnitDimensionless)
	TicketsAssigned       = stats.Int64("openmatch.dev/tickets_assigned", "Number of tickets assigned per request", stats.UnitDimensionless)
	TicketsReleased       = stats.Int64("openmatch.dev/tickets_released", "Number of tickets released per request", stats.UnitDimensionless)
)

// Match metrics
var (
	TotalBytesPerMatch = stats.Int64("openmatch.dev/total_bytes_per_match", "Total bytes per match", stats.UnitBytes)
	TicketsPerMatch    = stats.Int64("openmatch.dev/tickets_per_match", "Number of tickets per match", stats.UnitDimensionless)
)

// Query metrics
var (
	TicketsPerQuery          = stats.Int64("openmatch.dev/query/tickets_per_query", "Number of tickets per query", stats.UnitDimensionless)
	QueryCacheTotalItems     = stats.Int64("openmatch.dev/query/total_cache_items", "Total number of tickets query service cached", stats.UnitDimensionless)
	QueryCacheDeltaItems     = stats.Int64("openmatch.dev/query/delta_cache_items", "Number of tickets added by query service in the last update", stats.UnitDimensionless)
	QueryCacheWaitingQueries = stats.Int64("openmatch.dev/query/waiting_queries", "Number of waiting queries in the last update", stats.UnitDimensionless)
	QueryCacheUpdateLatency  = stats.Float64("openmatch.dev/query/update_latency", "Time elapsed of each query cache update", stats.UnitMilliseconds)
)

// Evaluator metrics
var (
	MatchesPerEvaluateRequest  = stats.Int64("openmatch.dev/evaluator/matches_per_request", "Number of matches sent to the evaluator per request", stats.UnitDimensionless)
	MatchesPerEvaluateResponse = stats.Int64("openmatch.dev/evaluator/matches_per_response", "Number of matches returned by the evaluator per response", stats.UnitDimensionless)
	CollidedMatchesPerEvaluate = stats.Int64("openmatch.dev/evaluator/collided_matches_per_call", "Number of collided matches per default evaluator call", stats.UnitDimensionless)
)

// Synchronizer metrics
var (
	SynchronizerIterationLatency        = stats.Float64("openmatch.dev/synchronizer/iteration_latency", "Time elapsed of each synchronizer iteration", stats.UnitMilliseconds)
	SynchronizerRegistrationWaitTime    = stats.Float64("openmatch.dev/synchronizer/registration_wait_time", "Time elapsed of registration wait time", stats.UnitMilliseconds)
	SynchronizerRegistrationMMFDoneTime = stats.Float64("openmatch.dev/synchronizer/registration_mmf_done_time", "Time elapsed wasted in registration window with done MMFs", stats.UnitMilliseconds)
)

// Health metrics
var (
	ProbeReadiness = stats.Int64("openmatch.dev/health/readiness", "Count of readiness probes", stats.UnitDimensionless)
)

func measureToDefaultView(s stats.Measure, metricName, metricDesc string, aggregation *view.Aggregation) *view.View {
	v := &view.View{
		Name:        metricName,
		Description: metricDesc,
		Measure:     s,
		Aggregation: aggregation,
	}
	return v
}
