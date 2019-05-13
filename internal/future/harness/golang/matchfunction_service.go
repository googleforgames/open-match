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
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"

	"github.com/sirupsen/logrus"
)

// matchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
// Input:
//  - MatchFunctionParams:
//			A structure that defines the resources that are available to the match function.
//			Developers can choose to add context to the structure such that match function has the ability
//			to cancel a stream response/request or to limit match function by sharing a static and protected view.
type matchFunction func(*MatchFunctionParams) []*pb.Match

// matchFunctionService implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	cfg           config.View
	functionName  string
	function      matchFunction
	mmlogicClient pb.MmLogicClient
}

// MatchFunctionParams : This harness example defines a protected view for the match function.
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
type MatchFunctionParams struct {
	Logger            *logrus.Entry
	ProfileName       string
	Properties        string
	Rosters           []*pb.Roster
	PoolNameToTickets map[string][]*pb.Ticket
}

// Run is this harness's implementation of the gRPC call defined in api/protobuf-spec/matchfunction.proto.
func (s *matchFunctionService) Run(req *pb.RunRequest, mfServer pb.MatchFunction_RunServer) error {
	return nil
}

func newMatchFunctionService(cfg config.View, fs *FunctionSettings) (*matchFunctionService, error) {
	mmlogicClient, err := getMMLogicClient(cfg)
	if err != nil {
		harnessLogger.Errorf("Failed to get MMLogic client, %v.", err)
		return nil, err
	}

	mmfService := &matchFunctionService{cfg: cfg, functionName: fs.FunctionName, function: fs.Func, mmlogicClient: mmlogicClient}
	return mmfService, nil
}
