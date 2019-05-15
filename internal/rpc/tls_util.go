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

	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"google.golang.org/grpc/credentials"
)

// clientCredentialsFromFile gets TransportCredentials from public certificate.
func clientCredentialsFromFile(certPath string) (credentials.TransportCredentials, error) {
	publicCertFileData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", certPath)
	}
	return clientCredentialsFromFileData(publicCertFileData, "")
}

// trustedCertificatesFromFile gets TransportCredentials from trusted public key file name
func trustedCertificatesFromFile(trustedKeyPath string) (*x509.CertPool, error) {
	trustedKeyFileData, err := ioutil.ReadFile(trustedKeyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read trusted key from %s", trustedKeyPath)
	}
	return trustedCertificates(trustedKeyFileData)
}

// trustedCertificates gets TransportCredentials from public certificate file contents.
func trustedCertificates(publicCertFileData []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return pool, nil
}

// clientCredentialsFromFileData gets TransportCredentials from public certificate file contents.
func clientCredentialsFromFileData(publicCertFileData []byte, serverOverride string) (credentials.TransportCredentials, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return credentials.NewClientTLSFromCert(pool, serverOverride), nil
}

// certificateFromFiles returns a tls.Certificate from PEM encoded X.509 public certificate and RSA private key files.
// nolint:deadcode unused
func certificateFromFiles(publicCertificatePath string, privateKeyPath string) (*tls.Certificate, error) {
	publicCertFileData, err := ioutil.ReadFile(publicCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", publicCertificatePath)
	}
	privateKeyFileData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read private key from %s", privateKeyPath)
	}
	return certificateFromFileData(publicCertFileData, privateKeyFileData)
}

// certificateFromFileData returns a tls.Certificate from PEM encoded file data.
func certificateFromFileData(publicCertFileData []byte, privateKeyFileData []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(publicCertFileData, privateKeyFileData)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cert, nil
}
