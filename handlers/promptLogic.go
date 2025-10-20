package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const FreeDailyLimit = 5

func CanUsePrompt(db *pgxpool.Pool, userID string) (bool, error) {
	var isPremium bool
	var premiumExpires, lastPromptDate *time.Time
	var used int

	err := db.QueryRow(context.Background(), `
		SELECT is_premium, premium_expires, daily_used_prompts, last_prompt_date
		FROM users WHERE id = $1
	`, userID).Scan(&isPremium, &premiumExpires, &used, &lastPromptDate)
	if err != nil {
		return false, err
	}

	now := time.Now()

	// Премиум — без лимита
	if isPremium && premiumExpires != nil && premiumExpires.After(now) {
		return true, nil
	}

	// Новая дата → сброс счётчика
	if lastPromptDate == nil || lastPromptDate.Format("2006-01-02") != now.Format("2006-01-02") {
		_, err = db.Exec(context.Background(), `
			UPDATE users SET daily_used_prompts = 1, last_prompt_date = $2 WHERE id = $1
		`, userID, now)
		return true, err
	}

	// Проверка лимита
	if used >= FreeDailyLimit {
		return false, nil
	}

	// Увеличиваем счётчик
	_, err = db.Exec(context.Background(), `
		UPDATE users SET daily_used_prompts = daily_used_prompts + 1 WHERE id = $1
	`, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}
