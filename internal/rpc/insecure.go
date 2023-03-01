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
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"open-match.dev/open-match/internal/telemetry"
)

type insecureServer struct {
	grpcListener net.Listener
	grpcServer   *grpc.Server

	httpListener net.Listener
	httpMux      *http.ServeMux
	proxyMux     *runtime.ServeMux
	httpServer   *http.Server
}

func (s *insecureServer) start(params *ServerParams) error {
	s.httpMux = params.ServeMux
	s.proxyMux = runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.HTTPBodyMarshaler{
			Marshaler: &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					UseProtoNames:   true,
					EmitUnpopulated: false,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
	)

	// Configure the gRPC server.
	s.grpcServer = grpc.NewServer(newGRPCServerOptions(params)...)
	// Bind gRPC handlers
	for _, handlerFunc := range params.handlersForGrpc {
		handlerFunc(s.grpcServer)
	}

	go func() {
		serverLogger.Infof("Serving gRPC: %s", s.grpcListener.Addr().String())
		gErr := s.grpcServer.Serve(s.grpcListener)
		if gErr != nil {
			return
		}
	}()

	// Configure the HTTP proxy server.
	// Bind gRPC handlers
	ctx, cancel := context.WithCancel(context.Background())

	for _, handlerFunc := range params.handlersForGrpcProxy {
		dialOpts := newGRPCDialOptions(params.enableMetrics, params.enableRPCLogging, params.enableRPCPayloadLogging)
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err := handlerFunc(ctx, s.proxyMux, s.grpcListener.Addr().String(), dialOpts); err != nil {
			cancel()
			return errors.WithStack(err)
		}
	}

	s.httpMux.Handle(telemetry.HealthCheckEndpoint, telemetry.NewHealthCheck(params.handlersForHealthCheck))
	s.httpMux.Handle("/", s.proxyMux)
	s.httpServer = &http.Server{
		Addr:    s.httpListener.Addr().String(),
		Handler: instrumentHTTPHandler(s.httpMux, params),
	}
	go func() {
		serverLogger.Infof("Serving HTTP: %s", s.httpListener.Addr().String())
		hErr := s.httpServer.Serve(s.httpListener)
		defer cancel()
		if hErr != nil && hErr != http.ErrServerClosed {
			serverLogger.Debugf("error serving HTTP: %s", hErr)
		}
	}()

	return nil
}

func (s *insecureServer) stop() error {
	// the servers also close their respective listeners.
	err := s.httpServer.Shutdown(context.Background())
	s.grpcServer.GracefulStop()
	return err
}

func newInsecureServer(grpcL, httpL net.Listener) *insecureServer {
	return &insecureServer{
		grpcListener: grpcL,
		httpListener: httpL,
	}
}
