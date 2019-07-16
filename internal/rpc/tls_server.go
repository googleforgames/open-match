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
	"net/http"
	"sync"

	"crypto/tls"
	"net"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"open-match.dev/open-match/internal/telemetry"
	"open-match.dev/open-match/internal/util/netlistener"
)

const (
	// https://http2.github.io/http2-spec/#rfc.section.3.1
	http2WithTLSVersionID = "h2"
)

type tlsServer struct {
	grpcLh       *netlistener.ListenerHolder
	grpcListener net.Listener
	grpcServer   *grpc.Server

	httpLh       *netlistener.ListenerHolder
	httpListener net.Listener
	httpMux      *http.ServeMux
	proxyMux     *runtime.ServeMux
	httpServer   *http.Server
}

func (s *tlsServer) start(params *ServerParams) (func(), error) {
	var serverStartWaiter sync.WaitGroup

	s.httpMux = params.ServeMux
	s.proxyMux = runtime.NewServeMux()

	grpcAddress := fmt.Sprintf("localhost:%d", s.grpcLh.Number())

	grpcListener, err := s.grpcLh.Obtain()
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	s.grpcListener = grpcListener

	rootCaCert, err := trustedCertificateFromFileData(params.rootCaPublicCertificateFileData)
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	certPoolForGrpcEndpoint, err := trustedCertificateFromFileData(params.publicCertificateFileData)
	if err != nil {
		return func() {}, errors.WithStack(err)
	}

	grpcTLSCertificate, err := certificateFromFileData(params.publicCertificateFileData, params.privateKeyFileData)
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	creds := credentials.NewServerTLSFromCert(grpcTLSCertificate)
	serverOpts := newGRPCServerOptions(params)
	serverOpts = append(serverOpts, grpc.Creds(creds))
	s.grpcServer = grpc.NewServer(serverOpts...)

	// Bind gRPC handlers
	for _, handlerFunc := range params.handlersForGrpc {
		handlerFunc(s.grpcServer)
	}

	serverStartWaiter.Add(1)
	go func() {
		serverStartWaiter.Done()
		serverLogger.Infof("Serving gRPC-TLS: %s", s.grpcLh.AddrString())
		gErr := s.grpcServer.Serve(s.grpcListener)
		if gErr != nil {
			serverLogger.Debugf("error closing gRPC-TLS server: %s", gErr)
		}
	}()

	// Start HTTP server
	httpListener, err := s.httpLh.Obtain()
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	s.httpListener = httpListener
	// Bind gRPC handlers
	ctx, cancel := context.WithCancel(context.Background())

	httpsToGrpcProxyOptions := newGRPCDialOptions(params.enableMetrics, params.enableRPCLogging)
	httpsToGrpcProxyOptions = append(httpsToGrpcProxyOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPoolForGrpcEndpoint, "")))

	for _, handlerFunc := range params.handlersForGrpcProxy {
		if err = handlerFunc(ctx, s.proxyMux, grpcAddress, httpsToGrpcProxyOptions); err != nil {
			cancel()
			return func() {}, errors.WithStack(err)
		}
	}

	// Bind HTTPS handlers
	s.httpMux.Handle(telemetry.HealthCheckEndpoint, telemetry.NewHealthCheck(params.handlersForHealthCheck))
	s.httpMux.Handle("/", s.proxyMux)
	s.httpServer = &http.Server{
		Addr:    s.httpListener.Addr().String(),
		Handler: s.httpMux,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{*grpcTLSCertificate},
			ClientCAs:    rootCaCert,
			// Commented as open-match does not support mutual authentication yet
			// ClientAuth:   tls.RequireAndVerifyClientCert,
			NextProtos: []string{http2WithTLSVersionID}, // https://github.com/grpc-ecosystem/grpc-gateway/issues/220
		},
	}
	serverStartWaiter.Add(1)
	go func() {
		serverStartWaiter.Done()
		tlsListener := tls.NewListener(s.httpListener, s.httpServer.TLSConfig)
		serverLogger.Infof("Serving HTTPS: %s", s.httpLh.AddrString())
		hErr := s.httpServer.Serve(tlsListener)
		defer cancel()
		if hErr != nil {
			serverLogger.Debugf("error closing server: %s", hErr)
		}
	}()

	// Wait for the servers to come up.
	return serverStartWaiter.Wait, nil
}

func (s *tlsServer) stop() {
	s.grpcServer.Stop()
	if err := s.grpcListener.Close(); err != nil {
		serverLogger.Debugf("error closing gRPC-TLS listener: %s", err)
	}

	if err := s.httpServer.Close(); err != nil {
		serverLogger.Debugf("error closing HTTPS server: %s", err)
	}

	if err := s.httpListener.Close(); err != nil {
		serverLogger.Debugf("error closing HTTPS listener: %s", err)
	}
}

func newTLSServer(grpcLh *netlistener.ListenerHolder, httpLh *netlistener.ListenerHolder) *tlsServer {
	return &tlsServer{
		grpcLh: grpcLh,
		httpLh: httpLh,
	}
}
