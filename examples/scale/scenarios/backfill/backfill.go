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

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName     = "all"
	openSlotsKey = "open-slots"
)

func Scenario() *BackfillScenario {
	ticketsPerMatch := 4
	return &BackfillScenario{
		TicketsPerMatch:           ticketsPerMatch,
		MaxTicketsPerNotFullMatch: 3,
		BackfillDeleteCond: func(b *pb.Backfill) bool {
			openSlots := getOpenSlots(b, ticketsPerMatch)
			return openSlots <= 0
		},
	}
}

type BackfillScenario struct {
	TicketsPerMatch           int
	MaxTicketsPerNotFullMatch int
	BackfillDeleteCond        func(*pb.Backfill) bool
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
	return statefullMMF(p, poolBackfills, poolTickets, s.TicketsPerMatch, s.MaxTicketsPerNotFullMatch)
}

// statefullMMF is a MMF implementation which is used in scenario when we want MMF to create not full match and fill it later.
// 1. The first FetchMatches is called
// 2. MMF grabs maxTicketsPerNotFullMatch tickets and makes a match and new backfill for it
// 3. MMF sets backfill's open slots to ticketsPerMatch - maxTicketsPerNotFullMatch
// 4. MMF returns the match as a result
// 5. The second FetchMatches is called
// 6. MMF gets previously created backfill
// 7. MMF gets backfill's open slots value
// 8. MMF grabs openSlots tickets and makes a match with previously created backfill
// 9. MMF sets backfill's open slots to 0
// 10. MMF returns the match as a result
func statefullMMF(p *pb.MatchProfile, poolBackfills map[string][]*pb.Backfill, poolTickets map[string][]*pb.Ticket, ticketsPerMatch int, maxTicketsPerNotFullMatch int) ([]*pb.Match, error) {
	var matches []*pb.Match

	for pool, backfills := range poolBackfills {
		tickets, ok := poolTickets[pool]

		if !ok || len(tickets) == 0 {
			// no tickets in pool
			continue
		}

		// process backfills first
		for _, b := range backfills {
			l := len(tickets)
			if l == 0 {
				// no tickets left
				break
			}

			openSlots := getOpenSlots(b, ticketsPerMatch)
			if openSlots <= 0 {
				// no free open slots
				continue
			}

			if l > openSlots {
				l = openSlots
			}

			setOpenSlots(b, openSlots-l)
			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				Tickets:       tickets[0:l],
				MatchProfile:  p.GetName(),
				MatchFunction: "backfill",
				Backfill:      b,
			})
			tickets = tickets[l:]
		}

		// create not full matches with backfill
		for {
			l := len(tickets)
			if l == 0 {
				// no tickets left
				break
			}

			if l > maxTicketsPerNotFullMatch {
				l = maxTicketsPerNotFullMatch
			}
			b := pb.Backfill{}
			setOpenSlots(&b, ticketsPerMatch-l)
			matches = append(matches, &pb.Match{
				MatchId:            fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
				Tickets:            tickets[0:l],
				MatchProfile:       p.GetName(),
				MatchFunction:      "backfill",
				Backfill:           &b,
				AllocateGameserver: true,
			})
			tickets = tickets[l:]
		}
	}

	return matches, nil
}

func getOpenSlots(b *pb.Backfill, defaultVal int) int {
	if b.Extensions == nil {
		return defaultVal
	}

	any, ok := b.Extensions[openSlotsKey]
	if !ok {
		return defaultVal
	}

	var val wrapperspb.Int32Value
	err := any.UnmarshalTo(&val)
	if err != nil {
		panic(err)
	}

	return int(val.Value)
}

func setOpenSlots(b *pb.Backfill, val int) {
	if b.Extensions == nil {
		b.Extensions = make(map[string]*anypb.Any)
	}

	any, err := anypb.New(&wrapperspb.Int32Value{Value: int32(val)})
	if err != nil {
		panic(err)
	}

	b.Extensions[openSlotsKey] = any
}

// statelessMMF is a MMF implementation which is used in scenario when we want MMF to fill backfills created by a Gameserver. It doesn't create
// or update any backfill.
// 1. FetchMatches is called
// 2. MMF gets a backfill
// 3. MMF grabs ticketsPerMatch tickets and makes a match with the backfill
// 4. MMF returns the match as a result
func statelessMMF(p *pb.MatchProfile, poolBackfills map[string][]*pb.Backfill, poolTickets map[string][]*pb.Ticket, ticketsPerMatch int) ([]*pb.Match, error) {
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

			if l > ticketsPerMatch && ticketsPerMatch > 0 {
				l = ticketsPerMatch
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
