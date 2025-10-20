package handlers

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(db *pgxpool.Pool, username, email, password, platform, deviceID string) (bool, error) {
	ctx := context.Background()

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
