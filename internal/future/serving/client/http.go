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

package client

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/future/serving"
)

// TODO: provides HTTPS client support

// InsecureHTTPFromConfig creates an HTTP client from a configuration.
func InsecureHTTPFromConfig(cfg config.View, prefix string) (*http.Client, string, error) {
	hostname := cfg.GetString(prefix + ".hostname")
	portNumber := cfg.GetInt(prefix + ".httpport")
	return HTTPFromParams(&Params{
		Hostname: hostname,
		Port:     portNumber,
	})
}

// HTTPFromParams creates a HTTP client from the parameters.
func HTTPFromParams(params *Params) (*http.Client, string, error) {
	address := fmt.Sprintf("%s:%d", params.Hostname, params.Port)
	if params.usingTLS() {
		baseURL := "https://" + address + "/"
		pool, err := serving.TrustedCertificates(params.PublicCertificate)
		if err != nil {
			return nil, "", err
		}
		tlsTransport := &http.Transport{
			TLSClientConfig: &tls.Config{
				ServerName: address,
				RootCAs:    pool,
			},
		}
		httpClient := &http.Client{
			Timeout:   time.Second * 10,
			Transport: tlsTransport,
		}
		return httpClient, baseURL, nil
	}
	baseURL := "http://" + address + "/"
	return &http.Client{Timeout: time.Second * 3}, baseURL, nil
}
