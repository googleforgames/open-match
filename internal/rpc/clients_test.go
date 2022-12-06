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
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/config"
	"open-match.dev/open-match/internal/telemetry"
	utilTesting "open-match.dev/open-match/internal/util/testing"
	"open-match.dev/open-match/pkg/pb"
	shellTesting "open-match.dev/open-match/testing"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

func TestSecureGRPCFromConfig(t *testing.T) {
	require := require.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(t, require, true, "localhost")
	defer closer()

	runSuccessGrpcClientTests(t, require, cfg, rpcParams)
}

func TestInsecureGRPCFromConfig(t *testing.T) {
	require := require.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(t, require, false, "localhost")
	defer closer()

	runSuccessGrpcClientTests(t, require, cfg, rpcParams)
}

func TestUnavailableGRPCFromConfig(t *testing.T) {
	require := require.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(t, require, false, "badhost")
	defer closer()

	runFailureGrpcClientTests(t, require, cfg, rpcParams, codes.Unavailable)
}

func TestHTTPSFromConfig(t *testing.T) {
	require := require.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(t, require, true, "localhost")
	defer closer()

	runHTTPClientTests(require, cfg, rpcParams)
}

func TestInsecureHTTPFromConfig(t *testing.T) {
	require := require.New(t)

	cfg, rpcParams, closer := configureConfigAndKeysForTesting(t, require, false, "localhost")
	defer closer()

	runHTTPClientTests(require, cfg, rpcParams)
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
			require := require.New(t)
			actual, err := sanitizeHTTPAddress(tc.address, tc.preferHTTPS)
			require.Equal(tc.expected, actual)
			require.Equal(tc.err, err)
		})
	}
}

func setupClientConnection(t *testing.T, require *require.Assertions, cfg config.View, rpcParams *ServerParams) *grpc.ClientConn {
	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)

	s := &Server{}
	t.Cleanup(func() {
		defer s.Stop()
	})
	err := s.Start(rpcParams)
	require.Nil(err)

	// Acquire grpc client
	grpcConn, err := GRPCClientFromConfig(cfg, "test")
	require.Nil(err)
	require.NotNil(grpcConn)
	return grpcConn
}

func runSuccessGrpcClientTests(t *testing.T, require *require.Assertions, cfg config.View, rpcParams *ServerParams) {
	grpcConn := setupClientConnection(t, require, cfg, rpcParams)

	// Confirm the client works as expected
	ctx := utilTesting.NewContext(t)
	feClient := pb.NewFrontendServiceClient(grpcConn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	require.Nil(err)
	require.NotNil(grpcResp)
}

func runFailureGrpcClientTests(t *testing.T, require *require.Assertions, cfg config.View, rpcParams *ServerParams, expectedCode codes.Code) {
	grpcConn := setupClientConnection(t, require, cfg, rpcParams)

	// Confirm the client works as expected
	ctx := utilTesting.NewContext(t)
	feClient := pb.NewFrontendServiceClient(grpcConn)
	grpcResp, err := feClient.CreateTicket(ctx, &pb.CreateTicketRequest{})
	require.Error(err)
	require.Nil(grpcResp)

	code := status.Code(err)
	require.Equal(expectedCode, code)
}

func runHTTPClientTests(require *require.Assertions, cfg config.View, rpcParams *ServerParams) {
	// Serve a fake frontend server and wait for its full start up
	ff := &shellTesting.FakeFrontend{}
	rpcParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)
	s := &Server{}
	defer s.Stop()
	err := s.Start(rpcParams)
	require.Nil(err)

	// Acquire http client
	httpClient, baseURL, err := HTTPClientFromConfig(cfg, "test")
	require.Nil(err)

	// Confirm the client works as expected
	httpReq, err := http.NewRequest(http.MethodGet, baseURL+telemetry.HealthCheckEndpoint, nil)
	require.Nil(err)
	require.NotNil(httpReq)

	httpResp, err := httpClient.Do(httpReq)
	require.Nil(err)
	require.NotNil(httpResp)
	defer func() {
		if httpResp != nil {
			httpResp.Body.Close()
		}
	}()

	body, err := ioutil.ReadAll(httpResp.Body)
	require.Nil(err)
	require.Equal(200, httpResp.StatusCode)
	require.Equal("ok", string(body))
}

// Generate a config view and optional TLS key manifests (optional) for testing
func configureConfigAndKeysForTesting(t *testing.T, require *require.Assertions, tlsEnabled bool, host string) (config.View, *ServerParams, func()) {
	// Create netlisteners on random ports used for rpc serving
	grpcL := MustListen()
	httpL := MustListen()
	rpcParams := NewServerParamsFromListeners(grpcL, httpL)

	// Generate a config view with paths to the manifests
	cfg := viper.New()
	cfg.Set("test.hostname", host)
	cfg.Set("test.grpcport", MustGetPortNumber(grpcL))
	cfg.Set("test.httpport", MustGetPortNumber(httpL))

	// Create temporary TLS key files for testing
	pubFile, err := ioutil.TempFile("", "pub*")
	require.Nil(err)

	if tlsEnabled {
		// Generate public and private key bytes
		pubBytes, priBytes, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{
			fmt.Sprintf("localhost:%s", MustGetPortNumber(grpcL)),
			fmt.Sprintf("localhost:%s", MustGetPortNumber(httpL)),
		})
		require.Nil(err)

		// Write certgen key bytes to the temp files
		err = ioutil.WriteFile(pubFile.Name(), pubBytes, 0400)
		require.Nil(err)

		// Generate a config view with paths to the manifests
		cfg.Set(configNameClientTrustedCertificatePath, pubFile.Name())

		rpcParams.SetTLSConfiguration(pubBytes, pubBytes, priBytes)
	}

	return cfg, rpcParams, func() { removeTempFile(t, pubFile.Name()) }
}

func MustListen() net.Listener {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}
	return l
}

func MustGetPortNumber(l net.Listener) string {
	_, port, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		panic(err)
	}
	return port
}

func removeTempFile(t *testing.T, paths ...string) {
	for _, path := range paths {
		err := os.Remove(path)
		if err != nil {
			t.Errorf("Can not remove the temporary file: %s, err: %s", path, err.Error())
		}
	}
}
