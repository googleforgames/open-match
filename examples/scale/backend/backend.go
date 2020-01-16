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
	"open-match.dev/open-match/examples/scale/profiles"
	"open-match.dev/open-match/examples/scale/scenarios"
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
	statProcessor  = scenarios.NewStatProcessor()

	mIterations          = telemetry.Counter("scale_backend_iterations", "fetch match iterations")
	mFetchMatchCalls     = telemetry.Counter("scale_backend_fetch_match_calls", "fetch match calls")
	mFetchMatchSuccesses = telemetry.Counter("scale_backend_fetch_match_successes", "fetch match successes")
	mFetchMatchErrors    = telemetry.Counter("scale_backend_fetch_match_errors", "fetch match errors")
	mMatchesReturned     = telemetry.Counter("scale_backend_matches_returned", "matches returned")
	mSumTicketsReturned  = telemetry.Counter("scale_backend_sum_tickets_returned", "tickets in matches returned")
	mMatchesAssigned     = telemetry.Counter("scale_backend_matches_assigned", "matches assigned")
	mMatchAssignsFailed  = telemetry.Counter("scale_backend_match_assigns_failed", "match assigns failed")
	mTicketsDeleted      = telemetry.Counter("scale_backend_tickets_deleted", "tickets deleted")
	mTicketDeletesFailed = telemetry.Counter("scale_backend_ticket_deletes_failed", "ticket deletes failed")
)

// Run triggers execution of functions that continuously fetch, assign and
// delete matches.
func BindService(p *rpc.ServerParams, cfg config.View) error {
	go run(cfg)
	return nil
}

func run(cfg config.View) {
	beConn, err := rpc.GRPCClientFromConfig(cfg, "api.backend")
	if err != nil {
		logger.Fatalf("failed to connect to Open Match Backend, got %v", err)
	}

	defer beConn.Close()
	be := pb.NewBackendClient(beConn)

	feConn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.Fatalf("failed to connect to Open Match Frontend, got %v", err)
	}

	defer feConn.Close()
	fe := pb.NewFrontendClient(feConn)

	startTime := time.Now()
	mprofiles := profiles.Generate(cfg)

	statProcessor.SetStat("TotalProfiles", len(mprofiles))

	w := logger.Writer()
	defer w.Close()

	for {
		// Keep pulling matches from Open Match backend
		var wg sync.WaitGroup
		for _, p := range mprofiles {
			wg.Add(1)
			go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
				defer wg.Done()
				runIteration(fe, be, p)
			}(&wg, p)
		}

		// Wait for all profiles to complete before proceeding.
		wg.Wait()
		statProcessor.SetStat("TimeElapsed", time.Since(startTime).String())
		telemetry.RecordUnitMeasurement(context.Background(), mIterations)
		statProcessor.Log(w)
	}
}

func runIteration(fe pb.FrontendClient, be pb.BackendClient, p *pb.MatchProfile) {
	ctx, span := trace.StartSpan(context.Background(), "scale.backend/FetchMatches")
	defer span.End()

	req := &pb.FetchMatchesRequest{
		Config: &pb.FunctionConfig{
			Host: "om-function",
			Port: 50502,
			Type: pb.FunctionConfig_GRPC,
		},
		Profiles: []*pb.MatchProfile{p},
	}

	telemetry.RecordUnitMeasurement(ctx, mFetchMatchCalls)
	stream, err := be.FetchMatches(ctx, req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, mFetchMatchErrors)
		statProcessor.RecordError("failed to get available stream client", err)
		return
	}

	processMatches(ctx, fe, be, stream)
}

func processMatches(ctx context.Context, fe pb.FrontendClient, be pb.BackendClient, stream pb.Backend_FetchMatchesClient) {
	for {
		// Pull the Match
		resp, err := stream.Recv()
		if err == io.EOF {
			telemetry.RecordUnitMeasurement(ctx, mFetchMatchSuccesses)
			return
		}

		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mFetchMatchErrors)
			statProcessor.RecordError("failed to get matches from stream client", err)
			return
		}

		telemetry.RecordUnitMeasurement(ctx, mMatchesReturned)
		statProcessor.IncrementStat("MatchCount", 1)

		ids := []string{}
		for _, t := range resp.GetMatch().Tickets {
			telemetry.RecordUnitMeasurement(ctx, mSumTicketsReturned)
			ids = append(ids, t.GetId())
		}
		// Assign Tickets
		if activeScenario.BackendAssignsTickets {
			if _, err := be.AssignTickets(context.Background(), &pb.AssignTicketsRequest{
				TicketIds: ids,
				Assignment: &pb.Assignment{
					Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
				},
			}); err != nil {
				telemetry.RecordUnitMeasurement(ctx, mMatchAssignsFailed)
				statProcessor.RecordError("failed to assign tickets", err)
				continue
			}

			telemetry.RecordUnitMeasurement(ctx, mMatchesAssigned)
			statProcessor.IncrementStat("Assigned", len(ids))
		}

		// Delete Tickets
		if activeScenario.BackendDeletesTickets {
			for _, id := range ids {
				req := &pb.DeleteTicketRequest{
					TicketId: id,
				}

				if _, err := fe.DeleteTicket(context.Background(), req); err != nil {
					telemetry.RecordUnitMeasurement(ctx, mTicketDeletesFailed)
					statProcessor.RecordError("failed to delete tickets", err)
					continue
				}

				telemetry.RecordUnitMeasurement(ctx, mTicketsDeleted)
				statProcessor.IncrementStat("Deleted", 1)
			}
		}
	}
}
