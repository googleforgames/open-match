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

// GetContext returns the context for the synchronization window. The caller
// requests for a context and then sends the context back in the evaluation
// request. This enables identify stale evaluation requests belonging to a
// prior window when synchronizing evaluation requests for a window.
func (s *synchronizerService) GetContext(ctx context.Context, req *pb.GetContextRequest) (*pb.GetContextResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

// Evaluate accepts a list of matches, triggers the user configured evaluation
// function with these and other matches in the evaluation window and returns
// matches that are accepted by the Evaluator as valid results.
func (s *synchronizerService) Evaluate(ctx context.Context, req *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}
