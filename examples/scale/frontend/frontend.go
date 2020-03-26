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

	mTicketsCreated        = telemetry.Counter("scale_frontend_tickets_created", "tickets created")
	mTicketCreationsFailed = telemetry.Counter("scale_frontend_ticket_creations_failed", "tickets created")
	mRunnersWaiting        = concurrentGauge(telemetry.Gauge("scale_frontend_runners_waiting", "runners waiting"))
	mRunnersCreating       = concurrentGauge(telemetry.Gauge("scale_frontend_runners_creating", "runners creating"))
)

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	go run(cfg)

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
	id, err := createTicket(ctx, fe)
	if err != nil {
		logger.WithError(err).Error("failed to create a ticket")
		return
	}

	_ = id
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
