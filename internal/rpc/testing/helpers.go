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

package testing

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/rpc"
	"open-match.dev/open-match/internal/util"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

// MustServe creates a test server and returns TestContext that can be used to create clients.
// This method pseudorandomly selects insecure and TLS mode to ensure both paths work.
func MustServe(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if r.Intn(2) == 0 {
		return MustServeInsecure(t, binder)
	}
	return MustServeTLS(t, binder)
}

// MustServeInsecure creates a test server without transport encryption and returns TestContext that can be used to create clients.
func MustServeInsecure(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	grpcLh := rpc.MustListen()
	proxyLh := rpc.MustListen()

	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", proxyLh.Number())

	p := rpc.NewServerParamsFromListeners(grpcLh, proxyLh)
	s := bindAndStart(t, p, binder)
	return &TestContext{
		t:            t,
		s:            s,
		grpcAddress:  grpcAddress,
		proxyAddress: proxyAddress,
		mc:           util.NewMultiClose(),
	}
}

// MustServeTLS creates a test server with TLS and returns TestContext that can be used to create clients.
func MustServeTLS(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	grpcLh := rpc.MustListen()
	proxyLh := rpc.MustListen()

	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", proxyLh.Number())
	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{grpcAddress, proxyAddress})
	if err != nil {
		t.Fatalf("cannot create certificates %v", err)
	}

	p := rpc.NewServerParamsFromListeners(grpcLh, proxyLh)
	p.SetTLSConfiguration(pub, pub, priv)
	s := bindAndStart(t, p, binder)

	return &TestContext{
		t:                  t,
		s:                  s,
		grpcAddress:        grpcAddress,
		proxyAddress:       proxyAddress,
		trustedCertificate: pub,
		mc:                 util.NewMultiClose(),
	}
}

func bindAndStart(t *testing.T, p *rpc.ServerParams, binder func(*rpc.ServerParams)) *rpc.Server {
	binder(p)
	s := &rpc.Server{}
	waitForStart, err := s.Start(p)
	if err != nil {
		t.Fatalf("failed to start server, %v", err)
	}
	waitForStart()
	return s
}

// TestContext provides methods to interact with the Open Match server.
type TestContext struct {
	t                  *testing.T
	s                  *rpc.Server
	grpcAddress        string
	proxyAddress       string
	trustedCertificate []byte
	mc                 *util.MultiClose
}

// AddCloseFunc adds a close function.
func (tc *TestContext) AddCloseFunc(closer func()) {
	tc.mc.AddCloseFunc(closer)
}

// Close shutsdown the server and frees the TCP port.
func (tc *TestContext) Close() {
	tc.mc.Close()
	tc.s.Stop()
}

// Context returns a context appropriate for calling an RPC.
func (tc *TestContext) Context() context.Context {
	return context.Background()
}

// MustGRPC returns a grpc client configured to connect to an endpoint.
func (tc *TestContext) MustGRPC() *grpc.ClientConn {
	conn, err := rpc.GRPCClientFromParams(tc.newClientParams(tc.grpcAddress))
	if err != nil {
		tc.t.Fatal(err)
	}
	return conn
}

// MustHTTP returns a HTTP(S) client configured to connect to an endpoint.
func (tc *TestContext) MustHTTP() (*http.Client, string) {
	client, endpoint, err := rpc.HTTPClientFromParams(tc.newClientParams(tc.proxyAddress))
	if err != nil {
		tc.t.Fatal(err)
	}
	return client, endpoint
}

func (tc *TestContext) newClientParams(address string) *rpc.ClientParams {
	return &rpc.ClientParams{
		Address:                 address,
		TrustedCertificate:      tc.trustedCertificate,
		EnableRPCLogging:        true,
		EnableRPCPayloadLogging: true,
		EnableMetrics:           false,
	}
}

// GetHostname returns the hostname of current text context
func (tc *TestContext) GetHostname() string {
	return "localhost"
}

// GetHTTPPort returns the http proxy port of current text context
func (tc *TestContext) GetHTTPPort() int {
	_, port := hostnameAndPort(tc.t, tc.proxyAddress)
	return port
}

// GetGRPCPort returns the grpc service port of current text context
func (tc *TestContext) GetGRPCPort() int {
	_, port := hostnameAndPort(tc.t, tc.grpcAddress)
	return port
}

func hostnameAndPort(t *testing.T, address string) (string, int) {
	// Coerce to a url.
	if !strings.Contains(address, "://") {
		address = "http://" + address
	}
	address = strings.Replace(address, "[::]", "localhost", -1)

	u, err := url.Parse(address)
	if err != nil {
		t.Fatalf("cannot parse address %s, %v", address, err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatalf("cannot convert port number %s, %v", u.Port(), err)
	}
	return u.Hostname(), port
}
