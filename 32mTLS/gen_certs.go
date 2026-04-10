// gen_certs.go — Generates all certificates needed for mTLS
//
// DEEP DIVE: Certificate Generation
// ==================================
// In mTLS, you need THREE things:
//   1. A Certificate Authority (CA) — the root of trust
//   2. A Server certificate — signed by the CA, proves "I am the server"
//   3. A Client certificate — signed by the CA, proves "I am the client"
//
// Both server and client trust the CA. So when either side presents its
// cert, the other can verify: "This cert was signed by a CA I trust."

package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

func main() {
	// ============================================================
	// STEP 1: Create the Certificate Authority (CA)
	// ============================================================
	// The CA is self-signed — it signs itself. Both the server and
	// client will be configured to trust this CA. In production,
	// this would be your organization's internal CA (e.g., HashiCorp
	// Vault, AWS Private CA, or cfssl).

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(err, "generate CA key")

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"My Company"},
			CommonName:   "My Company Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour), // 10 years
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true, // <-- THIS makes it a CA certificate
		MaxPathLen:            1,
	}

	// Self-signed: parent == template (signs itself)
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	must(err, "create CA cert")

	caCert, err := x509.ParseCertificate(caCertDER)
	must(err, "parse CA cert")

	savePEM("ca-cert.pem", "CERTIFICATE", caCertDER)
	saveKey("ca-key.pem", caKey)
	fmt.Println("✅ Created CA certificate (ca-cert.pem, ca-key.pem)")

	// ============================================================
	// STEP 2: Create the Server Certificate (signed by CA)
	// ============================================================
	// The server cert includes:
	//   - DNSNames/IPAddresses: what hostnames/IPs the cert is valid for
	//   - ExtKeyUsage: ServerAuth — marks this cert for TLS server use
	//
	// KEY INSIGHT: The server cert is signed by the CA (parent=caCert,
	// signingKey=caKey). This creates the chain of trust.

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(err, "generate server key")

	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"My Company"},
			CommonName:   "localhost",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),

		// What this cert can be used for:
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, // <-- SERVER auth

		// What hostnames/IPs this cert is valid for:
		DNSNames:    []string{"localhost"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
	}

	// Signed by CA: parent=caCert, key=caKey
	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	must(err, "create server cert")

	savePEM("server-cert.pem", "CERTIFICATE", serverCertDER)
	saveKey("server-key.pem", serverKey)
	fmt.Println("✅ Created Server certificate (server-cert.pem, server-key.pem)")

	// ============================================================
	// STEP 3: Create the Client Certificate (signed by CA)
	// ============================================================
	// The client cert is what makes this MUTUAL TLS.
	// Without it, you just have regular TLS (server-only auth).
	//
	// KEY INSIGHT: ExtKeyUsageClientAuth marks this for client use.
	// The server will check: "Was this client cert signed by a CA I trust?"

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	must(err, "generate client key")

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(3),
		Subject: pkix.Name{
			Organization: []string{"My Company"},
			CommonName:   "my-client-app", // identifies this specific client
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}, // <-- CLIENT auth
	}

	// Also signed by the same CA
	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	must(err, "create client cert")

	savePEM("client-cert.pem", "CERTIFICATE", clientCertDER)
	saveKey("client-key.pem", clientKey)
	fmt.Println("✅ Created Client certificate (client-cert.pem, client-key.pem)")
	fmt.Println("\n🔐 All certificates ready! Both server and client certs are signed by the same CA.")
}

// --- Helpers ---

func savePEM(filename, blockType string, data []byte) {
	f, err := os.Create(filename)
	must(err, "create "+filename)
	defer f.Close()
	pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}

func saveKey(filename string, key *ecdsa.PrivateKey) {
	der, err := x509.MarshalECPrivateKey(key)
	must(err, "marshal key")
	savePEM(filename, "EC PRIVATE KEY", der)
}

func must(err error, context string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "FATAL: %s: %v\n", context, err)
		os.Exit(1)
	}
}
