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

package evaluator

import (
	"context"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/statestore"
)

// The service implementing the Evaluator API that is called to evaluate matches.
type evaluatorService struct {
	cfg   config.View
	store statestore.Service
}

// newEvaluator creates and initializes the evaluator service.
func newEvaluator(cfg config.View) (*evaluatorService, error) {
	es := &evaluatorService{
		cfg: cfg,
	}

	// Initialize the state storage interface.
	var err error
	es.store, err = statestore.New(cfg)
	if err != nil {
		return nil, err
	}

	return es, nil
}

// Evaluate accepts a list of matches, triggers the user configured evaluation
// function with these and other matches in the evaluation window and returns
// matches that are accepted by the Evaluator as valid results.
func (s *evaluatorService) Evaluate(ctx context.Context, req *pb.EvaluateRequest) (*pb.EvaluateResponse, error) {
	return &pb.EvaluateResponse{}, nil
}
