// Copyright 2020 Google LLC
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

package backfill

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/examples/scale/metrics"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName = "all"
)

func Scenario() *BackfillScenario {
	return &BackfillScenario{
		BackfillsToCreate:             1000,
		TicketsPerMatch:               2,
		TotalTickets:                  -1,
		TicketQPS:                     100,
		GoroutinesWatchingAssignments: 100,
		GoroutinesAckTickets:          100,
		GoroutinesDeletingTickets:     100,
	}
}

type BackfillScenario struct {
	BackfillsToCreate             int
	TicketQPS                     int
	TotalTickets                  int
	TicketsPerMatch               int
	GoroutinesWatchingAssignments int
	GoroutinesAckTickets          int
	GoroutinesDeletingTickets     int
}

func (s *BackfillScenario) Profiles() []*pb.MatchProfile {
	return []*pb.MatchProfile{
		{
			Name: "entirePool",
			Pools: []*pb.Pool{
				{
					Name: poolName,
				},
			},
		},
	}
}

func (s *BackfillScenario) Ticket() *pb.Ticket {
	return &pb.Ticket{}
}

func (s *BackfillScenario) Backfill() *pb.Backfill {
	return &pb.Backfill{}
}

func (s *BackfillScenario) MatchFunction(p *pb.MatchProfile, poolBackfills map[string][]*pb.Backfill, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	var matches []*pb.Match

	for pool, backfills := range poolBackfills {
		tickets, ok := poolTickets[pool]

		if !ok || len(tickets) == 0 {
			// no tickets in pool
			continue
		}

		for _, b := range backfills {
			l := len(tickets)

			if l == 0 {
				// no tickets left
				break
			}

			if l > s.TicketsPerMatch && s.TicketsPerMatch > 0 {
				l = s.TicketsPerMatch
			}

			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				Tickets:       tickets[0:l],
				MatchProfile:  p.GetName(),
				MatchFunction: "backfill",
				Backfill:      b,
			})
			tickets = tickets[l:]
		}
	}

	return matches, nil
}

func (s *BackfillScenario) Evaluate(stream pb.Evaluator_EvaluateServer) error {
	tickets := map[string]struct{}{}
	backfills := map[string]struct{}{}
	matchIds := []string{}

outer:
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read evaluator input stream: %w", err)
		}

		m := req.GetMatch()

		if _, ok := backfills[m.Backfill.Id]; ok {
			continue outer
		}

		for _, t := range m.Tickets {
			if _, ok := tickets[t.Id]; ok {
				continue outer
			}
		}

		for _, t := range m.Tickets {
			tickets[t.Id] = struct{}{}
		}

		matchIds = append(matchIds, m.GetMatchId())
	}

	for _, id := range matchIds {
		err := stream.Send(&pb.EvaluateResponse{MatchId: id})
		if err != nil {
			return fmt.Errorf("failed to sending evaluator output stream: %w", err)
		}
	}

	return nil
}

func (s *BackfillScenario) Backend() func(pb.BackendServiceClient, pb.FrontendServiceClient, *logrus.Entry) error {
	return func(be pb.BackendServiceClient, fe pb.FrontendServiceClient, logger *logrus.Entry) error {
		w := logger.Writer()
		defer w.Close()

		matchesToAcknowledge := make(chan *pb.Match, 60000)

		for i := 0; i < s.GoroutinesAckTickets; i++ {
			go runAcknowledgeBackfills(fe, matchesToAcknowledge, logger)
		}

		// Don't go faster than this, as it likely means that FetchMatches is throwing
		// errors, and will continue doing so if queried very quickly.
		for range time.Tick(time.Millisecond * 250) {
			// Keep pulling matches from Open Match backend
			profiles := s.Profiles()
			var wg sync.WaitGroup

			for _, p := range profiles {
				wg.Add(1)
				go func(wg *sync.WaitGroup, p *pb.MatchProfile) {
					defer wg.Done()
					runFetchMatches(be, p, matchesToAcknowledge, logger)
				}(&wg, p)
			}

			// Wait for all profiles to complete before proceeding.
			wg.Wait()
			telemetry.RecordUnitMeasurement(context.Background(), metrics.Iterations)
		}

		return nil
	}
}

func runAcknowledgeBackfills(fe pb.FrontendServiceClient, matchesToAcknowledge <-chan *pb.Match, logger *logrus.Entry) {
	for m := range matchesToAcknowledge {
		ids := []string{}
		for _, t := range m.Tickets {
			ids = append(ids, t.GetId())
		}

		err := doAcknowledgeBackfill(fe, m.Backfill.Id)
		if err != nil {
			logger.WithError(err).Error("failed to acknowledge backfill")
			continue
		}
	}
}

func doAcknowledgeBackfill(fe pb.FrontendServiceClient, backfillId string) error {
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

func runFetchMatches(be pb.BackendServiceClient, p *pb.MatchProfile, matchesToAcknowledge chan<- *pb.Match, logger *logrus.Entry) {
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

		matchesToAcknowledge <- resp.GetMatch()
	}
}

type ticketToWatch struct {
	ticketId    string
	requestedAt time.Time
}

