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

// Package appmain contains the common application initialization code for Open Match servers.
package appmain

import (
	"context"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
)

var (
	logger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "app.main",
	})
)

// RunApplication starts and runs the given application forever.  For use in
// main functions to run the full application.
func RunApplication(serverName string, bindService Bind) {
	StartApplication(serverName, bindService, config.Read)
}

// Bind is a function which starts an application, and binds it to serving.
type Bind func(p *Params, b *Bindings) error

// Params are inputs to starting an application.
type Params struct {
	config config.View
}

// Config provides the configuration for the application.
func (p *Params) Config() config.View {
	return p.config
}

// Bindings allows applications to bind various functions to the running servers.
type Bindings struct {
	sp *rpc.ServerParams
}

// AddHealthCheckFunc allows an application to check if it is healthy, and
// contribute to the overall server health.
func (b *Bindings) AddHealthCheckFunc(f func(context.Context) error) {
	b.sp.AddHealthCheckFunc(f)
}

// AddHandleFunc adds a protobuf service to the grpc server which is starting.
func (b *Bindings) AddHandleFunc(handlerFunc rpc.GrpcHandler, grpcProxyHandler rpc.GrpcProxyHandler) {
	b.sp.AddHandleFunc(handlerFunc, grpcProxyHandler)
}

// StartApplication provides more control over an application than
// RunApplication.  It is for running in memory tests against your app.
func StartApplication(serverName string, bindService Bind, getCfg func() (config.View, error)) {
	// TODO: Provide way to shut down application, closing all servers and cleaning up resources.

	cfg, err := getCfg()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)
	sp, err := rpc.NewServerParamsFromConfig(cfg, "api."+serverName)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	if err := TemporaryBindWrapper(bindService, sp, cfg); err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("failed to bind %s service.", serverName)
	}

	rpc.MustServeForever(sp)
}

// TemporaryBindWrapper exists as a shim for testing only.  Once StartApplication
// can correctly clean up its resources, it will be used instead.
func TemporaryBindWrapper(bind Bind, sp *rpc.ServerParams, cfg config.View) error {
	p := &Params{
		config: cfg,
	}
	b := &Bindings{
		sp: sp,
	}
	return bind(p, b)
}
