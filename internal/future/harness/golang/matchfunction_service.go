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

// Package golang provides the Match Making Function service for Open Match golang harness.
package golang

import (
	"context"
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"

	"github.com/sirupsen/logrus"
)

var (
	matchfunctionLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "harness.golang.matchfunction_service",
	})
)

// matchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
// Input:
//  - matchFunctionView:
//			A structure that defines the resources that are available to the match function.
//			Developers can choose to add context to the structure such that match function has the ability
//			to cancel a stream response/request or to limit match function by sharing a static and protected view.
type matchFunction func(*MatchFunctionView) []*pb.Match

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	cfg           config.View
	functionName  string
	function      matchFunction
	mmlogicClient pb.MmLogicClient
}

// MatchFunctionView : This harness example defines a protected view for the match function.
//  - logger:
//			A logger used to generate error/debug logs
//  - profileName
//			The name of profile
//  - properties:
//			'Properties' of a MatchObject.
//  - rosters:
//			An array of Rosters. By convention, your input Roster contains players already in
//          the match, and the names of pools to search when trying to fill an empty slot.
//  - poolNameToTickets:
//			A map that contains mappings from pool name to a list of tickets that satisfied the filters in the pool
type MatchFunctionView struct {
	Logger            *logrus.Entry
	ProfileName       string
	Properties        string
	Rosters           []*pb.Roster
	PoolNameToTickets map[string][]*pb.Ticket
}

// Run is this harness's implementation of the gRPC call defined in api/protobuf-spec/matchfunction.proto.
func (s *matchFunctionService) Run(req *pb.RunRequest, mfServer pb.MatchFunction_RunServer) error {
	ctx := mfServer.Context()

	poolNameToTickets, err := s.getMatchManifest(ctx, req)
	if err != nil {
		return status.Error(codes.Code(codes.Aborted), err.Error())
	}

	// The matchfunction takes in some half-filled/empty rosters, a property bag, and a map[poolNames]tickets to generate match proposals
	mfView := &MatchFunctionView{
		Logger:            matchfunctionLogger,
		ProfileName:       req.Profile.Name,
		Properties:        req.Profile.Properties,
		Rosters:           req.Profile.Roster,
		PoolNameToTickets: poolNameToTickets,
	}
	// Run the customize match function!
	matchProposals := s.function(mfView)

	// Send the proposals back in stream
	for _, match := range matchProposals {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := mfServer.Send(&pb.RunResponse{Proposal: match})
			if err != nil {
				matchfunctionLogger.WithError(err).Error("Failed to send Run response to grpc server.")
				return status.Error(codes.Code(codes.Aborted), err.Error())
			}
		}
	}
	return nil
}

// getMatchManifest fetches all the data needed from the mmlogic API.
func (s *matchFunctionService) getMatchManifest(ctx context.Context, req *pb.RunRequest) (map[string][]*pb.Ticket, error) {
	poolNameToTickets := make(map[string][]*pb.Ticket)
	filterPools := req.Profile.Pool

	for _, pool := range filterPools {
		qtClient, err := s.mmlogicClient.QueryTickets(ctx, &pb.QueryTicketsRequest{Pool: pool})
		if err != nil {
			matchfunctionLogger.WithError(err).Error("Failed to get queryTicketClient from mmlogic.")
			return nil, err
		}

		// Aggregate tickets by poolName
		poolTickets := make([]*pb.Ticket, 0)
		for {
			qtResponse, err := qtClient.Recv()
			if err == io.EOF {
				matchfunctionLogger.Trace("Received all results from the queryTicketClient.")
				// Break when all results are received
				break
			}

			if err != nil {
				matchfunctionLogger.WithError(err).Error("Failed to receive a response from the queryTicketClient.")
				return nil, err
			}
			poolTickets = append(poolTickets, qtResponse.Ticket...)
		}
		poolNameToTickets[pool.Name] = poolTickets
	}

	return poolNameToTickets, nil
}
