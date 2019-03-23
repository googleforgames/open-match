package apisrv

import (
	"context"
	"errors"
	"math/rand"
	"time"

	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"go.opencensus.io/stats"
)

// MakeMatches is where your custom matchmaking logic lives.
// Input:
//  - profile : JSON blob input to the BackendAPI when you called ListMatches
//              or CreateMatch. It is the contents of the MatchObject 'Properties' field.
//  - rosters : an array of protobuf Roster messages. By convention, your input
//              Roster contains players already in the match, and the names of pools to search
//              when trying to fill an empty slot. Feel free to get more fancy
//              with it if you need more/different structure.
//  - pools : An array of protobuf PlayerPool messages. Contains all the
//            players returned by the MMLogic API when asking it for the player pools.
//            Already had ignorelists applied as of the time the player pool was
//            gathered. Already has values populated for attributes used in
//            filters, as it doesn't cost much to go ahead and grab those from
//            Redis when filtering.
// Output:
//  - (results) string : JSON blob to put in the MatchObject 'Properties' field
//              send back as the response to your ListMatches or CreateMatch call.
//  - (rosters) []*api.Roster : Filled team rosters.  Use is optional but
//				recommended; you'll need to construct at least one Roster with all the
//  			players you're selecting before calling the MMLogic CreateProposal()
//  			endpoint, as it is read and used to add those players to the ignorelist.
//  			You'll also need all the players you want to assign to your DGS in
//  			Roster(s) when you call the BackendAPI CreateAssignments()
//  			endpoint.  Might as well put them in rosters now.
//  - error :   Use if you need to return an unrecoverable error.
func makeMatches(ctx context.Context, profile string, rosters []*api.Roster, pools []*api.PlayerPool) (string, []*api.Roster, error) {

	// Open Match will try to marshal your JSON roster to an array of protobuf
	// Roster objects.  It's up to you if you want to fill these protobuf
	// Roster objects or just write your Rosters in your custom JSON blob.
	// This example uses the protobuf Rosters.
	//
	// This example is extraordinarily naive and has many edge cases.  It is
	// only intended to be used as a starting point for writing your own MMF!

	// Used for tracking metrics.
	var selectedPlayerCount int64

	// Loop through all the team rosters you send in when calling the Backend
	// API CreateMatch() or ListMatches() call.  Customize to fit your own structure.
	for ti, team := range rosters {

		mmfLog.Info(" Attempting to fill team:", team.Name)

		// Loop through all the players slots on this team roster.  Customize with
		// your own structure as you see fit.
		for si, slot := range team.Players {

			// Loop through all the pools we retrieved from state storage, and
			// see if there is a pool with a matching name to the poolName for
			// this player slot. Just one example of a way for your matchmaker
			// to specify which pools your MMF should search through to fill a
			// given player slot. Optional, feel free to change as you see fit.
			for _, pool := range pools {
				if slot.Pool == pool.Name && len(pool.Roster.Players) > 0 {

					/////////////////////////////////////////////////////////
					// These next few lines are where you would put your custom
					// logic, such as searching the pool for players with
					// similar latencies or skill ratings to the players you
					// have already selected.  This example doesn't do anything
					// but choose at random!
					mmfLog.Info("   Looking for player in pool: ", pool.Name, len(pool.Roster.Players))

					// Get a random index (subtract 1 because arrays are zero-indexed)
					// TODO probably an off by one errorr, just flooring to zero for GDC demo for safety
					randPlayerIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(pool.Roster.Players))
					if randPlayerIndex < 0 {
						randPlayerIndex = 0
					}

					// Get random player with this index
					selectedPlayer := pool.Roster.Players[randPlayerIndex]
					mmfLog.Infof("   Selected player index %v: %v", randPlayerIndex, selectedPlayer.Id)

					// Remove this player from the array as they are now used up.
					pool.Roster.Players[randPlayerIndex] = pool.Roster.Players[0]

					// This is a functional pop from a set.
					_, pool.Roster.Players = pool.Roster.Players[0], pool.Roster.Players[1:]
					mmfLog.Info("   Found player: ", selectedPlayer.Id, " strategy: RANDOM")

					// Write the player to the slot and loop.
					rosters[ti].Players[si] = selectedPlayer
					selectedPlayerCount++
					break
					/////////////////////////////////////////////////////////

				} else {
					// Weren't enough players left in the pool to fill all the
					// slots.  If your game can handle non-full teams, feel
					// free to customize this part as you see fit.
					return "{\"error\": \"insufficient players\"}", rosters, errors.New("insufficient players")
				}
			}
		}
	}
	mmfLog.Info(" Rosters complete.")
	mmfLog.Debug(rosters)

	// You can send back any arbitrary JSON in the first return value (the 'results'
	// string).  It will get sent back out the backend API in the Properties
	// field of the MatchObject message.
	// In this example, all the selected players are populated to the protobuf
	// Rosters array, so we'll just pass back the input profile back as the
	// results. If there was anything else arbitrary as output from the MMF, it
	// could easily be included here.
	results := profile

	// Emit some Open Census metrics.
	stats.Record(ctx, FnPlayersMatched.M(selectedPlayerCount))
	mmfLog.Infof(" Selected %v players", selectedPlayerCount)
	return results, rosters, nil
}
