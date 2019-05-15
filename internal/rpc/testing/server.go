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
	"testing"

	"github.com/stretchr/testify/assert"
	"open-match.dev/open-match/internal/rpc"
	netlistenerTesting "open-match.dev/open-match/internal/util/netlistener/testing"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

// MustServerParamsForTesting sets up a test server in insecure-mode.
func MustServerParamsForTesting() *rpc.ServerParams {
	return rpc.NewServerParamsFromListeners(netlistenerTesting.MustListen(), netlistenerTesting.MustListen())
}

// MustServerParamsForTestingTLS sets up a test server in TLS-mode.
func MustServerParamsForTestingTLS() *rpc.ServerParams {
	grpcLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()

	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{grpcLh.AddrString(), proxyLh.AddrString()})
	if err != nil {
		panic(err)
	}
	p := rpc.NewServerParamsFromListeners(grpcLh, proxyLh)
	p.SetTLSConfiguration(pub, pub, priv)

	return p
}

// TestServerBinding verifies that a server can start and shutdown.
func TestServerBinding(t *testing.T, binder func(*rpc.ServerParams)) {
	assert := assert.New(t)
	p := MustServerParamsForTesting()
	binder(p)
	s := &rpc.Server{}
	defer s.Stop()
	waitForStart, err := s.Start(p)
	assert.Nil(err)
	waitForStart()
}
