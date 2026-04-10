# mTLS in Go — Complete Deep Dive Reference

> A line-by-line, concept-by-concept reference for future you.
> Covers everything from "what is mTLS" to "every line of code explained."

---

## 1. What is TLS vs mTLS?

### Regular TLS (one-way)
- Client connects to server
- Server shows its certificate: "I am server.example.com"
- Client verifies: "Is this cert signed by a CA I trust?"
- If yes → encrypted channel. **Only the server proved its identity.**

### Mutual TLS (mTLS — two-way)
- Everything above, PLUS:
- Server says: "Now show ME your certificate"
- Client presents its certificate
- Server verifies: "Is this client cert signed by a CA I trust?"
- If yes → encrypted channel. **Both sides proved their identity.**

### When do you need mTLS?
- **Service-to-service communication** (microservices inside a cluster)
- **Zero-trust networks** (every connection must prove identity)
- **API security** (stronger than API keys — cryptographic identity)
- **IoT devices** authenticating to a backend
- **Kubernetes** pod-to-pod communication (Istio, Linkerd use mTLS)

---

## 2. The Handshake — Step by Step

```
  CLIENT                                       SERVER
    |                                             |
    |──── 1. ClientHello ────────────────────────>|
    |     (supported ciphers, TLS version)        |
    |                                             |
    |<─── 2. ServerHello + Server Certificate ────|
    |     (server's cert signed by CA)            |
    |                                             |
    |     Client verifies server cert             |
    |     against its RootCAs pool                |
    |                                             |
    |<─── 3. CertificateRequest ─────────────────|  ← THIS MAKES IT "MUTUAL"
    |     (server demands client's cert)          |
    |                                             |
    |──── 4. Client Certificate ─────────────────>|
    |     (client's cert signed by CA)            |
    |                                             |
    |     Server verifies client cert             |
    |     against its ClientCAs pool              |
    |                                             |
    |<──> 5. Key Exchange + Finished ────────────>|
    |                                             |
    |     🔐 Encrypted channel established        |
    |     Both sides authenticated                |
```

**The single line that turns TLS into mTLS on the server:**
```go
ClientAuth: tls.RequireAndVerifyClientCert
```

Without this, step 3 never happens — regular TLS.

---

## 3. Certificate Chain of Trust

```
    ┌─────────────────────────────┐
    │     Root CA (self-signed)   │
    │   "My Company Root CA"      │
    │   IsCA: true                │
    └──────────┬──────────────────┘
               │ signs
       ┌───────┴───────┐
       │               │
       ▼               ▼
 ┌────────────┐  ┌────────────┐
 │  Server    │  │  Client    │
 │  Certificate│  │  Certificate│
 │  CN=localhost│  │  CN=my-app │
 │  ExtKeyUsage│  │  ExtKeyUsage│
 │  =ServerAuth│  │  =ClientAuth│
 └────────────┘  └────────────┘
```

**Both server and client trust the Root CA.**
When either side presents its cert, the other checks:
"Was this signed by a CA in my trusted pool?" → Yes → Authenticated.

---

## 4. Project Structure

```
mtls-demo/
├── go.mod                  # Module definition
├── certs/
│   └── gen_certs.go        # Generates CA + server cert + client cert
├── server/
│   └── main.go             # HTTPS server requiring client certs
└── client/
    └── main.go             # HTTPS client presenting its cert
```

---

## 5. Certificate Generation — Line by Line

### File: `certs/gen_certs.go`

#### Imports explained

```go
import (
    "crypto/ecdsa"          // Elliptic Curve Digital Signature Algorithm (key type)
    "crypto/elliptic"       // Provides the P256 curve (industry standard)
    "crypto/rand"           // Cryptographically secure random number generator
    "crypto/x509"           // X.509 certificate parsing and creation
    "crypto/x509/pkix"      // PKIX (PKI X.509) name structures
    "encoding/pem"          // PEM encoding (the "-----BEGIN CERTIFICATE-----" format)
    "math/big"              // Arbitrary-precision integers (for serial numbers)
    "net"                   // net.IP for IP SANs
    "os"                    // File operations
    "time"                  // Certificate validity periods
)
```

**Why ECDSA P256?** It's the modern standard — 128-bit security with much smaller keys than RSA. A P256 key is 32 bytes; an equivalent RSA key would be 3072 bits (384 bytes). Faster handshakes, smaller certificates.

#### Creating the CA private key

```go
caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
```

