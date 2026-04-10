# mTLS in Go — Complete Deep Dive Project

## Project Structure

```
mtls-demo/
├── go.mod
├── certs/
│   └── gen_certs.go        # Step 1: Generate all certificates programmatically
├── server/
│   └── main.go             # Step 2: HTTPS server requiring client certs
├── client/
│   └── main.go             # Step 3: HTTPS client presenting its cert
└── README.md
```

## How to Run

```bash
# 1. Generate certificates (CA, server cert, client cert)
cd certs && go run gen_certs.go && cd ..

# 2. Start the server (in one terminal)
go run server/main.go

# 3. Run the client (in another terminal)
go run client/main.go
```

## What Happens

1. `gen_certs.go` creates a self-signed CA, then signs both a server cert and a client cert
2. The server loads its cert + the CA cert, and sets `ClientAuth: tls.RequireAndVerifyClientCert`
3. The client loads its cert + the CA cert, connects, and both sides verify each other
4. If you remove the client cert, the server rejects the connection — that's the "mutual" part

## Deep Dive: Key Go Types

### `tls.Config` (the heart of mTLS)

**Server side:**
- `Certificates` — the server's own cert+key pair
- `ClientAuth` — set to `RequireAndVerifyClientCert` to enforce mTLS
- `ClientCAs` — the CA pool used to verify incoming client certificates

**Client side:**
- `Certificates` — the client's own cert+key pair (this is what makes it *mutual*)
- `RootCAs` — the CA pool used to verify the server's certificate

### `x509.CertPool`

A set of trusted CA certificates. Both sides use this to verify the other's cert chain.
In production, you'd load your organization's internal CA here.

### Certificate Chain of Trust

```
Root CA (self-signed)
 ├── Server Certificate (signed by Root CA)
 └── Client Certificate (signed by Root CA)
```

Both server and client trust the Root CA. When either side presents its cert,
the other verifies: "Was this cert signed by a CA I trust?" If yes → authenticated.
