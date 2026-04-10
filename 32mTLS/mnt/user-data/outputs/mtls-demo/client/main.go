// client/main.go — HTTPS client that presents its certificate (mTLS)
//
// DEEP DIVE: Client-Side mTLS Configuration
// ==========================================
// The client's tls.Config mirrors the server's but with different roles:
//
//   1. Certificates  — "Here's MY cert to prove who I am TO THE SERVER"
//   2. RootCAs       — "I trust servers whose certs are signed by THESE CAs"
//
// Notice: the server uses ClientCAs (to verify clients), but the client
// uses RootCAs (to verify the server). Different field, same concept.
//
// WITHOUT the Certificates field, this would be normal TLS.
// WITH it, the client presents its cert → mutual authentication → mTLS.

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func main() {
	// ============================================================
	// STEP 1: Load the client's own certificate + private key
	// ============================================================
	// This is the cert the client presents to the server when the
	// server sends CertificateRequest (step 3 in the mTLS flow).
	//
	// If you comment this out, the server will reject you with:
	//   "tls: client didn't provide a certificate"

	clientCert, err := tls.LoadX509KeyPair("certs/client-cert.pem", "certs/client-key.pem")
	if err != nil {
		log.Fatalf("Failed to load client cert: %v", err)
	}

	// ============================================================
	// STEP 2: Load the CA certificate to verify the server
	// ============================================================
	// This CA pool is used to verify the SERVER's certificate.
	// The client asks: "Was the server's cert signed by a CA I trust?"
	//
	// This is the same CA that signed the server cert.
	// In production, you'd load your org's CA bundle here.

	caCert, err := os.ReadFile("certs/ca-cert.pem")
	if err != nil {
		log.Fatalf("Failed to read CA cert: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to add CA cert to pool")
	}

	// ============================================================
	// STEP 3: Configure TLS for the client
	// ============================================================
	// Two critical fields:
	//
	// Certificates: []tls.Certificate{clientCert}
	//   → "When the server asks for my cert, present THIS one."
	//   → This is the field that makes it MUTUAL TLS.
	//   → Without this, the server's RequireAndVerifyClientCert
	//     will reject the handshake.
	//
	// RootCAs: caCertPool
	//   → "I trust servers whose certs are signed by this CA."
	//   → Without this, Go would use the system cert store,
	//     which doesn't know about our self-signed CA.

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert}, // ← THE mTLS PART
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	// ============================================================
	// STEP 4: Create an HTTP client with our TLS config
	// ============================================================

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// ============================================================
	// STEP 5: Make the request — mTLS handshake happens here
	// ============================================================
	// Under the hood, Go's TLS library:
	//   1. Connects to the server
	//   2. Receives the server's certificate
	//   3. Verifies it against RootCAs ✓
	//   4. Receives CertificateRequest from server
	//   5. Sends our client certificate
	//   6. Server verifies our cert against its ClientCAs ✓
	//   7. Encrypted channel established — BOTH sides authenticated

	fmt.Println("🔐 Connecting to mTLS server...")
	resp, err := client.Get("https://localhost:8443/api/secret")
	if err != nil {
		log.Fatalf("❌ Request failed: %v\n\nThis usually means:\n"+
			"  - Server isn't running\n"+
			"  - Client cert not trusted by server's CA\n"+
			"  - Server cert not trusted by client's CA\n", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("✅ Response (status %d):\n%s", resp.StatusCode, body)

	// ============================================================
	// BONUS: Try without client cert to see the failure
	// ============================================================
	fmt.Println("\n--- Now trying WITHOUT client cert (should fail) ---")

	noMTLSConfig := &tls.Config{
		RootCAs:    caCertPool,    // Still trust the server's CA
		MinVersion: tls.VersionTLS12,
		// NO Certificates field — no client cert to present
	}

	noMTLSClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: noMTLSConfig,
		},
	}

	_, err = noMTLSClient.Get("https://localhost:8443/api/secret")
	if err != nil {
		fmt.Printf("❌ Expected failure: %v\n", err)
		fmt.Println("   ^ This proves mTLS is working — no cert = no access!")
	}
}
