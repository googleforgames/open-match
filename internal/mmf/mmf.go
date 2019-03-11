package mmf

import (
	"errors"
	"math/rand"
	"time"

	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
)

// MakeMatches is where your custom matchmaking logic lives.
// Input:
//  - profile : JSON blob input to the BackendAPI when you called ListMatches
//              or CreateMatch. It is the contents of the MatchObject 'Properties' field.
//  - rosters : an array of protobuf Roster messages. By convention, your input
//              Roster contains players already in the match and pools to search for each
//              empty slot. Feel free to get more fancy with it if you need more/different
//              structure.
//  - pools : An array of protobuf PlayerPool messages. Contains all the
//            players returned by the MMLogic API when asking it for the player pools.
//            Already had ignorelists applied as of the time the player pool was
//            gathered.
// Output:
//  - (results) string : JSON blob to put in the MatchObject 'Properties' field
//              send back as the response to your ListMatches or CreateMatch call.
//  - (rosters) []*api.Roster : Filled team rosters.  Use is optional but
//				recommended; you'll need to construct at least one Roster with all the
//  			players you're selecting before calling the MMLogic CreateProposal()
//  			endpoint, as it is read and used to add those players to the ignorelist.
//  			You'll also need all the players you want to assign to your DGS in
//  			Roster(s) when you call the BackendAPI CreateAssignments() endpoint.
//  - error :   Use if you need to return an unrecoverable error.
func makeMatches(profile string, rosters []*api.Roster, pools []*api.PlayerPool) (string, []*api.Roster, error) {

	// Open Match will try to marshal your JSON roster to an array of protobuf
	// Roster objects.  It's up to you if you want to fill these protobuf
	// Roster objects or just write your Rosters in your custom JSON blob.
	// This example uses the protobuf Rosters.
	//
	// This example is extraordinarily naive and has many edge cases.  It is
	// only intended to be used as a starting point for writing your own MMF!
	mmfLog.Info("In makeMatches")
	for ti, team := range rosters {
		mmfLog.Info(" Attempting to fill team:", team.Name)
		for si, slot := range team.Players {
			for _, pool := range pools {
				if slot.Pool == pool.Name && len(pool.Roster.Players) > 0 {
					mmfLog.Info("   Looking for player in pool: ", pool.Name, len(pool.Roster.Players))
					// Get a random index (subtract 1 because arrays are zero-indexed)
					// TODO probably an off by one errorr, just flooring to zero for GDC demo
					randPlayerIndex := rand.New(rand.NewSource(time.Now().UnixNano())).Intn(len(pool.Roster.Players))
					if randPlayerIndex < 0 {
						randPlayerIndex = 0
					}
					mmfLog.Infof("   Selected player index %v", randPlayerIndex)
					selectedPlayer := pool.Roster.Players[randPlayerIndex]
					mmfLog.Infof("   Selected player index %v: %v", randPlayerIndex, selectedPlayer.Id)
					// Remove this player from the array.
					pool.Roster.Players[randPlayerIndex] = pool.Roster.Players[0]
					_, pool.Roster.Players = pool.Roster.Players[0], pool.Roster.Players[1:]
					mmfLog.Info("   Found player: ", selectedPlayer.Id, " strategy: RANDOM")
					rosters[ti].Players[si] = selectedPlayer
					break
				} else {
					return "{\"error\": \"insufficient players\"}", rosters, errors.New("insufficient players")
				}
			}
		}
	}
	mmfLog.Info(" Rosters complete.")
	mmfLog.Debug(rosters)

	// You can send back any arbitrary JSON in the first return value (the
	// string).  In this example, all the players were chosen and populated to
	// the protobuf Rosters array, so we'll just pass back the input profile as
	// the results.
	results := profile

	return results, rosters, nil
}
