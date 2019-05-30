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
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"open-match.dev/open-match/internal/util/netlistener"
)

const (
	// https://http2.github.io/http2-spec/#rfc.section.3.1
	http2WithTLSVersionID = "h2"
)

var (
	tlsServerLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "tls_server",
	})
)

type tlsServer struct {
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

func (s *tlsServer) start(params *ServerParams) (func(), error) {
	s.grpcServeWaiter = make(chan error)
	s.httpServeWaiter = make(chan error)
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
	s.grpcServer = grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	// Bind gRPC handlers
	for _, handlerFunc := range params.handlersForGrpc {
		handlerFunc(s.grpcServer)
	}

	serverStartWaiter.Add(1)
	go func() {
		serverStartWaiter.Done()
		s.grpcServeWaiter <- s.grpcServer.Serve(s.grpcListener)
	}()

	// Start HTTP server
	httpListener, err := s.httpLh.Obtain()
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	s.httpListener = httpListener
	// Bind gRPC handlers
	ctx := context.Background()

	httpsToGrpcProxyOptions := []grpc.DialOption{grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(certPoolForGrpcEndpoint, ""))}

	for _, handlerFunc := range params.handlersForGrpcProxy {
		if err = handlerFunc(ctx, s.proxyMux, grpcAddress, httpsToGrpcProxyOptions); err != nil {
			return func() {}, errors.WithStack(err)
		}
	}

	// Bind HTTPS handlers
	s.httpMux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	s.httpMux.Handle("/", s.proxyMux)
	zpages.Handle(s.httpMux, "/debug")
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
		s.httpServeWaiter <- s.httpServer.Serve(tlsListener)
	}()

	// Wait for the servers to come up.
	return serverStartWaiter.Wait, nil
}

func (s *tlsServer) stop() {
	s.grpcServer.Stop()
	if err := s.grpcListener.Close(); err != nil {
		tlsServerLogger.Error(err)
	}

	if err := s.httpServer.Close(); err != nil {
		tlsServerLogger.Error(err)
	}

	if err := s.httpListener.Close(); err != nil {
		tlsServerLogger.Error(err)
	}
}

func newTLSServer(grpcLh *netlistener.ListenerHolder, httpLh *netlistener.ListenerHolder) *tlsServer {
	return &tlsServer{
		grpcLh: grpcLh,
		httpLh: httpLh,
	}
}
