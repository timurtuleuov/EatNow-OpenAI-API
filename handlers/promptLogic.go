package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

var FreeDailyLimit = viper.GetInt("server.free_daily_limit")

func CanUsePrompt(db *pgxpool.Pool, email string) (bool, error) {
	var isPremium bool
	var premiumExpires, lastPromptDate *time.Time
	var used int

	err := db.QueryRow(context.Background(), `
		SELECT is_premium, premium_expires, daily_used_prompts, last_prompt_date
		FROM users
		WHERE email = $1
		LIMIT 1
	`, email).Scan(&isPremium, &premiumExpires, &used, &lastPromptDate)

	// Новый пользователь
	if err != nil {
		_, err = db.Exec(context.Background(), `
			INSERT INTO users (email, daily_used_prompts, last_prompt_date)
			VALUES ($1, 1, NOW())
		`, email)
		return true, err
	}

	now := time.Now()

	// 🔓 Премиум — без лимита
	if isPremium && premiumExpires != nil && premiumExpires.After(now) {
		return true, nil
	}

	// 🔁 Новый день → сбрасываем счётчик
	if lastPromptDate == nil || lastPromptDate.Format("2006-01-02") != now.Format("2006-01-02") {
		_, err = db.Exec(context.Background(), `
			UPDATE users
			SET daily_used_prompts = 0, last_prompt_date = $2
			WHERE email = $1
		`, email, now)
		return true, err
	}

	// 🚫 Превышен лимит — проверяем бонус
	if used >= FreeDailyLimit {
		var bonusID string
		err := db.QueryRow(context.Background(), `
			SELECT id FROM user_bonuses
			WHERE email = $1
			  AND status = 'active'
			  AND (expires_at IS NULL OR expires_at > NOW())
			ORDER BY issued_at ASC
			LIMIT 1
		`, email).Scan(&bonusID)

		if err == nil && bonusID != "" {
			// 🪙 Используем бонус
			_, err := db.Exec(context.Background(), `
				UPDATE user_bonuses
				SET status = 'used', used_at = NOW()
				WHERE id = $1
			`, bonusID)
			if err != nil {
				return false, err
			}

			// 💥 Не трогаем счётчик, просто разрешаем промпт
			return true, nil
		}

		// ❌ Нет бонуса — лимит исчерпан
		return false, nil
	}

	// ✅ Всё ок, инкрементируем счётчик
	_, err = db.Exec(context.Background(), `
		UPDATE users
		SET daily_used_prompts = daily_used_prompts + 1,
		    last_prompt_date = NOW()
		WHERE email = $1
	`, email)
	return err == nil, err
}
