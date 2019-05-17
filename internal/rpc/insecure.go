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

package rpc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"

	"net"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/util/netlistener"
)

type insecureServer struct {
	grpcLh          *netlistener.ListenerHolder
	grpcListener    net.Listener
	grpcServeWaiter chan error
	grpcServer      *grpc.Server

	httpLh          *netlistener.ListenerHolder
	httpListener    net.Listener
	httpServeWaiter chan error
	httpMux         *http.ServeMux
	proxyMux        *runtime.ServeMux
	httpServer      *http.Server
}

func (s *insecureServer) start(params *ServerParams) (func(), error) {
	s.grpcServeWaiter = make(chan error)
	s.httpServeWaiter = make(chan error)
	var serverStartWaiter sync.WaitGroup

	s.httpMux = params.ServeMux
	s.proxyMux = runtime.NewServeMux()

	// Configure the gRPC server.
	grpcListener, err := s.grpcLh.Obtain()
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	s.grpcListener = grpcListener
	s.grpcServer = grpc.NewServer(grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	// Bind gRPC handlers
	for _, handlerFunc := range params.handlersForGrpc {
		handlerFunc(s.grpcServer)
	}

	serverStartWaiter.Add(1)
	go func() {
		serverStartWaiter.Done()
		s.grpcServeWaiter <- s.grpcServer.Serve(s.grpcListener)
	}()

	// Configure the HTTP proxy server.
	httpListener, err := s.httpLh.Obtain()
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	s.httpListener = httpListener

	// Bind gRPC handlers
	ctx := context.Background()

	for _, handlerFunc := range params.handlersForGrpcProxy {
		if err = handlerFunc(ctx, s.proxyMux, grpcListener.Addr().String(), []grpc.DialOption{grpc.WithInsecure()}); err != nil {
			return func() {}, errors.WithStack(err)
		}
	}

	s.httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	s.httpMux.Handle("/", s.proxyMux)
	zpages.Handle(s.httpMux, "/debug")
	s.httpServer = &http.Server{
		Addr:    s.httpListener.Addr().String(),
		Handler: s.httpMux,
	}
	serverStartWaiter.Add(1)
	go func() {
		serverStartWaiter.Done()
		s.httpServeWaiter <- s.httpServer.Serve(s.httpListener)
	}()

	return serverStartWaiter.Wait, nil
}

func (s *insecureServer) stop() {
	s.grpcServer.Stop()
	if err := s.grpcListener.Close(); err != nil {
		log.Printf("%s", err)
	}

	if err := s.httpServer.Close(); err != nil {
		log.Printf("%s", err)
	}

	if err := s.httpListener.Close(); err != nil {
		log.Printf("%s", err)
	}
}

func newInsecureServer(grpcLh *netlistener.ListenerHolder, httpLh *netlistener.ListenerHolder) *insecureServer {
	return &insecureServer{
		grpcLh: grpcLh,
		httpLh: httpLh,
	}
}
