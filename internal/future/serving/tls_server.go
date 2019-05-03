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

package serving

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"crypto/tls"
	"crypto/x509"
	"net"

	"open-match.dev/open-match/internal/util/netlistener"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/zpages"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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

// ClientCredentialsFromFile gets TransportCredentials from public certificate.
func ClientCredentialsFromFile(certPath string) (credentials.TransportCredentials, error) {
	publicCertFileData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", certPath)
	}
	return ClientCredentialsFromFileData(publicCertFileData, "")
}

// TrustedCertificates gets TransportCredentials from public certificate file contents.
func TrustedCertificates(publicCertFileData []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return pool, nil
}

// ClientCredentialsFromFileData gets TransportCredentials from public certificate file contents.
func ClientCredentialsFromFileData(publicCertFileData []byte, serverOverride string) (credentials.TransportCredentials, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return credentials.NewClientTLSFromCert(pool, serverOverride), nil
}

// CertificateFromFiles returns a tls.Certificate from PEM encoded X.509 public certificate and RSA private key files.
func CertificateFromFiles(publicCertificatePath string, privateKeyPath string) (*tls.Certificate, error) {
	publicCertFileData, err := ioutil.ReadFile(publicCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", publicCertificatePath)
	}
	privateKeyFileData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read private key from %s", privateKeyPath)
	}
	return CertificateFromFileData(publicCertFileData, privateKeyFileData)
}

// CertificateFromFileData returns a tls.Certificate from PEM encoded file data.
func CertificateFromFileData(publicCertFileData []byte, privateKeyFileData []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(publicCertFileData, privateKeyFileData)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cert, nil
}

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

func (s *tlsServer) start(params *Params) (func(), error) {
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

	rootCaCert, err := TrustedCertificates(params.rootCaPublicCertificateFileData)
	if err != nil {
		return func() {}, errors.WithStack(err)
	}
	credsForProxyToGrpc, err := ClientCredentialsFromFileData(params.publicCertificateFileData, "")
	if err != nil {
		return func() {}, errors.WithStack(err)
	}

	grpcTLSCertificate, err := CertificateFromFileData(params.publicCertificateFileData, params.privateKeyFileData)
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

	httpsToGrpcProxyOptions := []grpc.DialOption{grpc.WithTransportCredentials(credsForProxyToGrpc)}

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
			ClientAuth:   tls.RequireAndVerifyClientCert,
			NextProtos:   []string{http2WithTLSVersionID}, // https://github.com/grpc-ecosystem/grpc-gateway/issues/220
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
