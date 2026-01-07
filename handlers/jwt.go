package handlers

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("dsajdasdhasbdhasdyuashyudashdasnnduiashdyuasdhiasudhasoiu") // В продакшене используйте environment variables!

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
