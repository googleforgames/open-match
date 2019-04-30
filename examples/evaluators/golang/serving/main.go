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

// Package main is a sample Evaluator built using the Evaluator Harness. It evaluates
// multiple proposals and approves a subset of them. This sample demonstrates
// how to build a basic Evaluator using the Evaluator Harness . This example
// over-simplifies the actual evaluation decisions and hence should not be
// used as is for a real scenario.
package main

import (
	"context"
	"fmt"

	harness "github.com/GoogleCloudPlatform/open-match/internal/harness/evaluator/golang"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
)

func main() {
	// This invoke the harness to set up the Evaluator. The harness abstracts
	// fetching proposals ready for evaluation and the process of transforming
	// approved proposals into results that Open Match can relay to the caller
	// requesting for matches.
	harness.RunEvaluator(Evaluate)
}

// Evaluate is where your custom evaluation logic lives.
// Input:
//  - proposals : List of all the proposals to be consiered for evaluation. Each proposal will have
//                Rosters comprising of the players belonging to that proposal.
// Output:
//  - (proposals) : List of approved proposal IDs that can be returned as match results.
func Evaluate(ctx context.Context, proposals []*pb.MatchObject) ([]string, error) {
	// Map of approved and overloaded proposals. Using maps for easier lookup.
	approvedProposals := map[string]bool{}
	overloadedProposals := map[string]bool{}

	// Map of all the players encountered in the proposals. Each entry maps a player id to
	// the first match in which the player was encountered.
	allPlayers := map[string]string{}

	// Iterate over each proposal to either add to approved map or overloaded map.
	for _, proposal := range proposals {
		proposalID := proposal.Id
		approved := true
		players := getPlayersInProposal(proposal)
		// Iterate over each player in the proposal to check if the player was encountered before.
		for _, playerID := range players {
			if propID, found := allPlayers[playerID]; found {
				// Player was encountered in an earlier proposal. Mark the current proposal as overloaded (not approved).
				// Also, the first proposal where the player was encountered may have been marked approved. Remove that proposal
				// approved proposals and add to overloaded proposals since we encountered its player in current proposal too.
				approved = false
				delete(approvedProposals, propID)
				overloadedProposals[propID] = true
			} else {
				// Player encountered for the first time, add to all players map with the current proposal.
				allPlayers[playerID] = proposalID
			}

			if approved {
				approvedProposals[proposalID] = true
			} else {
				overloadedProposals[proposalID] = true
			}
		}
	}

	// Convert the maps to lists of overloaded, approved proposals.
	overloadedList := []string{}
	approvedList := []string{}
	for k := range overloadedProposals {
		overloadedList = append(overloadedList, k)
	}

	for k := range approvedProposals {
		approvedList = append(approvedList, k)
	}

	// Select proposals to approve from the overloaded proposals list.
	chosen, err := chooseProposals(overloadedList)
	if err != nil {
		return nil, fmt.Errorf("Failed to select approved list from overloaded proposals, %v", err)
	}

	// Add the chosen proposals to the approved list.
	approvedList = append(approvedList, chosen...)
	return approvedList, nil
}

// chooseProposals should look through all overloaded proposals (that is, have a player that is also
// in another proposed match) and choose the proposals to approve. This is where the core evaluation
// logic will be added.
func chooseProposals(overloaded []string) ([]string, error) {
	// As a basic example, we pick the first overloaded proposal for approval.
	approved := []string{}
	if len(overloaded) > 0 {
		approved = append(approved, overloaded[0])
	}
	return approved, nil
}

func getPlayersInProposal(proposal *pb.MatchObject) []string {
	var players []string
	for _, r := range proposal.Rosters {
		for _, p := range r.Players {
			players = append(players, p.Id)
		}
	}

	return players
}
