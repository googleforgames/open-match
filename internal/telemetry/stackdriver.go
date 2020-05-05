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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

func bindStackDriverMetrics(p Params, b Bindings) error {
	cfg := p.Config()

	if !cfg.GetBool("telemetry.stackdriverMetrics.enable") {
		logger.Info("StackDriver Metrics: Disabled")
		return nil
	}
	gcpProjectID := cfg.GetString("telemetry.stackdriverMetrics.gcpProjectId")
	metricPrefix := cfg.GetString("telemetry.stackdriverMetrics.prefix")

	logger.WithFields(logrus.Fields{
		"gcpProjectID": gcpProjectID,
		"metricPrefix": metricPrefix,
	}).Info("StackDriver Metrics: ENABLED")

	sd, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: gcpProjectID,
		// MetricPrefix helps uniquely identify your metrics.
		MetricPrefix: metricPrefix,
	})
	if err != nil {
		return errors.Wrap(err, "Failed to initialize OpenCensus exporter to Stack Driver")
	}

	view.RegisterExporter(sd)
	trace.RegisterExporter(sd)

	b.AddCloser(func() {
		view.UnregisterExporter(sd)
		trace.UnregisterExporter(sd)
		// It is imperative to invoke flush before your main function exits
		sd.Flush()
	})

	return nil
}
