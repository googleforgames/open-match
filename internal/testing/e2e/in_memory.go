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
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/app/evaluator"
	"open-match.dev/open-match/internal/app/evaluator/defaulteval"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/appmain"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/internal/telemetry"
	internalMmf "open-match.dev/open-match/internal/testing/mmf"
	"open-match.dev/open-match/internal/util"
	pb "open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/matchfunction/mmf"
)

// type inmemoryOM struct {
// 	mainTc *rpcTesting.TestContext
// 	mmfTc  *rpcTesting.TestContext
// 	evalTc *rpcTesting.TestContext
// 	t      *testing.T
// 	mc     *util.MultiClose
// }

// func (iom *inmemoryOM) withT(t *testing.T) OM {
// 	evalTc := createEvaluatorForTest(t)
// 	mainTc := createMinimatchForTest(t, evalTc)
// 	mmfTc := createMatchFunctionForTest(t, mainTc)

// 	om := &inmemoryOM{
// 		mainTc: mainTc,
// 		mmfTc:  mmfTc,
// 		evalTc: evalTc,
// 		t:      t,
// 		mc:     util.NewMultiClose(),
// 	}
// 	return om
// }

// func createZygote(m *testing.M) (OM, error) {
// 	return &inmemoryOM{}, nil
// }

// func (iom *inmemoryOM) MustFrontendGRPC() pb.FrontendServiceClient {
// 	conn := iom.mainTc.MustGRPC()
// 	iom.mc.AddCloseWithErrorFunc(conn.Close)
// 	return pb.NewFrontendServiceClient(conn)
// }

// func (iom *inmemoryOM) MustBackendGRPC() pb.BackendServiceClient {
// 	conn := iom.mainTc.MustGRPC()
// 	iom.mc.AddCloseWithErrorFunc(conn.Close)
// 	return pb.NewBackendServiceClient(conn)
// }

// func (iom *inmemoryOM) MustQueryServiceGRPC() pb.QueryServiceClient {
// 	conn := iom.mainTc.MustGRPC()
// 	iom.mc.AddCloseWithErrorFunc(conn.Close)
// 	return pb.NewQueryServiceClient(conn)
// }

// func (iom *inmemoryOM) MustMmfConfigGRPC() *pb.FunctionConfig {
// 	return &pb.FunctionConfig{
// 		Host: iom.mmfTc.GetHostname(),
// 		Port: int32(iom.mmfTc.GetGRPCPort()),
// 		Type: pb.FunctionConfig_GRPC,
// 	}
// }

// func (iom *inmemoryOM) MustMmfConfigHTTP() *pb.FunctionConfig {
// 	return &pb.FunctionConfig{
// 		Host: iom.mmfTc.GetHostname(),
// 		Port: int32(iom.mmfTc.GetHTTPPort()),
// 		Type: pb.FunctionConfig_REST,
// 	}
// }

// func (iom *inmemoryOM) HealthCheck() error {
// 	return nil
// }

// func (iom *inmemoryOM) Context() context.Context {
// 	return iom.mainTc.Context()
// }

// func (iom *inmemoryOM) cleanup() {
// 	iom.mc.Close()
// 	iom.mainTc.Close()
// 	iom.mmfTc.Close()
// 	iom.evalTc.Close()
// }

// func (iom *inmemoryOM) cleanupMain() error {
// 	return nil
// }

// // Create a minimatch test service with function bindings from frontendService, backendService, and queryService.
// // Instruct this service to start and connect to a fake storage service.
// func createMinimatchForTest(t *testing.T, evalTc *rpcTesting.TestContext) config.View {
// 	var closer func()
// 	cfg := viper.New()

// 	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
// 	// Server a minimatch for testing using random port at tc.grpcAddress & tc.proxyAddress
// 	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
// 		closer = statestoreTesting.New(t, cfg)
// 		cfg.Set("storage.page.size", 10)
// 		assert.Nil(t, appmain.TemporaryBindWrapper(minimatch.BindService, p, cfg))
// 	})
// 	// TODO: Revisit the Minimatch test setup in future milestone to simplify passing config
// 	// values between components. The backend needs to connect to to the synchronizer but when
// 	// it is initialized, does not know what port the synchronizer is on. To work around this,
// 	// the backend sets up a connection to the synchronizer at runtime and hence can access these
// 	// config values to establish the connection.
// 	cfg.Set("api.synchronizer.hostname", tc.GetHostname())
// 	cfg.Set("api.synchronizer.grpcport", tc.GetGRPCPort())
// 	cfg.Set("api.synchronizer.httpport", tc.GetHTTPPort())
// 	cfg.Set("synchronizer.registrationIntervalMs", "200ms")
// 	cfg.Set("synchronizer.proposalCollectionIntervalMs", "200ms")
// 	cfg.Set("api.evaluator.hostname", evalTc.GetHostname())
// 	cfg.Set("api.evaluator.grpcport", evalTc.GetGRPCPort())
// 	cfg.Set("api.evaluator.httpport", evalTc.GetHTTPPort())
// 	cfg.Set("synchronizer.enabled", true)
// 	cfg.Set(rpc.ConfigNameEnableRPCLogging, *testOnlyEnableRPCLoggingFlag)
// 	cfg.Set("logging.level", *testOnlyLoggingLevel)
// 	cfg.Set(telemetry.ConfigNameEnableMetrics, *testOnlyEnableMetrics)

// 	// TODO: This is very ugly. Need a better story around closing resources.
// 	tc.AddCloseFunc(closer)
// 	return tc
// }

