/*
package apisrv provides an implementation of the gRPC server defined in
../../../api/protobuf-spec/matchfunction.proto.

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

// Package apisrv provides the Match Making Function service for Open Match harness.
package apisrv

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/gogo/protobuf/proto"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/sirupsen/logrus"
)

// MatchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
type MatchFunction func(context.Context, *logrus.Entry, string, []*pb.Roster, []*pb.PlayerPool) (string, []*pb.Roster, error)

// MatchFunctionServer implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type MatchFunctionServer struct {
	FunctionName string
	Config       config.View
	Logger       *logrus.Entry
	Func         MatchFunction
	MMLogic      pb.MmLogicClient
}

// Run is this harness's implementation of the gRPC call defined in api/protobuf-spec/matchfunction.proto.
func (s *MatchFunctionServer) Run(ctx context.Context, req *pb.RunRequest) (*pb.RunResponse, error) {
	// Set up tagging for OpenCensus

	fnCtx, err := tag.New(ctx, tag.Insert(KeyMethod, "Run"))
	if err != nil {
		s.Logger.WithFields(logrus.Fields{"error": err.Error()}).Error("Failed setting up OpenCensus tagging")
		return nil, err
	}

	fnCtx, err = tag.New(fnCtx, tag.Insert(KeyFnName, s.FunctionName))
	if err != nil {
		s.Logger.WithFields(logrus.Fields{"error": err.Error()}).Error("Failed setting up OpenCensus tagging")
		return nil, err
	}

	stats.Record(fnCtx, HarnessRequests.M(1))
	start := time.Now()
	err = s.runMatchFunction(fnCtx, req)
	stats.Record(fnCtx, HarnessLatencySec.M(time.Since(start).Seconds()))
	if err != nil {
		s.Logger.WithFields(logrus.Fields{"error": err.Error()}).Error("harness.Run error")
		stats.Record(fnCtx, HarnessFailures.M(1))
		return &pb.RunResponse{}, err
	}

	return &pb.RunResponse{}, nil
}

// runMatchFunction fetches all the data needed from the mmlogic API and calls
// the match function. It then creates a proposal for the match returned.
func (s *MatchFunctionServer) runMatchFunction(ctx context.Context, req *pb.RunRequest) error {
	// Get a cancel-able context
	mmlogic := s.MMLogic
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Read the profile written to the Backend API
	response, err := mmlogic.GetProfile(ctx, &pb.GetProfileRequest{Match: &pb.MatchObject{Id: req.ProfileId}})
	if err != nil {
		return fmt.Errorf("Failed to get profile %v, %v", req.ProfileId, err)
	}

	profile := response.Match
	// Query the player data.
	playerPools := make([]*pb.PlayerPool, len(profile.Pools))
	numPlayers := 0
	for index, emptyPool := range profile.Pools {
		playerPools[index] = proto.Clone(emptyPool).(*pb.PlayerPool)
		playerPools[index].Roster = &pb.Roster{Players: []*pb.Player{}}

		// Loop through the Pool filter results stream.
		stream, err := mmlogic.GetPlayerPool(ctx, &pb.GetPlayerPoolRequest{PlayerPool: emptyPool})
		if err != nil {
			return fmt.Errorf("Failed to get player pool for %v, %v", emptyPool.Name, err)
		}

		for {
			response, err := stream.Recv()
			if err == io.EOF {
				// Break when all results are received
				break
			}

			if err != nil {
				return fmt.Errorf("Failed to get player pool for %v, %v", emptyPool.Name, err)
			}

			resultPool := response.PlayerPool
			// Update stats with the latest results
			emptyPool.Stats = resultPool.Stats

			// Put players into the Pool's Roster with the attributes we already retrieved through our filter queries.
			if resultPool.Roster != nil && len(resultPool.Roster.Players) > 0 {
				for _, player := range resultPool.Roster.Players {
					playerPools[index].Roster.Players = append(playerPools[index].Roster.Players, proto.Clone(player).(*pb.Player))
					numPlayers++
				}
			}
		}
	}

	// Generate a MatchObject message to write to state storage with the results in it.
	match := &pb.MatchObject{
		Id:         req.ResultId,
		Properties: profile.Properties,
		Pools:      profile.Pools,
	}

	// Return error when there are no players in the pools
	if numPlayers == 0 {
		s.Logger.Info("All player pools are empty, writing directly to ResultID to skip the evaluator")
		match.Error = "insufficient players"
	} else {
		// Run custom matchmaking logic to try to find a match
		stats.Record(ctx, FnRequests.M(1))
		start := time.Now()
		results, rosters, err := s.Func(ctx, s.Logger, profile.Properties, profile.Rosters, playerPools)
		stats.Record(ctx, FnLatencySec.M(time.Since(start).Seconds()))
		if err != nil {
			s.Logger.WithFields(logrus.Fields{
				"error": err.Error(),
				"id":    req.ResultId,
			}).Error("matchfunction returned an unrecoverable error")
			stats.Record(ctx, FnFailures.M(1))
			match.Error = err.Error()
		} else {
			// Prepare the proposal match object.
			match.Id = req.ProposalId
			match.Properties = results
			match.Rosters = rosters
		}
	}

	_, err = mmlogic.CreateProposal(ctx, &pb.CreateProposalRequest{Match: match})
	if err != nil {
		return fmt.Errorf("Failed to create a proposal for %v, %v", match.Id, err)
	}

	s.Logger.WithFields(logrus.Fields{"id": req.ProposalId}).Info("Proposal written successfully!")
	return nil
}
