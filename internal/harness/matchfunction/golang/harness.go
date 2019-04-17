/*
This application handles all the startup and connection scaffolding for
running a gRPC server serving the APIService as defined in
${OM_ROOT}/internal/pb/matchfunction.pb.go

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

package harness

import (
	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/harness/matchfunction/golang/apisrv"
	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	"github.com/GoogleCloudPlatform/open-match/internal/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/serving"
	"github.com/GoogleCloudPlatform/open-match/internal/signal"
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"

	log "github.com/sirupsen/logrus"
)

// HarnessParams is a collection of parameters used to create a MatchFunction server.
type HarnessParams struct {
	FunctionName   string
	ServicePortConfigName string
	ProxyPortConfigName string
	Func           apisrv.MatchFunction
}

// ServeMatchFunction is a hook for the main() method in the main executable.
func ServeMatchFunction(params *HarnessParams) {
	mfServer, err := newMatchFunctionServer(params)
	if err != nil {
		log.WithFields(log.Fields{
			"error": err.Error(),
		}).Info("Cannot construct the match function server.")
		return
	}

	// Instantiate the gRPC server with the bindings we've made.
	logger := mfServer.Logger
	grpcLh, err := netlistener.NewFromPortNumber(mfServer.Config.GetInt(params.ServicePortConfigName))
	proxyLh, err := netlistener.NewFromPortNumber(mfServer.Config.GetInt(params.ProxyPortConfigName))
	if err != nil {
		logger.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to create a TCP listener for the GRPC server")
		return
	}

	grpcServer := serving.NewGrpcServer(grpcLh, proxyLh, logger)
	grpcServer.AddService(func(server *grpc.Server) {
		pb.RegisterMatchFunctionServer(server, mfServer)
	})

	defer func() {
		err := grpcServer.Stop()
		if err != nil {
			logger.WithFields(log.Fields{"error": err.Error()}).Infof("Server shutdown error, %s.", err)
		}
	}()

	// Start serving traffic.
	err = grpcServer.Start()
	if err != nil {
		logger.WithFields(log.Fields{"error": err.Error()}).Fatal("Failed to start server")
	}

	// Exit when we see a signal
	wait, _ := signal.New()
	wait()
	logger.Info("Shutting down server")
}

// newMatchFunctionServer creates a MatchFunctionServer based on the harness parameters.
func newMatchFunctionServer(params *HarnessParams) (*apisrv.MatchFunctionServer, error) {
	log.AddHook(metrics.NewHook(apisrv.FnLogLines, apisrv.KeySeverity))
	logger := log.WithFields(log.Fields{
		"app":       "openmatch",
		"component": "matchfunction_service",
		"function":  params.FunctionName})

	// Add a hook to the logger to log the filename & line number.
	log.SetReportCaller(true)

	cfg, err := config.Read()
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
		return nil, err
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := []*view.View{}
	ocServerViews = append(ocServerViews, apisrv.DefaultFunctionViews...)
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...) // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)       // config loader view.
	logger.WithFields(log.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")

	promLh, err := netlistener.NewFromPortNumber(cfg.GetInt("metrics.port"))
	if err != nil {
		logger.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to create metrics TCP listener")
		return nil, err
	}
	metrics.ConfigureOpenCensusPrometheusExporter(promLh, cfg, ocServerViews)

	mfServer := &apisrv.MatchFunctionServer{
		Logger: logger,
		Config: cfg,
		Func:   params.Func,
	}

	return mfServer, nil
}
