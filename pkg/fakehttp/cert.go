package fakehttp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"
)

// Certificate is a test certificate with its ECDSA P-256 key.
type Certificate struct {
	Cert    *x509.Certificate
	Key     *ecdsa.PrivateKey
	CertPEM string
	KeyPEM  string
}

// TLSCertificate returns the certificate and key for tls.Config.Certificates.
func (c *Certificate) TLSCertificate() tls.Certificate {
	return tls.Certificate{Certificate: [][]byte{c.Cert.Raw}, PrivateKey: c.Key, Leaf: c.Cert}
}

// NewCA returns a self-signed CA certificate valid for one hour, also a server certificate for names.
func NewCA(names ...string) (*Certificate, error) {
	template := certificateTemplate("fakehttp CA", names)
	template.IsCA = true
	template.BasicConstraintsValid = true
	template.KeyUsage = x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	return newCertificate(template, nil)
}

// IssueClient returns a client certificate signed by ca, valid for one hour.
func (c *Certificate) IssueClient(commonName string) (*Certificate, error) {
	template := certificateTemplate(commonName, nil)
	template.KeyUsage = x509.KeyUsageDigitalSignature
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}

	return newCertificate(template, c)
}

func certificateTemplate(commonName string, names []string) *x509.Certificate {
	template := &x509.Certificate{
		Subject:   pkix.Name{CommonName: commonName, Organization: []string{"fakehttp"}},
		NotBefore: time.Now().Add(-time.Minute),
		NotAfter:  time.Now().Add(time.Hour),
	}

	for _, name := range names {
		if ip := net.ParseIP(name); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, name)
		}
	}

	return template
}

// newCertificate signs template with issuer, or self-signs it when issuer is nil.
func newCertificate(template *x509.Certificate, issuer *Certificate) (*Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("unable to generate key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("unable to generate serial number: %w", err)
	}

	template.SerialNumber = serial

	parent, signer := template, key
	if issuer != nil {
		parent, signer = issuer.Cert, issuer.Key
	}

	der, err := x509.CreateCertificate(rand.Reader, template, parent, &key.PublicKey, signer)
	if err != nil {
		return nil, fmt.Errorf("unable to create certificate: %w", err)
	}

	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate: %w", err)
	}

	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal key: %w", err)
	}

	return &Certificate{
		Cert:    cert,
		Key:     key,
		CertPEM: string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})),
		KeyPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyDER})),
	}, nil
}
