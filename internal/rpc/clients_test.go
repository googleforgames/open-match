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
	"fmt"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"io/ioutil"
	"net/http"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/telemetry"
	shellTesting "open-match.dev/open-match/internal/testing"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
	"os"
	"testing"
)

func TestSecureGRPCFromConfig(t *testing.T) {
	assert := assert.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(assert, true)
	defer closer()

	runGrpcClientTests(t, assert, cfg, rpcParams)
}

func TestInsecureGRPCFromConfig(t *testing.T) {
	assert := assert.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(assert, false)
	defer closer()

	runGrpcClientTests(t, assert, cfg, rpcParams)
}

func TestHTTPSFromConfig(t *testing.T) {
	assert := assert.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(assert, true)
	defer closer()

	runHTTPClientTests(assert, cfg, rpcParams)
}

func TestInsecureHTTPFromConfig(t *testing.T) {
	assert := assert.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(assert, false)
	defer closer()

	runHTTPClientTests(assert, cfg, rpcParams)
}

func TestSanitizeHTTPAddress(t *testing.T) {
	tests := []struct {
		address     string
		preferHTTPS bool
		expected    string
		err         error
	}{
		{"om-test:54321", false, "http://om-test:54321", nil},
		{"om-test:54321", true, "https://om-test:54321", nil},
		{"http://om-test:54321", false, "http://om-test:54321", nil},
		{"https://om-test:54321", false, "https://om-test:54321", nil},
		{"http://om-test:54321", true, "http://om-test:54321", nil},
		{"https://om-test:54321", true, "https://om-test:54321", nil},
	}

	for _, testCase := range tests {
		tc := testCase
		description := fmt.Sprintf("sanitizeHTTPAddress(%s, %t) => (%s, %v)", tc.address, tc.preferHTTPS, tc.expected, tc.err)
		t.Run(description, func(t *testing.T) {
			assert := assert.New(t)
			actual, err := sanitizeHTTPAddress(tc.address, tc.preferHTTPS)
			assert.Equal(tc.expected, actual)
			assert.Equal(tc.err, err)
		})
	}
}

func runGrpcClientTests(t *testing.T, assert *assert.Assertions, cfg config.View, rpcParams *ServerParams) {
	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)

	s := &Server{}
	defer s.Stop()
	waitForStart, err := s.Start(rpcParams)
	assert.Nil(err)
	waitForStart()

	// Acquire grpc client
	grpcConn, err := GRPCClientFromConfig(cfg, "test")
	assert.Nil(err)
	assert.NotNil(grpcConn)

	// Confirm the client works as expected
	ctx := utilTesting.NewContext(t)
	feClient := pb.NewFrontendServiceClient(grpcConn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	assert.Nil(err)
	assert.NotNil(grpcResp)
}

func runHTTPClientTests(assert *assert.Assertions, cfg config.View, rpcParams *ServerParams) {
	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	s := &Server{}
	defer s.Stop()
	waitForStart, err := s.Start(rpcParams)
	assert.Nil(err)
	waitForStart()

	// Acquire http client
	httpClient, baseURL, err := HTTPClientFromConfig(cfg, "test")
	assert.Nil(err)

	// Confirm the client works as expected
	httpReq, err := http.NewRequest(http.MethodGet, baseURL+telemetry.HealthCheckEndpoint, nil)
	assert.Nil(err)
	assert.NotNil(httpReq)

	httpResp, err := httpClient.Do(httpReq)
	assert.Nil(err)
	assert.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(httpResp.Body)
	assert.Nil(err)
	assert.Equal(200, httpResp.StatusCode)
	assert.Equal("ok", string(body))
}

// Generate a config view and optional TLS key manifests (optional) for testing
func configureConfigAndKeysForTesting(assert *assert.Assertions, tlsEnabled bool) (config.View, *ServerParams, func()) {
	// Create netlisteners on random ports used for rpc serving
	grpcLh := MustListen()
	httpLh := MustListen()
	rpcParams := NewServerParamsFromListeners(grpcLh, httpLh)

	// Generate a config view with paths to the manifests
	cfg := viper.New()
	cfg.Set("test.hostname", "localhost")
	cfg.Set("test.grpcport", grpcLh.Number())
	cfg.Set("test.httpport", httpLh.Number())

	// Create temporary TLS key files for testing
	pubFile, err := ioutil.TempFile("", "pub*")
	assert.Nil(err)

	if tlsEnabled {
		// Generate public and private key bytes
		pubBytes, priBytes, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{
			fmt.Sprintf("localhost:%d", grpcLh.Number()),
			fmt.Sprintf("localhost:%d", httpLh.Number()),
		})
		assert.Nil(err)

		// Write certgen key bytes to the temp files
		err = ioutil.WriteFile(pubFile.Name(), pubBytes, 0400)
		assert.Nil(err)

		// Generate a config view with paths to the manifests
		cfg.Set(configNameClientTrustedCertificatePath, pubFile.Name())

		rpcParams.SetTLSConfiguration(pubBytes, pubBytes, priBytes)
	}

	return cfg, rpcParams, func() { removeTempFile(assert, pubFile.Name()) }
}

func removeTempFile(assert *assert.Assertions, paths ...string) {
	for _, path := range paths {
		err := os.Remove(path)
		assert.Nil(err)
	}
}
