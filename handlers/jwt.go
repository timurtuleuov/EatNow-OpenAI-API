package handlers

import (
	"crypto/rand"
	"encoding/hex"

	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/spf13/viper"
)

var jwtKey = []byte(viper.GetString("server.jwt_key")) // В продакшене используйте environment variables!

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateJWT(email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // Токен на 24 часа
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
