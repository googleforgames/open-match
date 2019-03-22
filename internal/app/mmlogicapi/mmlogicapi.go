/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
${OM_ROOT}/internal/pb/mmlogic.pb.go

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
package mmlogicapi

import (
	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/app/mmlogicapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	"github.com/GoogleCloudPlatform/open-match/internal/signal"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	redigometrics "github.com/opencensus-integrations/redigo/redis"

	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
)

var (
	// Logrus structured logging setup
	mlLogFields = log.Fields{
		"app":       "openmatch",
		"component": "mmlogic",
	}
	mlLog = log.WithFields(mlLogFields)
)

func initializeApplication() (config.View, error) {
	// Logrus structured logging initialization
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(apisrv.MlLogLines, apisrv.KeySeverity))

	// Load configuration
	cfg, err := config.Read()
	if err != nil {
		mlLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
		return nil, err
	}

	if cfg.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel) // debug only, verbose - turn off in production!
		mlLog.Warn("Debug logging configured. Not recommended for production!")
	}

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := apisrv.DefaultMmlogicAPIViews                                   // Matchmaking logic API OpenCensus views.
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...)              // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)                    // config loader view.
	ocServerViews = append(ocServerViews, redigometrics.ObservabilityMetricViews...) // redis OpenCensus views.
	mlLog.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocServerViews)
	return cfg, nil
}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	cfg, err := initializeApplication()
	if err != nil {
		mlLog.Fatal(err)
	}

	// Connect to redis
	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		mlLog.Fatal(err)
	}
	defer pool.Close()

	// Instantiate the gRPC server with the connections we've made
	mlLog.Info("Attempting to start gRPC server")
	srv := apisrv.New(cfg, pool)

	// Run the gRPC server
	err = srv.Open()
	if err != nil {
		mlLog.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start gRPC server")
	}

	// Exit when we see a signal
	wait, _ := signal.New()
	wait()
	mlLog.Info("Shutting down gRPC server")
}
