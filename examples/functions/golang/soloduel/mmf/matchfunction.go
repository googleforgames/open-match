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

// Package mmf provides a sample match function that uses the GRPC harness to set up 1v1 matches.
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
	matchName = "a-simple-1v1-matchfunction"
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

		if len(thisMatch) >= 2 {
			matches = append(matches, &pb.Match{
				MatchId:       fmt.Sprintf("profile-%s-time-%s-num-%d", matchName, t, matchNum),
				MatchProfile:  matchName,
				MatchFunction: matchName,
				Tickets:       thisMatch,
			})

			thisMatch = make([]*pb.Ticket, 0, 2)
			matchNum++
		}
	}

	return matches, nil
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
