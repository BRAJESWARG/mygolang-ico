# JWT Internal Working - Deep Dive with Two Signing Strategies

## 1. WHAT IS JWT?

JWT (JSON Web Token) is a compact, self-contained way to transmit information between parties as a JSON object.

**Structure:** `Header.Payload.Signature`

Example:
```
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJ1c2VyX2lkIjoidXNlcl8xMjMiLCJ1c2VybmFtZSI6ImpvaG5fZG9lIn0.
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

---

## 2. JWT INTERNAL STRUCTURE (DEEP DIVE)

### PART 1: Header (Information about the token)
```json
{
  "alg": "HS256",     // Signing algorithm
  "typ": "JWT"        // Token type
}
```

**How it's created:**
1. Create JSON object
2. Convert to string: `{"alg":"HS256","typ":"JWT"}`
3. Base64URL encode: Remove padding, use URL-safe characters

**Algorithm options:**
- `HS256` - HMAC with SHA-256 (symmetric)
- `RS256` - RSA with SHA-256 (asymmetric)
- `ES256` - ECDSA with SHA-256 (asymmetric)

---

### PART 2: Payload (Claims - the actual data)
```json
{
  "user_id": "user_123",
  "username": "john_doe",
  "email": "john@example.com",
  "exp": 1699564800,     // Expiration time (Unix timestamp)
  "iat": 1699478400      // Issued at (Unix timestamp)
}
```

**Standard claims (optional but recommended):**
- `iss` - Issuer
- `sub` - Subject
- `aud` - Audience
- `exp` - Expiration time
- `nbf` - Not before
- `iat` - Issued at
- `jti` - JWT ID

**How it's created:**
1. Create JSON object with claims
2. Base64URL encode the JSON string

---

### PART 3: Signature (Proof of integrity and authenticity)

This is where the TWO STRATEGIES differ!

---

## 3. STRATEGY 1: HMAC-SHA256 (Symmetric Signing)

### How HMAC Works Internally:

```
Signing Input = Base64URL(Header) + "." + Base64URL(Payload)
             = "eyJhbGci..." + "." + "eyJ1c2Vy..."

HMAC-SHA256(Signing Input, Secret Key) = Signature
```

### Step-by-Step Process:

```
1. PREPARE SIGNING INPUT
   Input = Header.Payload
   
2. HASH WITH SECRET
   Hash(Input, Secret) using HMAC-SHA256
   
3. ENCODE RESULT
   Base64URL(Hash) = Signature
   
4. FINAL TOKEN
   Token = Header.Payload.Signature
```

### Key Characteristics:

- **Symmetric**: Uses ONE shared secret key
- **Fast**: Simple mathematical operation
- **Smaller**: Signature is relatively small
- **Risk**: Anyone with the secret can forge tokens

### Visual Example:

```
Secret Key: "my-super-secret-key-12345"

Message: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoianNvbiJ9

HMAC-SHA256 Algorithm:
  1. Block size = 64 bytes
  2. If key > 64 bytes: Hash it
  3. Pad key to 64 bytes
  4. Create outer & inner padding (XOR operations)
  5. Hash = SHA256(outer_pad + SHA256(inner_pad + message))
  6. Result: 32-byte signature (256 bits)

Signature: SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

### Verification Process:

```
Received Token:
  Header.Payload.ReceivedSignature

1. Extract Header and Payload
2. Recalculate: HMAC-SHA256(Header.Payload, Secret)
3. Compare: CalculatedSignature == ReceivedSignature?
4. If match: Token is authentic
5. If mismatch: Token was tampered or forged
```

---

## 4. STRATEGY 2: RSA-SHA256 (Asymmetric Signing)

### How RSA Works Internally:

```
Key Pair:
  - Private Key: Keep secret (create & verify signatures)
  - Public Key: Share freely (verify signatures only)

Signing:
  SHA256(Header.Payload) → Hash
  Hash encrypted with Private Key → Signature
  
Verification:
  Hash decrypted with Public Key
  Compare with new SHA256(Header.Payload)
```

### Step-by-Step Process:

