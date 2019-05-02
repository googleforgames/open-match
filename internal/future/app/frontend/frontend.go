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
	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/future/pb"
	"github.com/GoogleCloudPlatform/open-match/internal/future/serving"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

var (
	frontendLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "frontend",
	})
)

// RunApplication creates a server.
func RunApplication() {
	cfg, err := config.Read()
	if err != nil {
		frontendLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	p, err := serving.NewParamsFromConfig(cfg, "api.frontend")
	if err != nil {
		frontendLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	service := &frontendService{}
	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServer(s, service)
	}, pb.RegisterFrontendHandlerFromEndpoint)
	// TODO: Requires #330
	//serving.MustServeForever(p)
}