| Part | What it does |
|------|-------------|
| `ecdsa.GenerateKey` | Generates an ECDSA private+public key pair |
| `elliptic.P256()` | Uses the NIST P-256 curve (also called secp256r1) |
| `rand.Reader` | Uses the OS cryptographic random source (/dev/urandom on Linux) |
| `caKey` | The private key — KEEP THIS SECRET. Contains both private and public parts |

#### CA certificate template

```go
caTemplate := &x509.Certificate{
    SerialNumber: big.NewInt(1),
```
- `SerialNumber`: Unique identifier for this cert. In production, use a random 128-bit number. Here we use 1 for simplicity.

```go
    Subject: pkix.Name{
        Organization: []string{"My Company"},
        CommonName:   "My Company Root CA",
    },
```
- `Subject`: The identity embedded in the cert. `CommonName` is what shows up in cert viewers. `Organization` groups certs by company.

```go
    NotBefore: time.Now(),
    NotAfter:  time.Now().Add(10 * 365 * 24 * time.Hour),
```
- `NotBefore`/`NotAfter`: The validity window. This CA is valid for 10 years. CAs are long-lived; leaf certs should be short-lived (1 year or less).

```go
    KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
```
- `KeyUsageCertSign`: This key can sign other certificates (essential for a CA).
- `KeyUsageCRLSign`: This key can sign Certificate Revocation Lists.
- These are bitflags combined with `|`.

```go
    BasicConstraintsValid: true,
    IsCA:                  true,
    MaxPathLen:            1,
}
```
- `IsCA: true`: **THE critical field** — marks this as a Certificate Authority. Without this, the cert cannot sign other certs.
- `MaxPathLen: 1`: This CA can sign leaf certs (path length 0) but not intermediate CAs (limits the chain depth).
- `BasicConstraintsValid: true`: Must be true for IsCA and MaxPathLen to be included in the cert.

#### Creating the self-signed CA cert

```go
caCertDER, err := x509.CreateCertificate(
    rand.Reader,      // randomness source
    caTemplate,       // certificate template (what goes IN the cert)
    caTemplate,       // parent certificate (who signs it)
    &caKey.PublicKey,  // public key to embed in the cert
    caKey,            // private key to sign WITH
)
```

**Key insight: `parent == template` means self-signed.** The CA signs itself. For leaf certs (server/client), the parent will be the CA cert and the signing key will be the CA's private key.

The result `caCertDER` is the certificate in DER format (raw binary ASN.1). We'll PEM-encode it for storage.

#### Parsing back the CA cert

```go
caCert, err := x509.ParseCertificate(caCertDER)
```

We need the parsed `*x509.Certificate` object (not just the bytes) because we'll pass it as the `parent` parameter when signing the server and client certs.

#### Server certificate template

```go
serverTemplate := &x509.Certificate{
    SerialNumber: big.NewInt(2),          // Different from CA's serial
    Subject: pkix.Name{
        Organization: []string{"My Company"},
        CommonName:   "localhost",         // The hostname
    },
    NotBefore: time.Now(),
    NotAfter:  time.Now().Add(365 * 24 * time.Hour),  // 1 year (shorter than CA)
```

```go
    KeyUsage:    x509.KeyUsageDigitalSignature,
```
- `KeyUsageDigitalSignature`: This key can create digital signatures (needed for TLS handshake). Note: NO `KeyUsageCertSign` — leaf certs must NOT sign other certs.

```go
    ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
```
- **`ExtKeyUsageServerAuth`**: Marks this cert as valid for TLS server authentication. A client connecting to this server will check for this extension. If you tried to use a `ClientAuth` cert as a server cert, the TLS library would reject it.

```go
    DNSNames:    []string{"localhost"},
    IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
}
```
- `DNSNames` and `IPAddresses`: **Subject Alternative Names (SANs)**. These define what hostnames/IPs the cert is valid for. When a client connects to `https://localhost:8443`, Go checks: "Is `localhost` in the cert's SANs?" If not → rejected.
- In production, you'd put your real domain(s) and/or internal IPs here.

#### Signing the server cert with the CA

```go
serverCertDER, err := x509.CreateCertificate(
    rand.Reader,
    serverTemplate,    // what goes in the cert
    caCert,            // ← PARENT IS THE CA (not self-signed!)
    &serverKey.PublicKey,
    caKey,             // ← SIGNED WITH CA's PRIVATE KEY
)
```

**This creates the chain of trust:** The server cert is signed by the CA. Anyone who trusts the CA can verify this server cert.

#### Client certificate template

