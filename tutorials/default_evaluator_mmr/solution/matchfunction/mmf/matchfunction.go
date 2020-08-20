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

package mmf

import (
	"fmt"
	"log"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"

	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

// This match function fetches all the Tickets for all the pools specified in
// the profile. It uses a configured number of tickets from each pool to generate
// a Match Proposal. It continues to generate proposals till one of the pools
// runs out of Tickets.
const (
	matchName              = "basic-matchfunction"
	ticketsPerPoolPerMatch = 4
)

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *MatchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())

	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	if err != nil {
		log.Printf("Failed to query tickets for the given pools, got %s", err.Error())
		return err
	}

	// Generate proposals.
	proposals, err := makeMatches(req.GetProfile(), poolTickets)
	if err != nil {
		log.Printf("Failed to generate matches, got %s", err.Error())
		return err
	}

	log.Printf("Streaming %v proposals to Open Match", len(proposals))
	// Stream the generated proposals back to Open Match.
	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			log.Printf("Failed to stream proposals to Open Match, got %s", err.Error())
			return err
		}
	}

	return nil
}

func makeMatches(p *pb.MatchProfile, poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	var matches []*pb.Match
	count := 0
	for {
		insufficientTickets := false
		matchTickets := []*pb.Ticket{}
		for pool, tickets := range poolTickets {
			if len(tickets) < ticketsPerPoolPerMatch {
				// This pool is completely drained out. Stop creating matches.
				insufficientTickets = true
				break
			}

			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, tickets[0:ticketsPerPoolPerMatch]...)
			poolTickets[pool] = tickets[ticketsPerPoolPerMatch:]
		}

		if insufficientTickets {
			break
		}

		matchScore := scoreCalculator(matchTickets)
		evaluationInput, err := ptypes.MarshalAny(&pb.DefaultEvaluationCriteria{
			Score: matchScore,
		})

		if err != nil {
			log.Printf("Failed to marshal DefaultEvaluationCriteria, got %s.", err.Error())
			return nil, fmt.Errorf("Failed to marshal DefaultEvaluationCriteria, got %w", err)
		}

		log.Printf("Score for the generated match is %v", matchScore)
		matches = append(matches, &pb.Match{
			MatchId:       fmt.Sprintf("profile-%v-time-%v-%v", p.GetName(), time.Now().Format("2006-01-02T15:04:05.00"), count),
			MatchProfile:  p.GetName(),
			MatchFunction: matchName,
			Tickets:       matchTickets,
			Extensions: map[string]*any.Any{
				"evaluation_input": evaluationInput,
			},
		})

		count++
	}

	return matches, nil
}

// This match function defines the quality of a match as the difference of the highest and lowest skill
// ratings. When deduplicating overlapping matches, the evaluator will pick the match with the lowest score,
// thus picking the one match with a closer gap in skill levels. Negating the skill level allows the lower
scores to actually be the highest when considered. This will be polished in the custom evaluator example of this sample.
func computeQuality(tickets []*pb.Ticket) float64 {
	score := 0.0
	high := 0.0
	low := 0.0
	for _, ticket := range tickets {
		if high < ticket.SearchFields.DoubleArgs["mmr"] {
			high = ticket.SearchFields.DoubleArgs["mmr"]
		}
		if low > ticket.SearchFields.DoubleArgs["mmr"] {
			low = ticket.SearchFields.DoubleArgs["mmm"]
		}
	}
	quality = (high - low) * -1

	return score
}
