# Side-by-Side Code Comparison: Wrong vs Fixed

## Issue #1: RSA Signing Function

### ❌ WRONG CODE
```go
// signRSA - Internal method to create RSA signature
func (r *RSATokenizer) signRSA(message []byte) ([]byte, error) {
    hash := sha256.Sum256(message)
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, sha256.New(), hash[:])
    //                                                                 ^^^^^^^^^^^^^^
    //                                            This is wrong! Returns hash.Hash interface
    return signature, err
}
```

**Error:**
```
./main.go:194:64: cannot use sha256.New() (value of interface type hash.Hash) 
as crypto.Hash value in argument to rsa.SignPKCS1v15
```

### ✅ CORRECT CODE
```go
// signRSA - Internal method to create RSA signature
func (r *RSATokenizer) signRSA(message []byte) ([]byte, error) {
    hash := sha256.Sum256(message)
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hash[:])
    //                                                             ^^^^^^^^^^^^^^
    //                                         This is correct! A crypto.Hash constant
    return signature, err
}
```

**Why?**
- `sha256.New()` returns a `hash.Hash` interface (object that can hash data)
- `crypto.SHA256` is a `crypto.Hash` constant (identifier for the algorithm)
- The RSA function signature expects:
  ```go
  func SignPKCS1v15(rand io.Reader, priv *PrivateKey, hash crypto.Hash, hashed []byte) ([]byte, error)
  ```
  The 3rd parameter must be `crypto.Hash` constant, not a `hash.Hash` object!

---

## Issue #2: RSA Verification Function

### ❌ WRONG CODE
```go
// VerifyToken - Verify and parse RSA-signed token
func (r *RSATokenizer) VerifyToken(token string) (*Claims, error) {
    // ... token splitting code ...
    
    // Verify signature
    signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
    hash := sha256.Sum256([]byte(signingInput))
    
    err := rsa.VerifyPKCS1v15(r.publicKey, sha256.New(), hash[:], signature)
    //                                      ^^^^^^^^^^^^^^
    //                                 This is wrong!
    if err != nil {
        return nil, fmt.Errorf("signature verification failed: %v", err)
    }
    
    // ... rest of function ...
}
```

**Error:**
```
./main.go:216:41: cannot use sha256.New() (value of interface type hash.Hash) 
as crypto.Hash value in argument to rsa.VerifyPKCS1v15
```

### ✅ CORRECT CODE
```go
// VerifyToken - Verify and parse RSA-signed token
func (r *RSATokenizer) VerifyToken(token string) (*Claims, error) {
    // ... token splitting code ...
    
    // Verify signature
    signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
    hash := sha256.Sum256([]byte(signingInput))
    
    err := rsa.VerifyPKCS1v15(r.publicKey, crypto.SHA256, hash[:], signature)
    //                                      ^^^^^^^^^^^^^^
    //                                 This is correct!
    if err != nil {
        return nil, fmt.Errorf("signature verification failed: %v", err)
    }
    
    // ... rest of function ...
}
```

---

## Issue #3: String Separator Format

### ❌ WRONG CODE
```go
func main() {
    fmt.Println("=" * 60)
    //          ^^^^^^^^
    //    This syntax doesn't work in Go!
    
    fmt.Println("JWT INTERNAL WORKING - TWO SIGNING STRATEGIES")
    fmt.Println("=" * 60)
    
    // ... rest of main ...
}
```

**Error:**
```
./main.go:239:14: invalid operation: "=" * 60 (mismatched types untyped string and untyped int)
./main.go:241:14: invalid operation: "=" * 60 (mismatched types untyped string and untyped int)
```

**Why?**
- This syntax works in Python: `print("=" * 60)` creates `"============================================..."`
- In Go, you can't multiply strings by integers
- The `*` operator only works with numbers in Go

### ✅ CORRECT CODE
```go
func main() {
    fmt.Println(strings.Repeat("=", 60))
    //          ^^^^^^^^^^^^^^^^^^^^^^^
    //    Use strings.Repeat() instead!
    
    fmt.Println("JWT INTERNAL WORKING - TWO SIGNING STRATEGIES")
    fmt.Println(strings.Repeat("=", 60))
    
    // ... rest of main ...
}
```

**Alternative approaches:**
```go
// Method 1: strings.Repeat (best for clarity)
separator := strings.Repeat("=", 60)

// Method 2: fmt.Sprintf with padding
separator := fmt.Sprintf("%60s", "")  // Creates 60 spaces

// Method 3: Manual loop
var separator string
for i := 0; i < 60; i++ {
    separator += "="
}

// Method 4: Using bytes.Repeat
separator := string(bytes.Repeat([]byte("="), 60))
```

We use `strings.Repeat()` because it's the most idiomatic Go way.

