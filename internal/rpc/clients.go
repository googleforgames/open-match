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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	"go.opencensus.io/plugin/ocgrpc"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/telemetry"
)

const (
	configNameEnableRPCLogging = "logging.rpc"
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
	Hostname           string
	Port               int
	TrustedCertificate []byte
	EnableRPCLogging   bool
	EnableMetrics      bool
}

func (p *ClientParams) usingTLS() bool {
	return len(p.TrustedCertificate) > 0
}

// GRPCClientFromConfig creates a gRPC client connection from a configuration.
func GRPCClientFromConfig(cfg config.View, prefix string) (*grpc.ClientConn, error) {
	clientParams := &ClientParams{
		Hostname:         cfg.GetString(prefix + ".hostname"),
		Port:             cfg.GetInt(prefix + ".grpcport"),
		EnableRPCLogging: cfg.GetBool(configNameEnableRPCLogging),
		EnableMetrics:    cfg.GetBool(telemetry.ConfigNameEnableMetrics),
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
	grpcOptions := newGRPCDialOptions(cfg.GetBool(telemetry.ConfigNameEnableMetrics), cfg.GetBool(configNameEnableRPCLogging))

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
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)

	grpcOptions := newGRPCDialOptions(params.EnableMetrics, params.EnableRPCLogging)

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

	return grpc.Dial(address, grpcOptions...)
}

// HTTPClientFromConfig creates a HTTP client from from a configuration.
func HTTPClientFromConfig(cfg config.View, prefix string) (*http.Client, string, error) {
	clientParams := &ClientParams{
		Hostname:         cfg.GetString(prefix + ".hostname"),
		Port:             cfg.GetInt(prefix + ".httpport"),
		EnableRPCLogging: cfg.GetBool(configNameEnableRPCLogging),
		EnableMetrics:    cfg.GetBool(telemetry.ConfigNameEnableMetrics),
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
	// TODO: Make client Timeout configurable
	httpClient := &http.Client{Timeout: time.Second * 3}
	var baseURL string

	if cfg.GetString(configNameClientTrustedCertificatePath) != "" {
		var err error
		baseURL, err = sanitizeHTTPAddress(address, true)
		if err != nil {
			clientLogger.WithError(err).Error("cannot parse address")
			return nil, "", err
		}
		_, err = os.Stat(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("trusted certificate file may not exists.")
			return nil, "", err
		}

		trustedCertificate, err := ioutil.ReadFile(cfg.GetString(configNameClientTrustedCertificatePath))
		if err != nil {
			clientLogger.WithError(err).Error("failed to read tls trusted certificate to establish a secure grpc client.")
			return nil, "", err
		}

		pool, err := trustedCertificateFromFileData(trustedCertificate)
		if err != nil {
			clientLogger.WithError(err).Error("failed to get cert pool from file.")
			return nil, "", err
		}

		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: address,
				RootCAs:    pool,
			},
		}
	} else {
		var err error
		baseURL, err = sanitizeHTTPAddress(address, false)
		if err != nil {
			clientLogger.WithError(err).Error("cannot parse address")
			return nil, "", err
		}
	}

	return httpClient, baseURL, nil
}

// HTTPClientFromParams creates a HTTP client from the parameters.
func HTTPClientFromParams(params *ClientParams) (*http.Client, string, error) {
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)
	// TODO: Make client Timeout configurable
	httpClient := &http.Client{Timeout: time.Second * 3}
	var baseURL string

	if params.usingTLS() {
		var err error
		baseURL, err = sanitizeHTTPAddress(address, true)
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
				ServerName: address,
				RootCAs:    pool,
			},
		}
	} else {
		var err error
		baseURL, err = sanitizeHTTPAddress(address, false)
		if err != nil {
			clientLogger.WithError(err).Error("cannot parse address")
			return nil, "", err
		}
	}

	return httpClient, baseURL, nil
}

func newGRPCDialOptions(enableMetrics bool, enableRPCLogging bool) []grpc.DialOption {
	si := []grpc.StreamClientInterceptor{
		grpc_retry.StreamClientInterceptor(),
	}
	ui := []grpc.UnaryClientInterceptor{
		grpc_retry.UnaryClientInterceptor(),
	}
	if enableRPCLogging {
		grpcLogger := logrus.WithFields(logrus.Fields{
			"app":       "openmatch",
			"component": "grpc.client",
		})
		si = append(si, grpc_logrus.StreamClientInterceptor(grpcLogger))
		ui = append(ui, grpc_logrus.UnaryClientInterceptor(grpcLogger))
	}
	opts := []grpc.DialOption{
		grpc.WithStreamInterceptor(grpc_middleware.ChainStreamClient(si...)),
		grpc.WithUnaryInterceptor(grpc_middleware.ChainUnaryClient(ui...)),
	}
	if enableMetrics {
		opts = append(opts, grpc.WithStatsHandler(new(ocgrpc.ClientHandler)))
	}
	return opts
}