```go
clientTemplate := &x509.Certificate{
    SerialNumber: big.NewInt(3),
    Subject: pkix.Name{
        Organization: []string{"My Company"},
        CommonName:   "my-client-app",    // Identifies THIS specific client
    },
```
- `CommonName`: In mTLS, the server can read this to identify WHICH client connected. Useful for authorization: "Client `my-client-app` is allowed to access `/api/secret`, but `monitoring-bot` is not."

```go
    ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
```
- **`ExtKeyUsageClientAuth`**: The counterpart to `ServerAuth`. Marks this cert as valid for TLS client authentication. The server checks for this extension when verifying the client cert.

#### PEM encoding helper

```go
func savePEM(filename, blockType string, data []byte) {
    f, err := os.Create(filename)
    defer f.Close()
    pem.Encode(f, &pem.Block{Type: blockType, Bytes: data})
}
```

PEM is the text format you see in cert files:
```
-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJALRiMLAh...base64 encoded DER...
-----END CERTIFICATE-----
```

`blockType` is "CERTIFICATE" or "EC PRIVATE KEY" — it's the text after "BEGIN".

---

## 6. Server Code — Line by Line

### File: `server/main.go`

#### Loading the server's certificate

```go
serverCert, err := tls.LoadX509KeyPair(
    "certs/server-cert.pem",  // The certificate (public)
    "certs/server-key.pem",   // The private key (secret)
)
```

`tls.LoadX509KeyPair` reads both files and returns a `tls.Certificate` struct. This struct holds:
- The certificate chain (as DER bytes)
- The private key
- The parsed leaf certificate (lazily)

Go's TLS library uses this to:
1. Present the cert to clients during the handshake
2. Use the private key to complete the key exchange

#### Building the CA trust pool

```go
caCert, err := os.ReadFile("certs/ca-cert.pem")

caCertPool := x509.NewCertPool()
caCertPool.AppendCertsFromPEM(caCert)
```

| Line | Purpose |
|------|---------|
| `os.ReadFile(...)` | Read the CA cert file as raw bytes |
| `x509.NewCertPool()` | Create an empty pool of trusted certificates |
| `AppendCertsFromPEM(...)` | Parse the PEM data and add the CA cert to the pool |

**This pool answers the question:** "Which CAs do I trust for verifying CLIENT certificates?" You can add multiple CAs to support cert rotation or multiple clients from different organizations.

#### The TLS config — the heart of mTLS

```go
tlsConfig := &tls.Config{
```

##### Field 1: Certificates

```go
    Certificates: []tls.Certificate{serverCert},
```
- **What:** The server's own cert+key pair(s).
- **When used:** During step 2 of the handshake — the server presents this cert to the client.
- **Why a slice?** You can serve different certs for different SNI (Server Name Indication) hostnames. For most cases, one cert is enough.

##### Field 2: ClientAuth — THE mTLS SWITCH

```go
    ClientAuth: tls.RequireAndVerifyClientCert,
```
- **What:** Controls whether and how the server requests client certificates.
- **This single line is what makes it mTLS.** Without it (default = `NoClientCert`), you have regular TLS.

**All 5 levels explained:**

| Value | Behavior |
|-------|----------|
| `NoClientCert` | Default. Regular TLS. Server never asks for client cert. |
| `RequestClientCert` | Server asks for cert but accepts connections without one. No verification. |
| `RequireAnyClientCert` | Client MUST send a cert, but server doesn't verify the CA chain. (Rarely useful.) |
| `VerifyClientCertIfGiven` | If client sends a cert, verify it. If not, allow anyway. (Good for gradual rollout.) |
| `RequireAndVerifyClientCert` | Client MUST send a cert AND it must be signed by a trusted CA. **Full mTLS.** |

##### Field 3: ClientCAs

```go
    ClientCAs: caCertPool,
```
- **What:** The CA pool used to verify incoming client certificates.
- **When used:** During step 5 — after the client sends its cert, the server checks: "Is this cert signed by a CA in `ClientCAs`?"
- **If nil:** Go uses the system cert store (rarely what you want for mTLS).

##### Field 4: MinVersion

```go
    MinVersion: tls.VersionTLS12,
}
```
- **What:** Minimum TLS version to accept.
- **Why TLS 1.2?** TLS 1.0 and 1.1 are deprecated (RFC 8996). TLS 1.2 is the minimum for modern security. TLS 1.3 is even better but Go negotiates it automatically when both sides support it.

#### Reading the client's identity from the request

