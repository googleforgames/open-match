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

	errMap  = &sync.Map{}
	created uint64
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

	go doCreate(cfg, fe)

	select {}
}

func doCreate(cfg config.View, fe pb.FrontendClient) {
	concurrent := cfg.GetInt("testConfig.concurrentCreates")
	start := time.Now()
	for {
		var wg sync.WaitGroup
		for i := 0; i < concurrent; i++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				req := &pb.CreateTicketRequest{
					Ticket: tickets.Ticket(),
				}

				ctx, span := trace.StartSpan(context.Background(), "scale.frontend/CreateTicket")
				defer span.End()

				if _, err := fe.CreateTicket(ctx, req); err != nil {
					errMsg := fmt.Sprintf("failed to create a ticket: %w", err)
					errRead, ok := errMap.Load(errMsg)
					if !ok {
						errRead = 0
					}
					errMap.Store(errMsg, errRead.(int)+1)
				}
				atomic.AddUint64(&created, 1)
			}(&wg)
		}

		// Wait for all concurrent creates to complete.
		wg.Wait()
		errMap.Range(func(k interface{}, v interface{}) bool {
			logger.Infof("Got error %s: %#v", k, v)
			return true
		})
		logger.Infof("%v tickets created in %v", created, time.Since(start))
	}
}
