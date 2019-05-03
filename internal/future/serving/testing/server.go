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
	"github.com/GoogleCloudPlatform/open-match/internal/future/serving"
	netlistenerTesting "github.com/GoogleCloudPlatform/open-match/internal/util/netlistener/testing"
	certgenTesting "github.com/GoogleCloudPlatform/open-match/tools/certgen/testing"
)

// MustParamsForTesting sets up a test server in insecure-mode.
func MustParamsForTesting() *serving.Params {
	return serving.NewParamsFromListeners(netlistenerTesting.MustListen(), netlistenerTesting.MustListen())
}

// MustParamsForTestingTLS sets up a test server in TLS-mode.
func MustParamsForTestingTLS() *serving.Params {
	grpcLh := netlistenerTesting.MustListen()
	proxyLh := netlistenerTesting.MustListen()

	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting([]string{grpcLh.AddrString(), proxyLh.AddrString()})
	if err != nil {
		panic(err)
	}
	p := serving.NewParamsFromListeners(grpcLh, proxyLh)
	p.SetTLSConfiguration(pub, pub, priv)

	return p
}
