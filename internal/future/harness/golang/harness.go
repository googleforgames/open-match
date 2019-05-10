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

package golang

import (
	"fmt"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/serving"

	"github.com/sirupsen/logrus"
)

var (
	harnessLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "matchfunction.golang.harness",
	})
)

// FunctionSettings is a collection of parameters used to customize matchfunction views.
type FunctionSettings struct {
	FunctionName string
	Func         MatchFunction
}

// RunMatchFunction is a hook for the main() method in the main executable.
func RunMatchFunction(settings *FunctionSettings) {
	cfg, err := config.Read()
	if err != nil {
		harnessLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	p, err := serving.NewParamsFromConfig(cfg, "api.functions")
	if err != nil {
		harnessLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := BindService(p, cfg, settings); err != nil {
		harnessLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("failed to bind functions service.")
	}

	serving.MustServeForever(p)
}

// BindService creates the function service to the server Params.
func BindService(p *serving.Params, cfg config.View, fs *FunctionSettings) error {
	service, err := newMatchFunctionService(cfg, fs)
	if err != nil {
		return err
	}

	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterMatchFunctionServer(s, service)
	}, pb.RegisterMatchFunctionHandlerFromEndpoint)

	return nil
}

func newMatchFunctionService(cfg config.View, fs *FunctionSettings) (*matchFunctionService, error) {
	mmlogicClient, err := getMMLogicClient(cfg)
	if err != nil {
		harnessLogger.Errorf("Failed to get MMLogic client, %v.", err)
		return nil, err
	}

	// Placeholder for things that needs to be done with the cfg in future.

	mmfService := &matchFunctionService{cfg: cfg, functionName: fs.FunctionName, function: fs.Func, mmlogicClient: mmlogicClient}

	return mmfService, nil
}

// TODO: replace this method once the client side wrapper is done.
func getMMLogicClient(cfg config.View) (pb.MmLogicClient, error) {
	host := cfg.GetString("api.mmlogic.hostname")
	if len(host) == 0 {
		return nil, fmt.Errorf("Failed to get hostname for MMLogicAPI from the configuration")
	}

	port := cfg.GetString("api.mmlogic.port")
	if len(port) == 0 {
		return nil, fmt.Errorf("Failed to get port for MMLogicAPI from the configuration")
	}

	conn, err := grpc.Dial(fmt.Sprintf("%v:%v", host, port), grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %v, %v", fmt.Sprintf("%v:%v", host, port), err)
	}

	return pb.NewMmLogicClient(conn), nil
}
