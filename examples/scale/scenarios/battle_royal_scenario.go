package scenarios

import (
	"fmt"
	"math/rand"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

const (
	battleRoyalRegions = 20
	regionArg          = "region"
)

var (
	battleRoyalScenario = &Scenario{
		MMF:                          queryPoolsWrapper(battleRoyalMmf),
		Evaluator:                    fifoEvaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     100,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        true,
		Ticket:                       battleRoyalTicket,
		Profiles:                     battleRoyalProfile,
	}
)

func battleRoyalProfile() []*pb.MatchProfile {
	p := []*pb.MatchProfile{}

	for i := 0; i < battleRoyalRegions; i++ {
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

func battleRoyalTicket() *pb.Ticket {
	// Simple way to give an uneven distribution of region population.
	a := rand.Intn(battleRoyalRegions) + 1
	r := rand.Intn(a)

	return &pb.Ticket{
		SearchFields: &pb.SearchFields{
			StringArgs: map[string]string{
				regionArg: battleRoyalRegionName(r),
			},
		},
	}
}

func battleRoyalMmf(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
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

func battleRoyalRegionName(i int) string {
	return fmt.Sprintf("region_%d", i)
}