// // Create a mmf service using a started test server.
// // Inject the port config of queryService using that the passed in test server
// func createMatchFunctionForTest(t *testing.T, c *rpcTesting.TestContext) *rpcTesting.TestContext {
// 	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
// 	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
// 		cfg := viper.New()

// 		// The below configuration is used by GRPC harness to create an queryService client to query tickets.
// 		cfg.Set("api.query.hostname", c.GetHostname())
// 		cfg.Set("api.query.grpcport", c.GetGRPCPort())
// 		cfg.Set("api.query.httpport", c.GetHTTPPort())

// 		assert.Nil(t, appmain.TemporaryBindWrapper(internalMmf.BindServiceFor(mmf.MakeMatches), p, cfg))
// 	})
// 	return tc
// }

// // Create an evaluator service that will be used by the minimatch tests.
// func createEvaluatorForTest(t *testing.T) *rpcTesting.TestContext {
// 	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
// 		cfg := viper.New()
// 		assert.Nil(t, appmain.TemporaryBindWrapper(evaluator.BindServiceFor(defaulteval.Evaluate), p, cfg))
// 	})

// 	return tc
// }

/////////////////////////////////////
/////////////////////////////////////
/////////////////////////////////////
/////////////////////////////////////
/////////////////////////////////////
/////////////////////////////////////

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
	return pb.NewFrontendServiceClient(apptest.GRPCClient(t, iom.cfg, "api.backend"))
}

func (iom *inmemoryOM) MustBackendGRPC() pb.BackendServiceClient {
	return pb.NewBackendServiceClient(apptest.GRPCClient(t, iom.cfg, "api.backend"))
}

func (iom *inmemoryOM) MustQueryServiceGRPC() pb.QueryServiceClient {
	return pb.NewQueryServiceClient(apptest.GRPCClient(t, iom.cfg, "api.query"))
}

func (iom *inmemoryOM) MustMmfConfigGRPC() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.cfg.GetString("api.function.hostname"),
		Port: int32(iom.cfg.GetString("api.function.grpcport")),
		Type: pb.FunctionConfig_GRPC,
	}
}

func (iom *inmemoryOM) MustMmfConfigHTTP() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.cfg.GetString("api.function.hostname"),
		Port: int32(iom.cfg.GetString("api.function.httpport")),
		Type: pb.FunctionConfig_REST,
	}
}

func (iom *inmemoryOM) HealthCheck() error {
	return nil
}

func (iom *inmemoryOM) Context() context.Context {
	return iom.mainTc.Context()
}

func (iom *inmemoryOM) cleanup() {
}

func (iom *inmemoryOM) cleanupMain() error {
	return nil
}

func newInMemoryEnvironment(t *testing.T) config.View {
	cfg := viper.New()

	mredis := miniredis.NewMiniRedis()
	err := mredis.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start miniredis, %v", err)
	}
	t.Cleanup(mredis.Close)

	msentinal := minisentinel.NewSentinel(mredis)
	err = s.StartAddr("localhost:0")
	if err != nil {
		t.Fatalf("failed to start minisentinel, %v", err)
	}
	t.Cleanup(s.Close)

	grpcListener, err := net.Listener("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, grpcPort, err := net.SplitHostPort(grpcListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	httpListener, err := net.Listener("tcp", ":0")
	if err != nil {
		t.Fatal(err)
	}
	_, httpPort, err := net.SplitHostPort(httpListener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	cfg.Set("redis.sentinelEnabled", true)
	cfg.Set("redis.sentinelHostname", msentinal.Host())
	cfg.Set("redis.sentinelPort", msentinal.Port())
	cfg.Set("redis.sentinelMaster", msentinal.MasterInfo().Name)
	cfg.Set("redis.pool.maxIdle", 5)
	cfg.Set("redis.pool.maxActive", 5)
	cfg.Set("redis.pool.idleTimeout", 10*time.Second)
	cfg.Set("redis.pool.healthCheckTimeout", 100*time.Millisecond)
	cfg.Set("storage.ignoreListTTL", 500*time.Millisecond)
	cfg.Set("backoff.initialInterval", 30*time.Millisecond)
	cfg.Set("backoff.randFactor", .5)
	cfg.Set("backoff.multiplier", .5)
	cfg.Set("backoff.maxInterval", 300*time.Millisecond)
	cfg.Set("backoff.maxElapsedTime", 1000*time.Millisecond)
	cfg.Set("storage.page.size", 10)
	services := []string{"synchronizer", "backend", "frontend", "query", "function", "evaluator"}
	for _, name := range services {
		cfg.Set("api."+name+".hostname", "localhost")
		cfg.Set("api."+name+".grpcport", grpcPort)
		cfg.Set("api."+name+".httpport", httpPort)
	}
	cfg.Set("synchronizer.registrationIntervalMs", "200ms")
	cfg.Set("synchronizer.proposalCollectionIntervalMs", "200ms")
	cfg.Set("synchronizer.enabled", true)
	cfg.Set(rpc.ConfigNameEnableRPCLogging, *testOnlyEnableRPCLoggingFlag)
	cfg.Set("logging.level", *testOnlyLoggingLevel)
	cfg.Set(telemetry.ConfigNameEnableMetrics, *testOnlyEnableMetrics)

	apptest.TestApp(t, cfg, listeners, minimatch.BindService, internalMmf.BindServiceFor(mmf.MakeMatches), evaluator.BindServiceFor(defaulteval.Evaluate))
}