```go
mux.HandleFunc("/api/secret", func(w http.ResponseWriter, r *http.Request) {
    if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
        clientCN := r.TLS.PeerCertificates[0].Subject.CommonName
        clientOrg := r.TLS.PeerCertificates[0].Subject.Organization
```

| Field | What it is |
|-------|-----------|
| `r.TLS` | `*tls.ConnectionState` — info about the TLS connection. nil if not TLS. |
| `r.TLS.PeerCertificates` | The client's verified certificate chain. Index `[0]` = leaf cert. |
| `.Subject.CommonName` | The CN field from the client cert (e.g., "my-client-app"). |
| `.Subject.Organization` | The org field (e.g., ["My Company"]). |
| `.SerialNumber` | The cert's serial number — unique identifier. |

**By the time your handler runs, mTLS has already succeeded.** The TLS handshake completes before HTTP processing begins. If the client didn't have a valid cert, the connection was rejected at the transport layer — your handler never sees it.

**This means:** You can use `CommonName` or `Organization` for **authorization** (access control), not just authentication. Example:

```go
if clientCN == "admin-service" {
    // Allow admin operations
} else if clientCN == "read-only-service" {
    // Allow read-only operations
}
```

#### Starting the server

```go
server := &http.Server{
    Addr:      ":8443",
    Handler:   mux,
    TLSConfig: tlsConfig,
}

server.ListenAndServeTLS("", "")
```

- `":8443"` — Port 8443 is conventional for HTTPS with non-standard certs.
- `ListenAndServeTLS("", "")` — Empty strings because we already set certs in `tlsConfig.Certificates`. If you pass file paths here, Go uses `tlsConfig.Certificates` first anyway.

---

## 7. Client Code — Line by Line

### File: `client/main.go`

#### Loading the client's certificate

```go
clientCert, err := tls.LoadX509KeyPair(
    "certs/client-cert.pem",
    "certs/client-key.pem",
)
```

Same as the server loading its cert. This is the cert the client will present when the server sends `CertificateRequest` (step 3 of the handshake).

**Experiment:** Comment out this line and the `Certificates` field below. The server will reject you with:
```
tls: client didn't provide a certificate
```

#### Client TLS config

```go
tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{clientCert},  // ← THE mTLS PART
    RootCAs:      caCertPool,
    MinVersion:   tls.VersionTLS12,
}
```

**Server vs Client config comparison:**

| Field | Server | Client | Purpose |
|-------|--------|--------|---------|
| `Certificates` | Server's cert+key | Client's cert+key | "Here's MY identity" |
| `ClientAuth` | `RequireAndVerifyClientCert` | *(not used)* | Server demands client cert |
| `ClientCAs` | CA pool for verifying clients | *(not used)* | Server verifies client certs |
| `RootCAs` | *(not used)* | CA pool for verifying server | Client verifies server cert |

**Notice the symmetry:**
- Server uses `ClientCAs` to verify the client → "Is the client cert signed by a CA I trust?"
- Client uses `RootCAs` to verify the server → "Is the server cert signed by a CA I trust?"
- Both use `Certificates` to present their own identity.

#### Creating the HTTP client with TLS

```go
client := &http.Client{
    Transport: &http.Transport{
        TLSClientConfig: tlsConfig,
    },
}
```

- `http.Client` is Go's HTTP client.
- `http.Transport` is the low-level HTTP transport (connection pooling, TLS, proxies).
- `TLSClientConfig` plugs our mTLS config into the transport layer.
- Every request made with this `client` will present the client certificate.

#### Making the request

```go
resp, err := client.Get("https://localhost:8443/api/secret")
```

**Under the hood, this triggers the full mTLS handshake:**
1. TCP connect to localhost:8443
2. TLS ClientHello → server
3. Server sends its cert → client verifies against `RootCAs` ✓
4. Server sends CertificateRequest → client
5. Client sends its cert from `Certificates` → server
6. Server verifies client cert against its `ClientCAs` ✓
7. Key exchange → encrypted channel
8. HTTP GET /api/secret flows over the encrypted channel

All of this happens inside `client.Get()` — Go's `crypto/tls` package handles it transparently.

#### The failure demo

```go
noMTLSConfig := &tls.Config{
    RootCAs:    caCertPool,     // Still trust the server
    MinVersion: tls.VersionTLS12,
    // NO Certificates field!
}
```

Without `Certificates`, the client has nothing to present when the server demands a cert. The TLS handshake fails at step 4. The error:
```
tls: certificate required
```

