package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ============================================
// PART 1: JWT STRUCTURE & COMPONENTS
// ============================================

// Header contains token metadata
type Header struct {
	Alg string `json:"alg"` // Algorithm
	Typ string `json:"typ"` // Type
}

// Claims contains the token payload
type Claims struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	ExpiresAt int64  `json:"exp"` // Expiration time
	IssuedAt  int64  `json:"iat"` // Issued at
}

// ============================================
// STRATEGY 1: HMAC-SHA256 (Symmetric)
// ============================================

type HMACTokenizer struct {
	secret []byte
}

// NewHMACTokenizer creates an HMAC tokenizer with a shared secret
func NewHMACTokenizer(secret string) *HMACTokenizer {
	return &HMACTokenizer{
		secret: []byte(secret),
	}
}

// EncodeBase64 - URL-safe base64 encoding without padding
func encodeBase64(data []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(data), "=")
}

// DecodeBase64 - URL-safe base64 decoding with padding restoration
func decodeBase64(str string) ([]byte, error) {
	// Add padding if needed
	padding := 4 - (len(str) % 4)
	if padding != 4 {
		str += strings.Repeat("=", padding)
	}
	return base64.URLEncoding.DecodeString(str)
}

// CreateToken - Create a signed JWT token using HMAC
func (h *HMACTokenizer) CreateToken(claims Claims) (string, error) {
	// STEP 1: Create Header
	header := Header{
		Alg: "HS256",
		Typ: "JWT",
	}

	// STEP 2: Encode Header to JSON and then Base64
	headerJSON, _ := json.Marshal(header)
	headerB64 := encodeBase64(headerJSON)

	// STEP 3: Encode Claims to JSON and then Base64
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := encodeBase64(claimsJSON)

	// STEP 4: Create the signing input (header.payload)
	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)

	// STEP 5: Sign the input using HMAC-SHA256
	signature := h.signHMAC([]byte(signingInput))
	signatureB64 := encodeBase64(signature)

	// STEP 6: Combine all three parts
	token := fmt.Sprintf("%s.%s", signingInput, signatureB64)

	return token, nil
}

// signHMAC - Internal method to create HMAC signature
func (h *HMACTokenizer) signHMAC(message []byte) []byte {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(message)
	return mac.Sum(nil)
}

// VerifyToken - Verify and parse HMAC-signed token
func (h *HMACTokenizer) VerifyToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format: expected 3 parts, got %d", len(parts))
	}

	headerB64 := parts[0]
	claimsB64 := parts[1]
	signatureB64 := parts[2]

	// Verify signature
	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
	expectedSignature := h.signHMAC([]byte(signingInput))
	expectedSignatureB64 := encodeBase64(expectedSignature)

	if signatureB64 != expectedSignatureB64 {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Decode and parse claims
	claimsData, _ := decodeBase64(claimsB64)
	var claims Claims
	json.Unmarshal(claimsData, &claims)

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

// ============================================
// STRATEGY 2: RSA-SHA256 (Asymmetric)
// ============================================

type RSATokenizer struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

// NewRSATokenizer creates an RSA tokenizer with a key pair
func NewRSATokenizer(publicKey *rsa.PublicKey, privateKey *rsa.PrivateKey) *RSATokenizer {
	return &RSATokenizer{
		publicKey:  publicKey,
		privateKey: privateKey,
	}
}

// GenerateRSAKeyPair generates a new RSA key pair
func GenerateRSAKeyPair() (*rsa.PublicKey, *rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	return &privateKey.PublicKey, privateKey, nil
}

// CreateToken - Create a signed JWT token using RSA
func (r *RSATokenizer) CreateToken(claims Claims) (string, error) {
	// STEP 1: Create Header
	header := Header{
		Alg: "RS256",
		Typ: "JWT",
	}

	// STEP 2: Encode Header to JSON and then Base64
	headerJSON, _ := json.Marshal(header)
	headerB64 := encodeBase64(headerJSON)

	// STEP 3: Encode Claims to JSON and then Base64
	claimsJSON, _ := json.Marshal(claims)
	claimsB64 := encodeBase64(claimsJSON)

	// STEP 4: Create the signing input (header.payload)
	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)

	// STEP 5: Sign using RSA-SHA256
	signature, err := r.signRSA([]byte(signingInput))
	if err != nil {
		return "", err
	}
	signatureB64 := encodeBase64(signature)

	// STEP 6: Combine all three parts
	token := fmt.Sprintf("%s.%s", signingInput, signatureB64)

	return token, nil
}

// signRSA - Internal method to create RSA signature
func (r *RSATokenizer) signRSA(message []byte) ([]byte, error) {
	hash := sha256.Sum256(message)
	signature, err := rsa.SignPKCS1v15(rand.Reader, r.privateKey, crypto.SHA256, hash[:])
	return signature, err
}

