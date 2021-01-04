// Copyright 2020 Google LLC
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

package metrics

import (
	"context"
	"sync"

	"go.opencensus.io/stats"
	"open-match.dev/open-match/internal/telemetry"
)

var (
	Iterations          = telemetry.Counter("scale_backend_iterations", "fetch match iterations")
	FetchMatchCalls     = telemetry.Counter("scale_backend_fetch_match_calls", "fetch match calls")
	FetchMatchSuccesses = telemetry.Counter("scale_backend_fetch_match_successes", "fetch match successes")
	FetchMatchErrors    = telemetry.Counter("scale_backend_fetch_match_errors", "fetch match errors")
	MatchesReturned     = telemetry.Counter("scale_backend_matches_returned", "matches returned")
	SumTicketsReturned  = telemetry.Counter("scale_backend_sum_tickets_returned", "tickets in matches returned")
	MatchesAssigned     = telemetry.Counter("scale_backend_matches_assigned", "matches assigned")
	MatchAssignsFailed  = telemetry.Counter("scale_backend_match_assigns_failed", "match assigns failed")
	TicketsDeleted      = telemetry.Counter("scale_backend_tickets_deleted", "tickets deleted")
	TicketDeletesFailed = telemetry.Counter("scale_backend_ticket_deletes_failed", "ticket deletes failed")

	TicketsCreated        = telemetry.Counter("scale_frontend_tickets_created", "tickets created")
	TicketCreationsFailed = telemetry.Counter("scale_frontend_ticket_creations_failed", "tickets created")
	RunnersWaiting        = concurrentGauge(telemetry.Gauge("scale_frontend_runners_waiting", "runners waiting"))
	RunnersCreating       = concurrentGauge(telemetry.Gauge("scale_frontend_runners_creating", "runners creating"))

	BackfillsCreated        = telemetry.Counter("scale_frontend_backfills_created", "backfills_created")
	BackfillCreationsFailed = telemetry.Counter("scale_frontend_backfill_creations_failed", "backfill creations failed")
	TicketsTimeToAssignment = telemetry.HistogramWithBounds("scale_frontend_tickets_time_to_assignment", "tickets time to assignment", stats.UnitMilliseconds, []float64{0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000})
)

// Allows concurrent moficiation of a gauge value by modifying the concurrent
// value with a delta.
func concurrentGauge(s *stats.Int64Measure) func(delta int64) {
	m := sync.Mutex{}
	v := int64(0)
	return func(delta int64) {
		m.Lock()
		defer m.Unlock()

		v += delta
		telemetry.SetGauge(context.Background(), s, v)
	}
}

// stateGauge will have a single value be applied to one gauge at a time.
type StateGauge struct {
	f func(int64)
}

// start begins a stage measured in a gauge, stopping any previously started
// stage.
func (g *StateGauge) Start(f func(int64)) {
	g.Stop()
	g.f = f
	f(1)
}

// stop finishes the current stage by decrementing the gauge.
func (g *StateGauge) Stop() {
	if g.f != nil {
		g.f(-1)
		g.f = nil
	}
}
