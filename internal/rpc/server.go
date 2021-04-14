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
	"net"
	"net/http"
	"net/http/httputil"
	"time"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_tracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"
	grpc_validator "github.com/grpc-ecosystem/go-grpc-middleware/validator"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/plugin/ochttp/propagation/b3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
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

	grpcListener      net.Listener
	grpcProxyListener net.Listener

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
}

// NewServerParamsFromConfig returns server Params initialized from the configuration file.
func NewServerParamsFromConfig(cfg config.View, prefix string, listen func(network, address string) (net.Listener, error)) (*ServerParams, error) {
	serverLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": prefix,
	})

	grpcL, err := listen("tcp", fmt.Sprintf(":%d", cfg.GetInt(prefix+".grpcport")))
	if err != nil {
		return nil, errors.Wrap(err, "can't start listener for grpc")
	}
	httpL, err := listen("tcp", fmt.Sprintf(":%d", cfg.GetInt(prefix+".httpport")))
	if err != nil {
		surpressedErr := grpcL.Close() // Don't care about additional errors when stopping.
		_ = surpressedErr
		return nil, errors.Wrap(err, "can't start listener for http")
	}

	p := NewServerParamsFromListeners(grpcL, httpL)

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

	return p, nil
}

// NewServerParamsFromListeners returns server Params initialized with the ListenerHolder variables.
func NewServerParamsFromListeners(grpcL net.Listener, proxyL net.Listener) *ServerParams {
	return &ServerParams{
		ServeMux:             http.NewServeMux(),
		handlersForGrpc:      []GrpcHandler{},
		handlersForGrpcProxy: []GrpcProxyHandler{},
		grpcListener:         grpcL,
		grpcProxyListener:    proxyL,
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
}

// grpcServerWithProxy this will go away when insecure.go and tls.go are merged into the same server.
type grpcServerWithProxy interface {
	start(*ServerParams) error
	stop() error
}

// Start the gRPC+HTTP(s) REST server.
func (s *Server) Start(p *ServerParams) error {
	if p.usingTLS() {
		s.serverWithProxy = newTLSServer(p.grpcListener, p.grpcProxyListener)
	} else {
		s.serverWithProxy = newInsecureServer(p.grpcListener, p.grpcProxyListener)
	}
	return s.serverWithProxy.start(p)
}

// Stop the gRPC+HTTP(s) REST server.
func (s *Server) Stop() error {
	return s.serverWithProxy.stop()
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

	ui = append(ui, serverUnaryInterceptor)
	si = append(si, serverStreamInterceptor)

	if params.enableMetrics {
		opts = append(opts, grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	}

	return append(opts,
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(si...)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(ui...)),
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             10 * time.Second,
				PermitWithoutStream: true,
			},
		))
}

func serverStreamInterceptor(srv interface{},
	stream grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	err := handler(srv, stream)
	if err != nil {
		serverLogger.Error(err)
	}
	return err
}

func serverUnaryInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	h, err := handler(ctx, req)
	if err != nil {
		serverLogger.Error(err)
	}
	return h, err
}
