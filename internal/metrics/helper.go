/*
Package metrics is an internal package that provides helper functions for
OpenCensus instrumentation in Open Match.

Copyright 2018 Google LLC
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package metrics

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

var (
	metricsLogFields = log.Fields{
		"app":       "openmatch",
		"component": "metrics",
	}
	mhLog = log.WithFields(metricsLogFields)
)

// ConfigureOpenCensusPrometheusExporter reads from the provided viper
// config's 'metrics' section to set up a metrics endpoint that can be scraped
// by Promethus for  metrics gathering. The calling code can select any views
// it wants to  register, from any number of libraries, and pass them in as an
// array.
func ConfigureOpenCensusPrometheusExporter(cfg *viper.Viper, views []*view.View) {

	//var infoCtx, err = tag.New(context.Background(), tag.Insert(KeySeverity, "info"))
	metricsPort := cfg.GetInt("metrics.port")
	metricsEP := cfg.GetString("metrics.endpoint")
	metricsRP := cfg.GetInt("metrics.reportingPeriod")

	// Set OpenCensus to export to Prometheus
	pe, err := prometheus.NewExporter(prometheus.Options{Namespace: "open_match"})
	if err != nil {
		mhLog.WithFields(log.Fields{"error": err}).Fatal(
			"Failed to initialize OpenCensus exporter to Prometheus")
	}
	mhLog.Info("OpenCensus exporter to Promethus initialized")
	view.RegisterExporter(pe)

	// Register the OpenCensus views we want to export to Prometheus
	err = view.Register(views...)
	if err != nil {
		mhLog.Fatalf("Failed to register OpenCensus views for metrics gathering: %v", err)
	}
	mhLog.Info("Opencensus views registered")

	// Change the frequency of updates to the metrics endpoint
	view.SetReportingPeriod(time.Duration(metricsRP) * time.Second)
	mhLog.WithFields(log.Fields{
		"port":            metricsPort,
		"endpoint":        metricsEP,
		"retentionPeriod": metricsRP,
	}).Info("Opencensus measurement serving to Prometheus configured")

	// Start serving the /metrics http endpoint for Prometheus
	go func() {
		mux := http.NewServeMux()
		mux.Handle(metricsEP, pe)
		mhLog.WithFields(log.Fields{
			"port":     metricsPort,
			"endpoint": metricsEP,
		}).Info("Attempting to start http server for OpenCensus metrics on localhost")
		err := http.ListenAndServe(":"+strconv.Itoa(metricsPort), mux)
		if err != nil {
			mhLog.WithFields(log.Fields{
				"error":    err,
				"port":     metricsPort,
				"endpoint": metricsEP,
			}).Fatal("Failed to run Prometheus endpoint")
		}
		mhLog.WithFields(log.Fields{
			"port":     metricsPort,
			"endpoint": metricsEP,
		}).Info("Successfully started http server for OpenCensus metrics on localhost")
	}()
}

// Hook is a log hook that for counting log lines using OpenCensus.
type Hook struct {
	count       *stats.Int64Measure
	keySeverity tag.Key
}

// NewHook returns a new log.Hook for counting log lines using OpenCensus.
func NewHook(m *stats.Int64Measure, ks tag.Key) log.Hook {
	return Hook{
		count:       m,
		keySeverity: ks,
	}
}

// Fire is run every time log logs a line and increments the OpenCensus counter
func (h Hook) Fire(e *log.Entry) error {
	// get the entry's log level, and tag this opencensus counter increment with that level
	ctx, err := tag.New(context.Background(), tag.Insert(h.keySeverity, e.Level.String()))
	if err != nil {
		fmt.Println(err)
	}
	stats.Record(ctx, h.count.M(1))
	return err
}

// Levels returns all log levels, because we want the hook to always fire when a line is logged.
func (h Hook) Levels() []log.Level {
	return log.AllLevels
}
