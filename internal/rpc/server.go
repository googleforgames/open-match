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
	"io/ioutil"
	"net/http"
	"net/http/httputil"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_tracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/signal"
	"open-match.dev/open-match/internal/telemetry"
)

const (
	configNameServerPublicCertificateFile = "api.tls.certificateFile"
	configNameServerPrivateKeyFile        = "api.tls.privateKey"
	configNameServerRootCertificatePath   = "api.tls.rootCertificateFile"
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

// ServerParams holds all the parameters required to start a gRPC server.
type ServerParams struct {
	// ServeMux is the router for the HTTP server. You can use this to serve pages in addition to the HTTP proxy.
	// Do NOT register "/" handler because it's reserved for the proxy.
	ServeMux               *http.ServeMux
	handlersForGrpc        []GrpcHandler
	handlersForGrpcProxy   []GrpcProxyHandler
	handlersForHealthCheck []func(context.Context) error

	grpcListener      *ListenerHolder
	grpcProxyListener *ListenerHolder

	// Root CA public certificate in PEM format.
	rootCaPublicCertificateFileData []byte
	// Public certificate in PEM format.
	// If this field is the same as rootCaPublicCertificateFileData then the certificate is not backed by a CA.
	publicCertificateFileData []byte
	// Private key in PEM format.
	privateKeyFileData []byte

	enableRPCLogging        bool
	enableRPCPayloadLogging bool
	enableMetrics           bool
	closer                  func()
}

// NewServerParamsFromConfig returns server Params initialized from the configuration file.
func NewServerParamsFromConfig(cfg config.View, prefix string) (*ServerParams, error) {
	grpcLh, err := newFromPortNumber(cfg.GetInt(prefix + ".grpcport"))
	if err != nil {
		serverLogger.Fatal(err)
		return nil, err
	}
	httpLh, err := newFromPortNumber(cfg.GetInt(prefix + ".httpport"))
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
	p := NewServerParamsFromListeners(grpcLh, httpLh)

	certFile := cfg.GetString(configNameServerPublicCertificateFile)
	privateKeyFile := cfg.GetString(configNameServerPrivateKeyFile)
	if len(certFile) > 0 && len(privateKeyFile) > 0 {
		serverLogger.Debugf("Loading TLS certificate (%s) and private key (%s)", certFile, privateKeyFile)
		publicCertData, err := ioutil.ReadFile(certFile)
		if err != nil {
			p.invalidate()
			return nil, errors.WithStack(fmt.Errorf("cannot read TLS server public certificate file, %s, %s", certFile, err))
		}
		privateKeyData, err := ioutil.ReadFile(privateKeyFile)
		if err != nil {
			p.invalidate()
			return nil, errors.WithStack(fmt.Errorf("cannot read TLS server private key file, %s, %s", privateKeyFile, err))
		}
		// If there's no root CA certificate then use the public certificate as the trusted root.
		rootPublicCertData := publicCertData

		rootCertFile := cfg.GetString(configNameServerRootCertificatePath)
		if len(rootCertFile) > 0 {
			serverLogger.Debugf("Loading Root CA TLS certificate (%s)", rootCertFile)
			rootPublicCertData, err = ioutil.ReadFile(rootCertFile)
			if err != nil {
				p.invalidate()
				return nil, errors.WithStack(fmt.Errorf("cannot read TLS server root certificate file, %s, %s", rootCertFile, err))
			}
		}
		p.SetTLSConfiguration(rootPublicCertData, publicCertData, privateKeyData)
	}

	p.enableMetrics = cfg.GetBool(telemetry.ConfigNameEnableMetrics)
	p.enableRPCLogging = cfg.GetBool(ConfigNameEnableRPCLogging)
	p.enableRPCPayloadLogging = logging.IsDebugEnabled(cfg)
	// TODO: This isn't ideal since telemetry requires config for it to be initialized.
	// This forces us to initialize readiness probes earlier than necessary.
	p.closer = telemetry.Setup(prefix, p.ServeMux, cfg)
	return p, nil
}

// NewServerParamsFromListeners returns server Params initialized with the ListenerHolder variables.
func NewServerParamsFromListeners(grpcLh *ListenerHolder, proxyLh *ListenerHolder) *ServerParams {
	return &ServerParams{
		ServeMux:             http.NewServeMux(),
		handlersForGrpc:      []GrpcHandler{},
		handlersForGrpcProxy: []GrpcProxyHandler{},
		grpcListener:         grpcLh,
		grpcProxyListener:    proxyLh,
	}
}

// SetTLSConfiguration configures the server to run in TLS mode.
func (p *ServerParams) SetTLSConfiguration(rootCaPublicCertificateFileData []byte, publicCertificateFileData []byte, privateKeyFileData []byte) *ServerParams {
	p.rootCaPublicCertificateFileData = rootCaPublicCertificateFileData
	if len(p.rootCaPublicCertificateFileData) == 0 {
		p.rootCaPublicCertificateFileData = publicCertificateFileData
	}
	p.publicCertificateFileData = publicCertificateFileData
	p.privateKeyFileData = privateKeyFileData
	return p
}

// usingTLS returns true if a certificate is set.
func (p *ServerParams) usingTLS() bool {
	return len(p.publicCertificateFileData) > 0
}

// AddHandleFunc binds gRPC service handler and an associated HTTP proxy handler.
func (p *ServerParams) AddHandleFunc(handlerFunc GrpcHandler, grpcProxyHandler GrpcProxyHandler) {
	if handlerFunc != nil {
		p.handlersForGrpc = append(p.handlersForGrpc, handlerFunc)
	}
	if grpcProxyHandler != nil {
		p.handlersForGrpcProxy = append(p.handlersForGrpcProxy, grpcProxyHandler)
	}
}

// AddHealthCheckFunc adds a readiness probe to tell Kubernetes the service is able to handle traffic.
func (p *ServerParams) AddHealthCheckFunc(handlerFunc func(context.Context) error) {
	if handlerFunc != nil {
		p.handlersForHealthCheck = append(p.handlersForHealthCheck, handlerFunc)
	}
}

// invalidate closes all the TCP listeners that would otherwise leak if initialization fails.
func (p *ServerParams) invalidate() {
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
	closer          func()
}

// grpcServerWithProxy this will go away when insecure.go and tls.go are merged into the same server.
type grpcServerWithProxy interface {
	start(*ServerParams) (func(), error)
	stop()
}

// Start the gRPC+HTTP(s) REST server.
func (s *Server) Start(p *ServerParams) (func(), error) {
	if p.usingTLS() {
		s.serverWithProxy = newTLSServer(p.grpcListener, p.grpcProxyListener)
	} else {
		s.serverWithProxy = newInsecureServer(p.grpcListener, p.grpcProxyListener)
	}
	s.closer = p.closer
	return s.serverWithProxy.start(p)
}

// Stop the gRPC+HTTP(s) REST server.
func (s *Server) Stop() {
	s.serverWithProxy.stop()
	if s.closer != nil {
		s.closer()
	}
}

// startServingIndefinitely creates a server based on the params and begins serving the gRPC and HTTP proxy.
// It returns waitUntilKilled() which will wait indefinitely until crash or Ctrl+C is pressed.
// forceStopServingFunc() is also returned which is used to force kill the server for tests.
func startServingIndefinitely(params *ServerParams) (func(), func(), error) {
	s := &Server{}

	// Start serving traffic.
	waitForStart, err := s.Start(params)
	if err != nil {
		serverLogger.WithFields(logrus.Fields{
			"error": err.Error(),
		}).Fatal("Failed to start gRPC and HTTP servers.")
		return func() {}, func() {}, err
	}
	serverLogger.Info("Server has started.")
	// Exit when we see a signal
	waitUntilKilled, forceStopServingFunc := signal.New()

	waitForStart()
	serveUntilKilledFunc := func() {
		waitUntilKilled()
		s.Stop()
		serverLogger.Info("Shutting down server")
	}
	return serveUntilKilledFunc, forceStopServingFunc, nil
}

// MustServeForever is a convenience method for starting a server and running it indefinitely.
func MustServeForever(params *ServerParams) {
	serveUntilKilledFunc, _, err := startServingIndefinitely(params)
	if err != nil {
		return
	}
	serveUntilKilledFunc()
}

type loggingHTTPHandler struct {
	handler     http.Handler
	logPayloads bool
}

func (l *loggingHTTPHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	dumpReqLog, dumpReqErr := httputil.DumpRequest(req, l.logPayloads)
	fields := logrus.Fields{
		"method": req.Method,
		"url":    req.URL,
		"proto":  req.Proto,
	}
	if dumpReqErr == nil {
		serverLogger.WithFields(fields).Debug(string(dumpReqLog))
	} else {
		serverLogger.WithError(dumpReqErr).WithFields(fields).Debug("cannot dump request")
	}
	l.handler.ServeHTTP(w, req)
}

func instrumentHTTPHandler(handler http.Handler, params *ServerParams) http.Handler {
	if params.enableMetrics {
		handler = &ochttp.Handler{
			Handler:     handler,
			Propagation: &b3.HTTPFormat{},
		}
	}
	if params.enableRPCLogging {
		handler = &loggingHTTPHandler{
			handler:     handler,
			logPayloads: params.enableRPCPayloadLogging,
		}
	}
	return handler
}

func newGRPCServerOptions(params *ServerParams) []grpc.ServerOption {
	opts := []grpc.ServerOption{}
	si := []grpc.StreamServerInterceptor{
		grpc_recovery.StreamServerInterceptor(),
		grpc_validator.StreamServerInterceptor(),
		grpc_tracing.StreamServerInterceptor(),
	}
	ui := []grpc.UnaryServerInterceptor{
		grpc_recovery.UnaryServerInterceptor(),
		grpc_validator.UnaryServerInterceptor(),
		grpc_tracing.UnaryServerInterceptor(),
	}
	if params.enableRPCLogging {
		grpcLogger := logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "grpc.server",
		})
		grpcLogger.Level = logrus.DebugLevel
		if params.enableRPCPayloadLogging {
			logEverythingFromServer := func(_ context.Context, _ string, _ interface{}) bool {
				return true
			}
			si = append(si, grpc_logrus.PayloadStreamServerInterceptor(grpcLogger, logEverythingFromServer))
			ui = append(ui, grpc_logrus.PayloadUnaryServerInterceptor(grpcLogger, logEverythingFromServer))
		} else {
			si = append(si, grpc_logrus.StreamServerInterceptor(grpcLogger))
			ui = append(ui, grpc_logrus.UnaryServerInterceptor(grpcLogger))
		}
	}

	if params.enableMetrics {
		opts = append(opts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	}

	return append(opts,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(si...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(ui...)))
}
