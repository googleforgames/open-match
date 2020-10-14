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

// Package mmf provides a sample match function that uses the GRPC harness to set up matches based on latency requirements.
// This sample is a reference to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/pkg/matchfunction"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "latency-mmf"
)

const (
	ticketsPerMatch       = 0         // Update number of tickets needed
	latencyReqPerMillisec = 0         // Update with the max allowed latency for match (in milliseconds)
	latencyArg            = "latency" // Update with the key for the doubleArg that represents latency in your tickets
)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	grpc               *grpc.Server
	queryServiceClient pb.QueryServiceClient
	port               int
}

func makeMatches(poolTickets map[string][]*pb.Ticket) ([]*pb.Match, error) {
	tickets := map[string]*pb.Ticket{}
	for _, pool := range poolTickets {
		for _, ticket := range pool {
			tickets[ticket.GetId()] = ticket
		}
	}

	var matches []*pb.Match

	t := time.Now().Format("2006-01-02T15:04:05.00")

	thisMatch := make([]*pb.Ticket, 0, 2)
	matchNum := 0

	for _, ticket := range tickets {
		thisMatch = append(thisMatch, ticket)

		// Uncomment if match function relies on evaluator to score matches quality based on latency.
		if len(thisMatch) >= 2 {
			// Compute the match quality/score
			matchQuality := computeQuality(thisMatch)
			evaluationInput, err := ptypes.MarshalAny(&pb.DefaultEvaluationCriteria{
				Score: matchQuality,
			})

			if err != nil {
				log.Printf("Failed to marshal DefaultEvaluationCriteria, got %s.", err.Error())
				return nil, fmt.Errorf("Failed to marshal DefaultEvaluationCriteria, got %w", err)
			}

			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", matchName, t, matchNum),
				MatchProfile:  matchName,
				MatchFunction: matchName,
				Tickets:       thisMatch,
				// Uncomment to allow evaluator to determine match quality based on latency
				// Extensions: map[string]*any.Any{
				// 	"evaluation_input": evaluationInput,
				// },
			})

			thisMatch = make([]*pb.Ticket, 0, 2)
			matchNum++
		}
	}

	return matches, nil
}

// Function for computing the quality based on latency
func computeQuality(tickets []*pb.Ticket) float64 {
	// TO DO: Compute quality
	quality := 0.0

	// Here is a sample that compute the average latency for all tickets, computes the difference between each ticket and the average, and checks to ensure the difference is below the accepted average. The average latency is returned if the match is acceptable, otherwise it will return -1
	// sum := 0.0
	// averageLatency := 0.0
	// // Calulate average
	// for _, ticket := range tickets {
	// 	sum += ticket.SearchFields.DoubleArgs[latencyArg]
	// }
	// averageLatency = sum / float64(len(tickets))

	// // Calcuate the difference of each tickets latency and the average latency. If the diffence is above the accepted latency threshold, ticket match is invalid
	// for _, ticket := range tickets {
	// 	latencydiff := math.Abs(averageLatency - ticket.SearchFields.DoubleArgs[latencyArg])
	// 	if latencyReqPerMillisec < latencydiff {
	// 		return -1.0
	// 	}
	// }

	return quality
}

// Run is this match function's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	// Fetch tickets for the pools specified in the Match Profile.
	log.Printf("Generating proposals for function %v", req.GetProfile().GetName())

	poolTickets, err := matchfunction.QueryPools(stream.Context(), s.queryServiceClient, req.GetProfile().GetPools())
	if err != nil {
		log.Printf("Failed to query tickets for the given pools, got %s", err.Error())
		return err
	}

	// Generate proposals.
	proposals, err := makeMatches(poolTickets)
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
