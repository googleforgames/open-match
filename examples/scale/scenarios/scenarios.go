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

// ActiveScenario sets the scenario with preset parameters that we want to use for current Open Match benchmark run.
var ActiveScenario = battleRoyalScenario

// Scenario defines the controllable fields for Open Match benchmark scenarios
type Scenario struct {
	// TODO: supports the following controllable parameters

	// MatchFunction Configs
	// MatchOverlapRatio          float32
	// TicketSearchFieldsUnitSize int
	// TicketSearchFieldsNumber   int

	// GameFrontend Configs
	// TicketExtensionSize       int
	// PendingTicketNumber       int
	// MatchExtensionSize        int
	FrontendTotalTicketsToCreate int // TotalTicketsToCreate = -1 let scale-frontend create tickets forever
	FrontendTicketCreatedQPS     uint32

	// GameBackend Configs
	// ProfileNumber      int
	// FilterNumber       int
	BackendAssignsTickets bool
	BackendDeletesTickets bool

	Ticket   func() *pb.Ticket
	Profiles func() []*pb.MatchProfile

	MMF       matchFunction
	Evaluator evaluatorFunction
}

type matchFunction func(*pb.RunRequest, pb.MatchFunction_RunServer) error
type evaluatorFunction func(pb.Evaluator_EvaluateServer) error

func (mmf matchFunction) Run(req *pb.RunRequest, srv pb.MatchFunction_RunServer) error {
	return mmf(req, srv)
}

func (eval evaluatorFunction) Evaluate(srv pb.Evaluator_EvaluateServer) error {
	return eval(srv)
}
