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
	"sort"
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
	matchName              = "skill-matchfunction"
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
	// Create a colletion to hold match proposals
	var matches []*pb.Match
	count := 0
	for {
		insufficientTickets := false
		// Create a collection to hold tickets selected for a match
		matchTickets := []*pb.Ticket{}
		for pool, tickets := range poolTickets {
			// Set flag if there are not enough tickets to create a match
			if len(tickets) < ticketsPerPoolPerMatch {
				insufficientTickets = true
				break
			}
			// Sort the tickets based on skill
			sort.Slice(tickets, func(i, j int) bool {
				return tickets[i].SearchFields.DoubleArgs["mmr"] < tickets[j].SearchFields.DoubleArgs["mmr"]
			})

			// Remove the Tickets from this pool and add to the match proposal.
			matchTickets = append(matchTickets, tickets[0:ticketsPerPoolPerMatch]...)
			poolTickets[pool] = tickets[ticketsPerPoolPerMatch:]
		}

		if insufficientTickets {
			break
		}
		// Compute the match quality/score
		matchQuality := computeQuality(matchTickets)
		evaluationInput, err := ptypes.MarshalAny(&pb.DefaultEvaluationCriteria{
			Score: matchQuality,
		})

		if err != nil {
			log.Printf("Failed to marshal DefaultEvaluationCriteria, got %s.", err.Error())
			return nil, fmt.Errorf("Failed to marshal DefaultEvaluationCriteria, got %w", err)
		}
		// Output the match quality for our sanity
		log.Printf("Quality for the generated match is %v", matchQuality)
		// Create a match proposal
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

// Compute the quality as a difference in the highest and lowest player skill levels. This can be used to determine if the match is outside a given skill differential
func computeQuality(tickets []*pb.Ticket) float64 {
	quality := 0.0
	high := 0.0
	low := tickets[0].SearchFields.DoubleArgs["mmr"]
	for _, ticket := range tickets {
		if high < ticket.SearchFields.DoubleArgs["mmr"] {
			high = ticket.SearchFields.DoubleArgs["mmr"]
		}
		if low > ticket.SearchFields.DoubleArgs["mmr"] {
			low = ticket.SearchFields.DoubleArgs["mmr"]
		}
	}
	quality = high - low

	return quality
}
