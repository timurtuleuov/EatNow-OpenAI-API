package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

func GetUserFreePromptsCount(db *pgxpool.Pool, email string) (int, error) {
	var userFreePromptsCount int

	err := db.QueryRow(context.Background(), `
	SELECT daily_used_prompts FROM users WHERE email=$1
`, email).Scan(&userFreePromptsCount)

	if err != nil {
		return 0, err
	}

	return max(0, 5-userFreePromptsCount), nil
}