```
1. HASH THE INPUT
   Signing Input = Header.Payload
   Hash = SHA256(Signing Input)  [256 bits]
   
2. ENCRYPT HASH WITH PRIVATE KEY
   Signature = RSA_Encrypt(Hash, PrivateKey)
   Result: 2048-bit signature (256 bytes)
   
3. ENCODE RESULT
   Base64URL(Signature) = SignatureB64
   
4. FINAL TOKEN
   Token = Header.Payload.SignatureB64
```

### RSA Mathematical Foundation:

```
RSA Keys:
  - N = p × q (product of two large primes)
  - Private exponent: d
  - Public exponent: e (usually 65537)

Signing (PKCS#1 v1.5):
  Signature = (Hash)^d mod N
  
Verification:
  Hash' = (Signature)^e mod N
  Compare: Hash' == Original Hash?
```

### Key Characteristics:

- **Asymmetric**: Uses public/private key pair
- **Slower**: Complex mathematical operations
- **Larger**: Signature is much bigger
- **Secure**: Only private key holder can sign
- **Scalable**: Public key can be shared with anyone

### Visual Example:

```
Key Pair (2048-bit RSA):
  Private Key: (stored securely on server)
  -----BEGIN PRIVATE KEY-----
  MIIEvQIBADANBgkqhkiG9w0BAQE...
  -----END PRIVATE KEY-----
  
  Public Key: (shared with clients/other servers)
  -----BEGIN PUBLIC KEY-----
  MIIBIjANBgkqhkiG9w0BAQE...
  -----END PUBLIC KEY-----

Signing Process:
  1. Message: eyJhbGci... (Header.Payload)
  2. SHA256(Message) = hash value (32 bytes)
  3. RSA encrypt hash with private key
  4. Result: 256 bytes signature
  5. Base64URL encode: signature_b64

Verification Process:
  1. Received signature_b64 → decode → 256 bytes
  2. RSA decrypt with public key → original hash
  3. Recalculate SHA256(Header.Payload)
  4. Compare hashes
  5. If match: Valid signature
```

---

## 5. DETAILED COMPARISON

| Feature | HMAC-SHA256 | RSA-SHA256 |
|---------|------------|-----------|
| **Key Type** | Shared Secret | Public/Private Pair |
| **Key Size** | Usually 32+ bytes | 2048/4096 bits |
| **Signing Speed** | Fast ⚡ | Slow 🐢 |
| **Verification Speed** | Fast ⚡ | Slow 🐢 |
| **Signature Size** | 32 bytes | 256+ bytes |
| **Forgeability** | Medium risk | No risk |
| **Scalability** | Limited | Excellent |
| **Best For** | Internal services | Public APIs |

### Security Comparison:

**HMAC Risks:**
```
- Anyone with secret can forge tokens
- Secret must be transmitted securely
- Can't prove you created it (repudiation)
```

**RSA Benefits:**
```
- Private key stays on server
- Non-repudiation: Only you can sign
- Public key can be shared freely
- Suitable for distributed systems
```

---

## 6. ATTACK SCENARIOS & PREVENTION

### Scenario 1: Token Tampering

**Attacker tries:**
```
Original Token:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJ1c2VyX2lkIjoidXNlcl8xMjMiLCJpc19hZG1pbiI6ZmFsc2V9.
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c

Modified Token:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.
eyJ1c2VyX2lkIjoidXNlcl8xMjMiLCJpc19hZG1pbiI6dHJ1ZX0.
SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
```

**Prevention:**
- HMAC: Recompute signature. Won't match.
- RSA: Decrypt signature with public key. Hash won't match.
- **Both will reject the token** ✓

### Scenario 2: Secret Key Compromise (HMAC)

**If secret is compromised:**
- Attacker can create ANY valid token
- Can impersonate ANY user
- Cannot revoke existing tokens

**Mitigation:**
- Rotate keys immediately
- Invalidate all existing tokens
- Use short expiration times

### Scenario 3: Private Key Compromise (RSA)

**If private key is compromised:**
- Attacker can create ANY valid token
- Similar risk to HMAC

**Mitigation:**
- Revoke certificate immediately
- Generate new key pair
- Replace public key everywhere
- Can revoke via certificate systems (CRL)