// VerifyToken - Verify and parse RSA-signed token
func (r *RSATokenizer) VerifyToken(token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format: expected 3 parts, got %d", len(parts))
	}

	headerB64 := parts[0]
	claimsB64 := parts[1]
	signatureB64 := parts[2]

	// Decode signature
	signature, _ := decodeBase64(signatureB64)

	// Verify signature
	signingInput := fmt.Sprintf("%s.%s", headerB64, claimsB64)
	hash := sha256.Sum256([]byte(signingInput))

	err := rsa.VerifyPKCS1v15(r.publicKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		return nil, fmt.Errorf("signature verification failed: %v", err)
	}

	// Decode and parse claims
	claimsData, _ := decodeBase64(claimsB64)
	var claims Claims
	json.Unmarshal(claimsData, &claims)

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token has expired")
	}

	return &claims, nil
}

// ============================================
// DEMONSTRATION & TESTING
// ============================================

func main() {
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("JWT INTERNAL WORKING - TWO SIGNING STRATEGIES")
	fmt.Println(strings.Repeat("=", 60))

	// Create sample claims
	claims := Claims{
		UserID:    "user_123",
		Username:  "john_doe",
		Email:     "john@example.com",
		ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	// ============================================
	// STRATEGY 1: HMAC-SHA256 DEMO
	// ============================================
	fmt.Println("\n[STRATEGY 1: HMAC-SHA256 (Symmetric)]")
	fmt.Println("-" + strings.Repeat("-", 58))

	hmacSecret := "my-super-secret-key-12345"
	hmacTokenizer := NewHMACTokenizer(hmacSecret)

	// Create token
	hmacToken, _ := hmacTokenizer.CreateToken(claims)
	fmt.Printf("Token Created:\n%s\n\n", hmacToken)

	// Analyze token structure
	parts := strings.Split(hmacToken, ".")
	headerData, _ := decodeBase64(parts[0])
	claimsData, _ := decodeBase64(parts[1])

	fmt.Println("Token Structure:")
	fmt.Printf("  Header:    %s\n", string(headerData))
	fmt.Printf("  Claims:    %s\n", string(claimsData))
	fmt.Printf("  Signature: %s...\n\n", parts[2][:30])

	// Verify token
	verifiedClaims, err := hmacTokenizer.VerifyToken(hmacToken)
	if err != nil {
		fmt.Printf("❌ Verification Failed: %v\n", err)
	} else {
		fmt.Printf("✓ Verification Successful!\n")
		fmt.Printf("  User: %s (%s)\n\n", verifiedClaims.Username, verifiedClaims.Email)
	}

	// Test tampering detection
	fmt.Println("Testing Tamper Detection (HMAC):")
	tamperedToken := hmacToken[:len(hmacToken)-10] + "corrupted!"
	_, err = hmacTokenizer.VerifyToken(tamperedToken)
	if err != nil {
		fmt.Printf("✓ Tampered token rejected: %v\n\n", err)
	}

	// ============================================
	// STRATEGY 2: RSA-SHA256 DEMO
	// ============================================
	fmt.Println("[STRATEGY 2: RSA-SHA256 (Asymmetric)]")
	fmt.Println("-" + strings.Repeat("-", 58))

	// Generate RSA key pair
	publicKey, privateKey, _ := GenerateRSAKeyPair()
	rsaTokenizer := NewRSATokenizer(publicKey, privateKey)

	// Create token
	rsaToken, _ := rsaTokenizer.CreateToken(claims)
	fmt.Printf("Token Created:\n%s\n\n", rsaToken)

	// Analyze token structure
	parts = strings.Split(rsaToken, ".")
	headerData, _ = decodeBase64(parts[0])
	claimsData, _ = decodeBase64(parts[1])

	fmt.Println("Token Structure:")
	fmt.Printf("  Header:    %s\n", string(headerData))
	fmt.Printf("  Claims:    %s\n", string(claimsData))
	fmt.Printf("  Signature: %s...\n\n", parts[2][:30])

	// Verify token
	verifiedClaims, err = rsaTokenizer.VerifyToken(rsaToken)
	if err != nil {
		fmt.Printf("❌ Verification Failed: %v\n", err)
	} else {
		fmt.Printf("✓ Verification Successful!\n")
		fmt.Printf("  User: %s (%s)\n\n", verifiedClaims.Username, verifiedClaims.Email)
	}

	// Test tampering detection
	fmt.Println("Testing Tamper Detection (RSA):")
	tamperedToken = rsaToken[:len(rsaToken)-10] + "corrupted!"
	_, err = rsaTokenizer.VerifyToken(tamperedToken)
	if err != nil {
		fmt.Printf("✓ Tampered token rejected: %v\n\n", err)
	}

	// ============================================
	// COMPARISON
	// ============================================
	fmt.Println("[COMPARISON: HMAC vs RSA]")
	fmt.Println("-" + strings.Repeat("-", 58))
	fmt.Println(`
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
	`)
}
