package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

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
	FreeDailyLimit := viper.GetInt("server.free_daily_limit")
	// 🚫 Превышен лимит — проверяем бонус
	if used >= FreeDailyLimit {
		var bonusID int // У тебя в схеме SERIAL, это int
		err := db.QueryRow(context.Background(), `
			SELECT b.id FROM user_bonuses b
			JOIN users u ON b.user_id = u.id
			WHERE u.email = $1 
			AND b.status = 'active'
			AND (b.expires_at IS NULL OR b.expires_at > NOW())
			ORDER BY b.issued_at ASC
			LIMIT 1
		`, email).Scan(&bonusID)

		if err == nil {
			// 🪙 Используем бонус
			_, err := db.Exec(context.Background(), `
				UPDATE user_bonuses SET status = 'used', used_at = NOW() WHERE id = $1
			`, bonusID)
			if err != nil {
				return false, err
			}
			return true, nil
		}

		// Если ошибка не "строка не найдена", значит это реальная проблема с БД
		if err != nil && err.Error() != "no rows in result set" {
			return false, err
		}

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
