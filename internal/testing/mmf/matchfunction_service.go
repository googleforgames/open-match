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

// Package mmf provides the Match Making Function service for Open Match golang harness.
package mmf

import (
	"context"
	"io"

	"github.com/golang/protobuf/ptypes/any"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "matchfunction.harness.golang",
	})
)

// MatchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will query the Tickets for each Pool and will pass the map of Pool to Tickets to the function
// and the function will return a list of proposals.
// Input:
//  - MatchFunctionParams:
//			A structure that defines the resources that are available to the match function.
//			Developers can choose to add context to the structure such that match function has the ability
//			to cancel a stream response/request or to limit match function by sharing a static and protected view.
type MatchFunction func(*MatchFunctionParams) ([]*pb.Match, error)

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	cfg                config.View
	function           MatchFunction
	queryServiceClient pb.QueryServiceClient
}

// MatchFunctionParams is a protected view for the match function.
type MatchFunctionParams struct {
	// Logger is used to generate error/debug logs
	Logger *logrus.Entry

	// 'Name' from the MatchProfile.
	ProfileName string

	// 'Extensions' from the MatchProfile.
	Extensions map[string]*any.Any

	// A map that contains mappings from pool name to a list of tickets that satisfied the filters in the pool
	PoolNameToTickets map[string][]*pb.Ticket
}

// Run is this harness's implementation of the gRPC call defined in api/matchfunction.proto.
func (s *matchFunctionService) Run(req *pb.RunRequest, stream pb.MatchFunction_RunServer) error {
	poolNameToTickets, err := s.getMatchManifest(stream.Context(), req)
	if err != nil {
		return err
	}

	mfParams := &MatchFunctionParams{
		Logger: logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "matchfunction.implementation",
		}),
		ProfileName:       req.GetProfile().GetName(),
		Extensions:        req.GetProfile().GetExtensions(),
		PoolNameToTickets: poolNameToTickets,
	}
	// Run the customize match function!
	proposals, err := s.function(mfParams)
	if err != nil {
		return err
	}
	logger.WithFields(logrus.Fields{
		"proposals": proposals,
	}).Trace("proposals returned by match function")

	for _, proposal := range proposals {
		if err := stream.Send(&pb.RunResponse{Proposal: proposal}); err != nil {
			return err
		}
	}

	return nil
}

func newMatchFunctionService(cfg config.View, mf MatchFunction) (*matchFunctionService, error) {
	conn, err := rpc.GRPCClientFromConfig(cfg, "api.query")
	if err != nil {
		logger.Errorf("Failed to get QueryService connection, %v.", err)
		return nil, err
	}

	mmfService := &matchFunctionService{cfg: cfg, function: mf, queryServiceClient: pb.NewQueryServiceClient(conn)}
	return mmfService, nil
}

// getMatchManifest fetches all the data needed from the queryService API.
func (s *matchFunctionService) getMatchManifest(ctx context.Context, req *pb.RunRequest) (map[string][]*pb.Ticket, error) {
	poolNameToTickets := make(map[string][]*pb.Ticket)
	filterPools := req.GetProfile().GetPools()

	for _, pool := range filterPools {
		qtClient, err := s.queryServiceClient.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: pool}, grpc.WaitForReady(true))
		if err != nil {
			logger.WithError(err).Error("Failed to get queryTicketClient from queryService.")
			return nil, err
		}

		// Aggregate tickets by poolName
		poolTickets := make([]*pb.Ticket, 0)
		for {
			qtResponse, err := qtClient.Recv()
			if err == io.EOF {
				logger.Trace("Received all results from the queryTicketClient.")
				// Break when all results are received
				break
			}

			if err != nil {
				logger.WithError(err).Error("Failed to receive a response from the queryTicketClient.")
				return nil, err
			}
			poolTickets = append(poolTickets, qtResponse.Tickets...)
		}
		poolNameToTickets[pool.GetName()] = poolTickets
	}

	return poolNameToTickets, nil
}
