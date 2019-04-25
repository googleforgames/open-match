package testing

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	fakeAddress = "localhost:1024"
)

func TestCreateCertificateAndPrivateKeyForTestingAreValid(t *testing.T) {
	assert := assert.New(t)

	pubData, privData, err := CreateCertificateAndPrivateKeyForTesting(fakeAddress)
	assert.Nil(err)

	// Verify that we can load the public/private key pair.
	publicKeyPemBlock, remainder := pem.Decode(pubData)
	assert.Empty(remainder)
	pub, err := x509.ParseCertificate(publicKeyPemBlock.Bytes)
	assert.Nil(err)
	assert.NotNil(pub)
	privateKeyPemBlock, remainder := pem.Decode(privData)
	assert.Empty(remainder)
	pk, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes)
	assert.Nil(err)
	assert.NotNil(pk)
}

func TestCreateCertificateAndPrivateKeyForTestingAreDifferent(t *testing.T) {
	assert := assert.New(t)

	pubData1, privData1, err := CreateCertificateAndPrivateKeyForTesting(fakeAddress)
	assert.Nil(err)
	pubData2, privData2, err := CreateCertificateAndPrivateKeyForTesting(fakeAddress)
	assert.Nil(err)

	assert.NotEqual(string(pubData1), string(pubData2))
	assert.NotEqual(string(privData1), string(privData2))
}
