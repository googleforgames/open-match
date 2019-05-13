// ClientCredentialsFromFile gets TransportCredentials from public certificate.
func ClientCredentialsFromFile(certPath string) (credentials.TransportCredentials, error) {
	publicCertFileData, err := ioutil.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", certPath)
	}
	return ClientCredentialsFromFileData(publicCertFileData, "")
}

// TrustedCertificates gets TransportCredentials from public certificate file contents.
func TrustedCertificates(publicCertFileData []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return pool, nil
}

// ClientCredentialsFromFileData gets TransportCredentials from public certificate file contents.
func ClientCredentialsFromFileData(publicCertFileData []byte, serverOverride string) (credentials.TransportCredentials, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(publicCertFileData) {
		return nil, errors.WithStack(fmt.Errorf("ClientCredentialsFromFileData: failed to append certificates"))
	}
	return credentials.NewClientTLSFromCert(pool, serverOverride), nil
}

// CertificateFromFiles returns a tls.Certificate from PEM encoded X.509 public certificate and RSA private key files.
func CertificateFromFiles(publicCertificatePath string, privateKeyPath string) (*tls.Certificate, error) {
	publicCertFileData, err := ioutil.ReadFile(publicCertificatePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read public certificate from %s", publicCertificatePath)
	}
	privateKeyFileData, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read private key from %s", privateKeyPath)
	}
	return CertificateFromFileData(publicCertFileData, privateKeyFileData)
}

// CertificateFromFileData returns a tls.Certificate from PEM encoded file data.
func CertificateFromFileData(publicCertFileData []byte, privateKeyFileData []byte) (*tls.Certificate, error) {
	cert, err := tls.X509KeyPair(publicCertFileData, privateKeyFileData)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cert.Leaf, err = x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &cert, nil
}
