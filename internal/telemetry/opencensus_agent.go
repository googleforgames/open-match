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

package telemetry

import (
	"contrib.go.opencensus.io/exporter/ocagent"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func bindOpenCensusAgent(p Params, b Bindings) error {
	cfg := p.Config()

	if !cfg.GetBool("telemetry.opencensusAgent.enable") {
		logger.Info("OpenCensus Agent: Disabled")
		return nil
	}

	agentEndpoint := cfg.GetString("telemetry.opencensusAgent.agentEndpoint")
	logger.WithFields(logrus.Fields{
		"agentEndpoint": agentEndpoint,
	}).Info("OpenCensus Agent: ENABLED")

	oce, err := ocagent.NewExporter(ocagent.WithAddress(agentEndpoint), ocagent.WithInsecure(), ocagent.WithServiceName("open-match"))
	if err != nil {
		return errors.Wrap(err, "Failed to create a new ocagent exporter")
	}

	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)

	b.AddCloserErr(func() error {
		view.UnregisterExporter(oce)
		trace.UnregisterExporter(oce)
		// Before the program stops, please remember to stop the exporter.
		return oce.Stop()
	})

	return nil
}
