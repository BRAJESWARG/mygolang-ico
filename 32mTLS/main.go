// server/main.go — HTTPS server that REQUIRES client certificates (mTLS)
//
// DEEP DIVE: Server-Side mTLS Configuration
// ==========================================
// The magic happens in tls.Config. Three fields matter:
//
//   1. Certificates     — "Here's MY cert to prove who I am"
//   2. ClientAuth       — "I REQUIRE clients to show their cert too"
//   3. ClientCAs        — "I only trust client certs signed by THESE CAs"
//
// The ClientAuth field has 5 levels:
//   - NoClientCert           (default — regular TLS, no mTLS)
//   - RequestClientCert      (ask but don't require)
//   - RequireAnyClientCert   (require cert, but don't verify CA chain)
//   - VerifyClientCertIfGiven (verify if provided, but don't require)
//   - RequireAndVerifyClientCert  ← THIS is full mTLS

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
	// STEP 1: Load the server's own certificate + private key
	// ============================================================
	// This is the cert the server presents to clients during the
	// TLS handshake (step 2 in the mTLS flow).

	serverCert, err := tls.LoadX509KeyPair("certs/server-cert.pem", "certs/server-key.pem")
	if err != nil {
		log.Fatalf("Failed to load server cert: %v", err)
	}

	// ============================================================
	// STEP 2: Load the CA certificate into a trust pool
	// ============================================================
	// This CA pool is used to verify INCOMING client certificates.
	// The server asks: "Was this client's cert signed by a CA in my pool?"
	//
	// In production, this pool would contain your org's internal CA(s).
	// You might load multiple CAs to support cert rotation.

	caCert, err := os.ReadFile("certs/ca-cert.pem")
	if err != nil {
		log.Fatalf("Failed to read CA cert: %v", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		log.Fatal("Failed to add CA cert to pool")
	}

	// ============================================================
	// STEP 3: Configure TLS with mTLS enabled
	// ============================================================
	// THIS IS WHERE THE MAGIC HAPPENS.
	//
	// ClientAuth: tls.RequireAndVerifyClientCert
	//   → Server will REJECT any connection that doesn't present
	//     a valid client certificate signed by a trusted CA.
	//
	// ClientCAs: the CA pool from step 2
	//   → Used to verify the client's certificate chain.
	//
	// MinVersion: tls.VersionTLS12
	//   → Don't allow TLS 1.0/1.1 (deprecated, insecure).

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert, // ← THE mTLS SWITCH
		ClientCAs:    caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	// ============================================================
	// STEP 4: Create the HTTPS server with our TLS config
	// ============================================================

	mux := http.NewServeMux()

	// This handler extracts info from the verified client certificate.
	// By the time we reach here, mTLS has already succeeded — the client
	// is authenticated at the transport layer.
	mux.HandleFunc("/api/secret", func(w http.ResponseWriter, r *http.Request) {
		// r.TLS.PeerCertificates contains the client's verified cert chain.
		// Index [0] is the client's leaf certificate.
		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			clientCN := r.TLS.PeerCertificates[0].Subject.CommonName
			clientOrg := r.TLS.PeerCertificates[0].Subject.Organization

			log.Printf("✅ Authenticated client: CN=%s, Org=%v", clientCN, clientOrg)

			fmt.Fprintf(w, "Hello, %s! You are mTLS-authenticated.\n", clientCN)
			fmt.Fprintf(w, "Your cert org: %v\n", clientOrg)
			fmt.Fprintf(w, "Cert serial: %s\n", r.TLS.PeerCertificates[0].SerialNumber)
		} else {
			// This should never happen with RequireAndVerifyClientCert,
			// because the TLS handshake would fail before reaching here.
			http.Error(w, "No client certificate", http.StatusUnauthorized)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "OK\n")
	})

	server := &http.Server{
		Addr:      ":8443",
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	log.Println("🔐 mTLS server starting on https://localhost:8443")
	log.Println("   ClientAuth = RequireAndVerifyClientCert")
	log.Println("   Clients without valid certs will be REJECTED at TLS handshake")

	// ListenAndServeTLS: the cert/key args here are ignored because we
	// already set them in tlsConfig.Certificates. Pass empty strings.
	// (Or pass the paths again — Go will use tlsConfig.Certificates first.)
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
