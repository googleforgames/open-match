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
	"net/http"

	ocPrometheus "contrib.go.opencensus.io/exporter/prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"open-match.dev/open-match/internal/config"
)

const (
	// ConfigNameEnableMetrics indicates that telemetry is enabled.
	ConfigNameEnableMetrics = "telemetry.prometheus.enable"
)

func bindPrometheus(mux *http.ServeMux, cfg config.View) {
	if !cfg.GetBool("telemetry.prometheus.enable") {
		logger.Info("Prometheus Metrics: Disabled")
		return
	}

	endpoint := cfg.GetString("telemetry.prometheus.endpoint")
	registry := prometheus.NewRegistry()
	// Register standard prometheus instrumentation.
	registry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	registry.MustRegister(prometheus.NewGoCollector())
	promExporter, err := ocPrometheus.NewExporter(
		ocPrometheus.Options{
			Namespace: "",
			Registry:  registry,
		})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":    err,
			"endpoint": endpoint,
		}).Fatal(
			"Failed to initialize OpenCensus exporter to Prometheus")
	}

	// Register the Prometheus exporters as a stats exporter.
	view.RegisterExporter(promExporter)
	if err := view.Register(DefaultViews...); err != nil {
		logger.WithError(err).Fatalf("failed to register given views to exporters")
	}

	mux.Handle(endpoint, promExporter)

	logger.WithFields(logrus.Fields{
		"endpoint": endpoint,
	}).Info("Prometheus Metrics: ENABLED")
}
