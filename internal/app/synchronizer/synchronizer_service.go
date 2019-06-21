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

package synchronizer

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
)

// The service implementing the Synchronizer API that synchronizes the evaluation
// of results from Match functions.
type synchronizerService struct {
	cfg   config.View
	store statestore.Service
}

// Register associates this request with the current synchronization cycle and
// returns an identifier for this registration. The caller returns this
// identifier back in the evaluation
// request. This enables identify stale evaluation requests belonging to a
// prior window when synchronizing evaluation requests for a window.
func (s *synchronizerService) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// EvaluateProposals accepts a list of proposals and a registration identifier
// for this request. If the synchronization cycle to which the request was
// registered is completed, this request fails otherwise the proposals are
// added to the list of proposals to be evaluated in the current cycle. At the
//  end of the cycle, the user defined evaluation method is triggered and the
// matches accepted by it are returned as results.
func (s *synchronizerService) EvaluateProposals(ctx context.Context, req *pb.EvaluateProposalsRequest) (*pb.EvaluateProposalsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