**This proves mTLS is working** — the server enforces client authentication at the transport layer.

---

## 8. Go Types Quick Reference

### `tls.Config`
The central configuration struct for TLS in Go. Used by both client and server.

```go
type Config struct {
    Certificates []Certificate       // My cert(s) to present
    RootCAs      *x509.CertPool     // CAs I trust for verifying SERVERS (client-side)
    ClientCAs    *x509.CertPool     // CAs I trust for verifying CLIENTS (server-side)
    ClientAuth   ClientAuthType      // How strictly to require client certs
    MinVersion   uint16              // Minimum TLS version (VersionTLS12, VersionTLS13)
    // ... many more fields
}
```

### `tls.Certificate`
A certificate+key pair. Created by `tls.LoadX509KeyPair()`.

### `x509.CertPool`
A pool of trusted CA certificates. Created by `x509.NewCertPool()`, populated with `AppendCertsFromPEM()`.

### `x509.Certificate`
A parsed X.509 certificate. Used as a template for `x509.CreateCertificate()` and returned in `r.TLS.PeerCertificates`.

---

## 9. How to Run

```bash
# Terminal 1: Generate certificates
cd certs && go run gen_certs.go && cd ..
# Output: ca-cert.pem, ca-key.pem, server-cert.pem, server-key.pem,
#         client-cert.pem, client-key.pem

# Terminal 2: Start server
go run server/main.go
# Output: 🔐 mTLS server starting on https://localhost:8443

# Terminal 3: Run client
go run client/main.go
# Output:
#   ✅ Response (status 200):
#   Hello, my-client-app! You are mTLS-authenticated.
#   --- Now trying WITHOUT client cert (should fail) ---
#   ❌ Expected failure: ...certificate required
```

---

## 10. Testing with curl

```bash
# This WORKS — full mTLS with client cert:
curl --cacert certs/ca-cert.pem \
     --cert certs/client-cert.pem \
     --key certs/client-key.pem \
     https://localhost:8443/api/secret

# This FAILS — no client cert:
curl --cacert certs/ca-cert.pem \
     https://localhost:8443/api/secret
# Error: SSL handshake failure (alert: certificate required)
```

| curl flag | Purpose | Maps to Go field |
|-----------|---------|-----------------|
| `--cacert` | Trust this CA for verifying the server | `RootCAs` |
| `--cert` | Present this client certificate | `Certificates` (cert part) |
| `--key` | Client's private key | `Certificates` (key part) |

---

## 11. Production Considerations

### Certificate Rotation
- Use short-lived certs (hours/days, not years)
- Automate renewal with tools like cert-manager, Vault, or SPIFFE/SPIRE
- Go's `tls.Config.GetCertificate` and `GetClientCertificate` callbacks allow dynamic cert loading without server restart

### Certificate Revocation
- CRLs (Certificate Revocation Lists) or OCSP for revoking compromised certs
- Go's `x509.Certificate.CheckCRLSignature` for CRL verification

### Multiple CAs
```go
caCertPool.AppendCertsFromPEM(oldCA)
caCertPool.AppendCertsFromPEM(newCA)
// Both are trusted — supports CA rotation
```

### Authorization Beyond Authentication
mTLS proves identity, but you still need authorization:
```go
cn := r.TLS.PeerCertificates[0].Subject.CommonName
switch cn {
case "admin-service":
    // full access
case "readonly-service":
    // read-only
default:
    http.Error(w, "forbidden", 403)
}
```

### Using with gRPC
```go
creds := credentials.NewTLS(tlsConfig)
server := grpc.NewServer(grpc.Creds(creds))
// Same tls.Config, just wrapped in gRPC credentials
```

---

## 12. Common Errors and What They Mean

| Error | Cause | Fix |
|-------|-------|-----|
| `tls: client didn't provide a certificate` | Client didn't send a cert | Add `Certificates` to client's tls.Config |
| `x509: certificate signed by unknown authority` | Cert not signed by a trusted CA | Add the CA cert to the correct pool (RootCAs or ClientCAs) |
| `x509: certificate is valid for X, not Y` | SAN mismatch | Add the hostname/IP to DNSNames or IPAddresses in the cert template |
| `x509: certificate has expired` | NotAfter is in the past | Regenerate or rotate the certificate |
| `tls: private key does not match public key` | Cert and key files are mismatched | Ensure LoadX509KeyPair gets the matching cert+key pair |

---

*Last updated: April 2026. Based on Go's crypto/tls standard library.*
