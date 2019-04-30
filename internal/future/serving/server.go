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
	"io/ioutil"

	"github.com/sirupsen/logrus"

	"github.com/GoogleCloudPlatform/open-match/internal/config"
	"github.com/GoogleCloudPlatform/open-match/internal/signal"
	"github.com/GoogleCloudPlatform/open-match/internal/util/netlistener"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

var (
	serverLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "server",
	})
)

// GrpcHandler binds gRPC services.
type GrpcHandler func(*grpc.Server)

// GrpcProxyHandler binds HTTP handler to gRPC service.
type GrpcProxyHandler func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error

// Params yeah
type Params struct {
	handlersForGrpc      []GrpcHandler
	handlersForGrpcProxy []GrpcProxyHandler
	grpcListener         *netlistener.ListenerHolder
	grpcProxyListener    *netlistener.ListenerHolder

	// TLS configuration
	rootCaPublicCertificateFileData []byte
	publicCertificateFileData       []byte
	privateKeyFileData              []byte
}

// NewParamsFromConfig returns.
func NewParamsFromConfig(cfg config.View, prefix string) (*Params, error) {
	grpcLh, err := netlistener.NewFromPortNumber(cfg.GetInt(prefix + ".port"))
	if err != nil {
		serverLogger.Fatal(err)
		return nil, err
	}
	httpLh, err := netlistener.NewFromPortNumber(cfg.GetInt(prefix + ".proxyport"))
	if err != nil {
		closeErr := grpcLh.Close()
		if closeErr != nil {
			serverLogger.WithFields(logrus.Fields{
				"error": closeErr.Error(),
			}).Info("failed to gRPC close port")
		}
		serverLogger.Fatal(err)
		return nil, err
	}
	p := NewParamsFromListeners(grpcLh, httpLh)

	certFile := cfg.GetString("api.tls.certificatefile")
	privateKeyFile := cfg.GetString("api.tls.privatekey")
	if len(certFile) > 0 && len(privateKeyFile) > 0 {
		serverLogger.Printf("Loading TLS certificate from, %s", certFile)
		publicCertData, err := ioutil.ReadFile(certFile)
		if err != nil {
			p.invalidate()
			return nil, errors.WithStack(err)
		}
		privateKeyData, err := ioutil.ReadFile(privateKeyFile)
		if err != nil {
			p.invalidate()
			return nil, errors.WithStack(err)
		}
		p.SetTLSConfiguration(publicCertData, publicCertData, privateKeyData)
	}
	return p, nil
}

// NewParamsFromListeners returns.
func NewParamsFromListeners(grpcLh *netlistener.ListenerHolder, proxyLh *netlistener.ListenerHolder) *Params {
	return &Params{
		handlersForGrpc:      []GrpcHandler{},
		handlersForGrpcProxy: []GrpcProxyHandler{},
		grpcListener:         grpcLh,
		grpcProxyListener:    proxyLh,
	}
}

// SetTLSConfiguration configures the server to run in TLS mode.
func (p *Params) SetTLSConfiguration(rootCaPublicCertificateFileData []byte, publicCertificateFileData []byte, privateKeyFileData []byte) *Params {
	p.rootCaPublicCertificateFileData = rootCaPublicCertificateFileData
	if len(p.rootCaPublicCertificateFileData) == 0 {
		p.rootCaPublicCertificateFileData = publicCertificateFileData
	}
	p.publicCertificateFileData = publicCertificateFileData
	p.privateKeyFileData = privateKeyFileData
	return p
}

// usingTLS returns true if a certificate is set.
func (p *Params) usingTLS() bool {
	return len(p.publicCertificateFileData) > 0
}

// AddHandleFunc binds gRPC service handler and an associated HTTP proxy handler.
func (p *Params) AddHandleFunc(handlerFunc GrpcHandler, grpcProxyHandler GrpcProxyHandler) *Params {
	if handlerFunc != nil {
		p.handlersForGrpc = append(p.handlersForGrpc, handlerFunc)
	}
	if grpcProxyHandler != nil {
		p.handlersForGrpcProxy = append(p.handlersForGrpcProxy, grpcProxyHandler)
	}
	return p
}

// invalidate closes all the TCP listeners that would otherwise leak if initialization fails.
func (p *Params) invalidate() {
	if err := p.grpcListener.Close(); err != nil {
		serverLogger.Errorf("error closing grpc handler, %s", err)
	}
	if err := p.grpcProxyListener.Close(); err != nil {
		serverLogger.Errorf("error closing grpc-proxy handler, %s", err)
	}
}

// Server hosts a gRPC and HTTP server.
// All HTTP traffic is served from a common http.ServeMux.
type Server struct {
	serverWithProxy grpcServerWithProxy
}

// grpcServerWithProxy this will go away when insecure.go and tls.go are merged into the same server.
type grpcServerWithProxy interface {
	start(*Params) (func(), error)
	stop()
}

// Start the gRPC+HTTP(s) REST server.
func (s *Server) Start(p *Params) (func(), error) {
	if p.usingTLS() {
		s.serverWithProxy = newTLSServer(p.grpcListener, p.grpcProxyListener)
	} else {
		s.serverWithProxy = newInsecureServer(p.grpcListener, p.grpcProxyListener)
	}
	return s.serverWithProxy.start(p)
}

// Stop the gRPC+HTTP(s) REST server.
func (s *Server) Stop() {
	s.serverWithProxy.stop()
}

// New creates an instance of an Open Match server.
func New() *Server {
	return &Server{}
}

func startServingIndefinitely(params *Params) (func(), func(), error) {
	s := New()

	// Start serving traffic.
	waitForServing, err := s.Start(params)
	if err != nil {
		serverLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Failed to start gRPC and HTTP servers.")
		return func() {}, func() {}, err
	}
	serverLogger.Info("Server has started.")
	// Exit when we see a signal
	waitUntilKilled, stopServingFunc := signal.New()

	waitForServing()
	serveUntilKilledFunc := func() {
		waitUntilKilled()
		s.Stop()
		serverLogger.Info("Shutting down server")
	}
	return serveUntilKilledFunc, stopServingFunc, nil
}

// MustServeForever is a convenience method for starting a server and running it indefinitely.
func MustServeForever(params *Params) {
	serveUntilKilledFunc, _, err := startServingIndefinitely(params)
	if err != nil {
		return
	}
	serveUntilKilledFunc()
}