func (s *BackfillScenario) Frontend() func(pb.FrontendServiceClient, *logrus.Entry) error {
	return func(fe pb.FrontendServiceClient, logger *logrus.Entry) error {
		err := runCreateBackfills(fe, s.Backfill, s.BackfillsToCreate, logger)
		if err != nil {
			return err
		}

		ticketsToDelete := make(chan string, 60000)
		ticketsToWatch := make(chan ticketToWatch, 60000)

		for i := 0; i < s.GoroutinesWatchingAssignments; i++ {
			go runWatchAssignments(fe, ticketsToWatch, ticketsToDelete, logger)
		}

		for i := 0; i < s.GoroutinesDeletingTickets; i++ {
			go runDeleteTickets(fe, ticketsToDelete, logger)
		}

		go runCreateTickets(fe, s.Ticket, s.TotalTickets, s.TicketQPS, ticketsToWatch, logger)

		return nil
	}
}

func runDeleteTickets(fe pb.FrontendServiceClient, ticketsToDelete <-chan string, logger *logrus.Entry) {
	for id := range ticketsToDelete {
		ctx := context.Background()
		_, err := fe.DeleteTicket(ctx, &pb.DeleteTicketRequest{TicketId: id})

		if err != nil {
			logger.WithError(err).Errorf("failed to delete ticket: %s", id)
			telemetry.RecordUnitMeasurement(ctx, metrics.TicketDeletesFailed)
		} else {
			telemetry.RecordUnitMeasurement(ctx, metrics.TicketsDeleted)
		}
	}
}

func runWatchAssignments(fe pb.FrontendServiceClient, ticketsToWatch chan ticketToWatch, ticketsToDelete chan<- string, logger *logrus.Entry) {
	for t := range ticketsToWatch {
		ctx := context.Background()
		stream, err := fe.WatchAssignments(ctx, &pb.WatchAssignmentsRequest{TicketId: t.ticketId})
		if err != nil {
			logger.WithError(err).Errorf("failed to get ticket assignment: %s", t.ticketId)
			ticketsToDelete <- t.ticketId
			continue
		}

		var a *pb.Assignment
	outer:
		for a.GetConnection() == "" {
			resp, err := stream.Recv()
			if err != nil {
				logger.WithError(err).Errorf("failed to get ticket assignment: %s", t.ticketId)
				break outer
			}

			a = resp.Assignment
		}

		err = stream.CloseSend()
		if err != nil {
			logger.WithError(err).Error("failed to close WatchAssignments stream")
		}

		if a.GetConnection() != "" {
			ms := time.Since(t.requestedAt).Nanoseconds() / 1e6
			stats.Record(ctx, metrics.TicketsTimeToAssignment.M(ms))
		}

		ticketsToDelete <- t.ticketId
	}
}

func runCreateTickets(fe pb.FrontendServiceClient, f func() *pb.Ticket, totalTickets int, ticketQPS int, ticketsToWatch chan<- ticketToWatch, logger *logrus.Entry) {
	totalCreated := 0

	for range time.Tick(time.Second) {
		for i := 0; i < ticketQPS; i++ {
			if totalTickets == -1 || totalCreated < totalTickets {
				go doCreateTicket(fe, f, ticketsToWatch, logger)
			}
		}
	}
}

func doCreateTicket(fe pb.FrontendServiceClient, f func() *pb.Ticket, ticketsToWatch chan<- ticketToWatch, logger *logrus.Entry) {
	ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateTicket")
	defer span.End()

	g := metrics.StateGauge{}
	defer g.Stop()

	g.Start(metrics.RunnersWaiting)
	// A random sleep at the start of the worker evens calls out over the second
	// period, and makes timing between ticket creation calls a more realistic
	// poisson distribution.
	time.Sleep(time.Duration(rand.Int63n(int64(time.Second))))

	g.Start(metrics.RunnersCreating)
	req := pb.CreateTicketRequest{
		Ticket: f(),
	}

	resp, err := fe.CreateTicket(ctx, &req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, metrics.TicketCreationsFailed)
		logger.WithError(err).Error("failed to create a ticket")
		return
	}

	telemetry.RecordUnitMeasurement(ctx, metrics.TicketsCreated)
	ticketsToWatch <- ticketToWatch{ticketId: resp.Id, requestedAt: time.Now()}
}

func runCreateBackfills(fe pb.FrontendServiceClient, f func() *pb.Backfill, backfillsToCreate int, logger *logrus.Entry) error {
	for i := 0; i < backfillsToCreate; i++ {
		err := doCreateBackfill(fe, f, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

func doCreateBackfill(fe pb.FrontendServiceClient, f func() *pb.Backfill, logger *logrus.Entry) error {
	ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateBackfill")
	defer span.End()

	req := pb.CreateBackfillRequest{
		Backfill: f(),
	}

	_, err := fe.CreateBackfill(ctx, &req)
	if err != nil {
		telemetry.RecordUnitMeasurement(ctx, metrics.BackfillCreationsFailed)
		logger.WithError(err).Error("failed to create backfill")
		return err
	}

	telemetry.RecordUnitMeasurement(ctx, metrics.BackfillsCreated)
	return nil
}
