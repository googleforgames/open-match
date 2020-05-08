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

	"go.opencensus.io/stats/view"

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
func RunApplication(serviceName string, bindService Bind) {
	c := make(chan os.Signal, 1)
	// SIGTERM is signaled by k8s when it wants a pod to stop.
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)

	readConfig := func() (config.View, error) {
		return config.Read()
	}

	a, err := NewApplication(serviceName, bindService, readConfig, net.Listen)
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
	config      config.View
	serviceName string
}

// Config provides the configuration for the application.
func (p *Params) Config() config.View {
	return p.config
}

// ServiceName is a name for the currently running binary specified by
// RunApplication.
func (p *Params) ServiceName() string {
	return p.serviceName
}

// Bindings allows applications to bind various functions to the running servers.
type Bindings struct {
	sp       *rpc.ServerParams
	a        *App
	firstErr error
}

// AddHealthCheckFunc allows an application to check if it is healthy, and
// contribute to the overall server health.
func (b *Bindings) AddHealthCheckFunc(f func(context.Context) error) {
	b.sp.AddHealthCheckFunc(f)
}

// RegisterViews begins collecting data for the given views.
func (b *Bindings) RegisterViews(v ...*view.View) {
	if err := view.Register(v...); err != nil {
		if b.firstErr == nil {
			b.firstErr = err
		}
		return
	}

	b.AddCloser(func() {
		view.Unregister(v...)
	})
}

// AddHandleFunc adds a protobuf service to the grpc server which is starting.
func (b *Bindings) AddHandleFunc(handlerFunc rpc.GrpcHandler, grpcProxyHandler rpc.GrpcProxyHandler) {
	b.sp.AddHandleFunc(handlerFunc, grpcProxyHandler)
}

// TelemetryHandle adds a handler to the mux for serving debug info and metrics.
func (b *Bindings) TelemetryHandle(pattern string, handler http.Handler) {
	b.sp.ServeMux.Handle(pattern, handler)
}

// TelemetryHandleFunc adds a handlerfunc to the mux for serving debug info and metrics.
func (b *Bindings) TelemetryHandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	b.sp.ServeMux.HandleFunc(pattern, handler)
}

// AddCloser specifies a function to be called when the application is being
// stopped.  Closers are called in reverse order.
func (b *Bindings) AddCloser(c func()) {
	b.a.closers = append(b.a.closers, func() error {
		c()
		return nil
	})
}

// AddCloserErr specifies a function to be called when the application is being
// stopped.  Closers are called in reverse order.  The first error returned by
// a closer will be logged.
func (b *Bindings) AddCloserErr(c func() error) {
	b.a.closers = append(b.a.closers, c)
}

// App is used internally, and public only for apptest.  Do not use, and use apptest instead.
type App struct {
	closers []func() error
}

// NewApplication is used internally, and public only for apptest.  Do not use, and use apptest instead.
func NewApplication(serviceName string, bindService Bind, getCfg func() (config.View, error), listen func(network, address string) (net.Listener, error)) (*App, error) {
	a := &App{}

	cfg, err := getCfg()
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot read configuration.")
	}
	logging.ConfigureLogging(cfg)
	sp, err := rpc.NewServerParamsFromConfig(cfg, "api."+serviceName, listen)
	if err != nil {
		logger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatalf("cannot construct server.")
	}

	p := &Params{
		config:      cfg,
		serviceName: serviceName,
	}
	b := &Bindings{
		a:  a,
		sp: sp,
	}

	err = telemetry.Setup(p, b)
	if err != nil {
		surpressedErr := a.Stop() // Don't care about additional errors stopping.
		_ = surpressedErr
		return nil, err
	}

	err = bindService(p, b)
	if err != nil {
		surpressedErr := a.Stop() // Don't care about additional errors stopping.
		_ = surpressedErr
		return nil, err
	}
	if b.firstErr != nil {
		surpressedErr := a.Stop() // Don't care about additional errors stopping.
		_ = surpressedErr
		return nil, b.firstErr
	}

	s := &rpc.Server{}
	err = s.Start(sp)
	if err != nil {
		surpressedErr := a.Stop() // Don't care about additional errors stopping.
		_ = surpressedErr
		return nil, err
	}
	b.AddCloserErr(s.Stop)

	return a, nil
}

// Stop is used internally, and public only for apptest.  Do not use, and use apptest instead.
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
