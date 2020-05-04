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
	"flag"
	"sync"
	"testing"

	"github.com/pkg/errors"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/appmain/apptest"
	"open-match.dev/open-match/internal/config"
	mmfService "open-match.dev/open-match/internal/testing/mmf"
	"open-match.dev/open-match/pkg/pb"
)

var (
	testOnlyEnableMetrics        = flag.Bool("test_only_metrics", true, "Enables metrics exporting for tests.")
	testOnlyEnableRPCLoggingFlag = flag.Bool("test_only_rpc_logging", false, "Enables RPC Logging for tests. This output is very verbose.")
	testOnlyLoggingLevel         = flag.String("test_only_log_level", "info", "Sets the log level for tests.")
)

func newOM(t *testing.T) *om {
	om := &om{
		t: t,
	}
	t.Cleanup(func() {
		om.running.Wait()
		om.fLock.Lock()
		defer om.fLock.Unlock()
		// Set this cleanup before starting servers, so that servers will be
		// stopped before this runs.
		if om.mmf != nil && !om.mmfCalled {
			t.Error("MMF set but never called.")
		}
		if om.eval != nil && !om.evalCalled {
			t.Error("Evaluator set but never called.")
		}
	})

	om.cfg = start(t, om.evaluate, om.runMMF)
	om.fe = pb.NewFrontendServiceClient(apptest.GRPCClient(t, om.cfg, "api.frontend"))
	om.be = pb.NewBackendServiceClient(apptest.GRPCClient(t, om.cfg, "api.backend"))
	om.query = pb.NewQueryServiceClient(apptest.GRPCClient(t, om.cfg, "api.query"))

	return om
}

type om struct {
	t     *testing.T
	cfg   config.View
	fe    pb.FrontendServiceClient
	be    pb.BackendServiceClient
	query pb.QueryServiceClient

	running    sync.WaitGroup
	fLock      sync.Mutex
	mmfCalled  bool
	evalCalled bool
	mmf        mmfService.MatchFunction
	eval       evaluator.Evaluator
}

func (om *om) SetMMF(mmf mmfService.MatchFunction) {
	om.fLock.Lock()
	defer om.fLock.Unlock()

	if om.mmf == nil {
		om.mmf = mmf
		return
	}
	om.t.Fatal("Matchmaking function set multiple times")
}

func (om *om) runMMF(ctx context.Context, profile *pb.MatchProfile, out chan<- *pb.Match) error {
	om.running.Add(1)
	defer om.running.Done()
	om.fLock.Lock()
	mmf := om.mmf
	om.mmfCalled = true
	om.fLock.Unlock()

	if mmf == nil {
		return errors.New("MMF called without being set")
	}
	return mmf(ctx, profile, out)
}

func (om *om) SetEvaluator(eval evaluator.Evaluator) {
	om.fLock.Lock()
	defer om.fLock.Unlock()

	if om.eval == nil {
		om.eval = eval
		return
	}
	om.t.Fatal("Evaluator function set multiple times")
}

func (om *om) evaluate(ctx context.Context, in <-chan *pb.Match, out chan<- string) error {
	om.running.Add(1)
	defer om.running.Done()
	om.fLock.Lock()
	eval := om.eval
	om.evalCalled = true
	om.fLock.Unlock()

	if eval == nil {
		return errors.New("Evaluator called without being set")
	}
	return eval(ctx, in, out)
}

func (om *om) Frontend() pb.FrontendServiceClient {
	return om.fe
}

func (om *om) Backend() pb.BackendServiceClient {
	return om.be
}

func (om *om) Query() pb.QueryServiceClient {
	return om.query
}

func (om *om) MMFConfigGRPC() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: om.cfg.GetString("api." + apptest.ServiceName + ".hostname"),
		Port: int32(om.cfg.GetInt("api." + apptest.ServiceName + ".grpcport")),
		Type: pb.FunctionConfig_GRPC,
	}
}

func (om *om) MMFConfigHTTP() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: om.cfg.GetString("api." + apptest.ServiceName + ".hostname"),
		Port: int32(om.cfg.GetInt("api." + apptest.ServiceName + ".httpport")),
		Type: pb.FunctionConfig_REST,
	}
}

// configFile is the "cononical" test config.  It exactly matches the configmap
// which is used in the real cluster tests.
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
    hostname: "test"
    grpcport: "50509"
    httpport: "51509"
  test:
    hostname: "test"
    grpcport: "50509"
    httpport: "51509"


synchronizer:
  registrationIntervalMs: 100ms
  proposalCollectionIntervalMs: 100ms

storage:
  ignoreListTTL: 100ms
  page:
    size: 10

redis:
  sentinelPort: 26379
  sentinelMaster: om-redis-master
  sentinelHostname: om-redis.open-match.svc.cluster.local
  sentinelUsePassword: 
  usePassword: false
  passwordPath: /opt/bitnami/redis/secrets/redis-password
  pool:
    maxIdle: 200
    maxActive: 0
    idleTimeout: 0
    healthCheckTimeout: 300ms

telemetry:
  reportingPeriod: "1m"
  traceSamplingFraction: "0.01"
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
