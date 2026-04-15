package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
)

// StartServer launches an HTTPS server that REQUIRES a valid client certificate.
// Any client without a cert signed by our CA is rejected at the TLS layer —
// the HTTP handler never even runs.
func StartServer(serverCert tls.Certificate, caPool *x509.CertPool, addr string) *http.Server {

	// ── The critical mTLS config ──────────────────────────────────────────
	tlsConfig := &tls.Config{
		// Our own certificate (proves server identity to the client)
		Certificates: []tls.Certificate{serverCert},

		// ClientCAs: the pool of CAs we trust for CLIENT certificates.
		// Only clients whose cert was signed by one of these CAs will be allowed.
		ClientCAs: caPool,

		// RequireAndVerifyClientCert is the line that turns TLS → mTLS.
		// Without this, client certs are optional (normal TLS).
		ClientAuth: tls.RequireAndVerifyClientCert,

		MinVersion: tls.VersionTLS13, // TLS 1.3 only — best security
	}
	// ─────────────────────────────────────────────────────────────────────

	mux := http.NewServeMux()

	// /order — the protected endpoint
	mux.HandleFunc("/order", func(w http.ResponseWriter, r *http.Request) {
		// If we get here, the client cert was already verified by TLS.
		// We can safely inspect it for authorization logic.
		clientCN := r.TLS.PeerCertificates[0].Subject.CommonName

		fmt.Printf("[server] Request from verified client: %q\n", clientCN)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","message":"Order accepted","client":"%s"}`, clientCN)
	})

	// /health — also protected (all routes require mTLS with this config)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		clientCN := r.TLS.PeerCertificates[0].Subject.CommonName
		fmt.Fprintf(w, `{"status":"healthy","verified_client":"%s"}`, clientCN)
	})

	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	go func() {
		fmt.Printf("[server] Listening on https://%s (mTLS required)\n", addr)
		// ListenAndServeTLS with empty strings because certs are in TLSConfig
		if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			fmt.Printf("[server] Error: %v\n", err)
		}
	}()

	return server
}
