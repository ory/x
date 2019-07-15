package tlsx

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"github.com/pkg/errors"

	"github.com/ory/viper"
)

// ErrNoCertificatesConfigured is returned when no TLS configuration was found.
var ErrNoCertificatesConfigured = errors.New("no tls configuration was found")

// ErrInvalidCertificateConfiguration is returned when an invaloid TLS configuration was found.
var ErrInvalidCertificateConfiguration = errors.New("tls configuration is invalid")

// HTTPSCertificate returns loads a HTTP over TLS Certificate by looking at environment variables.
func HTTPSCertificate() ([]tls.Certificate, error) {
	prefix := "HTTPS_TLS"
	return Certificate(
		viper.GetString(prefix+"_CERT"), viper.GetString(prefix+"_KEY"),
		viper.GetString(prefix+"_CERT_PATH"), viper.GetString(prefix+"_KEY_PATH"),
	)
}

// HTTPSCertificateHelpMessage returns a help message for configuring HTTP over TLS Certificates.
func HTTPSCertificateHelpMessage() string {
	return CertificateHelpMessage("HTTPS_TLS")
}

// CertificateHelpMessage returns a help message for configuring TLS Certificates.
func CertificateHelpMessage(prefix string) string {
	return `- ` + prefix + `_CERT_PATH: The path to the TLS certificate (pem encoded).
	Example: ` + prefix + `_CERT_PATH=~/cert.pem

- ` + prefix + `_KEY_PATH: The path to the TLS private key (pem encoded).
	Example: ` + prefix + `_KEY_PATH=~/key.pem

- ` + prefix + `_CERT: Base64 encoded (without padding) string of the TLS certificate (PEM encoded) to be used for HTTP over TLS (HTTPS).
	Example: ` + prefix + `_CERT="-----BEGIN CERTIFICATE-----\nMIIDZTCCAk2gAwIBAgIEV5xOtDANBgkqhkiG9w0BAQ0FADA0MTIwMAYDVQQDDClP..."

- ` + prefix + `_KEY: Base64 encoded (without padding) string of the private key (PEM encoded) to be used for HTTP over TLS (HTTPS).
	Example: ` + prefix + `_KEY="-----BEGIN ENCRYPTED PRIVATE KEY-----\nMIIFDjBABgkqhkiG9w0BBQ0wMzAbBgkqhkiG9w0BBQwwDg..."
`
}

// Certificate returns loads a TLS Certificate by looking at environment variables.
func Certificate(
	certString, keyString string,
	certPath, keyPath string,
) ([]tls.Certificate, error) {
	if certString == "" && keyString == "" && certPath == "" && keyPath == "" {
		return nil, errors.WithStack(ErrNoCertificatesConfigured)
	} else if certString != "" && keyString != "" {
		tlsCertBytes, err := base64.StdEncoding.DecodeString(certString)
		if err != nil {
			return nil, fmt.Errorf("unable to base64 decode the TLS certificate: %v", err)
		}
		tlsKeyBytes, err := base64.StdEncoding.DecodeString(keyString)
		if err != nil {
			return nil, fmt.Errorf("unable to base64 decode the TLS private key: %v", err)
		}

		cert, err := tls.X509KeyPair(tlsCertBytes, tlsKeyBytes)
		if err != nil {
			return nil, fmt.Errorf("unable to load X509 key pair: %v", err)
		}
		return []tls.Certificate{cert}, nil
	}

	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, fmt.Errorf("unable to load X509 key pair from files: %v", err)
		}
		return []tls.Certificate{cert}, nil
	}

	return nil, errors.WithStack(ErrInvalidCertificateConfiguration)
}

// PublicKey returns the public key for a given key or nul.
func PublicKey(key interface{}) interface{} {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

// CreateSelfSignedTLSCertificate creates a self-signed TLS certificate.
func CreateSelfSignedTLSCertificate(key interface{}) (*tls.Certificate, error) {
	c, err := CreateSelfSignedCertificate(key)
	if err != nil {
		return nil, err
	}

	block, err := PEMBlockForKey(key)
	if err != nil {
		return nil, err
	}

	pemCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	pemKey := pem.EncodeToMemory(block)
	cert, err := tls.X509KeyPair(pemCert, pemKey)
	if err != nil {
		return nil, err
	}

	return &cert, nil
}

// CreateSelfSignedCertificate creates a self-signed x509 certificate.
func CreateSelfSignedCertificate(key interface{}) (cert *x509.Certificate, err error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return cert, errors.Errorf("failed to generate serial number: %s", err)
	}

	certificate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"ORY GmbH"},
			CommonName:   "ORY",
		},
		Issuer: pkix.Name{
			Organization: []string{"ORY GmbH"},
			CommonName:   "ORY",
		},
		NotBefore:             time.Now().UTC(),
		NotAfter:              time.Now().UTC().Add(time.Hour * 24 * 31),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certificate.IsCA = true
	certificate.KeyUsage |= x509.KeyUsageCertSign
	certificate.DNSNames = append(certificate.DNSNames, "localhost")
	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, PublicKey(key), key)
	if err != nil {
		return cert, errors.Errorf("failed to create certificate: %s", err)
	}

	cert, err = x509.ParseCertificate(der)
	if err != nil {
		return cert, errors.Errorf("failed to encode private key: %s", err)
	}
	return cert, nil
}

// PEMBlockForKey returns a PEM-encoded block for key.
func PEMBlockForKey(key interface{}) (*pem.Block, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}, nil
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}, nil
	default:
		return nil, errors.New("Invalid key type")
	}
}
