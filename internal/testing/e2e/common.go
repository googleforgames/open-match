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
	"time"

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
		om.fLock.Lock()
		defer om.fLock.Unlock()
		om.running.Wait()
		// Set this cleanup before starting servers, so that servers will be
		// stopped before this runs.
		if om.mmf != nil && !om.mmfCalled {
			t.Error("MMF set but never called.")
		}
		if om.eval != nil && !om.evalCalled {
			t.Error("Evaluator set but never called.")
		}
	})

	om.cfg, om.AdvanceTTLTime = start(t, om.evaluate, om.runMMF)
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

	// For local tests, advances the mini-redis ttl time.  For in cluster tests,
	// just sleeps.
	AdvanceTTLTime func(time.Duration)

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
	om.fLock.Lock()
	om.running.Add(1)
	defer om.running.Done()
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
	om.fLock.Lock()
	om.running.Add(1)
	defer om.running.Done()
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

// Testing constants which must match the configuration.  Not parsed in test so
// that parsing bugs can't hide logic bugs.
const registrationInterval = time.Millisecond * 200
const proposalCollectionInterval = time.Millisecond * 200
const pendingReleaseTimeout = time.Second * 1
const assignedDeleteTimeout = time.Millisecond * 200

// configFile is the "cononical" test config.  It exactly matches the configmap
// which is used in the real cluster tests.
const configFile = `
registrationInterval: 200ms
proposalCollectionInterval: 200ms
pendingReleaseTimeout: 1s
assignedDeleteTimeout: 200ms
queryPageSize: 10

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
    hostname: "open-match-backend"
    grpcport: "50505"
    httpport: "51505"
  frontend:
    hostname: "open-match-frontend"
    grpcport: "50504"
    httpport: "51504"
  query:
    hostname: "open-match-query"
    grpcport: "50503"
    httpport: "51503"
  synchronizer:
    hostname: "open-match-synchronizer"
    grpcport: "50506"
    httpport: "51506"
  swaggerui:
    hostname: "open-match-swaggerui"
    httpport: "51500"
  scale:
    httpport: "51509"
  evaluator:
    hostname: "open-match-test"
    grpcport: "50509"
    httpport: "51509"
  test:
    hostname: "open-match-test"
    grpcport: "50509"
    httpport: "51509"

redis:
  sentinelPort: 26379
  sentinelMaster: om-redis-master
  sentinelHostname: open-match-redis
  sentinelUsePassword: false
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
    agentEndpoint: ""
    collectorEndpoint: ""
  prometheus:
    enable: "false"
    endpoint: "/metrics"
    serviceDiscovery: "true"
  stackdriverMetrics:
    enable: "false"
    gcpProjectId: "intentionally-invalid-value"
    prefix: "open_match"
`
