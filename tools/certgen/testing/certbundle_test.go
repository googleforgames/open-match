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
	"testing"

	"github.com/stretchr/testify/require"
	certgenInternal "open-match.dev/open-match/tools/certgen/internal"
)

const (
	fakeAddress = "localhost:1024"
)

func TestCreateCertificateAndPrivateKeyForTestingAreValid(t *testing.T) {
	require := require.New(t)

	pubData, privData, err := CreateCertificateAndPrivateKeyForTesting([]string{fakeAddress})
	require.Nil(err)

	// Verify that we can load the public/private key pair.
	pub, pk, err := certgenInternal.ReadKeyPair(pubData, privData)
	require.Nil(err)
	require.NotNil(pub)
	require.NotNil(pk)
}

func TestCreateRootedCertificateAndPrivateKeyForTestingAreValid(t *testing.T) {
	require := require.New(t)

	rootPubData, rootPrivData, err := CreateRootCertificateAndPrivateKeyForTesting([]string{fakeAddress})
	require.Nil(err)
	pubData, privData, err := CreateDerivedCertificateAndPrivateKeyForTesting(rootPubData, rootPrivData, []string{fakeAddress})
	require.Nil(err)

	rootPub, rootPk, err := certgenInternal.ReadKeyPair(rootPubData, rootPrivData)
	require.Nil(err)
	require.NotNil(rootPk)
	pub, pk, err := certgenInternal.ReadKeyPair(pubData, privData)
	require.Nil(err)
	require.NotNil(pk)

	require.Nil(pub.CheckSignatureFrom(rootPub))
}

func TestCreateCertificateAndPrivateKeyForTestingAreDifferent(t *testing.T) {
	require := require.New(t)

	pubData1, privData1, err := CreateCertificateAndPrivateKeyForTesting([]string{fakeAddress})
	require.Nil(err)
	pubData2, privData2, err := CreateCertificateAndPrivateKeyForTesting([]string{fakeAddress})
	require.Nil(err)

	require.NotEqual(string(pubData1), string(pubData2))
	require.NotEqual(string(privData1), string(privData2))
}
