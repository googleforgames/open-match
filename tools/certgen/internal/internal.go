/*
Package internal holds the internal details of creating a self-signed certificate from either a Root CA or individual cert.

Copyright 2019 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/
package internal

// This code is an adapted version of https://golang.org/src/crypto/tls/generate_cert.go

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"log"
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
	CertificateAuthority      bool
	RootPublicCertificatePath string
	RootPrivateKeyPath        string

	RootPublicCertificateData []byte
	RootPrivateKeyData        []byte

	PublicCertificatePath string
	PrivateKeyPath        string
	ValidityDuration      time.Duration
	Hostnames             string
	RSAKeyLength          int
}

// GetHostnames returns all the hosts this certificate can handle.
func (p *Params) GetHostnames() []string {
	return strings.Split(p.Hostnames, ",")
}

// CreateCertificateAndPrivateKeyFiles writes the random public certificate and private key pair to disk.
func CreateCertificateAndPrivateKeyFiles(params *Params) error {
	if len(params.RootPublicCertificatePath) > 0 {
		if rootPublicCertificateData, err := ioutil.ReadFile(params.RootPublicCertificatePath); err == nil {
			params.RootPublicCertificateData = rootPublicCertificateData
		} else {
			return errors.WithStack(err)
		}
		if rootPrivateKeyData, err := ioutil.ReadFile(params.RootPrivateKeyPath); err == nil {
			params.RootPrivateKeyData = rootPrivateKeyData
		} else {
			return errors.WithStack(err)
		}
	}

	publicCertificatePemBytes, privateKeyPemBytes, err := CreateCertificateAndPrivateKey(params)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(params.PublicCertificatePath, publicCertificatePemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}
	if err = ioutil.WriteFile(params.PrivateKeyPath, privateKeyPemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}

	return nil
}

// ReadKeyPair works.
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
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  params.CertificateAuthority,
	}
	if params.CertificateAuthority {
		certTemplate.KeyUsage = certTemplate.KeyUsage | x509.KeyUsageCertSign
	}
	for _, hostname := range params.GetHostnames() {
		if ipAddress := net.ParseIP(hostname); ipAddress != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ipAddress)
		} else {
			certTemplate.DNSNames = append(certTemplate.DNSNames, hostname)
		}
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, params.RSAKeyLength)
	if err != nil {
		return []byte{}, []byte{}, errors.WithStack(err)
	}
	parentPrivateKey := privateKey

	parentTemplate := certTemplate
	if len(params.RootPublicCertificateData) > 0 {
		rootPublicCertificate, rootPrivateKey, err := ReadKeyPair(params.RootPublicCertificateData, params.RootPrivateKeyData)
		if err != nil {
			return nil, nil, err
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
