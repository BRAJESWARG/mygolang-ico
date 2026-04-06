// go get github.com/golang-jwt/jwt/v5

package main

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

var secretKey = []byte("my-secret-password")

// STEP 1: CREATE (sign) a token — like stamping a ticket
func createToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(1 * time.Hour).Unix(), // expires in 1 hour
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey) // stamp it with our secret
}

// STEP 2: VERIFY the token — check if ticket is genuine
func verifyToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil || !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	claims := token.Claims.(jwt.MapClaims)
	return claims["username"].(string), nil
}

func main() {
	token, _ := createToken("brajeswar")
	fmt.Println("Token:", token)

	user, _ := verifyToken(token)
	fmt.Println("Verified User:", user) // prints: brajeswar
}
