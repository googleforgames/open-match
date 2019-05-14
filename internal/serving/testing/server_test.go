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

package testing

import (
	"testing"

	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/serving"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	shellTesting "open-match.dev/open-match/internal/testing"
)

// TestMustParamsForTesting verifies that a server can stand up in insecure mode.
func TestMustParamsForTesting(t *testing.T) {
	runServerStartStopTest(t, MustParamsForTesting())
}

// TestMustParamsForTestingTLS verifies that a server can stand up in TLS-mode.
func TestMustParamsForTestingTLS(t *testing.T) {
	runServerStartStopTest(t, MustParamsForTestingTLS())
}

func runServerStartStopTest(t *testing.T, p *serving.Params) {
	assert := assert.New(t)
	ff := &shellTesting.FakeFrontend{}
	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServer(s, ff)
	}, pb.RegisterFrontendHandlerFromEndpoint)
	s := &serving.Server{}
	defer s.Stop()
	waitForStart, err := s.Start(p)
	assert.Nil(err)
	waitForStart()

	s.Stop()
}
