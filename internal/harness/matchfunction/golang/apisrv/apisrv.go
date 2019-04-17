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

package apisrv

import (
	"context"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"

	log "github.com/sirupsen/logrus"
)

// This is the function signature for the function to be implemented by the user.
// The harness will pass the Rosters and PlayerPool for the match profile to this
// function and it will return the Rosters to be populated in the proposal.
type MatchFunction func(context.Context, string, []*pb.Roster, []*pb.PlayerPool) (string, []*pb.Roster, error)

// MatchFunctionServer implements pb.MatchFunctionServer, the server generated
// by compiling the protobuf, by fulfilling the pb.MatchFunctionServer interface.
type MatchFunctionServer struct {
	Config config.View
	Logger *log.Entry
	Func   MatchFunction
}

func (*MatchFunctionServer) Run(context.Context, *pb.RunRequest) (*pb.RunResponse, error) {
	// TODO: Add Harness code here to perform the following:
	// Step 1 - Read the profile written to the Backend API
	// Step 2 - Select the player data that we want for our matchmaking logic.
	// Step 3 - Run custom matchmaking logic to try to find a match
	// Step 4 - Create Proposal for the returned results.
	// Step 5 - Export stats about this run.
	return &pb.RunResponse{}, nil
}
