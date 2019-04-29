package testing

import (
	"time"

	certgenInternal "open-match.dev/open-match/tools/certgen/internal"
)

// CreateCertificateAndPrivateKeyForTesting is for generating test certificates in unit tests.
func CreateCertificateAndPrivateKeyForTesting(address string) ([]byte, []byte, error) {
	return certgenInternal.CreateCertificateAndPrivateKey(&certgenInternal.Params{
		ValidityDuration: time.Hour * 1,
		Hostnames:        address,
		RSAKeyLength:     2048,
	})
}