---

## 7. PRACTICAL WORKFLOW

### Creating a Token (HMAC Example):

```go
claims := Claims{
    UserID: "user_123",
    Username: "john_doe",
    ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
}

// Step 1: Create header
header := Header{Alg: "HS256", Typ: "JWT"}
headerJSON := json.Marshal(header)
headerB64 := encodeBase64(headerJSON)

// Step 2: Encode claims
claimsJSON := json.Marshal(claims)
claimsB64 := encodeBase64(claimsJSON)

// Step 3: Create signing input
signingInput := headerB64 + "." + claimsB64

// Step 4: Sign with HMAC
mac := hmac.New(sha256.New, secret)
mac.Write(signingInput)
signature := mac.Sum(nil)
signatureB64 := encodeBase64(signature)

// Step 5: Combine
token := signingInput + "." + signatureB64
```

### Verifying a Token:

```go
parts := strings.Split(token, ".")
headerB64, claimsB64, signatureB64 := parts[0], parts[1], parts[2]

// Step 1: Recalculate signature
signingInput := headerB64 + "." + claimsB64
mac := hmac.New(sha256.New, secret)
mac.Write(signingInput)
expectedSignature := mac.Sum(nil)
expectedSignatureB64 := encodeBase64(expectedSignature)

// Step 2: Compare signatures
if signatureB64 != expectedSignatureB64 {
    return nil, fmt.Errorf("invalid signature")
}

// Step 3: Decode and validate claims
claimsData := decodeBase64(claimsB64)
var claims Claims
json.Unmarshal(claimsData, &claims)

// Step 4: Check expiration
if time.Now().Unix() > claims.ExpiresAt {
    return nil, fmt.Errorf("token expired")
}

return &claims, nil
```

---

## 8. CHOOSING YOUR STRATEGY

### Use HMAC-SHA256 if:
✓ Internal microservices only
✓ Single authentication server
✓ All services share a secret
✓ Performance is critical
✓ Key rotation is manageable

### Use RSA-SHA256 if:
✓ Public APIs
✓ Multiple independent services
✓ Third-party verification needed
✓ Non-repudiation is required
✓ Distributed/federated systems
✓ OAuth2 / OpenID Connect

---

## 9. REAL-WORLD SCENARIOS

### Scenario A: Internal Microservices

```
API Gateway (HMAC)
    ↓ shared secret
├── User Service (verify with secret)
├── Order Service (verify with secret)
└── Payment Service (verify with secret)

Perfect for HMAC because all services are internal.
```

### Scenario B: Public REST API

```
Server (RSA)
    │
    ├── Generates token with private key
    ├── Sends public key to clients
    │
    ↓ Client makes request with token
    
Server verifies using public key (can share securely).
Clients cannot forge tokens (don't have private key).
```

### Scenario C: OAuth2 / OIDC

```
Authorization Server (RSA)
    │
    ├── Signs tokens with private key
    ├── Publishes public key at /.well-known/jwks.json
    │
Resource Servers:
    ├── Download public key
    ├── Verify tokens using public key
    └── No need for shared secret!
```

---

## 10. SECURITY BEST PRACTICES

1. **Always validate expiration**: Check `exp` claim
2. **Validate algorithm**: Ensure `alg` matches expected
3. **Use HTTPS**: Always transmit tokens over HTTPS
4. **Short expiration**: 15-60 minutes for access tokens
5. **Refresh tokens**: Use separate, longer-lived refresh tokens
6. **Secure storage**: 
   - Server: Environment variables for secrets
   - Browser: HTTP-only, Secure cookies
   - Mobile: Secure enclave/keychain
7. **Rotation**: Regularly rotate keys
8. **Monitoring**: Track token usage and suspicious patterns

---

## Summary

JWT consists of three parts signed in two different ways:

**HMAC (Symmetric):**
- Both parties share a secret
- Fast but less secure for distributed systems
- Good for internal services

**RSA (Asymmetric):**
- Public/private key pair
- Slower but more secure for public APIs
- Perfect for distributed systems

Both provide integrity and authenticity verification but with different trust models.
