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
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
)

var (
	clientLogger = logrus.WithFields(logrus.Fields{
		"app":       "openmatch",
		"component": "client",
	})
)

// ClientParams contains the connection parameters to connect to an Open Match service.
type ClientParams struct {
	Hostname       string
	Port           int
	TrustedKeyPath string
}

func (p *ClientParams) usingTLS() bool {
	return len(p.TrustedKeyPath) > 0
}

// GRPCClientFromConfig creates a gRPC client connection from a configuration.
func GRPCClientFromConfig(cfg config.View, prefix string) (*grpc.ClientConn, error) {
	hostname := cfg.GetString(prefix + ".hostname")
	portNumber := cfg.GetInt(prefix + ".grpcport")

	clientParams := &ClientParams{
		Hostname: hostname,
		Port:     portNumber,
	}

	// If TLS support is enabled in the config, fill in the trusted key for decrpting server certificate.
	if cfg.GetBool("tls.enabled") {
		clientParams.TrustedKeyPath = cfg.GetString("tls.trustedKeyPath")
	}

	return GRPCClientFromParams(clientParams)
}

// GRPCClientFromParams creates a gRPC client connection from the parameters.
func GRPCClientFromParams(params *ClientParams) (*grpc.ClientConn, error) {
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)

	if params.usingTLS() {
		publicCertFileData, err := ioutil.ReadFile(params.TrustedKeyPath)
		creds, err := clientCredentialsFromFileData(publicCertFileData, address)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return grpc.Dial(address, grpc.WithTransportCredentials(creds))
	}
	return grpc.Dial(address, grpc.WithInsecure())
}

// HTTPClientFromConfig creates a HTTP client from from a configuration.
func HTTPClientFromConfig(cfg config.View, prefix string) (*http.Client, string, error) {
	hostname := cfg.GetString(prefix + ".hostname")
	portNumber := cfg.GetInt(prefix + ".httpport")

	clientParams := &ClientParams{
		Hostname: hostname,
		Port: portNumber,
	}

	// If TLS support is enabled in the config, fill in the trustedKeyPath
	if cfg.GetBool("tls.enabled") {
		clientParams.TrustedKeyPath = cfg.GetString("tls.trustedKeyPath")
	}

	return HTTPClientFromParams(clientParams)
}

// HTTPClientFromParams creates a HTTP client from the parameters.
func HTTPClientFromParams(params *ClientParams) (*http.Client, string, error) {
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)
	httpClient := &http.Client{Timeout: time.Second * 3}
	var baseURL string

	if params.usingTLS() {
		baseURL := "https://" + address
		tlsCert, err := clientCredentialsFromFile(params.TrustedKeyPath)
		if err != nil {
			return nil, "", err
		}

		pool, err := trustedCertificatesFromFile(params.TrustedKeyPath)
		if err != nil {
			return nil, "", err
		}

		tlsTransport := &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName:   address,
				RootCAs:      pool,
				Certificates: []tls.Certificate{*tlsCert},
			},
		}
		httpClient.Transport = tlsTransport
	}

	baseURL := "http://" + address
	return httpClient, baseURL, nil
}
