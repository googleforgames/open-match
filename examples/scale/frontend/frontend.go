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
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/examples/scale/scenarios"
	"open-match.dev/open-match/examples/scale/tickets"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale.frontend",
	})
	activeScenario     = scenarios.ActiveScenario
	statProcessor      = scenarios.NewStatProcessor()
	numOfRoutineCreate = 8

	totalCreated uint32
)

// Run triggers execution of the scale frontend component that creates
// tickets at scale in Open Match.
func Run() {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("cannot read configuration.")
	}

	logging.ConfigureLogging(cfg)
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("failed to get Frontend connection")
	}

	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	create(cfg, fe)
}

func create(cfg config.View, fe pb.FrontendClient) {
	w := logger.Writer()
	defer w.Close()

	ticketQPS := int(activeScenario.FrontendTicketCreatedQPS)
	ticketTotal := activeScenario.FrontendTotalTicketsToCreate

	for {
		currentCreated := int(atomic.LoadUint32(&totalCreated))
		if ticketTotal != -1 && currentCreated >= ticketTotal {
			break
		}

		// Each inner loop creates TicketCreatedQPS tickets
		var ticketPerRoutine, ticketModRoutine int
		start := time.Now()

		if ticketTotal == -1 || currentCreated+ticketQPS <= ticketTotal {
			ticketPerRoutine = ticketQPS / numOfRoutineCreate
			ticketModRoutine = ticketQPS % numOfRoutineCreate
		} else {
			ticketPerRoutine = (ticketTotal - currentCreated) / numOfRoutineCreate
			ticketModRoutine = (ticketTotal - currentCreated) % numOfRoutineCreate
		}

		var wg sync.WaitGroup
		for i := 0; i < numOfRoutineCreate; i++ {
			wg.Add(1)
			if i < ticketModRoutine {
				go createPerCycle(&wg, fe, ticketPerRoutine+1, start)
			} else {
				go createPerCycle(&wg, fe, ticketPerRoutine, start)
			}
		}

		// Wait for all concurrent creates to complete.
		wg.Wait()
		statProcessor.SetStat("TotalCreated", atomic.LoadUint32(&totalCreated))
		statProcessor.Log(w)
	}
}

func createPerCycle(wg *sync.WaitGroup, fe pb.FrontendClient, ticketPerRoutine int, start time.Time) {
	defer wg.Done()
	cycleCreated := 0

	for j := 0; j < ticketPerRoutine; j++ {
		req := &pb.CreateTicketRequest{
			Ticket: tickets.Ticket(),
		}

		ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateTicket")
		defer span.End()

		timeLeft := start.Add(time.Second).Sub(time.Now())
		if timeLeft <= 0 {
			break
		}
		ticketsLeft := ticketPerRoutine - cycleCreated

		time.Sleep(timeLeft / time.Duration(ticketsLeft))

		if _, err := fe.CreateTicket(ctx, req); err != nil {
			statProcessor.RecordError("failed to create a ticket", err)
		}
		cycleCreated++
	}

	atomic.AddUint32(&totalCreated, uint32(cycleCreated))
}
