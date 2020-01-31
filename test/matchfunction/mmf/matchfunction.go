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

// Package mmf provides a sample match function that uses the GRPC harness to set up
// the match making function as a service. This sample is a reference
// to demonstrate the usage of the GRPC harness and should only be used as
// a starting point for your match function. You will need to modify the
// matchmaking logic in this function based on your game's requirements.
package mmf

import (
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/any"
	"github.com/pkg/errors"
	"github.com/rs/xid"
	internalMmf "open-match.dev/open-match/internal/testing/mmf"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchName = "pool-based-match"
)

// MakeMatches is where your custom matchmaking logic lives.
// This is the core match making function that will be triggered by Open Match to generate matches.
// The goal of this function is to generate predictable matches that can be validated without flakyness.
// This match function loops through all the pools and generates one match per pool aggregating all players
// in that pool in the generated match.
func MakeMatches(params *internalMmf.MatchFunctionParams) ([]*pb.Match, error) {
	var result []*pb.Match
	for _, tickets := range params.PoolNameToTickets {
		if len(tickets) != 0 {
			evaluationInput, err := ptypes.MarshalAny(&pb.DefaultEvaluationCriteria{
				Score: scoreCalculator(tickets),
			})
			if err != nil {
				return nil, errors.Wrap(err, "Failed to marshal DefaultEvaluationCriteria.")
			}

			result = append(result, &pb.Match{
				MatchId:       xid.New().String(),
				MatchProfile:  params.ProfileName,
				MatchFunction: matchName,
				Tickets:       tickets,
				Extensions: map[string]*any.Any{
					"evaluation_input": evaluationInput,
				},
			})
		}
	}

	return result, nil
}

// This match function defines the quality of a match as the sum of the Double
// Args values of all tickets per match.  This is for testing purposes, and not
// an example of a good score calculation.
func scoreCalculator(tickets []*pb.Ticket) float64 {
	matchScore := 0.0
	for _, ticket := range tickets {
		for _, v := range ticket.GetSearchFields().GetDoubleArgs() {
			matchScore += v
		}
	}
	return matchScore
}
