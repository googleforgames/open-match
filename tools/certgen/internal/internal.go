package internal

// This code is an adapted version of https://golang.org/src/crypto/tls/generate_cert.go

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
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
	PublicKeyPath    string
	PrivateKeyPath   string
	ValidityDuration time.Duration
	Hostnames        string
	RSAKeyLength     int
}

// GetHostnames returns all the hosts this certificate can handle.
func (p *Params) GetHostnames() []string {
	return strings.Split(p.Hostnames, ",")
}

// CreateCertificateAndPrivateKeyFiles writes the random public certificate and private key pair to disk.
func CreateCertificateAndPrivateKeyFiles(params *Params) error {
	publicKeyPemBytes, privateKeyPemBytes, err := CreateCertificateAndPrivateKey(params)
	if err != nil {
		return err
	}
	if err = ioutil.WriteFile(params.PublicKeyPath, publicKeyPemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}
	if err = ioutil.WriteFile(params.PrivateKeyPath, privateKeyPemBytes, 0644); err != nil {
		return errors.WithStack(err)
	}

	return nil
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

	cert, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, publicKey(privateKey), privateKey)
	if err != nil {
		return []byte{}, []byte{}, errors.WithStack(err)
	}

	publicKeyPemBytes := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
	pemPrivateKey := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	privateKeyPemBytes := pem.EncodeToMemory(pemPrivateKey)

	return publicKeyPemBytes, privateKeyPemBytes, nil
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}
