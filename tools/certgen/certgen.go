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

// Package main is the certgen tool which generates public-certificate/private-key for Open Match to run in TLS mode.
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	certgenInternal "open-match.dev/open-match/tools/certgen/internal"
)

var (
	// serviceAddresses is a list of all the HTTP hostname:port combinations for generating a TLS certificate.
	// It appears that gRPC does not care about validating the port number so we only add the HTTP addresses here.
	serviceAddressList = []string{
		"om-backend:51505",
		"om-demo:51507",
		"om-demoevaluator:51508",
		"om-demofunction:51502",
		"om-e2eevaluator:51518",
		"om-e2ematchfunction:51512",
		"om-frontend:51504",
		"om-query:51503",
		"om-swaggerui:51500",
		"om-synchronizer:51506",
	}
	caFlag                    = flag.Bool("ca", false, "Create a root certificate. Use if you want a chain of trust with other certificates.")
	rootPublicCertificateFlag = flag.String("rootpubliccertificate", "", "(optional) Path to root certificate file. If set the output certificate is rooted from this certificate.")
	rootPrivateKeyFlag        = flag.String("rootprivatekey", "", "(required if --rootpubliccertificate is set) Path to private key paired from root certificate file.")
	publicCertificateFlag     = flag.String("publiccertificate", "public.cert", "Public key certificate path to be generated")
	privateKeyFlag            = flag.String("privatekey", "private.key", "Private key file path to be generated")
	validityDurationFlag      = flag.Duration("duration", time.Hour*24*365*5, "Lifetime for certificate validity (default is 5 years)")
	hostnamesFlag             = flag.String("hostnames", strings.Join(serviceAddressList, ","), "Comma separated list of host names.")
	rsaKeyLengthFlag          = flag.Int("rsa", 2048, "RSA Encryption Key bit length for certificate.")
)

func main() {
	flag.Parse()
	if err := createCertificateViaFlags(); err != nil {
		log.Fatal(err)
	}
}

func createCertificateViaFlags() error {
	params := &certgenInternal.Params{
		CertificateAuthority: *caFlag,
		ValidityDuration:     *validityDurationFlag,
		Hostnames:            strings.Split(*hostnamesFlag, ","),
		RSAKeyLength:         *rsaKeyLengthFlag,
	}

	if len(*rootPublicCertificateFlag) > 0 {
		if len(*rootPrivateKeyFlag) == 0 {
			return fmt.Errorf("--rootprivatekey is required if --rootpubliccertificate=%s is set", *rootPublicCertificateFlag)
		}
		if rootPublicCertificateData, err := ioutil.ReadFile(*rootPublicCertificateFlag); err == nil {
			params.RootPublicCertificateData = rootPublicCertificateData
		} else {
			return fmt.Errorf("cannot read the root public certificate from %s", *rootPublicCertificateFlag)
		}
		if rootPrivateKeyData, err := ioutil.ReadFile(*rootPrivateKeyFlag); err == nil {
			params.RootPrivateKeyData = rootPrivateKeyData
		} else {
			return fmt.Errorf("cannot read the root private key from %s", *rootPrivateKeyFlag)
		}
	}

	return certgenInternal.CreateCertificateAndPrivateKeyFiles(*publicCertificateFlag, *privateKeyFlag, params)
}
