package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetUserFreePromptsCount(db *pgxpool.Pool, email string) (int, error) {
	var balance int
	err := db.QueryRow(context.Background(), `
		SELECT balance FROM users WHERE email = $1
	`, email).Scan(&balance)
	if err != nil {
		return 0, err
	}
	return balance, nil
}
