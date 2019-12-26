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

import (

	// "sync"

	"github.com/sirupsen/logrus"
)

// TODO:
// - add images for scale-mmf and scale-evaluator, have this use it.
// - add an evaluator.
// - add ticket generation function and profiles to the scenario, can just pull from existing
//     packages for now

// after that start on getting metrics to show up from the scale-backend and scale-frontend.

var ActiveScenario = BasicScenario{Scenario{
	MmfServerPort:              50502,
	MatchOverlapRatio:          0,
	TicketSearchFieldsUnitSize: 0,
	TicketSearchFieldsNumber:   1,
	MmlogicAddr:                "om-mmlogic.open-match.svc.cluster.local:50503",
	EvaluatorServerPort:        50508,
	TicketExtensionSize:        0,
	PendingTicketNumber:        10000,
	MatchExtensionSize:         0,
	ShouldCreateTicketForever:  false,
	TicketQPS:                  100,
	ProfileNumber:              20,
	FilterNumber:               20,
	ShouldAssignTicket:         true,
	ShouldDeleteTicket:         true,
	Logger: logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "scale",
	}),
}}

// Scenario defines the controllable fields for Open Match benchmark scenarios
type Scenario struct {
	// MatchFunction Configs
	MmfServerPort              int32
	MatchOverlapRatio          float32
	TicketSearchFieldsUnitSize int
	TicketSearchFieldsNumber   int
	MmlogicAddr                string

	// Evaluator Configs
	EvaluatorServerPort int32

	// GameFrontend Configs
	TicketExtensionSize       int
	PendingTicketNumber       int
	MatchExtensionSize        int
	ShouldCreateTicketForever bool
	TicketQPS                 int

	// GameBackend Configs
	ProfileNumber      int
	FilterNumber       int
	ShouldAssignTicket bool
	ShouldDeleteTicket bool

	Logger *logrus.Entry
}
