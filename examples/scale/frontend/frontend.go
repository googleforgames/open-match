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
	"fmt"
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
	numOfRoutineCreate = 8

	errMap       = &sync.Map{}
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

	go create(cfg, fe)

	select {}
}

func create(cfg config.View, fe pb.FrontendClient) {
	for {
		if activeScenario.TotalTicketsToCreate != -1 && int(totalCreated) >= activeScenario.TotalTicketsToCreate {
			break
		}

		// Each inner loop creates TicketCreatedQPS tickets
		var ticketPerRoutine, ticketModRoutine int
		start := time.Now()
		if int(totalCreated+activeScenario.TicketCreatedQPS) <= activeScenario.TotalTicketsToCreate {
			ticketPerRoutine = int(activeScenario.TicketCreatedQPS) / numOfRoutineCreate
			ticketModRoutine = int(activeScenario.TicketCreatedQPS) % numOfRoutineCreate
		} else {
			ticketPerRoutine = (activeScenario.TotalTicketsToCreate - int(totalCreated)) / numOfRoutineCreate
			ticketModRoutine = (activeScenario.TotalTicketsToCreate - int(totalCreated)) % numOfRoutineCreate
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
		errMap.Range(func(k interface{}, v interface{}) bool {
			logger.Infof("Got error %s: %#v", k, v)
			return true
		})
		logger.Infof("%v tickets created in %v", totalCreated, time.Since(start))
	}
}

func createPerCycle(wg *sync.WaitGroup, fe pb.FrontendClient, ticketPerRoutine int, start time.Time) {
	defer wg.Done()
	var cycleCreated uint32

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
		ticketsLeft := uint32(ticketPerRoutine) - cycleCreated

		time.Sleep(timeLeft / time.Duration(ticketsLeft))

		if _, err := fe.CreateTicket(ctx, req); err != nil {
			errMsg := fmt.Sprintf("failed to create a ticket: %w", err)
			errRead, ok := errMap.Load(errMsg)
			if !ok {
				errRead = 0
			}
			errMap.Store(errMsg, errRead.(int)+1)
		}
		cycleCreated++
	}

	atomic.AddUint32(&totalCreated, cycleCreated)
}
