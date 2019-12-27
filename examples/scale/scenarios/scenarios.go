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

package scenarios

import "open-match.dev/open-match/pkg/pb"

// TODO:
// - add images for scale-mmf and scale-evaluator, have this use it.
// - add an evaluator.
// - add ticket generation function and profiles to the scenario, can just pull from existing
//     packages for now

// after that start on getting metrics to show up from the scale-backend and scale-frontend.

var ActiveScenario = basicScenario

type matchFunction func(*pb.RunRequest, pb.MatchFunction_RunServer) error
type evaluatorFunction func(pb.Evaluator_EvaluateServer) error

// Scenario defines the controllable fields for Open Match benchmark scenarios
type Scenario struct {
	// MatchFunction Configs
	// MatchOverlapRatio          float32
	// TicketSearchFieldsUnitSize int
	// TicketSearchFieldsNumber   int

	// GameFrontend Configs
	// TicketExtensionSize       int
	// PendingTicketNumber       int
	// MatchExtensionSize        int
	// ShouldCreateTicketForever bool
	// TicketCreatedQPS          int

	// GameBackend Configs
	// ProfileNumber      int
	// FilterNumber       int
	// ShouldAssignTicket bool
	// ShouldDeleteTicket bool

	MMF       matchFunction
	Evaluator evaluatorFunction
}

func (mmf matchFunction) Run(req *pb.RunRequest, srv pb.MatchFunction_RunServer) error {
	return mmf(req, srv)
}

func (eval evaluatorFunction) Evaluate(srv pb.Evaluator_EvaluateServer) error {
	return eval(srv)
}
