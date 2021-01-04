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
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
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
		"component": "scale.backend",
	})

	activeScenario = scenarios.ActiveScenario
)

// Run triggers execution of functions that continuously fetch, assign and
// delete matches.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	go run(p.Config())
	return nil
}

func run(cfg config.View) {
	beConn, err := rpc.GRPCClientFromConfig(cfg, "api.backend")
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to Open Match Backend")
	}

	defer beConn.Close()
	be := pb.NewBackendServiceClient(beConn)

	feConn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithError(err).Fatal("failed to connect to Open Match Frontend")
	}

	defer feConn.Close()

	fe := pb.NewFrontendServiceClient(feConn)
	backend := activeScenario.Backend()
	if backend == nil {
		backend = defaultBackend
	}

	err = backend(be, fe, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to run Open Match Backend")
	}
}

func defaultBackend(be pb.BackendServiceClient, fe pb.FrontendServiceClient, logger *logrus.Entry) error {
	w := logger.Writer()
	defer w.Close()

	matchesForAssignment := make(chan *pb.Match, 30000)
	ticketsForDeletion := make(chan string, 30000)

	for i := 0; i < 50; i++ {
		go runAssignments(be, matchesForAssignment, ticketsForDeletion)
		go runDeletions(fe, ticketsForDeletion)
	}

	// Don't go faster than this, as it likely means that FetchMatches is throwing
	// errors, and will continue doing so if queried very quickly.
	for range time.Tick(time.Millisecond * 250) {
		// Keep pulling matches from Open Match backend
		profiles := activeScenario.Profiles()
		var wg sync.WaitGroup

		for _, p := range profiles {
			wg.Add(1)
			go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
				defer wg.Done()
				runFetchMatches(be, p, matchesForAssignment)
			}(&wg, p)
		}

		// Wait for all profiles to complete before proceeding.
		wg.Wait()
		telemetry.RecordUnitMeasurement(context.Background(), metrics.Iterations)
	}

	return nil
}

func runFetchMatches(be pb.BackendServiceClient, p *pb.MatchProfile, matchesForAssignment chan<- *pb.Match) {
	ctx, span := trace.StartSpan(context.Background(), "scale.backend/FetchMatches")
	defer span.End()

	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: "open-match-function",
			Port: 50502,
			Type: pb.FunctionConfig_GRPC,
		},
		Profile: p,
	}

	telemetry.RecordUnitMeasurement(ctx, metrics.FetchMatchCalls)
	stream, err := be.FetchMatches(ctx, req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, metrics.FetchMatchErrors)
		logger.WithError(err).Error("failed to get available stream client")
		return
	}

	for {
		// Pull the Match
		resp, err := stream.Recv()
		if err == io.EOF {
			telemetry.RecordUnitMeasurement(ctx, metrics.FetchMatchSuccesses)
			return
		}

		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, metrics.FetchMatchErrors)
			logger.WithError(err).Error("failed to get matches from stream client")
			return
		}

		telemetry.RecordNUnitMeasurement(ctx, metrics.SumTicketsReturned, int64(len(resp.GetMatch().Tickets)))
		telemetry.RecordUnitMeasurement(ctx, metrics.MatchesReturned)

		matchesForAssignment <- resp.GetMatch()
	}
}

func runAssignments(be pb.BackendServiceClient, matchesForAssignment <-chan *pb.Match, ticketsForDeletion chan<- string) {
	ctx := context.Background()

	for m := range matchesForAssignment {
		ids := []string{}
		for _, t := range m.Tickets {
			ids = append(ids, t.GetId())
		}

		if activeScenario.BackendAssignsTickets {
			_, err := be.AssignTickets(context.Background(), &pb.AssignTicketsRequest{
				Assignments: []*pb.AssignmentGroup{
					{
						TicketIds: ids,
						Assignment: &pb.Assignment{
							Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
						},
					},
				},
			})
			if err != nil {
				telemetry.RecordUnitMeasurement(ctx, metrics.MatchAssignsFailed)
				logger.WithError(err).Error("failed to assign tickets")
				continue
			}

			telemetry.RecordUnitMeasurement(ctx, metrics.MatchesAssigned)
		}

		for _, id := range ids {
			ticketsForDeletion <- id
		}
	}
}

func runDeletions(fe pb.FrontendServiceClient, ticketsForDeletion <-chan string) {
	ctx := context.Background()

	for id := range ticketsForDeletion {
		if activeScenario.BackendDeletesTickets {
			req := &pb.DeleteTicketRequest{
				TicketId: id,
			}

			_, err := fe.DeleteTicket(context.Background(), req)

			if err == nil {
				telemetry.RecordUnitMeasurement(ctx, metrics.TicketsDeleted)
			} else {
				telemetry.RecordUnitMeasurement(ctx, metrics.TicketDeletesFailed)
				logger.WithError(err).Error("failed to delete tickets")
			}
		}
	}
}
