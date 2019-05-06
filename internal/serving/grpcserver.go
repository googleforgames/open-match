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

// Package serving provides a serving gRPC and HTTP proxy server harness for Open Match.
package serving

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/zpages"
	"open-match.dev/open-match/internal/util/netlistener"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// GrpcWrapper is a decoration around the standard GRPC server that sets up a bunch of things common to Open Match servers.
type GrpcWrapper struct {
	serviceLh, proxyLh        *netlistener.ListenerHolder
	serviceHandlerFuncs       []func(*grpc.Server)
	proxyHandlerFuncs         []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
	server                    *grpc.Server
	proxy                     *http.Server
	serviceLn, proxyLn        net.Listener
	logger                    *logrus.Entry
	grpcAwaiter, proxyAwaiter chan error
	grpcClient                *grpc.ClientConn
}

// NewGrpcServer creates a new GrpcWrapper.
func NewGrpcServer(serviceLh, proxyLh *netlistener.ListenerHolder, logger *logrus.Entry) *GrpcWrapper {
	return &GrpcWrapper{
		serviceLh:           serviceLh,
		proxyLh:             proxyLh,
		logger:              logger,
		serviceHandlerFuncs: []func(*grpc.Server){},
		proxyHandlerFuncs:   []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error{},
	}
}

// AddService adds a service registration function to be run when the server is created.
func (gw *GrpcWrapper) AddService(handlerFunc func(*grpc.Server)) {
	gw.serviceHandlerFuncs = append(gw.serviceHandlerFuncs, handlerFunc)
}

// AddProxy registers a reverse proxy from REST to gRPC when server is created.
func (gw *GrpcWrapper) AddProxy(handlerFunc func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error) {
	gw.proxyHandlerFuncs = append(gw.proxyHandlerFuncs, handlerFunc)
}

// Start begins the gRPC server.
func (gw *GrpcWrapper) Start() error {
	var wg sync.WaitGroup
	if err := gw.startGrpcServer(&wg); err != nil {
		return err
	}

	if err := gw.startHTTPProxy(&wg); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

// WaitForTermination blocks until the gRPC server is shutdown.
func (gw *GrpcWrapper) WaitForTermination() (error, error) {
	var grpcErr, proxyErr error

	if gw.grpcAwaiter != nil {
		grpcErr = <-gw.grpcAwaiter
		close(gw.grpcAwaiter)
		gw.grpcAwaiter = nil
	}

	if gw.proxyAwaiter != nil {
		proxyErr = <-gw.proxyAwaiter
		close(gw.proxyAwaiter)
		gw.proxyAwaiter = nil
	}
	return grpcErr, proxyErr
}

// Stop gracefully shutsdown the gRPC server.
func (gw *GrpcWrapper) Stop() error {
	if gw.server == nil {
		return nil
	}
	gw.server.GracefulStop()

	if gw.proxy == nil {
		return nil
	}

	err := gw.proxy.Shutdown(context.Background())
	if err != nil {
		return err
	}

	err = gw.serviceLn.Close()
	if err != nil {
		return err
	}
	err = gw.proxyLn.Close()
	if err != nil {
		return err
	}

	grpcClientErr := gw.grpcClient.Close()
	grpcErr, proxyErr := gw.WaitForTermination()

	gw.server = nil
	gw.serviceLn = nil
	gw.proxy = nil
	gw.proxyLn = nil

	if grpcClientErr != nil {
		return grpcClientErr
	}
	if grpcErr != nil {
		return grpcErr
	}
	if proxyErr != nil {
		return proxyErr
	}
	return nil
}

func (gw *GrpcWrapper) startHTTPProxy(wg *sync.WaitGroup) error {
	// Starting proxy server
	if gw.proxyLn != nil {
		return nil
	}

	wg.Add(1)

	proxyLn, err := gw.proxyLh.Obtain()

	if err != nil {
		gw.logger.WithFields(logrus.Fields{
			"error":     err.Error(),
			"proxyPort": gw.proxyLh.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.proxyLn = proxyLn

	gw.logger.WithFields(logrus.Fields{"proxyPort": gw.proxyLh.Number()}).Info("TCP net listener initialized")

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	proxyMux := runtime.NewServeMux()

	serviceEndpoint := gw.serviceLh.AddrString()
	gw.grpcClient, err = grpc.DialContext(ctx, serviceEndpoint, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		gw.logger.WithFields(logrus.Fields{
			"error":           err.Error(),
			"serviceEndpoint": serviceEndpoint,
		}).Error("grpc Dialing error")
		return err
	}

	for _, handlerFunc := range gw.proxyHandlerFuncs {
		if err := handlerFunc(ctx, proxyMux, gw.grpcClient); err != nil {
			return errors.WithStack(err)
		}
	}

	httpMux := http.NewServeMux()
	httpMux.HandleFunc("/healthz", healthzServer(gw.grpcClient))
	httpMux.HandleFunc("/swagger/", swaggerServer("."))
	httpMux.Handle("/", proxyMux)
	zpages.Handle(httpMux, "/debug")

	gw.proxy = &http.Server{Handler: httpMux}
	gw.proxyAwaiter = make(chan error)

	go func() {
		wg.Done()
		gw.logger.Infof("Serving proxy on %s", gw.proxyLh.AddrString())
		err := gw.proxy.Serve(proxyLn)
		gw.proxyAwaiter <- err
		if err != nil {
			gw.logger.WithFields(logrus.Fields{
				"error":           err.Error(),
				"serviceEndpoint": serviceEndpoint,
				"proxyPort":       gw.proxyLh.Number(),
			}).Error("proxy ListenAndServe() error")
		}
	}()

	return nil
}

func (gw *GrpcWrapper) startGrpcServer(wg *sync.WaitGroup) error {
	// Starting gRPC server
	if gw.serviceLn != nil {
		return nil
	}

	wg.Add(1)
	serviceLn, err := gw.serviceLh.Obtain()
	if err != nil {
		gw.logger.WithFields(logrus.Fields{
			"error":       err.Error(),
			"servicePort": gw.serviceLh.Number(),
		}).Error("net.Listen() error")
		return err
	}
	gw.serviceLn = serviceLn

	gw.logger.WithFields(logrus.Fields{
		"servicePort": gw.serviceLh.Number(),
	}).Info("TCP net listener initialized")

	server := grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	for _, handlerFunc := range gw.serviceHandlerFuncs {
		handlerFunc(server)
	}
	gw.server = server
	gw.grpcAwaiter = make(chan error)

	go func() {
		wg.Done()
		gw.logger.Infof("Serving gRPC on %s", gw.serviceLh.AddrString())
		err := gw.server.Serve(serviceLn)
		gw.grpcAwaiter <- err
		if err != nil {
			gw.logger.WithFields(logrus.Fields{"error": err.Error()}).Error("gRPC serve() error")
		}
	}()

	return nil
}

// healthzServer returns a simple health handler which returns ok.
func healthzServer(conn *grpc.ClientConn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if s := conn.GetState(); s != connectivity.Ready {
			http.Error(w, fmt.Sprintf("Service Unavailable: ClientConn is %s", s), http.StatusServiceUnavailable)
			return
		}
		fmt.Fprintln(w, "ok")
	}
}

// swaggerServer returns a file serving handler for .swagger.json files
func swaggerServer(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
			http.NotFound(w, r)
			return
		}

		p := strings.TrimPrefix(r.URL.Path, "/swagger/")
		p = path.Join(dir, p)
		http.ServeFile(w, r, p)
	}
}
