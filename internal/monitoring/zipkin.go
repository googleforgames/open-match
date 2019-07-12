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

//https://opencensus.io/quickstart/go/tracing/

import (
	"contrib.go.opencensus.io/exporter/zipkin"
	"go.opencensus.io/trace"

	openzipkin "github.com/openzipkin/zipkin-go"
	zipkinHTTP "github.com/openzipkin/zipkin-go/reporter/http"
	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
)

const (
	configNameZipkinEnable           = "monitoring.zipkin.enable"
	configNameZipkinEndpoint         = "monitoring.zipkin.endpoint"
	configNameZipkinReporterEndpoint = "monitoring.zipkin.reporterEndpoint"
)

var (
	zipkinLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "monitoring.zipkin",
	})
)

func bindZipkin(cfg config.View) {
	if !cfg.GetBool(configNameZipkinEnable) {
		zipkinLogger.Info("Zipkin Tracing: Disabled")
		return
	}
	zipkinEndpoint := cfg.GetString(configNameZipkinEndpoint)
	zipkinReporterEndpoint := cfg.GetString(configNameZipkinReporterEndpoint)
	// 1. Configure exporter to export traces to Zipkin.
	localEndpoint, err := openzipkin.NewEndpoint("open_match", zipkinEndpoint)
	if err != nil {
		zipkinLogger.WithFields(logrus.Fields{
			"error":                  err,
			"zipkinEndpoint":         zipkinEndpoint,
			"zipkinReporterEndpoint": zipkinReporterEndpoint,
		}).Fatal(
			"Failed to initialize OpenCensus exporter to Zipkin")
	}
	reporter := zipkinHTTP.NewReporter(zipkinReporterEndpoint)
	ze := zipkin.NewExporter(reporter, localEndpoint)
	trace.RegisterExporter(ze)

	// TODO: Provide a basic configuration for Zipkin trace samples.
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})

	zipkinLogger.WithFields(logrus.Fields{
		"zipkinEndpoint":         zipkinEndpoint,
		"zipkinReporterEndpoint": zipkinReporterEndpoint,
	}).Info("Zipkin Tracing: ENABLED")
}
