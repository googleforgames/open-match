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
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/future/pb"
	"open-match.dev/open-match/internal/future/serving/rpc"
	shellTesting "open-match.dev/open-match/internal/future/testing"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

func TestHTTPSFromConfig(t *testing.T) {
	runHTTPClientTests(t, true)
}
func TestInsecureHTTPFromConfig(t *testing.T) {
	runHTTPClientTests(t, false)
}

func runHTTPClientTests(t *testing.T, useTLS bool) {
	assert := assert.New(t)

	var (
		configPrefix      = "httpTest"
		clientHostNameKey = "httpTest.hostname"
		clientHTTPPortKey = "httpTest.httpport"
		httpClient        *http.Client
		baseURL           string
	)

	// Create netlisteners on random ports used for rpv serving
	grpcLh := netlistenerTesting.MustListen()
	httpLh := netlistenerTesting.MustListen()
	rpcParams := rpc.NewParamsFromListeners(grpcLh, httpLh)

	cfg := viper.New()
	cfg.Set(clientHostNameKey, "localhost")
	cfg.Set(clientHTTPPortKey, httpLh.Number())

	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", httpLh.Number())
	allHostnames := []string{grpcAddress, proxyAddress}
	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting(allHostnames)
	assert.Nil(err)

	// TLS setup
	if useTLS {
		rpcParams.SetTLSConfiguration(pub, pub, priv)
	}

	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServer(s, ff)
	}, pb.RegisterFrontendHandlerFromEndpoint)
	s := &rpc.Server{}
	defer s.Stop()
	waitForStart, err := s.Start(rpcParams)
	assert.Nil(err)
	waitForStart()

	// Acquire http client
	if useTLS {
		httpClient, baseURL, err = SecureHTTPFromConfig(cfg, configPrefix, pub, priv)
	} else {
		httpClient, baseURL, err = InsecureHTTPFromConfig(cfg, configPrefix)
	}

	// Confirm the client works as expected
	httpReq, err := http.NewRequest(http.MethodGet, baseURL+"/healthz", nil)
	assert.Nil(err)
	assert.NotNil(httpReq)

	httpResp, err := httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)

	body, err := ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("ok", string(body))
}
