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
	"fmt"
	"io"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName = "all"
)

func Scenario() *BackfillScenario {
	return &BackfillScenario{
		TicketsPerMatch: 2,
	}
}

type BackfillScenario struct {
	TicketsPerMatch int
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
