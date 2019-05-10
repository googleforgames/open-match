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

// Package harness provides the Match Making Function service for Open Match harness.
package harness

import (
	"context"

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

// MatchFunction is the function signature for the Match Making Function (MMF) to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
type MatchFunction func(context.Context, *logrus.Entry, string, []*pb.Roster, []*pb.Pool) (string, []*pb.Roster, error)

// MatchFunctionServer implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type matchFunctionService struct {
	cfg           config.View
	functionName  string
	function      MatchFunction
	mmlogicClient pb.MmLogicClient
}

// Run is this harness's implementation of the gRPC call defined in api/protobuf-spec/matchfunction.proto.
func (s *matchFunctionService) Run(*pb.RunRequest, pb.MatchFunction_RunServer) error {
	return nil
}

// runMatchFunction fetches all the data needed from the mmlogic API and calls
// the match function. It then creates a proposal for the match returned.
func (s *matchFunctionService) runMatchFunction(ctx context.Context, req *pb.RunRequest) error {
	return nil
}
