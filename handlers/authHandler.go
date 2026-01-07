package handlers

import (
	"context"
	"fmt"

	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(db *pgxpool.Pool, username, email, password, platform, deviceID string) (bool, error) {
	ctx := context.Background()
	var count int
	err := db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE email=$1;`,
		email,
	).Scan(&count)

	if err != nil {
		return false, fmt.Errorf("failed to check existing user: %w", err)
	}

	if count > 0 {
		return false, fmt.Errorf("user with this email or device already exists")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return false, fmt.Errorf("failed to hash password: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO users (name, password, email, platform, device_id)
		VALUES ($1, $2, $3, $4, $5)
	`, username, string(hashedPassword), email, platform, deviceID)
	if err != nil {
		return false, fmt.Errorf("failed to insert user: %w", err)
	}

	return true, nil
}

func CheckUserExistsAndAuth(db *pgxpool.Pool, email, deviceID, password string) (bool, error) {
	ctx := context.Background()

	var storedHash string
	var exists bool

	// Ищем пользователя по email или device_id
	err := db.QueryRow(ctx, `
		SELECT password FROM users WHERE email = $1;
	`, email).Scan(&storedHash)
	if err != nil {
		// Нет пользователя
		return false, nil
	}

	// Проверяем пароль через bcrypt
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		// Пароль не совпал
		return false, fmt.Errorf("invalid password")
	}

	exists = true
	return exists, nil
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")

		// Ожидаем формат "Bearer <token>"
		if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
			tokenString = tokenString[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			c.Abort()
			return
		}

		// Передаем email в контекст, чтобы использовать в других обработчиках
		c.Set("email", claims.Email)
		c.Next()
	}
}
