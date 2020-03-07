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
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	shellTesting "open-match.dev/open-match/internal/testing"
	"open-match.dev/open-match/pkg/pb"
	certgenTesting "open-match.dev/open-match/tools/certgen/testing"
)

// TestStartStopTlsServerWithCARootedCertificate verifies that we can have a gRPC+TLS+HTTPS server/client work with a single self-signed certificate.
func TestStartStopTlsServerWithSingleCertificate(t *testing.T) {
	assert := assert.New(t)
	grpcLh := MustListen()
	proxyLh := MustListen()
	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", proxyLh.Number())
	allHostnames := []string{grpcAddress, proxyAddress}
	pub, priv, err := certgenTesting.CreateCertificateAndPrivateKeyForTesting(allHostnames)
	assert.Nil(err)
	runTestStartStopTLSServer(t, &tlsServerTestParams{
		rootPublicCertificateFileData: pub,
		rootPrivateKeyFileData:        priv,
		publicCertificateFileData:     pub,
		privateKeyFileData:            priv,
		grpcLh:                        grpcLh,
		proxyLh:                       proxyLh,
		grpcAddress:                   grpcAddress,
		proxyAddress:                  proxyAddress,
	})
}

// TestStartStopTlsServerWithCARootedCertificate verifies that we can have a gRPC+TLS+HTTPS server/client work with a self-signed CA-rooted certificate.
func TestStartStopTlsServerWithCARootedCertificate(t *testing.T) {
	assert := assert.New(t)
	grpcLh := MustListen()
	proxyLh := MustListen()
	grpcAddress := fmt.Sprintf("localhost:%d", grpcLh.Number())
	proxyAddress := fmt.Sprintf("localhost:%d", proxyLh.Number())
	allHostnames := []string{grpcAddress, proxyAddress}
	rootPub, rootPriv, err := certgenTesting.CreateRootCertificateAndPrivateKeyForTesting(allHostnames)
	assert.Nil(err)

	pub, priv, err := certgenTesting.CreateDerivedCertificateAndPrivateKeyForTesting(rootPub, rootPriv, allHostnames)
	assert.Nil(err)

	runTestStartStopTLSServer(t, &tlsServerTestParams{
		rootPublicCertificateFileData: rootPub,
		rootPrivateKeyFileData:        rootPriv,
		publicCertificateFileData:     pub,
		privateKeyFileData:            priv,
		grpcLh:                        grpcLh,
		proxyLh:                       proxyLh,
		grpcAddress:                   grpcAddress,
		proxyAddress:                  proxyAddress,
	})
}

type tlsServerTestParams struct {
	rootPublicCertificateFileData []byte
	rootPrivateKeyFileData        []byte
	publicCertificateFileData     []byte
	privateKeyFileData            []byte
	grpcLh                        *ListenerHolder
	proxyLh                       *ListenerHolder
	grpcAddress                   string
	proxyAddress                  string
}

func runTestStartStopTLSServer(t *testing.T, tp *tlsServerTestParams) {
	assert := assert.New(t)

	ff := &shellTesting.FakeFrontend{}

	serverParams := NewServerParamsFromListeners(tp.grpcLh, tp.proxyLh)
	serverParams.AddHandleFunc(func(s *grpc.Server) {
		pb.RegisterFrontendServiceServer(s, ff)
	}, pb.RegisterFrontendServiceHandlerFromEndpoint)

	serverParams.SetTLSConfiguration(tp.rootPublicCertificateFileData, tp.publicCertificateFileData, tp.privateKeyFileData)
	s := newTLSServer(serverParams.grpcListener, serverParams.grpcProxyListener)
	defer s.stop()

	waitForStart, err := s.start(serverParams)
	assert.Nil(err)
	waitForStart()

	pool, err := trustedCertificateFromFileData(tp.rootPublicCertificateFileData)
	assert.Nil(err)

	conn, err := grpc.Dial(tp.grpcAddress, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, tp.grpcAddress)))
	assert.Nil(err)

	tlsCert, err := certificateFromFileData(tp.publicCertificateFileData, tp.privateKeyFileData)
	assert.Nil(err)
	tlsTransport := &http.Transport{
		TLSClientConfig: &tls.Config{
			ServerName:   tp.proxyAddress,
			RootCAs:      pool,
			Certificates: []tls.Certificate{*tlsCert},
		},
	}
	httpsEndpoint := fmt.Sprintf("https://%s", tp.proxyAddress)
	httpClient := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tlsTransport,
	}
	runGrpcWithProxyTests(t, assert, s, conn, httpClient, httpsEndpoint)
}
