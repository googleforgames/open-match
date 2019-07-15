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
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/internal/config"
)

func bindStackDriver(cfg config.View) func() {
	if !cfg.GetBool("telemetry.stackdriver.enable") {
		logger.Info("StackDriver Metrics: Disabled")
		return func() {}
	}
	gcpProjectID := cfg.GetString("telemetry.stackdriver.gcpProjectId")
	metricPrefix := cfg.GetString("telemetry.stackdriver.metricPrefix")
	sd, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: gcpProjectID,
		// MetricPrefix helps uniquely identify your metrics.
		MetricPrefix: metricPrefix,
	})
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":        err,
			"gcpProjectID": gcpProjectID,
			"metricPrefix": metricPrefix,
		}).Fatal("Failed to initialize OpenCensus exporter to Stack Driver")
	}

	// Register it as a metrics exporter
	view.RegisterExporter(sd)

	// Register it as a trace exporter
	trace.RegisterExporter(sd)

	logger.WithFields(logrus.Fields{
		"gcpProjectID": gcpProjectID,
		"metricPrefix": metricPrefix,
	}).Info("StackDriver Metrics: ENABLED")

	// It is imperative to invoke flush before your main function exits
	return sd.Flush
}
