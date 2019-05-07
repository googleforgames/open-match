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

package monitoring

import (
	"contrib.go.opencensus.io/exporter/jaeger"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/internal/config"
)

var (
	jaegerLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "monitoring.jaeger",
	})
)

func bindJaeger(cfg config.View) {
	if !cfg.GetBool("monitoring.jaeger.enable") {
		jaegerLogger.Info("Jaeger Tracing: Disabled")
		return
	}

	agentEndpointURI := cfg.GetString("monitoring.jaeger.agentEndpoint")
	collectorEndpointURI := cfg.GetString("monitoring.jaeger.collectorEndpoint")

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     agentEndpointURI,
		CollectorEndpoint: collectorEndpointURI,
		ServiceName:       "open_match",
	})
	if err != nil {
		jaegerLogger.WithFields(logrus.Fields{
			"error":             err,
			"agentEndpoint":     agentEndpointURI,
			"collectorEndpoint": collectorEndpointURI,
		}).Fatalf(
			"Failed to create the Jaeger exporter: %v", err)
	}

	jaegerLogger.WithFields(logrus.Fields{
		"agentEndpoint":     agentEndpointURI,
		"collectorEndpoint": collectorEndpointURI,
	}).Info("Jaeger Tracing: ENABLED")
	// And now finally register it as a Trace Exporter
	trace.RegisterExporter(je)
}
