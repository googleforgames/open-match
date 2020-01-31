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

// Package internal holds the internal details of creating a self-signed certificate from either a Root CA or individual cert.
package internal

// This code is an adapted version of https://golang.org/src/crypto/tls/generate_cert.go

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	country          = "US"
	organization     = "Open Match"
	organizationUnit = "OM"
	locality         = "Mountain View"
	province         = "CA"
)

// Params for creating an X.509 certificate and RSA private key pair for TLS.
type Params struct {
	// If true, indicates that this certificate is a root certificate.
	// Root certificates are used to establish a chain of trust.
	// This means that if the root certificate is trusted certificates derived from it are also trusted.
	CertificateAuthority bool

	// (optional) Root certificate that will be used to create the new derived certificate from.
	RootPublicCertificateData []byte
	// (optional) Root private that will be used to create the new derived certificate from.
	RootPrivateKeyData []byte

	// The duration from now when the certificate will expire.
	ValidityDuration time.Duration
	// List of hostnames that this certificate is valid for. Clients verify that this
	Hostnames []string
	// RSA encryption key length.
	RSAKeyLength int
}

// CreateCertificateAndPrivateKeyFiles writes the random public certificate and private key pair to disk.
func CreateCertificateAndPrivateKeyFiles(publicCertificateFilePath string, privateKeyFilePath string, params *Params) error {
	if len(publicCertificateFilePath) == 0 {
		return errors.New("public certificate file path must not be empty")
	}
	if len(privateKeyFilePath) == 0 {
		return errors.New("private key file path must not be empty")
	}
	if len(params.RootPublicCertificateData) > 0 && len(params.RootPrivateKeyData) == 0 {
		return errors.New("root public certificate data was set but root private key data was not")
	}
	if len(params.RootPublicCertificateData) == 0 && len(params.RootPrivateKeyData) > 0 {
		return errors.New("root private key data was set but root public certificate data was not")
	}
	if len(params.Hostnames) == 0 {
		return errors.New("hostname list was empty. At least 1 hostname is required for generating a certificate-key pair")
	}
	if int(math.Pow(math.Sqrt(float64(params.RSAKeyLength)), 2)) != params.RSAKeyLength {
		return fmt.Errorf("RSA key length %d must be a power of 2", params.RSAKeyLength)
	}
	if params.ValidityDuration == time.Millisecond*0 {
		return errors.New("validity duration is required, otherwise the certificate would immediately expire")
	}

	publicCertificatePemBytes, privateKeyPemBytes, err := CreateCertificateAndPrivateKey(params)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(publicCertificateFilePath, publicCertificatePemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}
	if err = ioutil.WriteFile(privateKeyFilePath, privateKeyPemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ReadKeyPair takes PEM-encoded public certificate/private key pairs and returns the Go classes for them so they can be used for encryption or signing.
func ReadKeyPair(publicCertFileData []byte, privateKeyFileData []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Verify that we can load the public/private key pair.
	publicCertPemBlock, remainder := pem.Decode(publicCertFileData)
	if len(remainder) > 0 {
		log.Printf("Public Certificate has a PEM remainder of %d bytes", len(remainder))
	}
	publicCertificate, err := x509.ParseCertificate(publicCertPemBlock.Bytes)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	privateKeyPemBlock, remainder := pem.Decode(privateKeyFileData)
	if len(remainder) > 0 {
		log.Printf("Public Certificate has a PEM remainder of %d bytes", len(remainder))
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes)
	if err != nil {
		return nil, nil, errors.WithStack(err)
	}
	return publicCertificate, privateKey, nil
}

// CreateCertificateAndPrivateKey returns the public certificate and private key pair as byte arrays.
func CreateCertificateAndPrivateKey(params *Params) ([]byte, []byte, error) {
	startTimestamp := time.Now()
	expirationTimestamp := startTimestamp.Add(params.ValidityDuration)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return []byte{}, []byte{}, errors.WithStack(err)
	}
	pkixName := pkix.Name{
		Country:            []string{country},
		Organization:       []string{organization},
		OrganizationalUnit: []string{organizationUnit},
		Locality:           []string{locality},
		Province:           []string{province},
		CommonName:         organization,
		SerialNumber:       serialNumber.String(),
	}
	certTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkixName,
		Issuer:                pkixName,
		NotBefore:             startTimestamp,
		NotAfter:              expirationTimestamp,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		SignatureAlgorithm:    x509.SHA256WithRSA,
		IsCA:                  params.CertificateAuthority,
	}
	if params.CertificateAuthority {
		certTemplate.KeyUsage |= x509.KeyUsageCertSign
	}
	hostnames := expandHostnames(params.Hostnames)
	for _, hostname := range hostnames {
		if ipAddress := net.ParseIP(hostname); ipAddress != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ipAddress)
		} else {
			certTemplate.DNSNames = append(certTemplate.DNSNames, hostname)
		}
	}

	certTemplate.DNSNames = append(certTemplate.DNSNames, "localhost")
	certTemplate.IPAddresses = append(certTemplate.IPAddresses, net.ParseIP("127.0.0.1"))

	privateKey, err := rsa.GenerateKey(rand.Reader, params.RSAKeyLength)
	if err != nil {
		return []byte{}, []byte{}, errors.WithStack(err)
	}
	parentPrivateKey := privateKey

	parentTemplate := certTemplate
	if len(params.RootPublicCertificateData) > 0 {
		rootPublicCertificate, rootPrivateKey, readErr := ReadKeyPair(params.RootPublicCertificateData, params.RootPrivateKeyData)
		if readErr != nil {
			return nil, nil, readErr
		}
		parentTemplate = *rootPublicCertificate
		parentPrivateKey = rootPrivateKey
	}

	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &parentTemplate, publicKey(privateKey), parentPrivateKey)
	if err != nil {
		return []byte{}, []byte{}, errors.WithStack(err)
	}

	publicCertificatePemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	pemPrivateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPemBytes := pem.EncodeToMemory(pemPrivateKey)

	return publicCertificatePemBytes, privateKeyPemBytes, nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func expandHostnames(hostnames []string) []string {
	expanded := []string{}
	for _, hostname := range hostnames {
		expanded = append(expanded, hostname)
		// If this is a hostname:port then also add just the hostname.
		parts := strings.SplitN(hostname, ":", -1)
		if len(parts) == 2 {
			expanded = append(expanded, parts[0])
		}
	}
	return expanded
}
