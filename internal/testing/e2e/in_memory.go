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
	"open-match.dev/open-match/internal/app/minimatch"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
	statestoreTesting "open-match.dev/open-match/internal/statestore/testing"
	pb "open-match.dev/open-match/pkg/pb"
)

const (
	// Map1BeginnerPool is an index.
	Map1BeginnerPool = "map1beginner"
	// Map1AdvancedPool is an index.
	Map1AdvancedPool = "map1advanced"
	// Map2BeginnerPool is an index.
	Map2BeginnerPool = "map2beginner"
	// Map2AdvancedPool is an index.
	Map2AdvancedPool = "map2advanced"
	// SkillAttribute is an index.
	SkillAttribute = "skill"
	// Map1Attribute is an index.
	Map1Attribute = "map1"
	// Map2Attribute is an index.
	Map2Attribute = "map2"
)

type inmemoryOM struct {
	tc *rpcTesting.TestContext
	t  *testing.T
	mc *multicloser
}

func (iom *inmemoryOM) withT(t *testing.T) OM {
	tc := createMinimatchForTest(t)
	om := &inmemoryOM{
		tc: tc,
		t:  t,
		mc: newMulticloser(),
	}
	return om
}

func createZygote(m *testing.M) (OM, error) {
	return &inmemoryOM{}, nil
}

func (iom *inmemoryOM) MustFrontendGRPC() pb.FrontendClient {
	conn := iom.tc.MustGRPC()
	iom.mc.addSilent(conn.Close)
	return pb.NewFrontendClient(conn)
}

func (iom *inmemoryOM) MustBackendGRPC() pb.BackendClient {
	conn := iom.tc.MustGRPC()
	iom.mc.addSilent(conn.Close)
	return pb.NewBackendClient(conn)
}

func (iom *inmemoryOM) MustMmLogicGRPC() pb.MmLogicClient {
	conn := iom.tc.MustGRPC()
	iom.mc.addSilent(conn.Close)
	return pb.NewMmLogicClient(conn)
}

func (iom *inmemoryOM) HealthCheck() error {
	return nil
}

func (iom *inmemoryOM) Context() context.Context {
	return iom.tc.Context()
}

func (iom *inmemoryOM) cleanup() {
	iom.mc.close()
	iom.tc.Close()
}

func (iom *inmemoryOM) cleanupMain() error {
	return nil
}

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
		cfg.Set("ticketIndices", []string{
			SkillAttribute,
			Map1Attribute,
			Map2Attribute,
		})
		if err := minimatch.BindService(p, cfg); err != nil {
			t.Fatalf("cannot bind minimatch: %s", err)
		}
	})
	// TODO: This is very ugly. Need a better story around closing resources.
	tc.AddCloseFunc(closer)
	return tc
}
