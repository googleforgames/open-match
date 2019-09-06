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

// Package golang provides the Evaluator service for Open Match golang harness.
package golang

import (
	"io"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/pkg/gopb"

	"github.com/sirupsen/logrus"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "evaluator.harness.golang",
	})
)

// Evaluator is the function signature for the Evaluator to be implemented by
// the user. The harness will pass the Matches to evaluate to the Evaluator
// and the Evaluator will return an accepted list of Matches.
type Evaluator func(*EvaluatorParams) ([]*gopb.Match, error)

// evaluatorService implements gopb.EvaluatorServer, the server generated by
// compiling the protobuf, by fulfilling the gopb.EvaluatorServer interface.
type evaluatorService struct {
	cfg      config.View
	evaluate Evaluator
}

// EvaluatorParams is the parameters to be passed by the harness to the evaluator.
//  - logger:
//			A logger used to generate error/debug logs
//  - Matches
//			Matches to be evaluated
type EvaluatorParams struct {
	Logger  *logrus.Entry
	Matches []*gopb.Match
}

// Evaluate is this harness's implementation of the gRPC call defined in
// api/evaluator.proto.
func (s *evaluatorService) Evaluate(stream gopb.Evaluator_EvaluateServer) error {
	var matches = []*gopb.Match{}
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		matches = append(matches, req.GetMatch())
	}

	// Run the customized evaluator!
	results, err := s.evaluate(&EvaluatorParams{
		Logger: logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "evaluator.implementation",
		}),
		Matches: matches,
	})
	if err != nil {
		return status.Error(codes.Aborted, err.Error())
	}

	for _, result := range results {
		if err := stream.Send(&gopb.EvaluateResponse{Match: result}); err != nil {
			return err
		}
	}

	logger.WithFields(logrus.Fields{
		"results": results,
	}).Debug("matches accepted by the evaluator")
	return nil
}
