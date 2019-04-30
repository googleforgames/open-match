/*

This server is currently a stand alone application that runs evaluation in a
conditional loop through its lifetime, calling the user's evaluation method
in each run. This abstracts the user's evaluation method from how Open Match
stores proposals, interprets results etc - having the user method to simply
focus on approving matches from a set of proposals.

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

// Package apisrv provides the evaluator service for Open Match harness.
package apisrv

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/set"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/ignorelist"
	"github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis/redispb"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/tag"
)

// EvaluateFunction is the evaluator function type.
type EvaluateFunction func(context.Context, []*pb.MatchObject) ([]string, error)

// Evaluator contains the configuration for various components of the Evaluator.
type Evaluator struct {
	Config   config.View
	Logger   *logrus.Entry
	Pool     *redis.Pool
	MMLogic  pb.MmLogicClient
	Evaluate EvaluateFunction
}

// EvaluateForever loops indefinitely to perform Open Match evaluations.
func (s *Evaluator) EvaluateForever() {
	logger := s.Logger

	// Get connection to redis
	redisConn := s.Pool.Get()
	defer func() {
		err := redisConn.Close()
		if err != nil {
			logger.Errorf("failed to close redis connection, %s", err)
		}
	}()
	start := time.Now()
	checkProposals := true
	pollInterval := s.Config.GetInt("evaluator.pollIntervalMs")
	maxWait := s.Config.GetInt64("evaluator.maxWaitMs")

	// This triggers an infinite loop running evaluation on the following conditions:
	//  - all MMFs are complete OR the evaluation interval is reached
	//  - AND at least one MMF has been started since the last we evaluated.
	for {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		cycleStart := time.Now()

		// Get number of running MMFs. 'nil' return from redigo indicates that no new MMFs have even
		// been started since the last time the evaluator ran.  We don't want to evaluate in that case.
		r, err := redishelpers.Retrieve(ctx, s.Pool, "concurrentMMFs")
		if err != nil {
			if err.Error() == "redigo: nil returned" {
				// No MMFs have run since we last evaluated; reset timer and loop
				start = time.Now()
				time.Sleep(time.Duration(pollInterval) * time.Millisecond)
			}

			continue
		}

		numRunning, err := strconv.Atoi(r)
		if err != nil {
			logger.Errorf("Failed to retrieve number of currently running MMFs, %v", err)
		}

		// Some MMFs ran since we last evaluated. We should evaluate the proposal either when all
		// MMFs are complete, or the timeout is reached.
		switch {
		// dividing by 1e6 looks gnarly but is actually the canonical implementation
		// https://github.com/golang/go/issues/5491
		case time.Since(start).Nanoseconds()/1e6 >= maxWait:
			logger.Infof("Maximum evaluator wait time (%v ms) exceeded", maxWait)
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalTrigger, "interval_exceeded"))
			checkProposals = true

		case numRunning <= 0: // Can be less than zero in some less common cases.
			logger.Info("All MMFs complete, trigger evaluation")
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalTrigger, "mmfs_completed"))
			checkProposals = true
		}

		if checkProposals {
			// Only run the evaluator if there are queued proposals.
			checkProposals = false
			results, err := redishelpers.Count(context.Background(), s.Pool, s.Config.GetString("queues.proposals.name"))
			switch {
			case err != nil:
				logger.Errorf("Failed to retrieve the length of the proposal queue from statestorage, %v", err)
			case results == 0:
				logger.Info("no proposals in the queue to evaluate")
			default:
				logger.Infof("%v proposals available, evaluating", results)
				go s.evaluator(ctx)
			}

			// Delete the counter.  This causes subsequent queries to return 'nil' until it is
			// re-created again on first write.  This enables us to skip evaluation logic if no MMF
			// has written to the counter since last evaluation.
			err = redishelpers.Delete(context.Background(), s.Pool, "concurrentMMFs")
			if err != nil {
				logger.WithFields(logrus.Fields{
					"error": err.Error(),
				}).Error("error deleting concurrent MMF counter")
			}
			start = time.Now()
		}

		// A sleep here is not critical but just a useful safety valve in case
		// things are broken, to keep the main loop from going all-out and spamming the log.
		logger.WithFields(logrus.Fields{"elapsed ": time.Since(cycleStart)}).Info("Evaluation complete. Sleeping for 1 sec.")
		time.Sleep(time.Duration(1000) * time.Millisecond)
	}
}

// evaluator queries all the proposals from Open Match and calls the user's evaluator with this collection
// of proposals. It receives approved proposals which it then updates in Open Match as results.
func (s *Evaluator) evaluator(ctx context.Context) error {
	logger := s.Logger
	logger.Info("Starting evaluation")

	// Get connection to redis
	redisConn := s.Pool.Get()
	defer func() {
		err := redisConn.Close()
		if err != nil {
			logger.Errorf("failed to close Redis connection, %v", err)
		}
	}()

	start := time.Now()
	// Get the number of proposals currently queued for evaluation.
	queueName := s.Config.GetString("queues.proposals.name")
	numProposals, err := redis.Int(redisConn.Do("SCARD", queueName))
	if err != nil {
		logger.Errorf("Failed to get proposal count from %v queue, %v", queueName, err)
		return err
	}

	// Get the proposal IDs for the currently queued proposals.
	proposalIds, err := redis.Strings(redisConn.Do("SPOP", queueName, numProposals))
	if err != nil {
		logger.Errorf("Failed to pop %v proposal ids from %v queue, %v", numProposals, queueName, err)
		return err
	}

	logger.Infof("Retrieved %v proposal ids from %v queue", numProposals, queueName)

	// Fetch all the proposals from Open Match state storage.
	proposals, err := s.getProposals(proposalIds)
	if err != nil {
		logger.Errorf("Failed to fetch proposals from redis, %v", err)
		return err
	}

	// Call the user's evaluation callback to get a list of approved proposals.
	approvedProposals, err := s.Evaluate(ctx, proposals)
	logger.Infof("List of approved proposals: %v", approvedProposals)

	// Create a map of proposals to access a proposal by its ID.
	proposalMap := make(map[string]*pb.MatchObject)
	for _, p := range proposals {
		proposalMap[p.Id] = p
	}

	// Lists of approved and rejected players
	approvedPlayers := []string{}
	potentiallyRejectedPlayers := []string{}

	// Approve proposals by renaming proposals (with results and rosters populated) to expected match ID.
	for _, proposalID := range approvedProposals {
		// The proposal is marked as a result by updating its name to the result name expected by the backend API.
		// Sample proposal id: proposal.80e43fa085844eebbf53fc736150ef96.testprofile
		// format: "proposal".unique_matchobject_id.profile_name
		// Expected result id format: unique_matchobject_id.profile_name
		values := strings.Split(proposalID, ".")
		moID, proID := values[1], values[2]
		resultID := moID + "." + proID

		a := getPlayersInProposal(proposalMap[proposalID])
		approvedPlayers = append(approvedPlayers, a...)

		// Approve the proposal
		logger.Infof("approving proposalID: %v -> %v", proposalID, resultID)
		_, err = redisConn.Do("RENAME", proposalID, resultID)
		if err != nil {
			// RENAME only fails if the source key doesn't exist
			logger.Error("Failure to move proposal to approved resultID! ", err)
		}
	}

	// To reject a proposal, remove any roster from the proposal, set error message and rename to expected match ID.
	rejectedProposals := set.Difference(proposalIds, approvedProposals)
	logger.Infof("List of rejected proposals: %v", rejectedProposals)
	for _, proposalID := range rejectedProposals {
		values := strings.Split(proposalID, ".")
		moID, proID := values[1], values[2]
		resultID := moID + "." + proID

		pr := getPlayersInProposal(proposalMap[proposalID])
		potentiallyRejectedPlayers = append(potentiallyRejectedPlayers, pr...)
		logger.Infof("rejecting proposalID: %v -> %v", proposalID, resultID)

		// Set error message
		_, err = redisConn.Do("HSET", proposalID, "error", "Proposed match rejected")
		if err != nil {
			logger.Errorf("Failure to set rejected error for proposal %v, %v", proposalID, err)
		}

		// Remove any rosters
		_, err = redisConn.Do("HSET", proposalID, "rosters", "")
		if err != nil {
			logger.Errorf("Failure to delete rejected rosters for proposal %v, %v", proposalID, err)
		}

		_, err = redisConn.Do("RENAME", proposalID, resultID)
		if err != nil {
			logger.Errorf("Failure to delete rejected proposal for proposal %v, %v", proposalID, err)
		}
	}

	// Remove players that are (only) in the rejected games from the ignorelist
	rejectedPlayers := set.Difference(potentiallyRejectedPlayers, approvedPlayers)
	logger.Infof("Approved Players: %v", approvedPlayers)
	logger.Infof("Rejected Players: %v", rejectedPlayers)

	if len(rejectedPlayers) > 0 {
		logger.Info("Removing rejectedPlayers from 'proposed' ignorelist")
		err = ignorelist.Remove(redisConn, "proposed", rejectedPlayers)
		if err != nil {
			logger.Errorf("Failed to remove rejected players from ignorelist, %v", err)
		}
	} else {
		logger.Infof("No rejected players to re-queue!")
	}

	logger.Infof("Finished evaluation in %v seconds.", time.Since(start).Seconds())
	return nil
}

func (s *Evaluator) getProposals(ids []string) ([]*pb.MatchObject, error) {
	var proposals []*pb.MatchObject
	for _, proposalID := range ids {
		proposal, err := s.getProposal(proposalID)
		if err != nil {
			return nil, fmt.Errorf("Failed to get proposal for id %v, %v", proposalID, err)
		}

		proposals = append(proposals, proposal)
	}

	return proposals, nil
}

func (s *Evaluator) getProposal(proposalID string) (*pb.MatchObject, error) {
	proposal := &pb.MatchObject{Id: proposalID}
	err := redispb.UnmarshalFromRedis(context.Background(), s.Pool, proposal)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch proposal id %v from redis, %v", proposalID, err)
	}

	return proposal, nil
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
