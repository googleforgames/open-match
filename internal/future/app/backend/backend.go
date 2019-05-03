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
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/serving"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	backendLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "backend",
	})
)

// RunApplication creates a server.
func RunApplication() {
	cfg, err := config.Read()
	if err != nil {
		backendLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	p, err := serving.NewParamsFromConfig(cfg, "api.backend")
	if err != nil {
		backendLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	service := &backendService{}
	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterBackendServer(s, service)
	}, pb.RegisterBackendHandlerFromEndpoint)
	serving.MustServeForever(p)
}
