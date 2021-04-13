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

	mIterations            = telemetry.Counter("scale_backend_iterations", "fetch match iterations")
	mFetchMatchCalls       = telemetry.Counter("scale_backend_fetch_match_calls", "fetch match calls")
	mFetchMatchSuccesses   = telemetry.Counter("scale_backend_fetch_match_successes", "fetch match successes")
	mFetchMatchErrors      = telemetry.Counter("scale_backend_fetch_match_errors", "fetch match errors")
	mMatchesReturned       = telemetry.Counter("scale_backend_matches_returned", "matches returned")
	mSumTicketsReturned    = telemetry.Counter("scale_backend_sum_tickets_returned", "tickets in matches returned")
	mMatchesAssigned       = telemetry.Counter("scale_backend_matches_assigned", "matches assigned")
	mMatchAssignsFailed    = telemetry.Counter("scale_backend_match_assigns_failed", "match assigns failed")
	mBackfillsDeleted      = telemetry.Counter("scale_backend_backfills_deleted", "backfills deleted")
	mBackfillDeletesFailed = telemetry.Counter("scale_backend_backfill_deletes_failed", "backfill deletes failed")
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
		logger.Fatalf("failed to connect to Open Match Backend, got %v", err)
	}

	defer beConn.Close()
	be := pb.NewBackendServiceClient(beConn)

	feConn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.Fatalf("failed to connect to Open Match Frontend, got %v", err)
	}

	defer feConn.Close()
	fe := pb.NewFrontendServiceClient(feConn)

	w := logger.Writer()
	defer w.Close()

	matchesToAssign := make(chan *pb.Match, 30000)

	if activeScenario.BackendAssignsTickets {
		for i := 0; i < 100; i++ {
			go runAssignments(be, matchesToAssign)
		}
	}

	backfillsToDelete := make(chan *pb.Backfill, 30000)

	if activeScenario.BackendDeletesBackfills {
		for i := 0; i < 100; i++ {
			go runDeleteBackfills(fe, backfillsToDelete)
		}
	}

	matchesToAcknowledge := make(chan *pb.Match, 30000)

	if activeScenario.BackendAcknowledgesBackfills {
		for i := 0; i < 100; i++ {
			go runAcknowledgeBackfills(fe, matchesToAcknowledge, backfillsToDelete)
		}
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
				runFetchMatches(be, p, matchesToAssign, matchesToAcknowledge)
			}(&wg, p)
		}

		// Wait for all profiles to complete before proceeding.
		wg.Wait()
		telemetry.RecordUnitMeasurement(context.Background(), mIterations)
	}
}

func runFetchMatches(be pb.BackendServiceClient, p *pb.MatchProfile, matchesToAssign chan<- *pb.Match, matchesToAcknowledge chan<- *pb.Match) {
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

	telemetry.RecordUnitMeasurement(ctx, mFetchMatchCalls)
	stream, err := be.FetchMatches(ctx, req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, mFetchMatchErrors)
		logger.WithError(err).Error("failed to get available stream client")
		return
	}

	for {
		// Pull the Match
		resp, err := stream.Recv()
		if err == io.EOF {
			telemetry.RecordUnitMeasurement(ctx, mFetchMatchSuccesses)
			return
		}

		if err != nil {
			telemetry.RecordUnitMeasurement(ctx, mFetchMatchErrors)
			logger.WithError(err).Error("failed to get matches from stream client")
			return
		}

		telemetry.RecordNUnitMeasurement(ctx, mSumTicketsReturned, int64(len(resp.GetMatch().Tickets)))
		telemetry.RecordUnitMeasurement(ctx, mMatchesReturned)

		if activeScenario.BackendAssignsTickets {
			matchesToAssign <- resp.GetMatch()
		}

		if activeScenario.BackendAcknowledgesBackfills {
			matchesToAcknowledge <- resp.GetMatch()
		}
	}
}

func runDeleteBackfills(fe pb.FrontendServiceClient, backfillsToDelete <-chan *pb.Backfill) {
	for b := range backfillsToDelete {
		if !activeScenario.BackfillDeleteCond(b) {
			continue
		}

		ctx := context.Background()
		_, err := fe.DeleteBackfill(ctx, &pb.DeleteBackfillRequest{BackfillId: b.Id})
		if err != nil {
			logger.WithError(err).Errorf("failed to delete backfill: %s", b.Id)
			telemetry.RecordUnitMeasurement(ctx, mBackfillDeletesFailed)
		} else {
			telemetry.RecordUnitMeasurement(ctx, mBackfillsDeleted)
		}
	}
}

func runAcknowledgeBackfills(fe pb.FrontendServiceClient, matchesToAcknowledge <-chan *pb.Match, backfillsToDelete chan<- *pb.Backfill) {
	for m := range matchesToAcknowledge {
		backfillId := m.Backfill.GetId()
		if backfillId == "" {
			continue
		}

		err := acknowledgeBackfill(fe, backfillId)
		if err != nil {
			logger.WithError(err).Errorf("failed to acknowledge backfill: %s", backfillId)
			continue
		}

		if activeScenario.BackendDeletesBackfills {
			backfillsToDelete <- m.Backfill
		}
	}
}

func acknowledgeBackfill(fe pb.FrontendServiceClient, backfillId string) error {
	ctx, span := trace.StartSpan(context.Background(), "scale.frontend/AcknowledgeBackfill")
	defer span.End()

	_, err := fe.AcknowledgeBackfill(ctx, &pb.AcknowledgeBackfillRequest{
		BackfillId: backfillId,
		Assignment: &pb.Assignment{
			Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
		},
	})
	return err
}

func runAssignments(be pb.BackendServiceClient, matchesToAssign <-chan *pb.Match) {
	ctx := context.Background()

	for m := range matchesToAssign {
		ids := []string{}
		for _, t := range m.Tickets {
			ids = append(ids, t.GetId())
		}

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
			telemetry.RecordUnitMeasurement(ctx, mMatchAssignsFailed)
			logger.WithError(err).Error("failed to assign tickets")
			continue
		}

		telemetry.RecordUnitMeasurement(ctx, mMatchesAssigned)
	}
}
