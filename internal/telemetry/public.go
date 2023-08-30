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
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"open-match.dev/open-match/internal/config"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "telemetry",
	})
)

// Setup configures the telemetry for the server.
func Setup(p Params, b Bindings) error {
	bindings := []func(p Params, b Bindings) error{
		configureOpenCensus,
		bindPrometheus,
		bindStackDriverMetrics,
		bindOpenCensusAgent,
		bindZpages,
		bindHelp,
		bindConfigz,
	}

	for _, f := range bindings {
		err := f(p, b)
		if err != nil {
			return err
		}
	}

	return nil
}

func configureOpenCensus(p Params, b Bindings) error {
	// There's no way to undo these options, but the next startup will override
	// them.

	samplingFraction := p.Config().GetFloat64("telemetry.traceSamplingFraction")
	logger.WithFields(logrus.Fields{
		"samplingFraction": samplingFraction,
	}).Info("Tracing sampler fraction set")
	trace.ApplyConfig(trace.Config{DefaultSampler: trace.ProbabilitySampler(samplingFraction)})

	periodString := p.Config().GetString("telemetry.reportingPeriod")
	reportingPeriod, err := time.ParseDuration(periodString)
	if err != nil {
		return errors.Wrap(err, "Unable to parse telemetry.reportingPeriod")
	}
	logger.WithFields(logrus.Fields{
		"reportingPeriod": reportingPeriod,
	}).Info("Telemetry reporting period set")
	// Change the frequency of updates to the metrics endpoint
	view.SetReportingPeriod(reportingPeriod)
	return nil
}

// Params allows appmain to bind telemetry without a circular dependency.
type Params interface {
	Config() config.View
	ServiceName() string
}

// Bindings allows appmain to bind telemetry without a circular dependency.
type Bindings interface {
	TelemetryHandle(pattern string, handler http.Handler)
	TelemetryHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
	AddCloser(c func())
	AddCloserErr(c func() error)
}
