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

	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/util"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "telemetry",
	})
)

// Setup configures the telemetry for the server.
func Setup(servicePrefix string, mux *http.ServeMux, cfg config.View) func() {
	mc := util.NewMultiClose()
	periodString := cfg.GetString("telemetry.reportingPeriod")
	reportingPeriod, err := time.ParseDuration(periodString)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error":           err,
			"reportingPeriod": periodString,
		}).Info("Failed to parse telemetry.reportingPeriod, defaulting to 1m")
		reportingPeriod = time.Minute * 1
	}

	bindJaeger(servicePrefix, cfg)
	bindPrometheus(mux, cfg)
	mc.AddCloseFunc(bindStackDriverMetrics(cfg))
	mc.AddCloseWithErrorFunc(bindOpenCensusAgent(cfg))
	bindZpages(mux, cfg)
	bindHelp(mux, cfg)
	bindConfigz(mux, cfg)

	// Change the frequency of updates to the metrics endpoint
	view.SetReportingPeriod(reportingPeriod)

	logger.WithFields(logrus.Fields{
		"reportingPeriod": reportingPeriod,
	}).Info("telemetry has been configured.")
	return mc.Close
}
