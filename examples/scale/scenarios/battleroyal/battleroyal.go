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

package battleroyal

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName  = "all"
	regionArg = "region"
)

func battleRoyalRegionName(i int) string {
	return fmt.Sprintf("region_%d", i)
}

func Scenario() *BattleRoyalScenario {
	return &BattleRoyalScenario{
		regions: 20,
	}
}

type BattleRoyalScenario struct {
	regions int
}

func (b *BattleRoyalScenario) Profiles() []*pb.MatchProfile {
	p := []*pb.MatchProfile{}

	for i := 0; i < b.regions; i++ {
		p = append(p, &pb.MatchProfile{
			Name: battleRoyalRegionName(i),
			Pools: []*pb.Pool{
				{
					Name: poolName,
					StringEqualsFilters: []*pb.StringEqualsFilter{
						{
							StringArg: regionArg,
							Value:     battleRoyalRegionName(i),
						},
					},
				},
			},
		})
	}
	return p
}

func (b *BattleRoyalScenario) Ticket() *pb.Ticket {
	// Simple way to give an uneven distribution of region population.
	a := rand.Intn(b.regions) + 1
	r := rand.Intn(a)

	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				regionArg: battleRoyalRegionName(r),
			},
		},
	}
}

func (b *BattleRoyalScenario) Backfill() *pb.Backfill {
	return nil
}

func (b *BattleRoyalScenario) MatchFunction(p *pb.MatchProfile, poolBackfills map[string][]*pb.Backfill, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	const playersInMatch = 100

	tickets := poolTickets[poolName]
	var matches []*pb.Match

	for i := 0; i+playersInMatch <= len(tickets); i += playersInMatch {
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
			Tickets:       tickets[i : i+playersInMatch],
			MatchProfile:  p.GetName(),
			MatchFunction: "battleRoyal",
		})
	}

	return matches, nil
}

// fifoEvaluate accepts all matches which don't contain the same ticket as in a
// previously accepted match.  Essentially first to claim the ticket wins.
func (b *BattleRoyalScenario) Evaluate(stream pb.Evaluator_EvaluateServer) error {
	used := map[string]struct{}{}

	// TODO: once the evaluator client supports sending and receiving at the
	// same time, don't buffer, just send results immediately.
	matchIDs := []string{}

outer:
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Error reading evaluator input stream: %w", err)
		}

		m := req.GetMatch()

		for _, t := range m.Tickets {
			if _, ok := used[t.Id]; ok {
				continue outer
			}
		}

		for _, t := range m.Tickets {
			used[t.Id] = struct{}{}
		}

		matchIDs = append(matchIDs, m.GetMatchId())
	}

	for _, mID := range matchIDs {
		err := stream.Send(&pb.EvaluateResponse{MatchId: mID})
		if err != nil {
			return fmt.Errorf("Error sending evaluator output stream: %w", err)
		}
	}

	return nil
}

func (b *BattleRoyalScenario) Backend() func(pb.BackendServiceClient, pb.FrontendServiceClient, *logrus.Entry) error {
	return nil
}

func (b *BattleRoyalScenario) Frontend() func(pb.FrontendServiceClient, *logrus.Entry) error {
	return nil
}
