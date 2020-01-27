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
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/internal/testing/evaluator"
	internalMmf "open-match.dev/open-match/internal/testing/mmf"
	"open-match.dev/open-match/internal/util"
	pb "open-match.dev/open-match/pkg/pb"
	"open-match.dev/open-match/test/matchfunction/mmf"

	"open-match.dev/open-match/test/evaluator/evaluate"
)

type inmemoryOM struct {
	mainTc *rpcTesting.TestContext
	mmfTc  *rpcTesting.TestContext
	evalTc *rpcTesting.TestContext
	t      *testing.T
	mc     *util.MultiClose
}

func (iom *inmemoryOM) withT(t *testing.T) OM {
	evalTc := createEvaluatorForTest(t)
	mainTc := createMinimatchForTest(t, evalTc)
	mmfTc := createMatchFunctionForTest(t, mainTc)

	om := &inmemoryOM{
		mainTc: mainTc,
		mmfTc:  mmfTc,
		evalTc: evalTc,
		t:      t,
		mc:     util.NewMultiClose(),
	}
	return om
}

func createZygote(m *testing.M) (OM, error) {
	return &inmemoryOM{}, nil
}

func (iom *inmemoryOM) MustFrontendGRPC() pb.FrontendServiceClient {
	conn := iom.mainTc.MustGRPC()
	iom.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewFrontendServiceClient(conn)
}

func (iom *inmemoryOM) MustBackendGRPC() pb.BackendServiceClient {
	conn := iom.mainTc.MustGRPC()
	iom.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewBackendServiceClient(conn)
}

func (iom *inmemoryOM) MustQueryServiceGRPC() pb.QueryServiceClient {
	conn := iom.mainTc.MustGRPC()
	iom.mc.AddCloseWithErrorFunc(conn.Close)
	return pb.NewQueryServiceClient(conn)
}

func (iom *inmemoryOM) MustMmfConfigGRPC() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.mmfTc.GetHostname(),
		Port: int32(iom.mmfTc.GetGRPCPort()),
		Type: pb.FunctionConfig_GRPC,
	}
}

func (iom *inmemoryOM) MustMmfConfigHTTP() *pb.FunctionConfig {
	return &pb.FunctionConfig{
		Host: iom.mmfTc.GetHostname(),
		Port: int32(iom.mmfTc.GetHTTPPort()),
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
	iom.mc.Close()
	iom.mainTc.Close()
	iom.mmfTc.Close()
	iom.evalTc.Close()
}

func (iom *inmemoryOM) cleanupMain() error {
	return nil
}

// Create a minimatch test service with function bindings from frontendService, backendService, and queryService.
// Instruct this service to start and connect to a fake storage service.
func createMinimatchForTest(t *testing.T, evalTc *rpcTesting.TestContext) *rpcTesting.TestContext {
	var closer func()
	cfg := viper.New()

	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	// Server a minimatch for testing using random port at tc.grpcAddress & tc.proxyAddress
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		closer = statestoreTesting.New(t, cfg)
		cfg.Set("storage.page.size", 10)
		assert.Nil(t, minimatch.BindService(p, cfg))
	})
	// TODO: Revisit the Minimatch test setup in future milestone to simplify passing config
	// values between components. The backend needs to connect to to the synchronizer but when
	// it is initialized, does not know what port the synchronizer is on. To work around this,
	// the backend sets up a connection to the synchronizer at runtime and hence can access these
	// config values to establish the connection.
	cfg.Set("api.synchronizer.hostname", tc.GetHostname())
	cfg.Set("api.synchronizer.grpcport", tc.GetGRPCPort())
	cfg.Set("api.synchronizer.httpport", tc.GetHTTPPort())
	cfg.Set("synchronizer.registrationIntervalMs", "200ms")
	cfg.Set("synchronizer.proposalCollectionIntervalMs", "200ms")
	cfg.Set("api.evaluator.hostname", evalTc.GetHostname())
	cfg.Set("api.evaluator.grpcport", evalTc.GetGRPCPort())
	cfg.Set("api.evaluator.httpport", evalTc.GetHTTPPort())
	cfg.Set("synchronizer.enabled", true)
	cfg.Set(rpc.ConfigNameEnableRPCLogging, *testOnlyEnableRPCLoggingFlag)
	cfg.Set("logging.level", *testOnlyLoggingLevel)
	cfg.Set(telemetry.ConfigNameEnableMetrics, *testOnlyEnableMetrics)

	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}

// Create a mmf service using a started test server.
// Inject the port config of queryService using that the passed in test server
func createMatchFunctionForTest(t *testing.T, c *rpcTesting.TestContext) *rpcTesting.TestContext {
	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()

		// The below configuration is used by GRPC harness to create an queryService client to query tickets.
		cfg.Set("api.query.hostname", c.GetHostname())
		cfg.Set("api.query.grpcport", c.GetGRPCPort())
		cfg.Set("api.query.httpport", c.GetHTTPPort())

		assert.Nil(t, internalMmf.BindService(p, cfg, &internalMmf.FunctionSettings{
			Func: mmf.MakeMatches,
		}))
	})
	return tc
}

// Create an evaluator service that will be used by the minimatch tests.
func createEvaluatorForTest(t *testing.T) *rpcTesting.TestContext {
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		assert.Nil(t, evaluator.BindService(p, cfg, evaluate.Evaluate))
	})

	return tc
}
