package scenarios

import (
	"fmt"
	"io"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/testing/customize/evaluator"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/evaluator/evaluate"
)

var (
	matchName = "roster-based-matchfunction"

	emptyRosterSpot = "EMPTY_ROSTER_SPOT"

	basicScenario = &Scenario{
		MMF:                          basicMatchFunction,
		Evaluator:                    basicEvaluate,
		FrontendTotalTicketsToCreate: -1,
		FrontendTicketCreatedQPS:     500,
		BackendAssignsTickets:        true,
		BackendDeletesTickets:        true,
	}

	logger = logrus.WithFields(logrus.Fields{
		"app":       "scale",
		"component": "scenarios.basic",
	})
)

func basicMatchFunction(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	poolTickets, err := matchfunction.QueryPools(stream.Context(), getMmlogicGRPCClient(), req.GetProfile().GetPools())
	if err != nil {
		return err
	}

	// Generate proposals.
	proposals, err := makeMatches(req.GetProfile(), poolTickets)
	if err != nil {
		return err
	}

	logger.WithFields(logrus.Fields{
		"proposals": proposals,
	}).Trace("proposals returned by match function")

	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			return err
		}
	}

	return nil
}

func basicEvaluate(stream pb.Evaluator_EvaluateServer) error {
	var matches = []*pb.Match{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		matches = append(matches, req.GetMatch())
	}

	// Run the customized evaluator!
	results, err := evaluate.Evaluate(&evaluator.Params{
		Logger: logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "evaluator.implementation",
		}),
		Matches: matches,
	})
	if err != nil {
		return status.Error(codes.Aborted, err.Error())
	}

	for _, result := range results {
		if err := stream.Send(&pb.EvaluateResponse{Match: result}); err != nil {
			return err
		}
	}

	logger.WithFields(logrus.Fields{
		"results": results,
	}).Debug("matches accepted by the evaluator")
	return nil
}

func makeMatches(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	// This roster based match function expects the match profile to have a
	// populated roster specifying the empty slots for each pool name and also
	// have the ticket pools referenced in the roster. It generates matches by
	// populating players from the specified pools into rosters.
	wantTickets, err := wantPoolTickets(p.GetRosters())
	if err != nil {
		return nil, err
	}

	var matches []*pb.Match
	count := 0
	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		matchRosters := []*pb.Roster{}

		// Loop through each pool wanted in the rosters and pick the number of
		// wanted players from the respective Pool.
		for poolName, want := range wantTickets {
			have, ok := poolTickets[poolName]
			if !ok {
				// A wanted Pool was not found in the Pools specified in the profile.
				insufficientTickets = true
				break
			}

			if len(have) < want {
				// The Pool in the profile has fewer tickets than what the roster needs.
				insufficientTickets = true
				break
			}

			// Populate the wanted tickets from the Tickets in the corresponding Pool.
			matchTickets = append(matchTickets, have[0:want]...)
			poolTickets[poolName] = have[want:]
			var ids []string
			for _, ticket := range matchTickets {
				ids = append(ids, ticket.Id)
			}

			matchRosters = append(matchRosters, &pb.Roster{
				Name:      poolName,
				TicketIds: ids,
			})
		}

		if insufficientTickets {
			// Ran out of Tickets. Matches cannot be created from the remaining Tickets.
			break
		}

		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), count),
			MatchProfile:  p.GetName(),
			MatchFunction: matchName,
			Tickets:       matchTickets,
			Rosters:       matchRosters,
		})

		count++
	}

	return matches, nil
}

// wantPoolTickets parses the roster to return a map of the Pool name to the
// number of empty roster slots for that Pool.
func wantPoolTickets(rosters []*pb.Roster) (map[string]int, error) {
	wantTickets := make(map[string]int)
	for _, r := range rosters {
		if _, ok := wantTickets[r.GetName()]; ok {
			// We do not expect multiple Roster Pools to have the same name.
			logger.Errorf("multiple rosters with same name not supported")
			return nil, status.Error(codes.InvalidArgument, "multiple rosters with same name not supported")
		}

		wantTickets[r.GetName()] = 0
		for _, slot := range r.GetTicketIds() {
			if slot == emptyRosterSpot {
				wantTickets[r.GetName()] = wantTickets[r.GetName()] + 1
			}
		}
	}

	return wantTickets, nil
}
