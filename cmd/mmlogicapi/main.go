/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
mmlogic/proto/mmlogic.pb.go

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
package main

import (
	"errors"
	"os"
	"os/signal"

	"github.com/GoogleCloudPlatform/open-match/cmd/mmlogicapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	redisHelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
)

var (
	// Logrus structured logging setup
	mlLogFields = log.Fields{
		"app":       "openmatch",
		"component": "mmlogic",
		"caller":    "mmlogicapi/main.go",
	}
	mlLog = log.WithFields(mlLogFields)

	// Viper config management setup
	cfg = viper.New()
	err = errors.New("")
)

func init() {
	// Logrus structured logging initialization
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(apisrv.MlLogLines, apisrv.KeySeverity))

	// Viper config management initialization
	cfg, err = config.Read()
	if err != nil {
		mlLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
	}

	if cfg.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel) // debug only, verbose - turn off in production!
		mlLog.Warn("Debug logging configured. Not recommended for production!")
	}

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := apisrv.DefaultMmlogicAPIViews                      // Matchmaking logic API OpenCensus views.
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...) // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)       // config loader view.
	// Waiting on https://github.com/opencensus-integrations/redigo/pull/1
	// ocServerViews = append(ocServerViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	mlLog.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocServerViews)
}

func main() {

	// Connect to redis
	pool := redisHelpers.ConnectionPool(cfg)
	defer pool.Close()

	// Instantiate the gRPC server with the connections we've made
	mlLog.Info("Attempting to start gRPC server")
	srv := apisrv.New(cfg, pool)

	// Run the gRPC server
	err := srv.Open()
	if err != nil {
		mlLog.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start gRPC server")
	}

	// Exit when we see a signal
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	mlLog.Info("Shutting down gRPC server")
}
