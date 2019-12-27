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
	"log"
	"os"
	"testing"

	"google.golang.org/grpc/resolver"
	pb "open-match.dev/open-match/pkg/pb"
)

var (
	testOnlyEnableMetrics        = flag.Bool("test_only_metrics", true, "Enables metrics exporting for tests.")
	testOnlyEnableRPCLoggingFlag = flag.Bool("test_only_rpc_logging", false, "Enables RPC Logging for tests. This output is very verbose.")
	testOnlyLoggingLevel         = flag.String("test_only_log_level", "info", "Sets the log level for tests.")
)

// OM is the interface for communicating with Open Match.
type OM interface {
	// MustFrontendGRPC returns a gRPC client to frontend server.
	MustFrontendGRPC() pb.FrontendClient
	// MustBackendGRPC returns a gRPC client to backend server.
	MustBackendGRPC() pb.BackendClient
	// MustMmLogicGRPC returns a gRPC client to mmlogic server.
	MustMmLogicGRPC() pb.MmLogicClient
	// HealthCheck probes the cluster for readiness.
	HealthCheck() error
	// MustMmfConfigGRPC returns a grpc match function config for backend server.
	MustMmfConfigGRPC() *pb.FunctionConfig
	// MustMmfConfigHTTP returns a http match function config for backend server.
	MustMmfConfigHTTP() *pb.FunctionConfig
	// Context provides a context to call remote methods.
	Context() context.Context

	cleanup()
	cleanupMain() error
	withT(t *testing.T) OM
}

// New creates a new e2e test interface.
func New(t *testing.T) (OM, func()) {
	om := zygote.withT(t)
	return om, om.cleanup
}

// RunMain provides the setup and teardown for Open Match e2e tests.
func RunMain(m *testing.M) {
	// Reset the gRPC resolver to passthrough for end-to-end out-of-cluster testings.
	// DNS resolver is unsupported for end-to-end local testings.
	resolver.SetDefaultScheme("passthrough")
	var exitCode int
	z, err := createZygote(m)
	if err != nil {
		log.Fatalf("failed to setup framework: %s", err)
	}
	defer func() {
		cErr := z.cleanupMain()
		if cErr != nil {
			log.Printf("failed to cleanup resources: %s", cErr)
		}
		os.Exit(exitCode)
	}()
	zygote = z
	exitCode = m.Run()
}
