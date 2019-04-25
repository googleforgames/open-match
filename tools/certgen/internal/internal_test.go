package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const (
	secretMessage = "this is a secret message"
)

func TestParamsGetHostname(t *testing.T) {
	assert := assert.New(t)
	p := &Params{
		Hostnames: "a.com,b.com,127.0.0.1",
	}
	assert.ElementsMatch([]string{"a.com", "b.com", "127.0.0.1"}, p.GetHostnames())
}

func TestCreateCertificate(t *testing.T) {
	assert := assert.New(t)

	tmpDir, err := ioutil.TempDir("", "certtest")
	defer func() {
		assert.Nil(os.RemoveAll(tmpDir))
	}()
	assert.Nil(err)
	publicCertPath := filepath.Join(tmpDir, "public.cert")
	privateKeyPath := filepath.Join(tmpDir, "private.key")
	err = CreateCertificateAndPrivateKeyFiles(&Params{
		PublicKeyPath:    publicCertPath,
		PrivateKeyPath:   privateKeyPath,
		ValidityDuration: time.Hour * 1,
		Hostnames:        "a.com,b.com,127.0.0.1",
		RSAKeyLength:     2048,
	})
	assert.Nil(err)

	assert.FileExists(publicCertPath)
	assert.FileExists(privateKeyPath)

	publicCertFileData, err := ioutil.ReadFile(publicCertPath)
	assert.Nil(err)
	privateKeyFileData, err := ioutil.ReadFile(privateKeyPath)
	assert.Nil(err)

	// Verify that we can load the public/private key pair.
	publicCertPemBlock, remainder := pem.Decode(publicCertFileData)
	assert.Empty(remainder)
	pub, err := x509.ParseCertificate(publicCertPemBlock.Bytes)
	assert.Nil(err)
	assert.NotNil(pub)
	privateKeyPemBlock, remainder := pem.Decode(privateKeyFileData)
	assert.Empty(remainder)
	pk, err := x509.ParsePKCS1PrivateKey(privateKeyPemBlock.Bytes)
	assert.Nil(err)
	assert.NotNil(publicCertFileData)
	assert.NotNil(pk)

	// Verify that the public/private key pair can RSA encrypt/decrypt.
	pubKey, ok := pub.PublicKey.(*rsa.PublicKey)
	assert.True(ok, "pub.PublicKey is not of type, *rsa.PublicKey, %v", pub.PublicKey)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(secretMessage), []byte{})
	assert.Nil(err)
	assert.NotEqual(string(ciphertext), secretMessage)
	cleartext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, []byte(ciphertext), []byte{})
	assert.Nil(err)
	assert.Equal(string(cleartext), string(secretMessage))
}
