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

package backfillbattleroyal

import (
	"fmt"
	"io"
	"math/rand"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName  = "all"
	regionArg = "region"
)

func battleRoyalRegionName(i int) string {
	return fmt.Sprintf("region_%d", i)
}

func Scenario() *BackfillScenario {
	return &BackfillScenario{
		regions: 20,
	}
}

type BackfillScenario struct {
	regions int
}

func (b *BackfillScenario) Profiles() []*pb.MatchProfile {
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

func (b *BackfillScenario) Ticket() *pb.Ticket {
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

// MatchFunction produces half full matches first and adds a Backfill,
// if backfill is found it would include rest of the tickets (second half) along with Backfill from Redis
func (b *BackfillScenario) MatchFunction(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket, poolBackfills map[string][]*pb.Backfill) ([]*pb.Match, error) {
	const playersInMatch = 100

	tickets := poolTickets[poolName]
	backfills := poolBackfills[poolName]
	var matches []*pb.Match

	if len(backfills) == 0 && len(tickets) > 0 {
		backfill := &pb.Backfill{
			SearchFields: tickets[0].SearchFields,
			Extensions:   tickets[0].Extensions,
		}
		for i := 0; i+playersInMatch/2 <= len(tickets)/2; i += playersInMatch / 2 {
			matches = append(matches, &pb.Match{
				MatchId:            fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				Tickets:            tickets[i : i+playersInMatch/2],
				MatchProfile:       p.GetName(),
				MatchFunction:      "battleRoyal",
				Backfill:           backfill,
				AllocateGameserver: true,
			})
		}
	} else {
		// Second round of MMF: collect all tickets which are left
		for i, j := 0, 0; j < len(backfills) && i+playersInMatch/2 <= len(tickets); i += playersInMatch / 2 {
			matches = append(matches, &pb.Match{
				MatchId:            fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				Tickets:            tickets[i : i+playersInMatch/2],
				MatchProfile:       p.GetName(),
				MatchFunction:      "battleRoyal",
				Backfill:           backfills[j],
				AllocateGameserver: false,
			})
			j++
		}
	}

	return matches, nil
}

// Evaluate fifoEvaluate accepts all matches which don't contain the same ticket as in a
// previously accepted match.  Essentially first to claim the ticket wins.
func (b *BackfillScenario) Evaluate(stream pb.Evaluator_EvaluateServer) error {
	used := map[string]struct{}{}

	// TODO: once the evaluator client supports sending and recieving at the
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
