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
	"github.com/spf13/viper"
	harness "open-match.dev/open-match/internal/harness/golang"
	harnessTesting "open-match.dev/open-match/internal/harness/golang/testing"

	"open-match.dev/open-match/internal/rpc"
)

const (
	mmfName        = "test-matchfunction"
	mmfPrefix      = "testmmf"
	mmfHost        = "localhost"
	mmfGRPCPort    = "50511"
	mmfHTTPPort    = "51511"
	mmfGRPCPortInt = 50511
	mmfHTTPPortInt = 51511
)

// serveMatchFunction creates a GRPC server and starts it to server the match function forever.
func serveMatchFunction() (func(), error) {
	cfg := viper.New()
	// Set match function configuration.
	cfg.Set("testmmf.hostname", mmfHost)
	cfg.Set("testmmf.grpcport", mmfGRPCPort)
	cfg.Set("testmmf.httpport", mmfHTTPPort)

	// The below configuration is used by GRPC harness to create an mmlogic client to query tickets.
	cfg.Set("api.mmlogic.hostname", minimatchHost)
	cfg.Set("api.mmlogic.grpcport", minimatchGRPCPort)
	cfg.Set("api.mmlogic.httpport", minimatchHTTPPort)

	p, err := rpc.NewServerParamsFromConfig(cfg, mmfPrefix)
	if err != nil {
		return nil, err
	}

	if err := harness.BindService(p, cfg, &harness.FunctionSettings{
		FunctionName: mmfName,
		Func:         harnessTesting.MakeMatches,
	}); err != nil {
		return nil, err
	}

	s := &rpc.Server{}
	waitForStart, err := s.Start(p)
	if err != nil {
		return nil, err
	}
	go func() {
		waitForStart()
	}()

	return func() { s.Stop() }, nil
}
