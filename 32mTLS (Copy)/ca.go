package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

// CertBundle holds a TLS certificate and its raw PEM-encoded CA pool
type CertBundle struct {
	TLSCert tls.Certificate
	CACert  *x509.Certificate
	CAPool  *x509.CertPool
}

// GenerateCA creates a self-signed Certificate Authority in memory.
// In production, you'd load this from disk or a secrets manager.
func GenerateCA() (*x509.Certificate, *ecdsa.PrivateKey, []byte, error) {
	// Step 1: Generate a private key for the CA using P-256 elliptic curve
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	// Step 2: Define the CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"mTLS Demo CA"},
			CommonName:   "Root CA",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(24 * time.Hour), // short-lived for demo

		// These three fields are what make this a CA:
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	// Step 3: Self-sign the CA cert (parent = template itself)
	caDERBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, nil, nil, err
	}

	// Step 4: Parse back to get the *x509.Certificate struct
	caCert, err := x509.ParseCertificate(caDERBytes)
	if err != nil {
		return nil, nil, nil, err
	}

	// PEM-encode the CA cert for use in cert pools
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDERBytes})

	return caCert, caKey, caPEM, nil
}

// SignCertificate creates and signs a leaf certificate (server or client) using the CA.
func SignCertificate(caCert *x509.Certificate, caKey *ecdsa.PrivateKey, commonName string, isServer bool) (tls.Certificate, error) {
	// Step 1: Generate a fresh key for this leaf cert
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Step 2: Set key usage based on role
	// Servers need ServerAuth; clients need ClientAuth
	extKeyUsage := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	if isServer {
		extKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	// Step 3: Build the certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"mTLS Demo"},
			CommonName:   commonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: extKeyUsage,
	}

	// For servers: the cert must include the hostname it serves as a SAN
	if isServer {
		template.DNSNames = []string{"localhost"}
	}

	// Step 4: Sign with CA (parent = caCert, not template)
	leafDERBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Step 5: PEM-encode both the cert and the private key
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: leafDERBytes})

	leafKeyDER, err := x509.MarshalECPrivateKey(leafKey)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: leafKeyDER})

	// Step 6: Load into tls.Certificate (what Go's TLS stack actually uses)
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return tls.Certificate{}, err
	}

	return tlsCert, nil
}
