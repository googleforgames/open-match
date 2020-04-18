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

// Taken from https://opencensus.io/quickstart/go/metrics/#1
import (
	ocPrometheus "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
)

const (
	// ConfigNameEnableMetrics indicates that telemetry is enabled.
	ConfigNameEnableMetrics = "telemetry.prometheus.enable"
)

func bindPrometheus(p Params, b Bindings) error {
	cfg := p.Config()

	if !cfg.GetBool("telemetry.prometheus.enable") {
		logger.Info("Prometheus Metrics: Disabled")
		return nil
	}

	endpoint := cfg.GetString("telemetry.prometheus.endpoint")

	logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Info("Prometheus Metrics: ENABLED")

	registry := prometheus.NewRegistry()
	// Register standard prometheus instrumentation.
	err := registry.Register(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	if err != nil {
		return errors.Wrap(err, "Failed to register prometheus collector")
	}
	err = registry.Register(prometheus.NewGoCollector())
	if err != nil {
		return errors.Wrap(err, "Failed to register prometheus collector")
	}

	promExporter, err := ocPrometheus.NewExporter(
		ocPrometheus.Options{
			Namespace: "",
			Registry:  registry,
		})
	if err != nil {
		return errors.Wrap(err, "Failed to initialize OpenCensus exporter to Prometheus")
	}

	// Register the Prometheus exporters as a stats exporter.
	view.RegisterExporter(promExporter)
	b.AddCloser(func() {
		view.UnregisterExporter(promExporter)
	})

	b.TelemetryHandle(endpoint, promExporter)
	return nil
}
