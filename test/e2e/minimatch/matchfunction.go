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

	"github.com/rs/xid"
	"github.com/spf13/viper"
	harness "open-match.dev/open-match/internal/harness/golang"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	"open-match.dev/open-match/examples/functions/golang/pool"
)

func createMatchFunctionForTest(t *testing.T, depTc *rpcTesting.TestContext) (*rpcTesting.TestContext, string) {
	mmfName := "test-matchfunction"

	// TODO: Use insecure for now since minimatch and mmf only works with the same secure mode
	tc := rpcTesting.MustServeInsecure(t, func(p *rpc.ServerParams) {
		cfg := viper.New()

		// The below configuration is used by GRPC harness to create an mmlogic client to query tickets.
		cfg.Set("api.mmlogic.hostname", depTc.GetHostname())
		cfg.Set("api.mmlogic.grpcport", depTc.GetGRPCPort())
		cfg.Set("api.mmlogic.httpport", depTc.GetHTTPPort())

		if err := harness.BindService(p, cfg, &harness.FunctionSettings{
			Func:         pool.MakeMatches,
		}); err != nil {
			t.Error(err)
		}
	})
	return tc, mmfName
}
