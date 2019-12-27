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

	// The buffered channels attempt to decouple fetch, assign and delete. It is
	// best effort and these operations may still block each other if buffers are full.
	matches   = make(chan *pb.Match, 2000)
	deleteIds = make(chan string, 2000)

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

	go doFetch(cfg, be)
	go doAssign(be)
	go doDelete(fe)

	select {}

}

// doFetch continuously fetches all profiles in a loop and queues up the fetched
// matches for assignment.
func doFetch(cfg config.View, be pb.BackendClient) {
	startTime := time.Now()
	mprofiles := profiles.Generate(cfg)

	for {
		var wg sync.WaitGroup
		for _, p := range mprofiles {
			wg.Add(1)
			go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
				defer wg.Done()
				fetch(be, p)
			}(&wg, p)
		}

		// Wait for all FetchMatches calls to complete before proceeding.
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

func fetch(be pb.BackendClient, p *pb.MatchProfile) {
	for {
		req := &pb.FetchMatchesRequest{
			Config: &pb.FunctionConfig{
				Host: "om-function",
				Port: 50502,
				Type: pb.FunctionConfig_GRPC,
			},
			Profiles: []*pb.MatchProfile{p},
		}

		ctx, span := trace.StartSpan(context.Background(), "scale.backend/FetchMatches")
		defer span.End()

		stream, err := be.FetchMatches(ctx, req)
		if err != nil {
			errMsg := fmt.Sprintf("failed to get available stream client: %w", err)
			errRead, ok := errMap.Load(errMsg)
			if !ok {
				errRead = 0
			}
			errMap.Store(errMsg, errRead.(int)+1)
			return
		}

		for {
			resp, err := stream.Recv()
			if err == io.EOF {
				return
			}

			if err != nil {
				errMsg := fmt.Sprintf("failed to stream in the halfway: %w", err)
				errRead, ok := errMap.Load(errMsg)
				if !ok {
					errRead = 0
				}
				errMap.Store(errMsg, errRead.(int)+1)
				return
			}

			matches <- resp.GetMatch()
			atomic.AddUint64(&matchCount, 1)
		}
	}
}

// doAssign continuously assigns matches that were queued in the matches channel
// by doFetch and after successful assignment, queues all the tickets to deleteIds
// channel for deletion by doDelete.
func doAssign(be pb.BackendClient) {

	for {
		match := <-matches
		ids := []string{}
		for _, t := range match.Tickets {
			ids = append(ids, t.Id)
		}

		req := &pb.AssignTicketsRequest{
			TicketIds: ids,
			Assignment: &pb.Assignment{
				Connection: fmt.Sprintf("%d.%d.%d.%d:2222", rand.Intn(256), rand.Intn(256), rand.Intn(256), rand.Intn(256)),
			},
		}

		if _, err := be.AssignTickets(context.Background(), req); err != nil {
			errMsg := fmt.Sprintf("failed to assign tickets: %w", err)
			errRead, ok := errMap.Load(errMsg)
			if !ok {
				errRead = 0
			}
			errMap.Store(errMsg, errRead.(int)+1)
		}

		atomic.AddUint64(&assigned, uint64(len(ids)))
		for _, id := range ids {
			deleteIds <- id
		}
	}
}

// doDelete deletes all the tickets whose ids get added to the deleteIds channel.
func doDelete(fe pb.FrontendClient) {
	for {
		id := <-deleteIds
		req := &pb.DeleteTicketRequest{
			TicketId: id,
		}

		if _, err := fe.DeleteTicket(context.Background(), req); err != nil {
			errMsg := fmt.Sprintf("failed to delete tickets: %w", err)
			errRead, ok := errMap.Load(errMsg)
			if !ok {
				errRead = 0
			}
			errMap.Store(errMsg, errRead.(int)+1)
		}

		atomic.AddUint64(&deleted, 1)
	}
}
