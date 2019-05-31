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

package backend

import (
	"testing"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
	rpcTesting "open-match.dev/open-match/internal/rpc/testing"
)

func TestServerBinding(t *testing.T) {
	bs := func(p *rpc.ServerParams) {
		p.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterBackendServer(s, &backendService{})
		}, pb.RegisterBackendHandlerFromEndpoint)
	}

	rpcTesting.TestServerBinding(t, bs)
}

func TestAssignTickets(t *testing.T) {
	// TODO: add test when test helper #462 is checked in
	// https://github.com/GoogleCloudPlatform/open-match/pull/462
}
