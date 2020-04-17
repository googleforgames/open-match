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
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/telemetry"
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
	c := make(chan os.Signal)
	// SIGTERM is signaled by k8s when it wants a pod to stop.
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	a, err := StartApplication(serverName, bindService, config.Read, net.Listen)
	if err != nil {
		logger.Fatal(err)
	}

	<-c
	err = a.Stop()
	if err != nil {
		logger.Fatal(err)
	}
	logger.Info("Application stopped successfully.")
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

func (p *Params) ServiceName() string {
	////////////////////////TODO
	return ""
}

func (p *Params) Now() func() time.Time {
	////////////////////////TODO
	return time.Now
}

// Bindings allows applications to bind various functions to the running servers.
type Bindings struct {
	sp *rpc.ServerParams
	a  *App
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

func (b *Bindings) TelementryHandle(pattern string, handler http.Handler) {
	b.sp.ServeMux.Handle(pattern, handler)
}

func (b *Bindings) TelementryHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	b.sp.ServeMux.HandleFunc(pattern, handler)
}

func (b *Bindings) AddCloser(c func()) {
	b.a.closers = append(b.a.closers, func() error {
		c()
		return nil
	})
}

func (b *Bindings) AddCloserErr(c func() error) {
	b.a.closers = append(b.a.closers, c)
}

type App struct {
	closers []func() error
}

// StartApplication provides more control over an application than
// RunApplication.  It is for running in memory tests against your app.
func StartApplication(serverName string, bindService Bind, getCfg func() (config.View, error), listen func(network, address string) (net.Listener, error)) (*App, error) {
	a := &App{}

	cfg, err := getCfg()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)
	sp, err := rpc.NewServerParamsFromConfig(cfg, "api."+serverName, listen)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	p := &Params{
		config: cfg,
	}
	b := &Bindings{
		a:  a,
		sp: sp,
	}

	err = telemetry.Setup(p, b)
	if err != nil {
		a.Stop()
		return nil, err
	}

	err = bindService(p, b)
	if err != nil {
		a.Stop()
		return nil, err
	}

	s := &rpc.Server{}
	err = s.Start(sp)
	if err != nil {
		a.Stop()
		return nil, err
	}
	b.AddCloser(s.Stop)

	// rpc.MustServeForever(sp)

	return a, nil
}

func (a *App) Stop() error {
	// Use closers in reverse order: Since dependencies are created before
	// their dependants, this helps ensure no dependencies are closed
	// unexpectedly.
	var firstErr error
	for i := len(a.closers) - 1; i >= 0; i-- {
		err := a.closers[i]()
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
