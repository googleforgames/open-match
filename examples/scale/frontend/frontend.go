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
	doCreate(cfg)
}

func doCreate(cfg config.View) {
	concurrent := cfg.GetInt("testConfig.concurrent-creates")
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.frontend")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("failed to get Frontend connection")
	}

	defer conn.Close()
	fe := pb.NewFrontendClient(conn)

	var created uint64
	var failed uint64
	start := time.Now()
	for {
		var wg sync.WaitGroup
		for i := 0; i <= concurrent; i++ {
			wg.Add(1)
			go func(wg *sync.WaitGroup) {
				defer wg.Done()
				req := &pb.CreateTicketRequest{
					Ticket: tickets.Ticket(cfg),
				}

				if _, err := fe.CreateTicket(context.Background(), req); err != nil {
					logger.WithFields(logrus.Fields{
						"error": err.Error(),
					}).Error("failed to create a ticket.")
					atomic.AddUint64(&failed, 1)
					return
				}

				atomic.AddUint64(&created, 1)
			}(&wg)
		}

		// Wait for all concurrent creates to complete.
		wg.Wait()
		logger.Infof("%v tickets created, %v failed in %v", created, failed, time.Since(start))
	}
}
