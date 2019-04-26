/*
certgen generates self-signed certificates for Open Match to run in TLS mode.

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
package main

// This code is an adapted version of https://golang.org/src/crypto/tls/generate_cert.go

import (
	"flag"
	"log"
	"time"

	certgenInternal "github.com/GoogleCloudPlatform/open-match/tools/certgen/internal"
)

var (
	caFlag                    = flag.Bool("ca", false, "Public key certificate path")
	rootPublicCertificateFlag = flag.String("rootpubliccertificate", "", "Public key certificate path")
	rootPrivateKeyFlag        = flag.String("rootprivatekey", "", "Private key PEM file path")
	publicCertificateFlag     = flag.String("publiccertificate", "public.cert", "Public key certificate path")
	privateKeyFlag            = flag.String("privatekey", "private.key", "Private key PEM file path")
	validityDurationFlag      = flag.Duration("duration", time.Hour*24*365*5, "Lifetime for certificate validity (default is 5 years)")
	hostnamesFlag             = flag.String("hostnames", "om-frontendapi,om-backendapi,om-mmforc,om-function,om-mmlogicapi,om-evaluator", "Comma separated list of host names.")
	rsaKeyLengthFlag          = flag.Int("rsa", 2048, "RSA Encryption Key bit length for certificate.")
)

func main() {
	flag.Parse()
	if err := createCertificateViaFlags(); err != nil {
		log.Fatal(err)
	}
}

func createCertificateViaFlags() error {
	return certgenInternal.CreateCertificateAndPrivateKeyFiles(&certgenInternal.Params{
		CertificateAuthority:      *caFlag,
		RootPublicCertificatePath: *rootPublicCertificateFlag,
		RootPrivateKeyPath:        *rootPrivateKeyFlag,
		PublicCertificatePath:     *publicCertificateFlag,
		PrivateKeyPath:            *privateKeyFlag,
		ValidityDuration:          *validityDurationFlag,
		Hostnames:                 *hostnamesFlag,
		RSAKeyLength:              *rsaKeyLengthFlag,
	})
}
