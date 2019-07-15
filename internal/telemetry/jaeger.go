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
	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/internal/config"
)

func bindJaeger(cfg config.View) {
	if !cfg.GetBool("telemetry.jaeger.enable") {
		logger.Info("Jaeger Tracing: Disabled")
		return
	}

	agentEndpointURI := cfg.GetString("telemetry.jaeger.agentEndpoint")
	collectorEndpointURI := cfg.GetString("telemetry.jaeger.collectorEndpoint")

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     agentEndpointURI,
		CollectorEndpoint: collectorEndpointURI,
		ServiceName:       "open_match",
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":             err,
			"agentEndpoint":     agentEndpointURI,
			"collectorEndpoint": collectorEndpointURI,
		}).Fatalf(
			"Failed to create the Jaeger exporter: %v", err)
	}

	// And now finally register it as a Trace Exporter
	trace.RegisterExporter(je)

	logger.WithFields(logrus.Fields{
		"agentEndpoint":     agentEndpointURI,
		"collectorEndpoint": collectorEndpointURI,
	}).Info("Jaeger Tracing: ENABLED")
}
