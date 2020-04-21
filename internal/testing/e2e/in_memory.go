// +build !e2ecluster
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

package e2e

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/Bose/minisentinel"
	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/spf13/viper"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/app/evaluator/defaulteval"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/appmain/apptest"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
	internalMmf "open-match.dev/open-match/internal/testing/mmf"
	pb "open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/matchfunction/mmf"
)

type inmemoryOM struct {
	cfg config.View
	t   *testing.T
}

func (iom *inmemoryOM) withT(t *testing.T) OM {
	cfg := newInMemoryEnvironment(t)

	om := &inmemoryOM{
		cfg: cfg,
		t:   t,
	}
	return om
}

func createZygote(m *testing.M) (OM, error) {
	return &inmemoryOM{}, nil
}

func (iom *inmemoryOM) MustFrontendGRPC() pb.FrontendServiceClient {
	return pb.NewFrontendServiceClient(apptest.GRPCClient(iom.t, iom.cfg, "api.backend"))
}

func (iom *inmemoryOM) MustBackendGRPC() pb.BackendServiceClient {
	return pb.NewBackendServiceClient(apptest.GRPCClient(iom.t, iom.cfg, "api.frontend"))
}

func (iom *inmemoryOM) MustQueryServiceGRPC() pb.QueryServiceClient {
	return pb.NewQueryServiceClient(apptest.GRPCClient(iom.t, iom.cfg, "api.query"))
}

func (iom *inmemoryOM) MustMmfConfigGRPC() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.cfg.GetString("api.function.hostname"),
		Port: int32(iom.cfg.GetInt("api.function.grpcport")),
		Type: pb.FunctionConfig_GRPC,
	}
}

func (iom *inmemoryOM) MustMmfConfigHTTP() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.cfg.GetString("api.function.hostname"),
		Port: int32(iom.cfg.GetInt("api.function.httpport")),
		Type: pb.FunctionConfig_REST,
	}
}

func (iom *inmemoryOM) HealthCheck() error {
	return nil
}

func (iom *inmemoryOM) Context() context.Context {
	return context.Background()
}

func newInMemoryEnvironment(t *testing.T) config.View {

	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}
	t.Cleanup(mredis.Close)

	msentinal := minisentinel.NewSentinel(mredis)
	err = msentinal.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start minisentinel, %v", err)
	}
	t.Cleanup(msentinal.Close)

	grpcListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, grpcPort, err := net.SplitHostPort(grpcListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	httpListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, httpPort, err := net.SplitHostPort(httpListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	listeners := []net.Listener{grpcListener, httpListener}

	cfg := viper.New()
	cfg.SetConfigType("yaml")
	err = cfg.ReadConfig(strings.NewReader(configFile))
	if err != nil {
		t.Fatal(err)
	}

	cfg.Set("redis.sentinelHostname", msentinal.Host())
	cfg.Set("redis.sentinelPort", msentinal.Port())
	cfg.Set("redis.sentinelMaster", msentinal.MasterInfo().Name)
	services := []string{apptest.ServiceName, "synchronizer", "backend", "frontend", "query", "function", "evaluator"}
	for _, name := range services {
		cfg.Set("api."+name+".hostname", "localhost")
		cfg.Set("api."+name+".grpcport", grpcPort)
		cfg.Set("api."+name+".httpport", httpPort)
	}
	cfg.Set("storage.page.size", 10)
	cfg.Set(rpc.ConfigNameEnableRPCLogging, *testOnlyEnableRPCLoggingFlag)
	cfg.Set("logging.level", *testOnlyLoggingLevel)
	cfg.Set(telemetry.ConfigNameEnableMetrics, *testOnlyEnableMetrics)

	apptest.TestApp(t, cfg, listeners, minimatch.BindService, internalMmf.BindServiceFor(mmf.MakeMatches), evaluator.BindServiceFor(defaulteval.Evaluate))
	return cfg
}

// configFile is the "cononical" test config.  It exactly matches the configmap
// which is used in the real cluster tests.
// TODO: The above is a lie.  There should be a test which confirm that this
// does actually match the test config map.
const configFile = `
logging:
  level: debug
  format: text
  rpc: false

backoff:
  initialInterval: 100ms
  maxInterval: 500ms
  multiplier: 1.5
  randFactor: 0.5
  maxElapsedTime: 3000ms

api:
  backend:
    hostname: "om-backend"
    grpcport: "50505"
    httpport: "51505"
  frontend:
    hostname: "om-frontend"
    grpcport: "50504"
    httpport: "51504"
  query:
    hostname: "om-query"
    grpcport: "50503"
    httpport: "51503"
  synchronizer:
    hostname: "om-synchronizer"
    grpcport: "50506"
    httpport: "51506"
  swaggerui:
    hostname: "om-swaggerui"
    httpport: "51500"
  scale:
    httpport: "51509"
  evaluator:
    hostname: "om-evaluator"
    grpcport: "50508"
    httpport: "51508"

synchronizer:
  registrationIntervalMs: 250ms
  proposalCollectionIntervalMs: 20000ms

storage:
  ignoreListTTL: 500ms
  page:
    size: 10000

redis:
  sentinelPort: 26379
  sentinelMaster: om-redis-master
  sentinelHostname: om-redis.open-match.svc.cluster.local
  sentinelUsePassword: 
  usePassword: false
  passwordPath: /opt/bitnami/redis/secrets/redis-password
  pool:
    maxIdle: 500
    maxActive: 500
    idleTimeout: 0
    healthCheckTimeout: 300ms

telemetry:
  reportingPeriod: "1m"
  traceSamplingFraction: 0.005
  zpages:
    enable: "true"
  jaeger:
    enable: "false"
    agentEndpoint: "open-match-jaeger-agent:6831"
    collectorEndpoint: "http://open-match-jaeger-collector:14268/api/traces"
  prometheus:
    enable: "false"
    endpoint: "/metrics"
    serviceDiscovery: "true"
  stackdriverMetrics:
    enable: "false"
    gcpProjectId: "sredig-gaming-test"
    prefix: "open_match"
`
