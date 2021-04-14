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
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
	"open-match.dev/open-match/internal/telemetry"
)

const (
	// https://http2.github.io/http2-spec/#rfc.section.3.1
	http2WithTLSVersionID = "h2"
)

type tlsServer struct {
	grpcListener net.Listener
	grpcServer   *grpc.Server

	httpListener net.Listener
	httpMux      *http.ServeMux
	proxyMux     *runtime.ServeMux
	httpServer   *http.Server
}

func (s *tlsServer) start(params *ServerParams) error {
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

	_, grpcPort, err := net.SplitHostPort(s.grpcListener.Addr().String())
	if err != nil {
		return err
	}
	grpcAddress := fmt.Sprintf("localhost:%s", grpcPort)

	rootCaCert, err := trustedCertificateFromFileData(params.rootCaPublicCertificateFileData)
	if err != nil {
		return errors.WithStack(err)
	}
	certPoolForGrpcEndpoint, err := trustedCertificateFromFileData(params.publicCertificateFileData)
	if err != nil {
		return errors.WithStack(err)
	}

	grpcTLSCertificate, err := certificateFromFileData(params.publicCertificateFileData, params.privateKeyFileData)
	if err != nil {
		return errors.WithStack(err)
	}
	creds := credentials.NewServerTLSFromCert(grpcTLSCertificate)
	serverOpts := newGRPCServerOptions(params)
	serverOpts = append(serverOpts, grpc.Creds(creds))
	s.grpcServer = grpc.NewServer(serverOpts...)

	// Bind gRPC handlers
	for _, handlerFunc := range params.handlersForGrpc {
		handlerFunc(s.grpcServer)
	}

	go func() {
		serverLogger.Infof("Serving gRPC-TLS: %s", s.grpcListener.Addr().String())
		gErr := s.grpcServer.Serve(s.grpcListener)
		if gErr != nil {
			serverLogger.Debugf("error closing gRPC-TLS server: %s", gErr)
		}
	}()

	// Start HTTP server
	// Bind gRPC handlers
	ctx, cancel := context.WithCancel(context.Background())

	httpsToGrpcProxyOptions := newGRPCDialOptions(params.enableMetrics, params.enableRPCLogging, params.enableRPCPayloadLogging)
	httpsToGrpcProxyOptions = append(httpsToGrpcProxyOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPoolForGrpcEndpoint, "")))

	for _, handlerFunc := range params.handlersForGrpcProxy {
		if err = handlerFunc(ctx, s.proxyMux, grpcAddress, httpsToGrpcProxyOptions); err != nil {
			cancel()
			return errors.WithStack(err)
		}
	}

	// Bind HTTPS handlers
	s.httpMux.Handle(telemetry.HealthCheckEndpoint, telemetry.NewHealthCheck(params.handlersForHealthCheck))
	s.httpMux.Handle("/", s.proxyMux)
	s.httpServer = &http.Server{
		Addr:    s.httpListener.Addr().String(),
		Handler: instrumentHTTPHandler(s.httpMux, params),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*grpcTLSCertificate},
			ClientCAs:    rootCaCert,
			// Commented as open-match does not support mutual authentication yet
			// ClientAuth:   tls.RequireAndVerifyClientCert,
			NextProtos: []string{http2WithTLSVersionID}, // https://github.com/grpc-ecosystem/grpc-gateway/issues/220
		},
	}
	go func() {
		tlsListener := tls.NewListener(s.httpListener, s.httpServer.TLSConfig)
		serverLogger.Infof("Serving HTTPS: %s", s.httpListener.Addr().String())
		hErr := s.httpServer.Serve(tlsListener)
		defer cancel()
		if hErr != nil && hErr != http.ErrServerClosed {
			serverLogger.Debugf("error serving HTTP: %s", hErr)
		}
	}()

	return nil
}

func (s *tlsServer) stop() error {
	// the servers also close their respective listeners.
	err := s.httpServer.Shutdown(context.Background())
	s.grpcServer.GracefulStop()
	return err
}

func newTLSServer(grpcL, httpL net.Listener) *tlsServer {
	return &tlsServer{
		grpcListener: grpcL,
		httpListener: httpL,
	}
}
