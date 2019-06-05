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

// Package testing provides easy to use methods for creating certificates using a Root CA or individual.
package testing

import (
	"time"

	certgenInternal "open-match.dev/open-match/tools/certgen/internal"
)

// CreateCertificateAndPrivateKeyForTesting is for generating test certificates in unit tests.
func CreateCertificateAndPrivateKeyForTesting(hostnameList []string) ([]byte, []byte, error) {
	return certgenInternal.CreateCertificateAndPrivateKey(&certgenInternal.Params{
		ValidityDuration: time.Hour * 1,
		Hostnames:        hostnameList,
		RSAKeyLength:     2048,
	})
}

// CreateRootCertificateAndPrivateKeyForTesting creates a Root CA certificate that can be used as the parent from certificate-private key pairs created from CreateDerivedCertificateAndPrivateKeyForTesting.
func CreateRootCertificateAndPrivateKeyForTesting(hostnameList []string) ([]byte, []byte, error) {
	return certgenInternal.CreateCertificateAndPrivateKey(&certgenInternal.Params{
		ValidityDuration:     time.Hour * 1,
		Hostnames:            hostnameList,
		RSAKeyLength:         2048,
		CertificateAuthority: true,
	})
}

// CreateDerivedCertificateAndPrivateKeyForTesting creates a certificate rooted in the Root certificate from CreateRootCertificateAndPrivateKeyForTesting().
func CreateDerivedCertificateAndPrivateKeyForTesting(rootPublicCertificateData []byte, rootPrivateKeyData []byte, hostnameList []string) ([]byte, []byte, error) {
	return certgenInternal.CreateCertificateAndPrivateKey(&certgenInternal.Params{
		RootPublicCertificateData: rootPublicCertificateData,
		RootPrivateKeyData:        rootPrivateKeyData,
		ValidityDuration:          time.Hour * 1,
		Hostnames:                 hostnameList,
		RSAKeyLength:              2048,
		CertificateAuthority:      false,
	})
}