---

## Issue #4: Missing Import

### ❌ WRONG CODE
```go
import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    // Missing: "crypto"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

func main() {
    // ...
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hash[:])
    //                                                                           ^^^^
    //                                            This won't compile without "crypto" import!
}
```

**Error (implicit):**
- If you tried to compile with `crypto.SHA256` without importing `"crypto"`:
```
undefined: crypto
```

### ✅ CORRECT CODE
```go
import (
    "crypto/hmac"
    "crypto/rand"
    "crypto/rsa"
    "crypto/sha256"
    "crypto"           // ← Added this!
    "encoding/base64"
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

func main() {
    // ...
    signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hash[:])
    //                                                            ^^^^^^^^^^^^^^
    //                                        Now this works because "crypto" is imported!
}
```

**Key Point:**
- The `"crypto/rsa"` package contains RSA functions
- The `"crypto"` package contains the `crypto.Hash` constants
- Even though `crypto/rsa` is imported, you still need to explicitly import `"crypto"` to use `crypto.SHA256`
- This is because of Go's import system - sub-packages don't automatically export parent package symbols

---

## Summary of All Changes

### Changed Lines

| Line | Original | Fixed | Reason |
|------|----------|-------|--------|
| ~1 (import) | *(missing)* | `"crypto"` | Need for `crypto.SHA256` |
| ~194 | `sha256.New()` | `crypto.SHA256` | Wrong type passed to RSA function |
| ~216 | `sha256.New()` | `crypto.SHA256` | Wrong type passed to RSA function |
| ~240 | `"=" * 60` | `strings.Repeat("=", 60)` | Go syntax doesn't support string multiplication |
| ~242 | `"=" * 60` | `strings.Repeat("=", 60)` | Go syntax doesn't support string multiplication |

---

## Type System Explanation

### Go's Type System: The Key Insight

Go has **strong typing** which is why these errors occurred. Let's understand:

```go
// These are DIFFERENT types even though they relate to SHA256!

// 1. crypto.Hash - A constant (identifier)
const SHA256 crypto.Hash = 4  // Just a number, not a function or object

// 2. hash.Hash - An interface (an object contract)
type Hash interface {
    Write(p []byte) (n int, err error)
    Sum(b []byte) []byte
    Reset()
    Size() int
    BlockSize() int
}

// 3. sha256.New() - A function that returns hash.Hash
func New() hash.Hash {
    return &digest{h: h0, x: make([]byte, 0, chunk), len: 0}
}

// So:
sha256.New()    // Returns: hash.Hash (an interface)
crypto.SHA256   // Returns: crypto.Hash (a constant/enum value)

// They're not compatible even though they both relate to SHA256!
```

**Why the distinction?**
- `crypto.Hash` constants are used to **identify** algorithms for RSA, ECDSA, etc.
- `hash.Hash` interfaces are used to **compute** hashes incrementally
- RSA functions need to know which hash algorithm to use (thus `crypto.Hash`)
- But they don't need a hash object (that's why not `hash.Hash`)

**Analogy:**
- `crypto.Hash` = "What type of hash?" (concept)
- `hash.Hash` = "A hash object that can compute" (implementation)

---

## Compilation Process

### Before Fixes
```
$ go run main.go

# Compiler errors:
error 1: Line 194 - wrong type for SignPKCS1v15
error 2: Line 216 - wrong type for VerifyPKCS1v15  
error 3: Line 239 - invalid operation (string * int)
error 4: Line 241 - invalid operation (string * int)

Compilation FAILED ❌
```

### After Fixes
```
$ go run main.go

============================================================
JWT INTERNAL WORKING - TWO SIGNING STRATEGIES
============================================================

[STRATEGY 1: HMAC-SHA256 (Symmetric)]
------------------------------------------------------------
Token Created:
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...

...

Compilation SUCCESSFUL ✅
Program runs and shows output ✅
```

---

## Key Learning Points

1. **Go's Strong Typing** - Types matter! `hash.Hash` ≠ `crypto.Hash`
2. **Packages vs Sub-packages** - Import what you use, not just parent packages
3. **Go Idioms** - Use `strings.Repeat()` for string repetition, not multiplication
4. **RSA Function Signatures** - They take `crypto.Hash` constants, not `hash.Hash` objects
5. **Pre-computed Hashes** - RSA functions expect the hash as bytes, not a hasher object

---

## Now You're Ready!

With these fixes, the code compiles and runs perfectly. You can:

✅ See HMAC token creation and verification  
✅ See RSA token creation and verification  
✅ Compare both strategies side-by-side  
✅ Understand JWT internals deeply  
✅ Learn about cryptographic signing  

```bash
cd jwt-demo
go run main.go
```

Happy learning! 🎉
