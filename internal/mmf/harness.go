package mmf

import (
	"context"
	"io"
	"time"

	"github.com/golang/protobuf/proto"
	log "github.com/sirupsen/logrus"

	api "github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/spf13/viper"
)

var (
	// Logrus structured logging setup
	mmfLogFields = log.Fields{
		"app":       "openmatch",
		"component": "function_mmlogic_harness",
	}
	mmfLog = log.WithFields(mmfLogFields)
)

// Step 2 - Talk to Redis.
// This example uses the MM Logic API in OM to read/write to/from redis.

// Run is used to kick off the served MMF.
func Run(fnArgs *api.Arguments, cfg *viper.Viper, mmlogic api.MmLogicClient) error {
	ctx, cancel := context.WithCancel(context.Background())
	var start time.Time
	defer cancel()
	mmfLog.Debug("args: ", fnArgs.Request)

	// Step 3 - Read the profile written to the Backend API
	profile, err := mmlogic.GetProfile(ctx, &api.MatchObject{Id: fnArgs.Request.ProfileId})
	if err != nil {
		mmfLog.Error(err)
		return err
	}
	mmfLog.Debug("Profile: ", profile)

	// Step 4 - Select the player data from Redis that we want for our matchmaking logic.
	playerPools := make([]*api.PlayerPool, len(profile.Pools))
	numPlayers := 0
	for index, emptyPool := range profile.Pools {
		poolLog := mmfLog.WithFields(log.Fields{"poolName": emptyPool.Name})
		playerPools[index] = proto.Clone(emptyPool).(*api.PlayerPool)
		playerPools[index].Roster = &api.Roster{Players: []*api.Player{}}
		poolLog.Info("Retrieving pool")

		// DEBUG: Print how long the filtering takes
		if cfg.IsSet("debug") && cfg.GetBool("debug") {
			start = time.Now()
		}

		// Pool filter results are streamed in chunks as they can be too large
		// to send in one grpc message.  Loop to get them all.
		stream, err := mmlogic.GetPlayerPool(ctx, emptyPool)
		if err != nil {
			poolLog.Error(err)
			return err
		}
		for {
			partialResults, err := stream.Recv()
			if err == io.EOF {
				// Break when all results are received
				break
			}
			if err != nil {
				poolLog.Error(err)
				return err
			}

			// Update stats with the latest results
			emptyPool.Stats = partialResults.Stats

			// Put players into the Pool's Roster with their attributes.
			if partialResults.Roster != nil && len(partialResults.Roster.Players) > 0 {
				for _, player := range partialResults.Roster.Players {
					playerPools[index].Roster.Players = append(playerPools[index].Roster.Players, proto.Clone(player).(*api.Player))
					numPlayers++
				}
			}

			if cfg.IsSet("debug") && cfg.GetBool("debug") {
				poolLog.WithFields(log.Fields{
					"elapsed": time.Now().Sub(start).Seconds(),
					"count":   len(playerPools[index].Roster.Players),
				}).Debug("retrieval stats")
			}
		}
	}

	// Generate a MatchObject message to write to state storage with the
	// results in it.  By default, assume we weren't successful (write to error
	// ID) until proven otherwise.
	mo := &api.MatchObject{
		Id:         fnArgs.Request.ResultId,
		Properties: profile.Properties,
		Pools:      profile.Pools,
	}

	// Return error when there are no players in the pools
	if numPlayers == 0 {
		if cfg.IsSet("debug") && cfg.GetBool("debug") {
			mmfLog.Info("All player pools are empty, writing to error to skip the evaluator")
		}
		// Writing to the error key skips the evaluator
		mo.Error = "insufficient players"
	} else {

		////////////////////////////////////////////////////////////////
		// Step 5 - Run custom matchmaking logic to try to find a match
		// This is in the file mmf.go
		results, rosters, err := makeMatches(profile.Properties, profile.Rosters, playerPools)
		////////////////////////////////////////////////////////////////

		if err != nil {
			mmfLog.WithFields(log.Fields{
				"error": err.Error(),
				"id":    fnArgs.Request.ResultId,
			}).Error("MMF returned an unrecoverable error")
			mo.Error = err.Error()
		} else {
			// No error!
			// Prepare the output match object. Actually able to set to the real Proposal ID now!
			mo.Id = fnArgs.Request.ProposalId
			mo.Properties = results
			mo.Rosters = rosters
		}

	}

	// DEBUG
	if cfg.IsSet("debug") && cfg.GetBool("debug") {
		mmfLog.Debug("Output MatchObject:", mo)
	}

	// Step 6 - Write the outcome of the matchmaking logic back to state storage.
	// Step 7 - Remove the selected players from consideration by other MMFs.
	// CreateProposal does both of these for you, and some other items as well.
	success, err := mmlogic.CreateProposal(ctx, mo)
	if err != nil {
		mmfLog.Error(err)
	}
	mmfLog.WithFields(log.Fields{"id": fnArgs.Request.ProposalId, "success": success.Success}).Info("MMF write to state storage")

	// [OPTIONAL] Step 8 - Export stats about this run.
	// TODO
	return nil
}
