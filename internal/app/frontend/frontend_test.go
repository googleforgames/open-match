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

package frontend

import (
	"testing"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/serving"
	servingTesting "open-match.dev/open-match/internal/serving/testing"
)

func TestServerBinding(t *testing.T) {
	bs := func(p *serving.Params) {
		p.AddHandleFunc(func(s *grpc.Server) {
			pb.RegisterFrontendServer(s, &frontendService{})
		}, pb.RegisterFrontendHandlerFromEndpoint)
	}

	servingTesting.TestServerBinding(t, bs)
}
