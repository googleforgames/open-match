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

package frontend

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	totalBytesPerTicket     = stats.Int64("open-match.dev/frontend/total_bytes_per_ticket", "Total bytes per ticket", stats.UnitBytes)
	searchFieldsPerTicket   = stats.Int64("open-match.dev/frontend/searchfields_per_ticket", "Searchfields per ticket", stats.UnitDimensionless)
	totalBytesPerBackfill   = stats.Int64("open-match.dev/frontend/total_bytes_per_backfill", "Total bytes per backfill", stats.UnitBytes)
	searchFieldsPerBackfill = stats.Int64("open-match.dev/frontend/searchfields_per_backfill", "Searchfields per backfill", stats.UnitDimensionless)

	totalBytesPerTicketView = &view.View{
		Measure:     totalBytesPerTicket,
		Name:        "open-match.dev/frontend/total_bytes_per_ticket",
		Description: "Total bytes per ticket",
		Aggregation: telemetry.DefaultBytesDistribution,
	}
	searchFieldsPerTicketView = &view.View{
		Measure:     searchFieldsPerTicket,
		Name:        "open-match.dev/frontend/searchfields_per_ticket",
		Description: "SearchFields per ticket",
		Aggregation: telemetry.DefaultCountDistribution,
	}
	totalBytesPerBackfillView = &view.View{
		Measure:     totalBytesPerBackfill,
		Name:        "open-match.dev/frontend/total_bytes_per_backfill",
		Description: "Total bytes per backfill",
		Aggregation: telemetry.DefaultBytesDistribution,
	}
	searchFieldsPerBackfillView = &view.View{
		Measure:     searchFieldsPerBackfill,
		Name:        "open-match.dev/frontend/searchfields_per_backfill",
		Description: "SearchFields per backfill",
		Aggregation: telemetry.DefaultCountDistribution,
	}
)

// BindService creates the frontend service and binds it to the serving harness.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	service := &frontendService{
		cfg:   p.Config(),
		store: statestore.New(p.Config()),
	}

	b.AddHealthCheckFunc(service.store.HealthCheck)
	b.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, service)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	b.RegisterViews(
		totalBytesPerTicketView,
		searchFieldsPerTicketView,
		totalBytesPerBackfillView,
		searchFieldsPerBackfillView,
	)
	return nil
}
