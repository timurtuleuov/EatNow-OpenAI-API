package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"openai/internal/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GrantBonus(db *pgxpool.Pool, email, bonusType string, amount int) error {
	tag, err := db.Exec(context.Background(), `
		UPDATE users
		SET balance = balance + $1
		WHERE email = $2
	`, amount, email)
	if err != nil {
		return fmt.Errorf("failed to grant bonus: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found: %s", email)
	}

	slog.Info("bonus_granted",
		logger.KeyUser, email,
		"type", bonusType,
		"amount", amount,
	)
	return nil
}

func AddBalance(db *pgxpool.Pool, email string, amount int) error {
	_, err := db.Exec(context.Background(), `
		UPDATE users SET balance = balance + $1 WHERE email = $2
	`, amount, email)
	return err
}
