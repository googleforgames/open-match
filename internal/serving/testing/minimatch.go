/*
Copyright 2019 Google LLC

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

// Package testing provides test helpers for the serving package.
package testing

import (
	"fmt"
	"net/http"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/opencensus-integrations/redigo/redis"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/metrics"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/serving"
	redishelpers "open-match.dev/open-match/internal/statestorage/redis"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
)

// MiniMatchServer is an OpenMatchServer with additional context for testing.
type MiniMatchServer struct {
	*serving.OpenMatchServer
	mRedis *miniredis.Miniredis
}

// GetFrontendClient gets the frontend client.
func (mm *MiniMatchServer) GetFrontendClient() (pb.FrontendClient, error) {
	port := mm.Config.GetInt("api.frontend.port")
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewFrontendClient(conn), nil
}

// GetFrontendProxyClient gets the REST proxy of the frontend client
func (mm *MiniMatchServer) GetFrontendProxyClient() (*http.Client, string) {
	httpClient := &http.Client{
		Timeout: time.Second * 3,
	}

	baseURL := fmt.Sprintf("http://localhost:%d", mm.Config.GetInt("api.frontend.proxyport"))

	return httpClient, baseURL
}

// GetBackendClient gets the backend client.
func (mm *MiniMatchServer) GetBackendClient() (pb.BackendClient, error) {
	port := mm.Config.GetInt("api.backend.port")
	conn, err := grpc.Dial(fmt.Sprintf(":%d", port), grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return pb.NewBackendClient(conn), nil
}

// GetBackendProxyClient gets the REST proxy of the backend client
func (mm *MiniMatchServer) GetBackendProxyClient() (*http.Client, string) {
	httpClient := &http.Client{
		Timeout: time.Second * 3,
	}

	baseURL := fmt.Sprintf("http://localhost:%d", mm.Config.GetInt("api.backend.proxyport"))

	return httpClient, baseURL
}

// Stop shuts down Mini Match
func (mm *MiniMatchServer) Stop() {
	err := mm.OpenMatchServer.Stop()
	if err != nil {
		mm.Logger.WithFields(logrus.Fields{"error": err.Error()}).Debug("Failed to stop server")
	}
	mm.mRedis.Close()
}

// MustMiniMatch requires Mini Match to be created successfully.
func MustMiniMatch(params []*serving.ServerParams) (*MiniMatchServer, func()) {
	mm, closer, err := NewMiniMatch(params)
	if err != nil {
		panic(err)
	}
	return mm, closer
}

// NewMiniMatch creates and starts an OpenMatchServer context for testing.
func NewMiniMatch(params []*serving.ServerParams) (*MiniMatchServer, func(), error) {
	mm, err := createOpenMatchServer(params)
	if err != nil {
		return nil, func() {}, err
	}
	logger := mm.Logger
	// Start serving traffic.
	err = mm.Start()
	if err != nil {
		logger.WithFields(logrus.Fields{"error": err.Error()}).Fatal("Failed to start server")
		return nil, func() {}, err
	}
	closer := func() {
		mm.Stop()
	}
	return mm, closer, nil
}

func createOpenMatchServer(paramsList []*serving.ServerParams) (*MiniMatchServer, error) {
	logger := logrus.WithFields(paramsList[0].BaseLogFields)

	cfg := viper.New()
	cfg.Set("logging.level", "debug")
	cfg.Set("logging.format", "text")
	// TODO: Re-enable once, https://github.com/sirupsen/logrus/issues/954 is fixed.
	cfg.Set("logging.source", false)

	promListener := netlistenerTesting.MustListen()

	cfg.Set("metrics.port", promListener.Number())
	cfg.Set("metrics.endpoint", "/metrics")
	cfg.Set("metrics.reportingPeriod", "5s")

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocServerViews := []*view.View{}
	/* TODO: Views are conflicting, so not loading them.
	for _, params := range paramsList {
		ocServerViews = append(ocServerViews, params.CustomMeasureViews...)
	}
	*/
	ocServerViews = append(ocServerViews, ocgrpc.DefaultServerViews...)      // gRPC OpenCensus views.
	ocServerViews = append(ocServerViews, config.CfgVarCountView)            // config loader view.
	ocServerViews = append(ocServerViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	logger.WithFields(logrus.Fields{"viewscount": len(ocServerViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(promListener, cfg, ocServerViews)

	// Connect to redis
	mredis, err := miniredis.Run()
	if err != nil {
		return nil, err
	}

	// TODO: Clean this up so that we can deterministically close Redis if initialization fails. Or defer redis start.
	closeOnFailure := func() {
		mredis.Close()
	}

	cfg.Set("redis.hostname", mredis.Host())
	cfg.Set("redis.port", mredis.Port())
	cfg.Set("redis.pool.maxIdle", 1000)
	cfg.Set("redis.pool.idleTimeout", time.Second)
	cfg.Set("redis.pool.maxActive", 1000)
	cfg.Set("playerIndices", []string{
		"char.cleric",
		"char.knight",
		"char.paladin",
		"map.aleroth",
		"map.oasis",
		"mmr.rating",
		"mode.battleroyale",
		"mode.ctf",
		"mode.demo",
	})

	pool, err := redishelpers.ConnectionPool(cfg)
	if err != nil {
		closeOnFailure()
		return nil, err
	}

	serviceLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()

	for _, params := range paramsList {
		cfg.Set(params.ServicePortConfigName, serviceLh.Number())
		cfg.Set(params.ProxyPortConfigName, proxyLh.Number())
	}

	// Instantiate the gRPC server with the connections we've made
	logger.Info("Attempting to start gRPC server")
	grpcServer := serving.NewGrpcServer(serviceLh, proxyLh, logger)

	mmServer := &MiniMatchServer{
		OpenMatchServer: &serving.OpenMatchServer{
			Config:     cfg,
			GrpcServer: grpcServer,
			Logger:     logger,
			RedisPool:  pool,
		},
		mRedis: mredis,
	}
	for _, params := range paramsList {
		for _, binding := range params.Bindings {
			binding(mmServer.OpenMatchServer)
		}
	}
	return mmServer, nil
}
