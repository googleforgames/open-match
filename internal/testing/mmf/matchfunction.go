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

// Package mmf provides the Match Making Function service for Open Match golang harness.
package mmf

import (
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/app"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/pkg/pb"
)

// FunctionSettings is a collection of parameters used to customize matchfunction views.
type FunctionSettings struct {
	Func MatchFunction
}

// RunMatchFunction is a hook for the main() method in the main executable.
func RunMatchFunction(settings *FunctionSettings) {
	app.RunApplication("functions", getCfg, func(p *rpc.ServerParams, cfg config.View) error {
		return BindService(p, cfg, settings)
	})
}

// BindService creates the function service to the server Params.
func BindService(p *rpc.ServerParams, cfg config.View, fs *FunctionSettings) error {
	service, err := newMatchFunctionService(cfg, fs)
	if err != nil {
		return err
	}

	p.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterMatchFunctionServer(s, service)
	}, pb.RegisterMatchFunctionHandlerFromEndpoint)

	return nil
}

func getCfg() (config.View, error) {
	cfg := viper.New()

	cfg.Set("api.functions.hostname", "om-function")
	cfg.Set("api.functions.grpcport", 50502)
	cfg.Set("api.functions.httpport", 51502)

	cfg.Set("api.mmlogic.hostname", "om-mmlogic")
	cfg.Set("api.mmlogic.grpcport", 50503)

	return cfg, nil
}
