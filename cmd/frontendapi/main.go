/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
frontendapi/proto/frontend.pb.go

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
	"time"

	"github.com/GoogleCloudPlatform/open-match/cmd/frontendapi/apisrv"
	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
)

var (
	// Logrus structured logging setup
	feLogFields = log.Fields{
		"app":       "openmatch",
		"component": "frontend",
		"caller":    "frontendapi/main.go",
	}
	feLog = log.WithFields(feLogFields)

	// Viper config management setup
	cfg = viper.New()
	err = errors.New("")
)

func init() {
	// Logrus structured logging initialization
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(apisrv.FeLogLines, apisrv.KeySeverity))

	// Viper config management initialization
	cfg, err = config.Read()
	if err != nil {
		feLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
	}

	if cfg.GetBool("debug") == true {
		log.SetLevel(log.DebugLevel) // debug only, verbose - turn off in production!
		feLog.Warn("Debug logging configured. Not recommended for production!")
	}

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := apisrv.DefaultFrontendAPIViews                     // FrontendAPI OpenCensus views.
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...) // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)       // config loader view.
	// Waiting on https://github.com/opencensus-integrations/redigo/pull/1
	// ocServerViews = append(ocServerViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	feLog.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocServerViews)
}

func main() {

	// Connect to redis
	pool := redisConnect(cfg)
	defer pool.Close()

	// Instantiate the gRPC server with the connections we've made
	feLog.WithFields(log.Fields{"testfield": "test"}).Info("Attempting to start gRPC server")
	srv := apisrv.New(cfg, pool)

	// Run the gRPC server
	err := srv.Open()
	if err != nil {
		feLog.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start gRPC server")
	}

	// Exit when we see a signal
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt)
	<-terminate
	feLog.Info("Shutting down gRPC server")
}

// redisConnect reads the configuration and attempts to instantiate a redis connection
// pool based on the configured hostname and port.
// TODO: needs to be reworked to use redis sentinel when we're ready to support it.
func redisConnect(cfg *viper.Viper) *redis.Pool {

	// As per https://www.iana.org/assignments/uri-schemes/prov/redis
	// redis://user:secret@localhost:6379/0?foo=bar&qux=baz
	redisURL := "redis://" + cfg.GetString("redis.hostname") + ":" + cfg.GetString("redis.port")
	// TODO: check if auth details are in the config, and append them if they
	// are.  Right now, assumes your redis instance is unsecured!

	feLog.WithFields(log.Fields{"redisURL": redisURL}).Info("Attempting to connect to Redis")
	pool := redis.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 60 * time.Second,
		Dial:        func() (redis.Conn, error) { return redis.DialURL(redisURL) },
	}

	feLog.Info("Connected to Redis")
	return &pool
}
