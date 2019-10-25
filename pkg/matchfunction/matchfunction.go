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

// Package golang provides the Match Making Function service for Open Match golang harness.
package matchfunction

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "matchfunction",
		"component": "matchfunction.library",
	})
)

func FetchPoolTickets(ctx context.Context, mmlClient pb.MmLogicClient, req *pb.RunRequest) (map[string][]*pb.Ticket, error) {
	var poolTickets = make(map[string][]*pb.Ticket)
	for _, pool := range req.GetProfile().GetPools() {
		qtClient, err := mmlClient.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: pool})
		if err != nil {
			logger.WithError(err).Error("Failed to get queryTicketClient from mmlogic.")
			return nil, err
		}

		// Aggregate tickets by poolName
		var tickets []*pb.Ticket
		for {
			qtResponse, err := qtClient.Recv()
			if err == io.EOF {
				logger.Trace("Received all results from the queryTicketClient.")
				// Break when all results are received
				break
			}

			if err != nil {
				logger.WithError(err).Error("Failed to receive a response from the queryTicketClient.")
				return nil, err
			}

			tickets = append(tickets, qtResponse.Tickets...)
		}

		poolTickets[pool.GetName()] = tickets
	}

	return poolTickets, nil
}
