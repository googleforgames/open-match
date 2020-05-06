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
	ticketsPerQuery     = stats.Int64("open-match.dev/query/tickets_per_query", "Number of tickets per query", stats.UnitDimensionless)
	cacheTotalItems     = stats.Int64("open-match.dev/query/total_cache_items", "Total number of tickets query service cached", stats.UnitDimensionless)
	cacheFetchedItems   = stats.Int64("open-match.dev/query/fetched_items", "Number of fetched items in total", stats.UnitDimensionless)
	cacheWaitingQueries = stats.Int64("open-match.dev/query/waiting_queries", "Number of waiting queries in the last update", stats.UnitDimensionless)
	cacheUpdateLatency  = stats.Float64("open-match.dev/query/update_latency", "Time elapsed of each query cache update", stats.UnitMilliseconds)

	ticketsPerQueryView = &view.View{
		Measure:     ticketsPerQuery,
		Name:        "open-match.dev/query/tickets_per_query",
		Description: "Tickets per query",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
	cacheTotalItemsView = &view.View{
		Measure:     cacheTotalItems,
		Name:        "open-match.dev/query/total_cache_items",
		Description: "Total number of tickets query service cached",
		Aggregation: view.LastValue(),
	}
	cacheFetchedItemsView = &view.View{
		Measure:     cacheFetchedItems,
		Name:        "open-match.dev/query/total_fetched_items",
		Description: "Total number of fetched tickets",
		Aggregation: view.Count(),
	}
	cacheUpdateView = &view.View{
		Measure:     cacheWaitingQueries,
		Name:        "open-match.dev/query/cache_updates",
		Description: "Number of query cache updates in total",
		Aggregation: view.Count(),
	}
	cacheWaitingQueriesView = &view.View{
		Measure:     cacheWaitingQueries,
		Name:        "open-match.dev/query/waiting_requests",
		Description: "Number of waiting requests in total",
		Aggregation: telemetry.DefaultCountDistribution,
	}
	cacheUpdateLatencyView = &view.View{
		Measure:     cacheUpdateLatency,
		Name:        "open-match.dev/query/update_latency",
		Description: "Time elapsed of each query cache update",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
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
	b.RegisterViews(
		ticketsPerQueryView,
		cacheTotalItemsView,
		cacheUpdateView,
		cacheWaitingQueriesView,
		cacheUpdateLatencyView,
	)
	return nil
}
