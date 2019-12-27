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
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"go.opencensus.io/plugin/ocgrpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_tracing "github.com/grpc-ecosystem/go-grpc-middleware/tracing/opentracing"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ochttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/resolver"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/logging"
	"open-match.dev/open-match/internal/telemetry"
)

const (
	// ConfigNameEnableRPCLogging is the config name for enabling RPC logging.
	ConfigNameEnableRPCLogging = "logging.rpc"
	// configNameClientTrustedCertificatePath is the same as the root CA cert that the server trusts.
	configNameClientTrustedCertificatePath = configNameServerRootCertificatePath
)

var (
	clientLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "client",
	})
)

// ClientParams contains the connection parameters to connect to an Open Match service.
type ClientParams struct {
	Address                 string
	TrustedCertificate      []byte
	EnableRPCLogging        bool
	EnableRPCPayloadLogging bool
	EnableMetrics           bool
}

// nolint:gochecknoinits
func init() {
	// Using gRPC's DNS resolver to create clients.
	// This is a workaround for load balancing gRPC applications under k8s environments.
	// See https://kubernetes.io/blog/2018/11/07/grpc-load-balancing-on-kubernetes-without-tears/ for more details.
	// https://godoc.org/google.golang.org/grpc/resolver#SetDefaultScheme
	resolver.SetDefaultScheme("dns")
}

func (p *ClientParams) usingTLS() bool {
	return len(p.TrustedCertificate) > 0
}

// GRPCClientFromConfig creates a gRPC client connection from a configuration.
func GRPCClientFromConfig(cfg config.View, prefix string) (*grpc.ClientConn, error) {
	clientParams := &ClientParams{
		Address:                 toAddress(cfg.GetString(prefix+".hostname"), cfg.GetInt(prefix+".grpcport")),
		EnableRPCLogging:        cfg.GetBool(ConfigNameEnableRPCLogging),
		EnableRPCPayloadLogging: logging.IsDebugEnabled(cfg),
		EnableMetrics:           cfg.GetBool(telemetry.ConfigNameEnableMetrics),
	}

	// If TLS support is enabled in the config, fill in the trusted certificates for decrpting server certificate.
	if cfg.GetString(configNameClientTrustedCertificatePath) != "" {
		_, err := os.Stat(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("trusted certificate file may not exists.")
			return nil, err
		}

		clientParams.TrustedCertificate, err = ioutil.ReadFile(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("failed to read tls trusted certificate to establish a secure grpc client.")
			return nil, err
		}
	}

	return GRPCClientFromParams(clientParams)
}

// GRPCClientFromEndpoint creates a gRPC client connection from endpoint.
func GRPCClientFromEndpoint(cfg config.View, address string) (*grpc.ClientConn, error) {
	// TODO: investigate if it is possible to keep a cache of the certpool and transport credentials
	grpcOptions := newGRPCDialOptions(cfg.GetBool(telemetry.ConfigNameEnableMetrics), cfg.GetBool(ConfigNameEnableRPCLogging), logging.IsDebugEnabled(cfg))

	if cfg.GetString(configNameClientTrustedCertificatePath) != "" {
		_, err := os.Stat(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("trusted certificate file may not exists.")
			return nil, err
		}

		trustedCertificate, err := ioutil.ReadFile(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("failed to read tls trusted certificate to establish a secure grpc client.")
			return nil, err
		}

		pool, err := trustedCertificateFromFileData(trustedCertificate)
		if err != nil {
			clientLogger.WithError(err).Error("failed to get transport credentials from file.")
			return nil, errors.WithStack(err)
		}
		tc := credentials.NewClientTLSFromCert(pool, "")

		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(tc))
	} else {
		grpcOptions = append(grpcOptions, grpc.WithInsecure())
	}

	return grpc.Dial(address, grpcOptions...)
}

// GRPCClientFromParams creates a gRPC client connection from the parameters.
func GRPCClientFromParams(params *ClientParams) (*grpc.ClientConn, error) {
	grpcOptions := newGRPCDialOptions(params.EnableMetrics, params.EnableRPCLogging, params.EnableRPCPayloadLogging)

	if params.usingTLS() {
		trustedCertPool, err := trustedCertificateFromFileData(params.TrustedCertificate)
		if err != nil {
			clientLogger.WithError(err).Error("failed to get transport credentials from file.")
			return nil, errors.WithStack(err)
		}
		grpcOptions = append(grpcOptions, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(trustedCertPool, "")))
	} else {
		grpcOptions = append(grpcOptions, grpc.WithInsecure())
	}

	return grpc.Dial(params.Address, grpcOptions...)
}

// HTTPClientFromConfig creates a HTTP client from from a configuration.
func HTTPClientFromConfig(cfg config.View, prefix string) (*http.Client, string, error) {
	clientParams := &ClientParams{
		Address:                 toAddress(cfg.GetString(prefix+".hostname"), cfg.GetInt(prefix+".httpport")),
		EnableRPCLogging:        cfg.GetBool(ConfigNameEnableRPCLogging),
		EnableRPCPayloadLogging: logging.IsDebugEnabled(cfg),
		EnableMetrics:           cfg.GetBool(telemetry.ConfigNameEnableMetrics),
	}

	// If TLS support is enabled in the config, fill in the trusted certificates for decrpting server certificate.
	if cfg.GetString(configNameClientTrustedCertificatePath) != "" {
		_, err := os.Stat(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("trusted certificate file may not exists.")
			return nil, "", err
		}

		clientParams.TrustedCertificate, err = ioutil.ReadFile(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("failed to read tls trusted certificate to establish a secure grpc client.")
			return nil, "", err
		}
	}

	return HTTPClientFromParams(clientParams)
}

