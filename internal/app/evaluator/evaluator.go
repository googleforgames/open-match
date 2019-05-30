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

package evaluator

import (
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/pb"
	"open-match.dev/open-match/internal/rpc"
)

var (
	evaluatorLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "evaluator",
	})
)

// FunctionSettings is a collection of parameters used to define the evaluator service.
type FunctionSettings struct {
	Func evaluatorFunction
}

// RunApplication creates a server.
func RunApplication(settings *FunctionSettings) {
	cfg, err := config.Read()
	if err != nil {
		evaluatorLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}

	p, err := rpc.NewServerParamsFromConfig(cfg, "api.evaluator")
	if err != nil {
		evaluatorLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := BindService(p, cfg, settings); err != nil {
		evaluatorLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("failed to bind evaluator service.")
	}

	rpc.MustServeForever(p)
}

// BindService creates the evaluator service and binds it to the serving harness.
func BindService(p *rpc.ServerParams, cfg config.View, fs *FunctionSettings) error {
	service, err := newEvaluator(cfg, fs)
	if err != nil {
		return err
	}

	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterEvaluatorServer(s, service)
	}, pb.RegisterEvaluatorHandlerFromEndpoint)

	return nil
}
