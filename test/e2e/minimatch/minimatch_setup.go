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
	"open-match.dev/open-match/internal/app/backend"
	"open-match.dev/open-match/internal/app/frontend"
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/app/mmlogic"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
)

// Create a minimatch test service with function bindings from frontend, backend, and mmlogic.
// Instruct this service to start and connect to a fake storage service.
func createMinimatchForTest(t *testing.T) *rpcTesting.TestContext {
	var closer func()

	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	// Server a minimatch for testing using random port at tc.grpcAddress & tc.proxyAddress
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closer = statestoreTesting.New(t, cfg)

		cfg.Set("storage.page.size", 10)
		// Set up the attributes that a ticket will be indexed for.
		cfg.Set("playerIndices", []string{
			skillattribute,
			map1attribute,
			map2attribute,
		})
		assert.Nil(t, minimatch.BindService(p, cfg))
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}

func createMmlogicForTest(t *testing.T) *rpcTesting.TestContext {
	var closer func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closer = statestoreTesting.New(t, cfg)

		cfg.Set("storage.page.size", 10)
		assert.Nil(t, mmlogic.BindService(p, cfg))
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}

func createBackendForTest(t *testing.T) *rpcTesting.TestContext {
	var closer func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closer = statestoreTesting.New(t, cfg)

		cfg.Set("storage.page.size", 10)
		assert.Nil(t, backend.BindService(p, cfg))
		assert.Nil(t, frontend.BindService(p, cfg))
		assert.Nil(t, mmlogic.BindService(p, cfg))
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}

func createStore(t *testing.T) *rpcTesting.TestContext {
	var closer func()
	tc := rpcTesting.MustServe(t, func(p *rpc.ServerParams) {
		cfg := viper.New()
		closer = statestoreTesting.New(t, cfg)

		cfg.Set("playerIndices", []string{"testindex1", "testindex2"})
		assert.Nil(t, frontend.BindService(p, cfg))
	})
	tc.AddCloseFunc(closer)
	return tc
}
