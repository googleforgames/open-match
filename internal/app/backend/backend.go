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
	totalBytesPerMatch      = stats.Int64("open-match.dev/backend/total_bytes_per_match", "Total bytes per match", stats.UnitBytes)
	ticketsPerMatch         = stats.Int64("open-match.dev/backend/tickets_per_match", "Number of tickets per match", stats.UnitDimensionless)
	ticketsReleased         = stats.Int64("open-match.dev/backend/tickets_released", "Number of tickets released per request", stats.UnitDimensionless)
	ticketsAssigned         = stats.Int64("open-match.dev/backend/tickets_assigned", "Number of tickets assigned per request", stats.UnitDimensionless)
	ticketsTimeToAssignment = stats.Int64("open-match.dev/backend/ticket_time_to_assignment", "Time to assignment for tickets", stats.UnitMilliseconds)

	totalMatchesView = &view.View{
		Measure:     totalBytesPerMatch,
		Name:        "open-match.dev/backend/total_matches",
		Description: "Total number of matches",
		Aggregation: view.Count(),
	}
	totalBytesPerMatchView = &view.View{
		Measure:     totalBytesPerMatch,
		Name:        "open-match.dev/backend/total_bytes_per_match",
		Description: "Total bytes per match",
		Aggregation: telemetry.DefaultBytesDistribution,
	}
	ticketsPerMatchView = &view.View{
		Measure:     ticketsPerMatch,
		Name:        "open-match.dev/backend/tickets_per_match",
		Description: "Tickets per ticket",
		Aggregation: telemetry.DefaultCountDistribution,
	}
	ticketsAssignedView = &view.View{
		Measure:     ticketsAssigned,
		Name:        "open-match.dev/backend/tickets_assigned",
		Description: "Number of tickets assigned per request",
		Aggregation: view.Sum(),
	}
	ticketsReleasedView = &view.View{
		Measure:     ticketsReleased,
		Name:        "open-match.dev/backend/tickets_released",
		Description: "Number of tickets released per request",
		Aggregation: view.Sum(),
	}

	ticketsTimeToAssignmentView = &view.View{
		Measure:     ticketsTimeToAssignment,
		Name:        "open-match.dev/backend/ticket_time_to_assignment",
		Description: "Time to assignment for tickets",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
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
	b.RegisterViews(
		totalMatchesView,
		totalBytesPerMatchView,
		ticketsPerMatchView,
		ticketsAssignedView,
		ticketsReleasedView,
		ticketsTimeToAssignmentView,
	)
	return nil
}
