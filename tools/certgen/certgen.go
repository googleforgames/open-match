package main

// This code is an adapted version of https://golang.org/src/crypto/tls/generate_cert.go

import (
	"flag"
	"log"
	"time"

	certgenInternal "github.com/GoogleCloudPlatform/open-match/tools/certgen/internal"
)

var (
	publicKeyFlag        = flag.String("publickey", "public.cert", "Public key certificate path")
	privateKeyFlag       = flag.String("privatekey", "private.key", "Private key PEM file path")
	validityDurationFlag = flag.Duration("duration", time.Hour*24*365*5, "Lifetime for certificate validity (default is 5 years)")
	hostnamesFlag        = flag.String("hostnames", "om-frontendapi,om-backendapi,om-mmforc,om-function,om-mmlogicapi,om-evaluator", "Comma separated list of host names.")
	rsaKeyLengthFlag     = flag.Int("rsa", 2048, "RSA Encryption Key bit length for certificate.")
)

func main() {
	flag.Parse()
	if err := createCertificateViaFlags(); err != nil {
		log.Fatal(err)
	}
}

func createCertificateViaFlags() error {
	return certgenInternal.CreateCertificateAndPrivateKeyFiles(&certgenInternal.Params{
		PublicKeyPath:    *publicKeyFlag,
		PrivateKeyPath:   *privateKeyFlag,
		ValidityDuration: *validityDurationFlag,
		Hostnames:        *hostnamesFlag,
		RSAKeyLength:     *rsaKeyLengthFlag,
	})
}
