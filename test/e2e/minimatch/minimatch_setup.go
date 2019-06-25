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

package minimatch

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	"open-match.dev/open-match/internal/testing/e2e"
)

// Create a minimatch test service with function bindings from frontend, backend, and mmlogic.
// Instruct this service to start and connect to a fake storage service.
func createMinimatchForTest(t *testing.T, evalTc *rpcTesting.TestContext) *rpcTesting.TestContext {
	var closer func()
	cfg := viper.New()

	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	// Server a minimatch for testing using random port at tc.grpcAddress & tc.proxyAddress
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		closer = statestoreTesting.New(t, cfg)
		cfg.Set("storage.page.size", 10)
		// Set up the attributes that a ticket will be indexed for.
		cfg.Set("playerIndices", []string{
			e2e.SkillAttribute,
			e2e.Map1Attribute,
			e2e.Map2Attribute,
		})
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
	cfg.Set("synchronizer.registrationIntervalMs", "3000ms")
	cfg.Set("synchronizer.proposalCollectionIntervalMs", "3000ms")
	cfg.Set("api.evaluator.hostname", evalTc.GetHostname())
	cfg.Set("api.evaluator.grpcport", evalTc.GetGRPCPort())
	cfg.Set("api.evaluator.httpport", evalTc.GetHTTPPort())
	// TODO: Uncomment this when the synchronization logic is checked in to enable synchronizer.
	// cfg.Set("synchronizer.enabled", true)

	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}
