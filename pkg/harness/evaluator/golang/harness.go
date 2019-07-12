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
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"

	"github.com/sirupsen/logrus"
)

// RunEvaluator is a hook for the main() method in the main executable.
func RunEvaluator(eval Evaluator) {
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}

	p, err := rpc.NewServerParamsFromConfig(cfg, "api.evaluator")
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := BindService(p, cfg, eval); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("failed to bind evaluator service.")
	}

	rpc.MustServeForever(p)
}

// BindService creates the evaluator service to the server Params.
func BindService(p *rpc.ServerParams, cfg config.View, eval Evaluator) error {
	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterEvaluatorServer(s, &evaluatorService{cfg: cfg, evaluate: eval})
	}, pb.RegisterEvaluatorHandlerFromEndpoint)

	return nil
}
