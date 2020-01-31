package scenarios

import (
	"fmt"
	"io"
	"time"

	"open-match.dev/open-match/pkg/pb"
)

const (
	poolName = "all"
)

var (
	firstMatchScenario = &Scenario{
		MMF:                          queryPoolsWrapper(firstMatchMmf),
		Evaluator:                    fifoEvaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     100,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        true,
		Ticket:                       firstMatchTicket,
		Profiles:                     firstMatchProfile,
	}
)

func firstMatchProfile() []*pb.MatchProfile {
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

func firstMatchTicket() *pb.Ticket {
	return &pb.Ticket{}
}

func firstMatchMmf(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	tickets := poolTickets[poolName]
	var matches []*pb.Match

	for i := 0; i+1 < len(tickets); i += 2 {
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), len(matches)),
			Tickets:       []*pb.Ticket{tickets[i], tickets[i+1]},
			MatchProfile:  p.GetName(),
			MatchFunction: "rangeExpandingMatchFunction",
		})
	}

	return matches, nil
}

// fifoEvaluate accepts all matches which don't contain the same ticket as in a
// previously accepted match.  Essentially first to claim the ticket wins.
func fifoEvaluate(stream pb.Evaluator_EvaluateServer) error {
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
