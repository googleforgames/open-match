/*
This is a sample match function that uses the GRPC harness to set up
the match making function as a service. This sample is a reference
to demonstrate the usage of the GRPC harness and should only be used as
a starting point for your match function. You will need to modify the
matchmaking logic in this function based on your game's requirements.

Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	harness "github.com/googleforgames/open-match/internal/harness/matchfunction/golang"
	"github.com/googleforgames/open-match/internal/pb"

	log "github.com/sirupsen/logrus"
)

func main() {

	// Invoke the harness to setup a GRPC service that handles requests to run the
	// match function. The harness itself queries open match for player pools for
	// the specified request and passes the pools to the match function to generate
	// proposals.
	harness.ServeMatchFunction(&harness.HarnessParams{
		FunctionName:          "simple-matchfunction",
		ServicePortConfigName: "api.functions.port",
		ProxyPortConfigName:   "api.functions.proxyport",
		Func:                  makeMatches,
	})
}

// makeMatches is where your custom matchmaking logic lives.
// Input:
//  - profile : 'Properties' of a MatchObject specified in the ListMatches or CreateMatch call.
//  - rosters : An array of Rosters. By convention, your input Roster contains players already in
//              the match, and the names of pools to search when trying to fill an empty slot.
//  - pools   : An array PlayerPool messages. Contains all the players returned by the MMLogic API
//              upon querying for the player pools. It already had ignorelists applied as of the
//              time the player pool was queried.
// Output:
//  - (results) JSON blob to populated in the MatchObject 'Properties' field sent to the ListMatches
//              or CreateMatch call.
//  - (rosters) Populated team rosters.  Use is optional but recommended;
//              you'll need to construct at least one Roster with all the players you're selecting
//              as it is used to add those players to the ignorelist. You'll also need all
//              the players you want to assign to your DGS in Roster(s) when you call the
//              BackendAPI CreateAssignments() endpoint.  Might as well put them in rosters now.
//  - error :   Use if you need to return an unrecoverable error.
func makeMatches(ctx context.Context, logger *log.Entry, profile string, rosters []*pb.Roster, pools []*pb.PlayerPool) (string, []*pb.Roster, error) {

	// Open Match will try to marshal your JSON roster to an array of protobuf Roster objects. It's
	// up to you if you want to fill these protobuf Roster objects or just write your Rosters in your
	// custom JSON blob. This example uses the protobuf Rosters.

	// Used for tracking metrics.
	var selectedPlayerCount int64

	// Loop through all the team rosters sent in the call to create a match.
	for ti, team := range rosters {
		logger.Infof(" Attempting to fill team: %v", team.Name)

		// Loop through all the players slots on this team roster.
		for si, slot := range team.Players {

			// Loop through all the pools and check if there is a pool with a matching name to the
			// poolName for this player slot. Just one example of a way for your matchmaker to
			// specify which pools your MMF should search through to fill a given player slot.
			// Optional, feel free to change as you see fit.
			for _, pool := range pools {
				if slot.Pool == pool.Name && len(pool.Roster.Players) > 0 {

					/////////////////////////////////////////////////////////
					// These next few lines are where you would put your custom logic, such as
					// searching the pool for players with similar latencies or skill ratings
					// to the players you have already selected.  This example doesn't do anything
					// but choose at random!
					logger.Infof("Looking for player in pool: %v, size: %v", pool.Name, len(pool.Roster.Players))
					randPlayerIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(pool.Roster.Players))

					// Get random player with this index
					selectedPlayer := pool.Roster.Players[randPlayerIndex]
					logger.Infof("Selected player index %v: %v", randPlayerIndex, selectedPlayer.Id)

					// Remove this player from the array as they are now used up.
					pool.Roster.Players[randPlayerIndex] = pool.Roster.Players[0]

					// This is a functional pop from a set.
					_, pool.Roster.Players = pool.Roster.Players[0], pool.Roster.Players[1:]

					// Write the player to the slot and loop.
					rosters[ti].Players[si] = selectedPlayer
					selectedPlayerCount++
					break
					/////////////////////////////////////////////////////////

				} else {
					// Weren't enough players left in the pool to fill all the slots so this example errors out.
					// For this example, this is an error condition and so the match result will have the error
					// populated. If your game can handle partial results, customize this to NOT return an error
					// and instaead populate the result with any properties that may be needed to evaluate the proposal.
					return "", rosters, errors.New("insufficient players")
				}
			}
		}
	}
	logger.Info(" Rosters complete.")

	// You can send back any arbitrary JSON in the first return value (the 'results' string).  It
	// will get sent back out the backend API in the Properties field of the MatchObject message.
	// In this example, all the selected players are populated to the Rosters array, so we'll just
	// pass back the input profile back as the results. If there was anything else arbitrary as
	// output from the MMF, it could easily be included here.
	results := profile

	logger.Infof("Selected %v players", selectedPlayerCount)
	return results, rosters, nil
}
