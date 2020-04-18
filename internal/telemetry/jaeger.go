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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/trace"
)

func bindJaeger(p Params, b Bindings) error {
	cfg := p.Config()

	if !cfg.GetBool("telemetry.jaeger.enable") {
		logger.Info("Jaeger Tracing: Disabled")
		return nil
	}

	agentEndpointURI := cfg.GetString("telemetry.jaeger.agentEndpoint")
	collectorEndpointURI := cfg.GetString("telemetry.jaeger.collectorEndpoint")
	serviceName := p.ServiceName()

	logger.WithFields(logrus.Fields{
		"agentEndpoint":     agentEndpointURI,
		"collectorEndpoint": collectorEndpointURI,
		"serviceName":       serviceName,
	}).Info("Jaeger Tracing: ENABLED")

	je, err := jaeger.NewExporter(jaeger.Options{
		AgentEndpoint:     agentEndpointURI,
		CollectorEndpoint: collectorEndpointURI,
		ServiceName:       serviceName,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to create the Jaeger exporter")
	}

	trace.RegisterExporter(je)
	b.AddCloser(func() {
		trace.UnregisterExporter(je)
	})

	return nil
}
