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
	"log"
	pb "open-match.dev/open-match/pkg/pb"
	"os"
	"testing"
)

// OM is the interface for communicating with Open Match.
type OM interface {
	// MustFrontendGRPC returns a gRPC client to frontend server.
	MustFrontendGRPC() (pb.FrontendClient, func())
	// MustBackendGRPC returns a gRPC client to backend server.
	MustBackendGRPC() (pb.BackendClient, func())
	// MustMmLogicGRPC returns a gRPC client to mmlogic server.
	MustMmLogicGRPC() (pb.MmLogicClient, func())
	// HealthCheck probes the cluster for readiness.
	HealthCheck() error
	// Context provides a context to call remote methods.
	Context() context.Context

	cleanup() error
	cleanupMain() error
	withT(t *testing.T) OM
}

// New creates a new e2e test interface.
func New(t *testing.T) (OM, func()) {
	om := zygote.withT(t)
	return om, func() {
		if err := om.cleanup(); err != nil {
			t.Errorf("failed to cleanup minimatch: %s", err)
		}
	}
}

// RunMain provides the setup and teardown for Open Match e2e tests.
func RunMain(m *testing.M) {
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

func closeSilently(f func() error) func() {
	return func() {
		if err := f(); err != nil {
			log.Printf("failed to close, %s", err)
		}
	}
}
