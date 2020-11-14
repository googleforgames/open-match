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

package internal

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	secretMessage = "this is a secret message"
)

func TestCreateCertificate(t *testing.T) {
	require := require.New(t)

	tmpDir, err := ioutil.TempDir("", "certtest")
	defer func() {
		require.Nil(os.RemoveAll(tmpDir))
	}()
	require.Nil(err)
	publicCertPath := filepath.Join(tmpDir, "public.cert")
	privateKeyPath := filepath.Join(tmpDir, "private.key")
	err = CreateCertificateAndPrivateKeyFiles(
		publicCertPath,
		privateKeyPath,
		&Params{
			ValidityDuration: time.Hour * 1,
			Hostnames:        []string{"a.com", "b.com", "127.0.0.1"},
			RSAKeyLength:     2048,
		})
	require.Nil(err)

	require.FileExists(publicCertPath)
	require.FileExists(privateKeyPath)

	publicCertFileData, err := ioutil.ReadFile(publicCertPath)
	require.Nil(err)
	privateKeyFileData, err := ioutil.ReadFile(privateKeyPath)
	require.Nil(err)

	// Verify that we can load the public/private key pair.
	pub, pk, err := ReadKeyPair(publicCertFileData, privateKeyFileData)
	require.Nil(err)
	require.NotNil(pub)
	require.NotNil(pk)

	// Verify that the public/private key pair can RSA encrypt/decrypt.
	pubKey, ok := pub.PublicKey.(*rsa.PublicKey)
	require.True(ok, "pub.PublicKey is not of type, *rsa.PublicKey, %v", pub.PublicKey)
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pubKey, []byte(secretMessage), []byte{})
	require.Nil(err)
	require.NotEqual(string(ciphertext), secretMessage)
	cleartext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pk, ciphertext, []byte{})
	require.Nil(err)
	require.Equal(string(cleartext), string(secretMessage))
}

func TestBadValues(t *testing.T) {
	testCases := []struct {
		errorString string
		pub         string
		priv        string
		params      *Params
	}{
		{"root public certificate data was set but root private key data was not", "pub.cert", "priv.key",
			&Params{RootPublicCertificateData: []byte("A"), ValidityDuration: time.Second, Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2048}},
		{"root private key data was set but root public certificate data was not", "pub.cert", "priv.key",
			&Params{RootPrivateKeyData: []byte("A"), ValidityDuration: time.Second, Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2048}},
		{"public certificate file path must not be empty", "", "priv.key", &Params{ValidityDuration: time.Second, Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2048}},
		{"private key file path must not be empty", "pub.cert", "", &Params{ValidityDuration: time.Second, Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2048}},
		{"hostname list was empty. At least 1 hostname is required for generating a certificate-key pair", "pub.cert", "priv.key", &Params{}},
		{"RSA key length 2047 must be a power of 2", "pub.cert", "priv.key", &Params{ValidityDuration: time.Second, Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2047}},
		{"hostname list was empty. At least 1 hostname is required for generating a certificate-key pair", "pub.cert", "priv.key", &Params{ValidityDuration: time.Second, RSAKeyLength: 2048}},
		{"validity duration is required, otherwise the certificate would immediately expire", "pub.cert", "priv.key", &Params{Hostnames: []string{"127.0.0.1"}, RSAKeyLength: 2048}},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.errorString, func(t *testing.T) {
			err := CreateCertificateAndPrivateKeyFiles(
				testCase.pub,
				testCase.priv,
				testCase.params)
			if err == nil {
				t.Errorf("Expected an error with text, '%s'", testCase.errorString)
			} else if err.Error() != testCase.errorString {
				t.Errorf("Expected an error with text, '%s', got '%s'", testCase.errorString, err)
			}
		})
	}
}

func TestExpandHostnames(t *testing.T) {
	testCases := []struct {
		input    []string
		expected []string
	}{
		{
			[]string{},
			[]string{},
		},
		{
			[]string{"hello"},
			[]string{"hello"},
		},
		{
			[]string{"hello:1234"},
			[]string{"hello:1234", "hello"},
		},
	}
	for _, testCase := range testCases {
		testCase := testCase
		t.Run(fmt.Sprintf("expandHostnames(%s) => %s", testCase.input, testCase.expected), func(t *testing.T) {
			actual := expandHostnames(testCase.input)
			require.Equal(t, testCase.expected, actual)
		})
	}
}
