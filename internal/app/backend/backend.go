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

package backend

import (
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	totalBytesPerMatch = stats.Int64("openmatch.dev/total_bytes_per_match", "Total bytes per match", stats.UnitBytes)
	ticketsPerMatch    = stats.Int64("openmatch.dev/tickets_per_match", "Number of tickets per match", stats.UnitDimensionless)
	ticketsReleased    = stats.Int64("openmatch.dev/tickets_released", "Number of tickets released per request", stats.UnitDimensionless)
	ticketsAssigned    = stats.Int64("openmatch.dev/tickets_assigned", "Number of tickets assigned per request", stats.UnitDimensionless)

	totalMatchesView = telemetry.MeasureToView(
		totalBytesPerMatch,
		"openmatch.dev/total_matches",
		"Total number of matches",
		view.Count(),
	)
	totalBytesPerMatchView = telemetry.MeasureToView(
		totalBytesPerMatch,
		"openmatch.dev/total_bytes_per_match",
		"Total bytes per match",
		telemetry.DefaultBytesDistribution,
	)
	ticketsPerMatchView = telemetry.MeasureToView(
		ticketsPerMatch,
		"openmatch.dev/tickets_per_match",
		"Tickets per ticket",
		telemetry.DefaultCountDistribution,
	)
	ticketsAssignedView = telemetry.MeasureToView(
		ticketsAssigned,
		"openmatch.dev/tickets_assigned",
		"Number of tickets assigned per request",
		view.Sum(),
	)
	ticketsReleasedView = telemetry.MeasureToView(
		ticketsReleased,
		"openmatch.dev/tickets_released",
		"Number of tickets released per request",
		view.Sum(),
	)
)

// BindService creates the backend service and binds it to the serving harness.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	service := &backendService{
		synchronizer: newSynchronizerClient(p.Config()),
		store:        statestore.New(p.Config()),
		cc:           rpc.NewClientCache(p.Config()),
	}

	b.AddHealthCheckFunc(service.store.HealthCheck)
	b.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterBackendServiceServer(s, service)
	}, pb.RegisterBackendServiceHandlerFromEndpoint)
	if err := view.Register(
		totalMatchesView,
		totalBytesPerMatchView,
		ticketsPerMatchView,
		ticketsAssignedView,
		ticketsReleasedView,
	); err != nil {
		logger.WithError(err).Fatalf("failed to register given views to exporters")
	}
	return nil
}
