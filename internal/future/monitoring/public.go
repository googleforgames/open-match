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
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"go.opencensus.io/stats/view"
	"open-match.dev/open-match/internal/config"
)

var (
	publicLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "monitoring",
	})
)

// Setup configures the monitoring for the server.
func Setup(mux *http.ServeMux, cfg config.View) {
	periodString := cfg.GetString("monitoring.reportingPeriod")
	reportingPeriod, err := time.ParseDuration(periodString)
	if err != nil {
		publicLogger.WithFields(logrus.Fields{
			"error":           err,
			"retentionPeriod": periodString,
		}).Info("Failed to parse monitoring.reportingPeriod, defaulting to 10s")
		reportingPeriod = time.Second * 10
	}

	bindJaeger(mux, cfg)
	bindPrometheus(mux, cfg)
	bindStackDriver(mux, cfg)
	bindZipkin(mux, cfg)
	bindZpages(mux, cfg)

	// Change the frequency of updates to the metrics endpoint
	view.SetReportingPeriod(reportingPeriod)

	// Register the OpenCensus views we want to export to Prometheus
	/*
		err = view.Register(views...)
		if err != nil {
			publicLogger.Fatalf("Failed to register OpenCensus views for metrics gathering: %v", err)
		}
	*/

	publicLogger.WithFields(logrus.Fields{
		"retentionPeriod": reportingPeriod,
	}).Info("Monitoring has been configured.")
}
