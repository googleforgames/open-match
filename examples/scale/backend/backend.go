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
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.backend",
	})

	activeScenario = scenarios.ActiveScenario
	statProcessor  = scenarios.NewStatProcessor()
)

// Run triggers execution of functions that continuously fetch, assign and
// delete matches.
func Run() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}

	logging.ConfigureLogging(cfg)
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
				run(fe, be, p)
			}(&wg, p)
		}

		// Wait for all profiles to complete before proceeding.
		wg.Wait()
		statProcessor.SetStat("TimeElapsed", time.Since(startTime).String())
		statProcessor.Log(w)
	}
}

func run(fe pb.FrontendClient, be pb.BackendClient, p *pb.MatchProfile) {
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

	stream, err := be.FetchMatches(ctx, req)
	if err != nil {
		statProcessor.RecordError("failed to get available stream client", err)
		return
	}

	processMatches(fe, be, stream)
}

func processMatches(fe pb.FrontendClient, be pb.BackendClient, stream pb.Backend_FetchMatchesClient) {
	for {
		// Pull the Match
		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			statProcessor.RecordError("failed to get matches from stream client", err)
			return
		}

		statProcessor.IncrementStat("MatchCount", 1)

		ids := []string{}
		for _, t := range resp.GetMatch().Tickets {
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
				statProcessor.RecordError("failed to assign tickets", err)
				continue
			}

			statProcessor.IncrementStat("Assigned", len(ids))
		}

		// Delete Tickets
		if activeScenario.BackendDeletesTickets {
			for _, id := range ids {
				req := &pb.DeleteTicketRequest{
					TicketId: id,
				}

				if _, err := fe.DeleteTicket(context.Background(), req); err != nil {
					statProcessor.RecordError("failed to delete tickets", err)
					continue
				}

				statProcessor.IncrementStat("Deleted", 1)
			}
		}
	}
}
