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
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/ipb"
	"open-match.dev/open-match/internal/statestore"
	"open-match.dev/open-match/internal/telemetry"
)

var (
	iterationLatency        = stats.Float64("open-match.dev/synchronizer/iteration_latency", "Time elapsed of each synchronizer iteration", stats.UnitMilliseconds)
	registrationWaitTime    = stats.Float64("open-match.dev/synchronizer/registration_wait_time", "Time elapsed of registration wait time", stats.UnitMilliseconds)
	registrationMMFDoneTime = stats.Float64("open-match.dev/synchronizer/registration_mmf_done_time", "Time elapsed wasted in registration window with done MMFs", stats.UnitMilliseconds)

	iterationLatencyView = &view.View{
		Measure:     iterationLatency,
		Name:        "open-match.dev/synchronizer/iteration_latency",
		Description: "Time elapsed of each synchronizer iteration",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
	registrationWaitTimeView = &view.View{
		Measure:     registrationWaitTime,
		Name:        "open-match.dev/synchronizer/registration_wait_time",
		Description: "Time elapsed of registration wait time",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
	registrationMMFDoneTimeView = &view.View{
		Measure:     registrationMMFDoneTime,
		Name:        "open-match.dev/synchronizer/registration_mmf_done_time",
		Description: "Time elapsed wasted in registration window with done MMFs",
		Aggregation: telemetry.DefaultMillisecondsDistribution,
	}
)

// BindService creates the synchronizer service and binds it to the serving harness.
func BindService(p *appmain.Params, b *appmain.Bindings) error {
	store := statestore.New(p.Config())
	service := newSynchronizerService(p.Config(), newEvaluator(p.Config()), store)
	b.AddHealthCheckFunc(store.HealthCheck)
	b.AddHandleFunc(func(s *grpc.Server) {
		ipb.RegisterSynchronizerServer(s, service)
	}, nil)
	b.RegisterViews(
		iterationLatencyView,
		registrationWaitTimeView,
		registrationMMFDoneTimeView,
	)
	return nil
}
