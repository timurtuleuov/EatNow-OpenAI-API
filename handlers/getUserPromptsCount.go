package handlers

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

func GetUserFreePromptsCount(db *pgxpool.Pool, email string) (int, error) {
	var userFreePromptsCount int
	FreeDailyLimit := viper.GetInt("server.free_daily_limit")

	err := db.QueryRow(context.Background(), `
	SELECT daily_used_prompts FROM users WHERE email=$1
`, email).Scan(&userFreePromptsCount)

	if err != nil {
		return 0, err
	}

	return max(0, FreeDailyLimit-userFreePromptsCount), nil
}
