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

// Package evaluator provides the Evaluator service for Open Match golang harness.
package evaluator

import (
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

func BindServiceFor(eval Evaluator) func(p *rpc.ServerParams, cfg config.View) error {
	return func(p *rpc.ServerParams, cfg config.View) error {
		p.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterEvaluatorServer(s, &evaluatorService{evaluate: eval})
		}, pb.RegisterEvaluatorHandlerFromEndpoint)

		return nil
	}
}
