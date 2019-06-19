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
	"open-match.dev/open-match/examples/functions/golang/pool"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	mmfHarness "open-match.dev/open-match/pkg/harness/function/golang"
)

// Create a mmf service using a started test server.
// Inject the port config of mmlogic using that the passed in test server
func createMatchFunctionForTest(t *testing.T, c *rpcTesting.TestContext) *rpcTesting.TestContext {
	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()

		// The below configuration is used by GRPC harness to create an mmlogic client to query tickets.
		cfg.Set("api.mmlogic.hostname", c.GetHostname())
		cfg.Set("api.mmlogic.grpcport", c.GetGRPCPort())
		cfg.Set("api.mmlogic.httpport", c.GetHTTPPort())

		assert.Nil(t, mmfHarness.BindService(p, cfg, &mmfHarness.FunctionSettings{
			Func: pool.MakeMatches,
		}))
	})
	return tc
}
