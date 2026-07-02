package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var secretKey []byte

func init() {
	key := os.Getenv("JWT_SECRET")
	if key == "" {
		key = "tian-niu-dev-secret-change-in-production"
		fmt.Fprintln(os.Stderr, "[WARNING] JWT_SECRET not set, using insecure default.")
	}
	secretKey = []byte(key)
}

// GenerateToken generates a JWT token
func GenerateToken(userID, username string) (string, error) {
	if len(secretKey) < 16 {
		return "", fmt.Errorf("JWT secret key too short (must be >= 16 bytes)")
	}
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // 24 hours expiry
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secretKey)
}

// ParseToken parses a JWT token
func ParseToken(tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return secretKey, nil
	}, jwt.WithValidMethods([]string{"HS256"}))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, err
}
