package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
)

// MakeRequest sends an mTLS request to the server.
// clientCert: the client's own cert (proves its identity)
// caPool:     the CA pool used to verify the SERVER's certificate
func MakeRequest(clientCert tls.Certificate, caPool *x509.CertPool, url string) (string, error) {

	// ── The critical mTLS config ──────────────────────────────────────────
	tlsConfig := &tls.Config{
		// Certificates: our cert that we present to the server
		Certificates: []tls.Certificate{clientCert},

		// RootCAs: the CA pool we trust for verifying the SERVER's cert.
		// Without this, Go would use the system cert pool (which won't trust
		// our self-signed CA).
		RootCAs: caPool,

		MinVersion: tls.VersionTLS13,
	}
	// ─────────────────────────────────────────────────────────────────────

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client := &http.Client{Transport: transport}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// MakeRequestNoCert sends a request WITHOUT a client certificate.
// This simulates an unauthorized caller — should be rejected by the server.
func MakeRequestNoCert(caPool *x509.CertPool, url string) error {
	tlsConfig := &tls.Config{
		RootCAs:    caPool,
		MinVersion: tls.VersionTLS13,
		// No Certificates field — we don't present a client cert
	}

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	_, err := client.Get(url)
	return err // we expect this to fail
}
