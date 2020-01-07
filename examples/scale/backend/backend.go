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
	"sync/atomic"
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

	errMap     = &sync.Map{}
	matchCount uint64
	assigned   uint64
	deleted    uint64
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
		errMap.Range(func(k interface{}, v interface{}) bool {
			logger.Infof("Got error %s: %#v", k, v)
			return true
		})
		logger.Infof(
			"FetchedMatches:%v, AssignedTickets:%v, DeletedTickets:%v in time %v, Total profiles: %v",
			atomic.LoadUint64(&matchCount),
			atomic.LoadUint64(&assigned),
			atomic.LoadUint64(&deleted),
			time.Since(startTime).Seconds(),
			len(mprofiles),
		)
	}
}

func run(fe pb.FrontendClient, be pb.BackendClient, p *pb.MatchProfile) {
	for {
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
			processError("failed to get available stream client", err)
			return
		}

		processMatches(fe, be, stream)
	}
}

func processMatches(fe pb.FrontendClient, be pb.BackendClient, stream pb.Backend_FetchMatchesClient) {
	for {
		// Pull the Match
		resp, err := stream.Recv()
		if err == io.EOF {
			return
		}

		if err != nil {
			processError("failed to get matches from stream client", err)
			return
		}

		atomic.AddUint64(&matchCount, 1)

		ids := []string{}
		for _, t := range resp.GetMatch().Tickets {
			ids = append(ids, t.GetId())
		}
		// Assign Tickets
		if !activeScenario.BackendAssignsTickets {
			goto Delete
		}

		if _, err := be.AssignTickets(context.Background(), &pb.AssignTicketsRequest{
			TicketIds: ids,
			Assignment: &pb.Assignment{
				Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			},
		}); err != nil {
			processError("failed to assign tickets", err)
			continue
		}

		atomic.AddUint64(&assigned, uint64(len(ids)))

		// Delete Tickets
	Delete:
		if !activeScenario.BackendDeletesTickets {
			goto End
		}

		for _, id := range ids {
			req := &pb.DeleteTicketRequest{
				TicketId: id,
			}

			if _, err := fe.DeleteTicket(context.Background(), req); err != nil {
				processError("failed to delete tickets", err)
				continue
			}

			atomic.AddUint64(&deleted, 1)
		}

	End:
		// Placeholder for future logging/stat aggregation tasks.
	}
}

func processError(desc string, err error) {
	errMsg := fmt.Sprintf("%s: %w", desc, err)
	errRead, ok := errMap.Load(errMsg)
	if !ok {
		errRead = 0
	}
	errMap.Store(errMsg, errRead.(int)+1)
}
