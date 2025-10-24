package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const FreeDailyLimit = 5

func CanUsePrompt(db *pgxpool.Pool, deviceID string) (bool, error) {
	var isPremium bool
	var premiumExpires, lastPromptDate *time.Time
	var used int

	err := db.QueryRow(context.Background(), `
		SELECT is_premium, premium_expires, daily_used_prompts, last_prompt_date
		FROM users
		WHERE device_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, deviceID).Scan(&isPremium, &premiumExpires, &used, &lastPromptDate)
	if err != nil {
		// нет пользователя с таким device_id — значит новый девайс
		// можно разрешить 1-й промпт
		return true, nil
	}

	now := time.Now()

	// Премиум — без лимита
	if isPremium && premiumExpires != nil && premiumExpires.UTC().After(now.UTC()) {
		return true, nil
	}

	// Новая дата → сброс счётчика
	if lastPromptDate == nil || lastPromptDate.Format("2006-01-02") != now.Format("2006-01-02") {
		_, err = db.Exec(context.Background(), `
			UPDATE users
			SET daily_used_prompts = 1, last_prompt_date = $2
			WHERE device_id = $1
		`, deviceID, now)
		return true, err
	}

	// Проверка лимита
	if used >= FreeDailyLimit {
		return false, nil
	}

	// Увеличиваем счётчик
	_, err = db.Exec(context.Background(), `
		UPDATE users
		SET daily_used_prompts = daily_used_prompts + 1
		WHERE device_id = $1
	`, deviceID)
	return err == nil, err
}
