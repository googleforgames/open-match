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
	"time"

	"github.com/sirupsen/logrus"
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
	mOutstandingRunners    = telemetry.Gauge("scale_frontend_outstanding_runners", "outstanding runners")
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

	ticker := time.NewTicker(time.Second)

	totalCreated := 0
	outstanding := int64(0)
	runnerDone := make(chan struct{})

	for {
		select {
		case <-ticker.C:
			for i := 0; i < ticketQPS; i++ {
				if ticketTotal == -1 || totalCreated < ticketTotal {
					go runner(fe, runnerDone)
					outstanding++
				}
			}

		case <-runnerDone:
			outstanding--
		}

		telemetry.SetGauge(context.Background(), mOutstandingRunners, outstanding)
	}
}

func runner(fe pb.FrontendServiceClient, done chan struct{}) {
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	{ // Always create ticket.
		req := &pb.CreateTicketRequest{
			Ticket: activeScenario.Ticket(),
		}

		ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateTicket")
		defer span.End()

		if _, err := fe.CreateTicket(ctx, req); err == nil {
			telemetry.RecordUnitMeasurement(ctx, mTicketsCreated)
		} else {
			logger.WithError(err).Error("failed to create a ticket")
			telemetry.RecordUnitMeasurement(ctx, mTicketCreationsFailed)
		}
	}

	done <- struct{}{}
}
