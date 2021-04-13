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
	"context"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/examples/scale/scenarios"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.frontend",
	})
	activeScenario = scenarios.ActiveScenario

	mTicketsCreated          = telemetry.Counter("scale_frontend_tickets_created", "tickets created")
	mTicketCreationsFailed   = telemetry.Counter("scale_frontend_ticket_creations_failed", "tickets created")
	mRunnersWaiting          = concurrentGauge(telemetry.Gauge("scale_frontend_runners_waiting", "runners waiting"))
	mRunnersCreating         = concurrentGauge(telemetry.Gauge("scale_frontend_runners_creating", "runners creating"))
	mTicketsDeleted          = telemetry.Counter("scale_frontend_tickets_deleted", "tickets deleted")
	mTicketDeletesFailed     = telemetry.Counter("scale_frontend_ticket_deletes_failed", "ticket deletes failed")
	mBackfillsCreated        = telemetry.Counter("scale_frontend_backfills_created", "backfills_created")
	mBackfillCreationsFailed = telemetry.Counter("scale_frontend_backfill_creations_failed", "backfill creations failed")
	mTicketsTimeToAssignment = telemetry.HistogramWithBounds("scale_frontend_tickets_time_to_assignment", "tickets time to assignment", stats.UnitMilliseconds, []float64{0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 5000, 10000, 20000, 50000, 100000, 200000, 500000, 1000000})
)

type ticketToWatch struct {
	id        string
	createdAt time.Time
}

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	go run(p.Config())

	return nil
}

func run(cfg config.View) {
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("failed to get Frontend connection")
	}
	fe := pb.NewFrontendServiceClient(conn)

	if activeScenario.FrontendCreatesBackfillsOnStart {
		createBackfills(fe, activeScenario.FrontendTotalBackfillsToCreate)
	}

	ticketQPS := int(activeScenario.FrontendTicketCreatedQPS)
	ticketTotal := activeScenario.FrontendTotalTicketsToCreate
	totalCreated := 0

	for range time.Tick(time.Second) {
		for i := 0; i < ticketQPS; i++ {
			if ticketTotal == -1 || totalCreated < ticketTotal {
				go runner(fe)
			}
		}
	}
}

func runner(fe pb.FrontendServiceClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := stateGauge{}
	defer g.stop()

	g.start(mRunnersWaiting)
	// A random sleep at the start of the worker evens calls out over the second
	// period, and makes timing between ticket creation calls a more realistic
	// poisson distribution.
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	g.start(mRunnersCreating)
	createdAt := time.Now()
	id, err := createTicket(ctx, fe)
	if err != nil {
		logger.WithError(err).Error("failed to create a ticket")
		return
	}

	err = watchAssignments(ctx, fe, ticketToWatch{id: id, createdAt: createdAt})
	if err != nil {
		logger.WithError(err).Errorf("failed to get ticket assignment: %s", id)
	} else {
		ms := time.Since(createdAt).Nanoseconds() / 1e6
		stats.Record(ctx, mTicketsTimeToAssignment.M(ms))
	}

	if activeScenario.FrontendDeletesTickets {
		err = deleteTicket(ctx, fe, id)
		if err != nil {
			logger.WithError(err).Errorf("failed to delete ticket: %s", id)
		}
	}
}

func createTicket(ctx context.Context, fe pb.FrontendServiceClient) (string, error) {
	ctx, span := trace.StartSpan(ctx, "scale.frontend/CreateTicket")
	defer span.End()

	req := &pb.CreateTicketRequest{
		Ticket: activeScenario.Ticket(),
	}

	resp, err := fe.CreateTicket(ctx, req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, mTicketCreationsFailed)
		return "", err
	}

	telemetry.RecordUnitMeasurement(ctx, mTicketsCreated)
	return resp.Id, nil
}

func watchAssignments(ctx context.Context, fe pb.FrontendServiceClient, ticket ticketToWatch) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	stream, err := fe.WatchAssignments(ctx, &pb.WatchAssignmentsRequest{TicketId: ticket.id})
	if err != nil {
		return err
	}

	var a *pb.Assignment
	for a.GetConnection() == "" {
		resp, err := stream.Recv()
		if err != nil {
			return err
		}

		a = resp.Assignment
	}

	return nil
}

func createBackfills(fe pb.FrontendServiceClient, numBackfillsToCreate int) error {
	for i := 0; i < numBackfillsToCreate; i++ {
		err := createBackfill(fe)
		if err != nil {
			return err
		}
	}

	return nil
}

func createBackfill(fe pb.FrontendServiceClient) error {
	ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateBackfill")
	defer span.End()

	req := pb.CreateBackfillRequest{
		Backfill: activeScenario.Backfill(),
	}

	_, err := fe.CreateBackfill(ctx, &req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, mBackfillCreationsFailed)
		logger.WithError(err).Error("failed to create backfill")
		return err
	}

	telemetry.RecordUnitMeasurement(ctx, mBackfillsCreated)
	return nil
}

func deleteTicket(ctx context.Context, fe pb.FrontendServiceClient, ticketId string) error {
	_, err := fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: ticketId})
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, mTicketDeletesFailed)
	} else {
		telemetry.RecordUnitMeasurement(ctx, mTicketsDeleted)
	}

	return err
}

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
type stateGauge struct {
	f func(int64)
}

// start begins a stage measured in a gauge, stopping any previously started
// stage.
func (g *stateGauge) start(f func(int64)) {
	g.stop()
	g.f = f
	f(1)
}

// stop finishes the current stage by decrementing the gauge.
func (g *stateGauge) stop() {
	if g.f != nil {
		g.f(-1)
		g.f = nil
	}
}
