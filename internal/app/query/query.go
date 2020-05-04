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

package query

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	ticketsPerQuery     = stats.Int64("openmatch.dev/query/tickets_per_query", "Number of tickets per query", stats.UnitDimensionless)
	cacheTotalItems     = stats.Int64("openmatch.dev/query/total_cache_items", "Total number of tickets query service cached", stats.UnitDimensionless)
	cacheDeltaItems     = stats.Int64("openmatch.dev/query/delta_cache_items", "Number of tickets added by query service in the last update", stats.UnitDimensionless)
	cacheWaitingQueries = stats.Int64("openmatch.dev/query/waiting_queries", "Number of waiting queries in the last update", stats.UnitDimensionless)
	cacheUpdateLatency  = stats.Float64("openmatch.dev/query/update_latency", "Time elapsed of each query cache update", stats.UnitMilliseconds)

	ticketsPerQueryView = telemetry.MeasureToView(
		ticketsPerQuery,
		"openmatch.dev/query/tickets_per_query",
		"Tickets per query",
		telemetry.DefaultMillisecondsDistribution,
	)
	cacheTotalItemsView = telemetry.MeasureToView(
		cacheTotalItems,
		"openmatch.dev/query/total_cache_items",
		"Total number of tickets query service cached",
		view.LastValue(),
	)
	cacheDeltaItemsView = telemetry.MeasureToView(
		cacheDeltaItems,
		"openmatch.dev/query/delta_cache_items",
		"Number of tickets cache delta items",
		view.LastValue(),
	)
	cacheUpdateView = telemetry.MeasureToView(
		cacheWaitingQueries,
		"openmatch.dev/query/cache_updates",
		"Number of query cache updates in total",
		view.Count(),
	)
	cacheWaitingQueriesView = telemetry.MeasureToView(
		cacheWaitingQueries,
		"openmatch.dev/query/waiting_requests",
		"Number of waiting requests in total",
		telemetry.DefaultMillisecondsDistribution,
	)
	cacheUpdateLatencyView = telemetry.MeasureToView(
		cacheUpdateLatency,
		"openmatch.dev/query/update_latency",
		"Time elapsed of each query cache update",
		telemetry.DefaultMillisecondsDistribution,
	)
)

// BindService creates the query service and binds it to the serving harness.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	service := &queryService{
		cfg: p.Config(),
		tc:  newTicketCache(b, p.Config()),
	}

	b.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterQueryServiceServer(s, service)
	}, pb.RegisterQueryServiceHandlerFromEndpoint)
	if err := view.Register(
		ticketsPerQueryView,
		cacheTotalItemsView,
		cacheDeltaItemsView,
		cacheUpdateView,
		cacheWaitingQueriesView,
		cacheUpdateLatencyView,
	); err != nil {
		logger.WithError(err).Fatalf("failed to register given views to exporters")
	}
	return nil
}
