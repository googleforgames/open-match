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

	"google.golang.org/grpc"
	"open-match.dev/open-match/internal/rpc"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

// MustServe creates a test server and returns TestContext that can be used to create clients.
// This method pseudorandomly selects insecure and TLS mode to ensure both paths work.
func MustServe(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	if rand.Intn(2) == 0 {
		return MustServeInsecure(t, binder)
	}
	return MustServeTLS(t, binder)
}

// MustServeInsecure creates a test server without transport encryption and returns TestContext that can be used to create clients.
func MustServeInsecure(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	grpcLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()

	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", proxyLh.Number())

	p := rpc.NewServerParamsFromListeners(grpcLh, proxyLh)
	s := bindAndStart(t, p, binder)
	return &TestContext{
		t:            t,
		s:            s,
		grpcAddress:  grpcAddress,
		proxyAddress: proxyAddress,
	}
}

// MustServeTLS creates a test server with TLS and returns TestContext that can be used to create clients.
func MustServeTLS(t *testing.T, binder func(*rpc.ServerParams)) *TestContext {
	grpcLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()

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

// TestServerBinding verifies that a server can start and shutdown.
func TestServerBinding(t *testing.T, binder func(*rpc.ServerParams)) {
	tc := MustServe(t, binder)
	tc.Close()
}

// TestContext provides methods to interact with the Open Match server.
type TestContext struct {
	t                  *testing.T
	s                  *rpc.Server
	grpcAddress        string
	proxyAddress       string
	trustedCertificate []byte
	closers            []func()
}

// AddCloseFunc adds a close function.
func (tc *TestContext) AddCloseFunc(closer func()) {
	tc.closers = append(tc.closers, closer)
}

// Close shutsdown the server and frees the TCP port.
func (tc *TestContext) Close() {
	for _, closer := range tc.closers {
		closer()
	}

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
	hostname, port := hostnameAndPort(tc.t, address)
	return &rpc.ClientParams{
		Hostname:           hostname,
		Port:               port,
		TrustedCertificate: tc.trustedCertificate,
	}
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
