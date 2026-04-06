# JWT Project - Compilation Fixes

## Issues Found & Fixed

### ❌ Error 1: `sha256.New()` Type Mismatch in RSA Signing

**Original Code:**
```go
func (r *RSATokenizer) signRSA(message []byte) ([]byte, error) {
    hash := sha256.Sum256(message)
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, sha256.New(), hash[:])
    return signature, err
}
```

**Error Message:**
```
./main.go:194:64: cannot use sha256.New() (value of interface type hash.Hash) 
as crypto.Hash value in argument to rsa.SignPKCS1v15
```

**Problem:**
- `sha256.New()` returns a `hash.Hash` interface (which is a hash instance)
- `rsa.SignPKCS1v15()` expects a `crypto.Hash` constant (which identifies the algorithm)
- These are two different types!

**Fixed Code:**
```go
func (r *RSATokenizer) signRSA(message []byte) ([]byte, error) {
    hash := sha256.Sum256(message)
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hash[:])
    return signature, err
}
```

**Explanation:**
- `crypto.SHA256` is a constant that identifies SHA-256 algorithm
- `hash[:]` passes the computed hash bytes
- This matches the function signature: `SignPKCS1v15(rand, key, hashType, hashedBytes)`

---

### ❌ Error 2: `sha256.New()` Type Mismatch in RSA Verification

**Original Code:**
```go
err := rsa.VerifyPKCS1v15(r.publicKey, sha256.New(), hash[:], signature)
```

**Error Message:**
```
./main.go:216:41: cannot use sha256.New() (value of interface type hash.Hash) 
as crypto.Hash value in argument to rsa.VerifyPKCS1v15
```

**Fixed Code:**
```go
err := rsa.VerifyPKCS1v15(r.publicKey, crypto.SHA256, hash[:], signature)
```

**Same reasoning as Error 1** - use the `crypto.SHA256` constant instead of `sha256.New()`.

---

### ❌ Error 3: String Multiplication in Separators

**Original Code:**
```go
fmt.Println("=" * 60)
fmt.Println("JWT INTERNAL WORKING - TWO SIGNING STRATEGIES")
fmt.Println("=" * 60)
```

**Error Message:**
```
./main.go:239:14: invalid operation: "=" * 60 (mismatched types untyped string and untyped int)
```

**Problem:**
- Go doesn't support string multiplication syntax like Python
- `"=" * 60` tries to multiply a string by an integer, which isn't allowed

**Fixed Code:**
```go
fmt.Println(strings.Repeat("=", 60))
fmt.Println("JWT INTERNAL WORKING - TWO SIGNING STRATEGIES")
fmt.Println(strings.Repeat("=", 60))
```

**Explanation:**
- `strings.Repeat(s string, count int) string` repeats a string N times
- This is the Go way to create repeated strings

---

### ❌ Error 4: Missing Import

**Problem:**
- `crypto.SHA256` constant requires the `crypto` package import
- It wasn't included in the original imports

**Fixed:**
```go
import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto"              // ← Added this!
    "encoding/base64"
    "encoding/json"
    "fmt"
    "strings"
    "time"
)
```

---

## Summary of Changes

| Error | Line(s) | Type | Fix |
|-------|---------|------|-----|
| #1 | ~194 | Type mismatch | `sha256.New()` → `crypto.SHA256` |
| #2 | ~216 | Type mismatch | `sha256.New()` → `crypto.SHA256` |
| #3 | ~239-241 | Syntax error | `"=" * 60` → `strings.Repeat("=", 60)` |
| #4 | ~4-14 | Missing import | Added `"crypto"` import |

---

## Key Learnings

### 1. **crypto.Hash vs hash.Hash**

In Go, there's an important distinction:

```go
// crypto.Hash - A CONSTANT that identifies an algorithm
const (
    SHA256 crypto.Hash = 4  // Just an identifier
    SHA512 crypto.Hash = 6  // Just an identifier
)

// hash.Hash - An INTERFACE for hashing objects
type Hash interface {
    Write(p []byte) (n int, err error)
    Sum(b []byte) []byte
    Reset()
    Size() int
    BlockSize() int
}
```

**When to use what:**
- **`crypto.SHA256`** - Specify which hash algorithm to RSA signing functions
- **`sha256.New()`** - Create an instance to hash data incrementally

### 2. **String Repetition in Go**

Go doesn't have Python's string multiplication:

```go
// Python (works)
separator = "=" * 60

// Go (doesn't work)
separator := "=" * 60  // ❌ Error!

// Go (correct ways)
separator := strings.Repeat("=", 60)  // ✓ Best
separator := fmt.Sprintf("%*s", 60, "=")  // ✓ Alternative (pads with spaces)
```

### 3. **RSA Function Signatures**

```go
// RSA Signing
func SignPKCS1v15(
    rand io.Reader,           // Random source
    priv *PrivateKey,         // Private key
    hash crypto.Hash,         // Algorithm (constant!)
    hashed []byte,            // Pre-computed hash
) ([]byte, error)

// RSA Verification  
func VerifyPKCS1v15(
    pub *PublicKey,           // Public key
    hash crypto.Hash,         // Algorithm (constant!)
    hashed []byte,            // Pre-computed hash
    sig []byte,               // Signature to verify
) error
```

Notice:
- Both take `crypto.Hash` constant (not `hash.Hash` interface)
- Both take pre-computed `hashed []byte` (not the message)
- You compute the hash first, then pass it to RSA functions

---

## How to Run Now

```bash
# Navigate to project directory
cd jwt-demo

# Run the corrected program
go run main.go
```

## Expected Output

```
============================================================
JWT INTERNAL WORKING - TWO SIGNING STRATEGIES
============================================================

[STRATEGY 1: HMAC-SHA256 (Symmetric)]
------------------------------------------------------------
Token Created:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcl8xMjMi...

Token Structure:
  Header:    {"alg":"HS256","typ":"JWT"}
  Claims:    {"user_id":"user_123","username":"john_doe",...}
  Signature: SflKxwRJSMeKKF2QT4fwpMeJ...

✓ Verification Successful!
  User: john_doe (john@example.com)

Testing Tamper Detection (HMAC):
✓ Tampered token rejected: signature verification failed

[STRATEGY 2: RSA-SHA256 (Asymmetric)]
------------------------------------------------------------
Token Created:
eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoidXNlcl8xMjMi...

Token Structure:
  Header:    {"alg":"RS256","typ":"JWT"}
  Claims:    {"user_id":"user_123","username":"john_doe",...}
  Signature: JVFt8p0WkL9zQ2vM4xN5pQ...

✓ Verification Successful!
  User: john_doe (john@example.com)

Testing Tamper Detection (RSA):
✓ Tampered token rejected: signature verification failed

[COMPARISON: HMAC vs RSA]
------------------------------------------------------------
HMAC (Symmetric):
  ✓ Faster (simpler algorithm)
  ✓ Smaller signatures
  ✗ Secret key must be shared
  ✗ Anyone with the key can create tokens
  Use case: Internal microservices, single server

RSA (Asymmetric):
  ✓ Public key can be shared freely
  ✓ Only server with private key can create tokens
  ✓ Easy for distributed systems
  ✗ Slower (complex algorithm)
  ✗ Larger signatures
  Use case: Public APIs, multiple servers, OAuth
```

---

## Additional Resources

- **Go crypto/rsa docs**: https://pkg.go.dev/crypto/rsa
- **Go crypto package**: https://pkg.go.dev/crypto
- **JWT Standard**: https://tools.ietf.org/html/rfc7519

All files are now corrected and ready to run! 🎉
