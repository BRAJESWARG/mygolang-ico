package main

import (
	"context"
	"crypto/x509"
	"fmt"
	"time"
)

func main() {
	fmt.Println("═══════════════════════════════════════════")
	fmt.Println("         mTLS Deep Dive Demo in Go         ")
	fmt.Println("═══════════════════════════════════════════\n")

	// ── Phase 1: Certificate Authority ───────────────────────────────────
	fmt.Println("── Phase 1: Generating Certificate Authority ──")
	caCert, caKey, caPEM, err := GenerateCA()
	must(err, "generate CA")
	fmt.Printf("✓ CA created: %q\n\n", caCert.Subject.CommonName)

	// Build a cert pool that both client and server will use to
	// verify each other's certificates.
	caPool := x509.NewCertPool()
	caPool.AppendCertsFromPEM(caPEM)

	// ── Phase 2: Issue Leaf Certificates ─────────────────────────────────
	fmt.Println("── Phase 2: Issuing Leaf Certificates ──")

	serverCert, err := SignCertificate(caCert, caKey, "order-service", true)
	must(err, "sign server cert")
	fmt.Println("✓ Server cert issued: CN=order-service (ExtKeyUsage: ServerAuth, SAN: localhost)")

	clientCert, err := SignCertificate(caCert, caKey, "payment-service", false)
	must(err, "sign client cert")
	fmt.Println("✓ Client cert issued: CN=payment-service (ExtKeyUsage: ClientAuth)\n")

	// ── Phase 3: Start mTLS Server ───────────────────────────────────────
	fmt.Println("── Phase 3: Starting mTLS Server ──")
	server := StartServer(serverCert, caPool, "localhost:8443")
	time.Sleep(100 * time.Millisecond) // let server start

	// ── Phase 4: Authorized Request ──────────────────────────────────────
	fmt.Println("\n── Phase 4: Authorized Client Request ──")
	fmt.Println("[client] Connecting with valid client cert...")

	body, err := MakeRequest(clientCert, caPool, "https://localhost:8443/order")
	must(err, "authorized request")
	fmt.Printf("[client] Response: %s\n", body)

	// ── Phase 5: Health Check ─────────────────────────────────────────────
	fmt.Println("\n── Phase 5: Health Check ──")
	body, err = MakeRequest(clientCert, caPool, "https://localhost:8443/health")
	must(err, "health check")
	fmt.Printf("[client] Response: %s\n", body)

	// ── Phase 6: Unauthorized Attempt ────────────────────────────────────
	fmt.Println("\n── Phase 6: Unauthorized Client (no cert) ──")
	fmt.Println("[intruder] Attempting connection WITHOUT a client certificate...")

	err = MakeRequestNoCert(caPool, "https://localhost:8443/order")
	if err != nil {
		fmt.Printf("[intruder] ✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("[intruder] ✗ Should have been rejected but wasn't!")
	}

	// ── Phase 7: Wrong CA (completely untrusted cert) ────────────────────
	fmt.Println("\n── Phase 7: Rogue Client (cert from different CA) ──")
	fmt.Println("[rogue] Generating a self-signed cert not from our CA...")

	// Create a completely separate CA — its certs are unknown to our server
	rogueCACert, rogueCAKey, _, err := GenerateCA()
	must(err, "generate rogue CA")
	rogueCert, err := SignCertificate(rogueCACert, rogueCAKey, "rogue-client", false)
	must(err, "sign rogue cert")

	err = MakeRequest(rogueCert, caPool, "https://localhost:8443/order")
	if err != nil {
		fmt.Printf("[rogue]    ✓ Correctly rejected: %v\n", err)
	} else {
		fmt.Println("[rogue]    ✗ Should have been rejected!")
	}

	// ── Done ─────────────────────────────────────────────────────────────
	fmt.Println("\n═══════════════════════════════════════════")
	fmt.Println("  Demo complete. Shutting down server.")
	fmt.Println("═══════════════════════════════════════════")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}

func must(err error, context string) {
	if err != nil {
		panic(fmt.Sprintf("FATAL [%s]: %v", context, err))
	}
}
