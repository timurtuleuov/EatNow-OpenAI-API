package handlers

import (
	"context"
	"fmt"

	"time"

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

func CheckUserExistsAndAuth(db *pgxpool.Pool, email, password string) (bool, error) {
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

func SaveRefreshToken(db *pgxpool.Pool, email, token string) error {
	ctx := context.Background()

	expiresAt := time.Now().Add(6 * 30 * 24 * time.Hour)

	hashedToken, err := bcrypt.GenerateFromPassword([]byte(token), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO refresh_tokens (user_email, token, expires_at, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (user_email) 
		DO UPDATE SET 
			token = EXCLUDED.token,
			expires_at = EXCLUDED.expires_at,
			created_at = NOW();
	`

	_, err = db.Exec(ctx, query, email, string(hashedToken), expiresAt)
	return err
}

func VerifyRefreshToken(db *pgxpool.Pool, email, rawToken string) error {
	ctx := context.Background()

	var hashedToken string
	var expiresAt time.Time

	// Берем только одну, самую свежую запись по дате создания
	err := db.QueryRow(ctx, `
        SELECT token, expires_at 
        FROM refresh_tokens 
        WHERE user_email = $1 
        ORDER BY created_at DESC 
        LIMIT 1
    `, email).Scan(&hashedToken, &expiresAt)

	if err != nil {
		// Если записей нет, Scan вернет ошибку
		return fmt.Errorf("no session found for this user")
	}

	// 1. Проверяем срок годности
	if time.Now().After(expiresAt) {
		return fmt.Errorf("refresh token expired")
	}

	// 2. Сверяем хэш bcrypt с тем, что прислал клиент
	err = bcrypt.CompareHashAndPassword([]byte(hashedToken), []byte(rawToken))
	if err != nil {
		return fmt.Errorf("invalid refresh token")
	}

	return nil
}

func GetMe(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email") // Достаем из JWT middleware

		var isPremium bool
		var expiresAt *time.Time

		err := db.QueryRow(context.Background(),
			"SELECT is_premium, premium_expires FROM users WHERE email = $1",
			email).Scan(&isPremium, &expiresAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// Проверка: если в БД true, но время вышло — отдаем false
		if isPremium && expiresAt != nil && expiresAt.Before(time.Now()) {
			isPremium = false
			// Можно тут же фоном обновить БД, чтобы не считать лишний раз
		}

		c.JSON(http.StatusOK, gin.H{
			"email":      email,
			"is_premium": isPremium,
		})
	}
}

func UserIsPremium(db *pgxpool.Pool, email string) bool {

	var isPremium bool

	err := db.QueryRow(context.Background(),
		"SELECT is_premium FROM users WHERE email = $1",
		email).Scan(&isPremium)

	if err != nil {
		return false
	}
	return isPremium
}
