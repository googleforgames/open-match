/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
${OM_ROOT}/internal/pb/backend.pb.go

All the actual important bits are in the API Server source code: apisrv/apisrv.go

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
package backendapi

import (
	"errors"
	"os"
	"os/signal"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/app/backendapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
)

var (
	// Logrus structured logging setup
	beLogFields = log.Fields{
		"app":       "openmatch",
		"component": "backend",
	}
	beLog = log.WithFields(beLogFields)

	// Viper config management setup
	cfg = viper.New()
	err = errors.New("")
)

func initializeApplication() {
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(apisrv.BeLogLines, apisrv.KeySeverity))

	// Viper config management initialization
	cfg, err = config.Read()
	if err != nil {
		beLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := apisrv.DefaultBackendAPIViews                      // BackendAPI OpenCensus views.
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...) // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)       // config loader view.
	// Waiting on https://github.com/opencensus-integrations/redigo/pull/1
	// ocServerViews = append(ocServerViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	beLog.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocServerViews)
}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	initializeApplication()
	
	// Connect to redis
	pool := redishelpers.ConnectionPool(cfg)
	defer pool.Close()

	// Instantiate the gRPC server with the connections we've made
	beLog.Info("Attempting to start gRPC server")
	srv := apisrv.New(cfg, pool)

	// Run the gRPC server
	err := srv.Open()
	if err != nil {
		beLog.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start gRPC server")
	}

	// Exit when we see a signal
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	beLog.Info("Shutting down gRPC server")
}