func sanitizeHTTPAddress(address string, preferHTTPS bool) (string, error) {
	lca := strings.ToLower(address)
	if !strings.HasPrefix(lca, "http://") && !strings.HasPrefix(lca, "https://") {
		if preferHTTPS {
			address = "https://" + address
		} else {
			address = "http://" + address
		}
	}
	u, err := url.Parse(address)
	if err != nil {
		return "", errors.New(fmt.Sprintf("%s is not a valid HTTP(S) address", address))
	}

	return u.String(), nil
}

// HTTPClientFromEndpoint creates a HTTP client from from endpoint.
func HTTPClientFromEndpoint(cfg config.View, address string) (*http.Client, string, error) {
	// TODO: investigate if it is possible to keep a cache of the certpool and transport credentials
	params := &ClientParams{
		Address:                 address,
		EnableRPCLogging:        cfg.GetBool(ConfigNameEnableRPCLogging),
		EnableRPCPayloadLogging: logging.IsDebugEnabled(cfg),
		EnableMetrics:           cfg.GetBool(telemetry.ConfigNameEnableMetrics),
	}
	if cfg.GetString(configNameClientTrustedCertificatePath) != "" {
		_, err := os.Stat(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("trusted certificate file may not exists.")
			return nil, "", err
		}

		trustedCertificate, err := ioutil.ReadFile(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("failed to read tls trusted certificate to establish a secure grpc client.")
			return nil, "", err
		}
		params.TrustedCertificate = trustedCertificate
	}
	return HTTPClientFromParams(params)
}

// HTTPClientFromParams creates a HTTP client from the parameters.
func HTTPClientFromParams(params *ClientParams) (*http.Client, string, error) {
	// TODO: Make client Timeout configurable
	httpClient := &http.Client{Timeout: time.Second * 3}
	var baseURL string

	if params.usingTLS() {
		var err error
		baseURL, err = sanitizeHTTPAddress(params.Address, true)
		if err != nil {
			clientLogger.WithError(err).Error("cannot parse address")
			return nil, "", err
		}

		pool, err := trustedCertificateFromFileData(params.TrustedCertificate)
		if err != nil {
			clientLogger.WithError(err).Error("failed to get cert pool from file.")
			return nil, "", err
		}

		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: params.Address,
				RootCAs:    pool,
			},
		}
	} else {
		var err error
		baseURL, err = sanitizeHTTPAddress(params.Address, false)
		if err != nil {
			clientLogger.WithError(err).Error("cannot parse address")
			return nil, "", err
		}
	}

	if params.EnableRPCLogging {
		attachTransport(httpClient, func(transport http.RoundTripper) http.RoundTripper {
			return &loggingHTTPClient{
				transport:   transport,
				logPayloads: params.EnableRPCPayloadLogging,
			}
		})
	}

	if params.EnableMetrics {
		attachTransport(httpClient, func(transport http.RoundTripper) http.RoundTripper {
			return &ochttp.Transport{
				Base: transport,
			}
		})
	}

	return httpClient, baseURL, nil
}

func newGRPCDialOptions(enableMetrics bool, enableRPCLogging bool, enableRPCPayloadLogging bool) []grpc.DialOption {
	si := []grpc.StreamClientInterceptor{
		grpc_tracing.StreamClientInterceptor(),
	}
	ui := []grpc.UnaryClientInterceptor{
		grpc_tracing.UnaryClientInterceptor(),
	}
	if enableRPCLogging {
		grpcLogger := logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "grpc.client",
		})
		grpcLogger.Level = logrus.DebugLevel
		if enableRPCPayloadLogging {
			logEverythingFromClient := func(_ context.Context, _ string) bool {
				return true
			}
			si = append(si, grpc_logrus.PayloadStreamClientInterceptor(grpcLogger, logEverythingFromClient))
			ui = append(ui, grpc_logrus.PayloadUnaryClientInterceptor(grpcLogger, logEverythingFromClient))
		} else {
			si = append(si, grpc_logrus.StreamClientInterceptor(grpcLogger))
			ui = append(ui, grpc_logrus.UnaryClientInterceptor(grpcLogger))
		}
	}
	opts := []grpc.DialOption{
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(si...)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(ui...)),
		grpc.WithBalancerName(roundrobin.Name),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                20 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}
	if enableMetrics {
		opts = append(opts, grpc.WithStatsHandler(new(ocgrpc.ClientHandler)))
	}
	return opts
}

func toAddress(hostname string, port int) string {
	return fmt.Sprintf("%s:%d", hostname, port)
}

type loggingHTTPClient struct {
	transport   http.RoundTripper
	logPayloads bool
}

func (c *loggingHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	dumpReqLog, dumpReqErr := httputil.DumpRequest(req, c.logPayloads)
	fields := logrus.Fields{
		"method": req.Method,
		"url":    req.URL,
		"proto":  req.Proto,
	}
	if dumpReqErr == nil {
		clientLogger.WithFields(fields).Debug(string(dumpReqLog))
	} else {
		clientLogger.WithError(dumpReqErr).WithFields(fields).Debug("cannot dump request")
	}
	resp, err := c.transport.RoundTrip(req)
	if err == nil {
		dumpRespLog, dumpRespErr := httputil.DumpResponse(resp, c.logPayloads)
		if dumpRespErr == nil {
			clientLogger.WithFields(fields).Debug(string(dumpRespLog))
		} else {
			clientLogger.WithError(dumpRespErr).WithFields(fields).Debug("request was successful but cannot dump response")
		}
	} else {
		clientLogger.WithError(err).WithFields(fields).Debug("request failed")
	}
	return resp, err
}

func attachTransport(client *http.Client, wrapper func(http.RoundTripper) http.RoundTripper) {
	transport := client.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	client.Transport = wrapper(transport)
}
