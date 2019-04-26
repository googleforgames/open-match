/*
This application handles the scaffolding for the Evaluator. Currently this is
simply does some initializations for the components needed by the Evaluator.
In future, this application will initialize a GRPC server harness that Open
Match will call to synchronize evaluation.

Note that this method only has the initialization code for the evaluator. All
the actual important bits are in the API Server source code: apisrv/apisrv.go

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

package harness

import (
	"fmt"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/harness/evaluator/golang/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	redishelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
)

// RunEvaluator is a hook for the main() method in the evaluator executable.
func RunEvaluator(fn apisrv.EvaluateFunction) {
	evaluator, err := newEvaluator(fn)
	if err != nil {
		log.Errorf("Cannot construct the Evaluator, %v", err)
		return
	}

	evaluator.EvaluateForever()
}

// newEvaluator creates and initializes an Evaluator.
func newEvaluator(fn apisrv.EvaluateFunction) (*apisrv.Evaluator, error) {
	log.AddHook(metrics.NewHook(apisrv.EvaluatorLogLines, apisrv.KeySeverity))
	logger := log.WithFields(log.Fields{
		"app":       "openmatch",
		"component": "evaluator"})
	log.SetReportCaller(true)

	// Initialize the configuration
	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
		return nil, err
	}

	// Configure Open Match logging defaults
	logging.ConfigureLogging(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocEvaluatorViews := []*view.View{}
	ocEvaluatorViews = append(ocEvaluatorViews, apisrv.DefaultEvaluatorViews...)
	ocEvaluatorViews = append(ocEvaluatorViews, config.CfgVarCountView) // config loader view.

	logger.WithFields(log.Fields{"viewscount": len(ocEvaluatorViews)}).Info("Loaded OpenCensus views")

	promLh, err := netlistener.NewFromPortNumber(cfg.GetInt("metrics.port"))
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to create metrics TCP listener")
		return nil, err
	}
	metrics.ConfigureOpenCensusPrometheusExporter(promLh, cfg, ocEvaluatorViews)

	// Get the MMLogic client.
	mmlogic, err := getMMLogicClient(cfg)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Failed to get MMLogic client")
		return nil, err
	}

	// Get redis connection pool.
	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to connect to redis")
		return nil, err
	}

	evaluator := &apisrv.Evaluator{
		Logger:   logger,
		Config:   cfg,
		Pool:     pool,
		MMLogic:  mmlogic,
		Evaluate: fn,
	}

	return evaluator, nil
}

func getMMLogicClient(cfg config.View) (pb.MmLogicClient, error) {
	host := cfg.GetString("api.mmlogic.hostname")
	if len(host) == 0 {
		return nil, fmt.Errorf("Failed to get hostname for MMLogicAPI from the configuration")
	}

	port := cfg.GetString("api.mmlogic.port")
	if len(port) == 0 {
		return nil, fmt.Errorf("Failed to get port for MMLogicAPI from the configuration")
	}

	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", host, port), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %v, %v", fmt.Sprintf("%v:%v", host, port), err)
	}

	return pb.NewMmLogicClient(conn), nil
}
