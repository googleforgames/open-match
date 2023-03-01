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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/pkg/pb"
)

var (
	matchesPerEvaluateRequest  = stats.Int64("open-match.dev/evaluator/matches_per_request", "Number of matches sent to the evaluator per request", stats.UnitDimensionless)
	matchesPerEvaluateResponse = stats.Int64("open-match.dev/evaluator/matches_per_response", "Number of matches returned by the evaluator per response", stats.UnitDimensionless)

	matchesPerEvaluateRequestView = &view.View{
		Measure:     matchesPerEvaluateRequest,
		Name:        "open-match.dev/evaluator/matches_per_request",
		Description: "Number of matches sent to the evaluator per request",
		Aggregation: telemetry.DefaultCountDistribution,
	}
	matchesPerEvaluateResponseView = &view.View{
		Measure:     matchesPerEvaluateResponse,
		Name:        "open-match.dev/evaluator/matches_per_response",
		Description: "Number of matches sent to the evaluator per response",
		Aggregation: telemetry.DefaultCountDistribution,
	}
)

// BindServiceFor creates the evaluator service and binds it to the serving harness.
func BindServiceFor(eval Evaluator) appmain.Bind {
	return func(p *appmain.Params, b *appmain.Bindings) error {
		b.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterEvaluatorServer(s, &evaluatorService{evaluate: eval})
		}, pb.RegisterEvaluatorHandlerFromEndpoint)
		b.RegisterViews(
			matchesPerEvaluateRequestView,
			matchesPerEvaluateResponseView,
		)
		return nil
	}
}
