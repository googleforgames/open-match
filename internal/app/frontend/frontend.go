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
	totalBytesPerTicket   = stats.Int64("openmatch.dev/total_bytes_per_ticket", "Total bytes per ticket", stats.UnitBytes)
	searchFieldsPerTicket = stats.Int64("openmatch.dev/searchfields_per_ticket", "Searchfields per ticket", stats.UnitDimensionless)

	totalBytesPerTicketView = telemetry.MeasureToView(
		totalBytesPerTicket,
		"openmatch.dev/total_bytes_per_ticket",
		"Total bytes per ticket",
		telemetry.DefaultBytesDistribution,
	)
	searchFieldsPerTicketView = telemetry.MeasureToView(
		searchFieldsPerTicket,
		"openmatch.dev/searchfields_per_ticket",
		"SearchFields per ticket",
		telemetry.DefaultCountDistribution,
	)
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
	if err := view.Register(
		totalBytesPerTicketView,
		searchFieldsPerTicketView,
	); err != nil {
		logger.WithError(err).Fatalf("failed to register given views to exporters")
	}
	return nil
}
