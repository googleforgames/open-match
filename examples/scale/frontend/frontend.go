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
	"open-match.dev/open-match/examples/scale/metrics"
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
)

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	go run(p.Config())

	return nil
}

func run(cfg config.View) {
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithError(err).Fatal("failed to get Open Match Frontend connection")
	}

	fe := pb.NewFrontendServiceClient(conn)
	frontend := activeScenario.Frontend()
	if frontend == nil {
		frontend = defaultFrontend
	}

	err = frontend(fe, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to run Open Match frontend")
	}
}

func defaultFrontend(fe pb.FrontendServiceClient, logger *logrus.Entry) error {
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

	return nil
}

func runner(fe pb.FrontendServiceClient) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	g := metrics.StateGauge{}
	defer g.Stop()

	g.Start(metrics.RunnersWaiting)
	// A random sleep at the start of the worker evens calls out over the second
	// period, and makes timing between ticket creation calls a more realistic
	// poisson distribution.
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	g.Start(metrics.RunnersCreating)
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
		telemetry.RecordUnitMeasurement(ctx, metrics.TicketCreationsFailed)
		return "", err
	}

	telemetry.RecordUnitMeasurement(ctx, metrics.TicketsCreated)
	return resp.Id, nil
}
