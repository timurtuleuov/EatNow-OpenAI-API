package handlers

import (
	"database/sql"
	"time"
)

const FreeDailyLimit = 5

func CanUsePrompt(db *sql.DB, userID string) (bool, error) {

	var isPremium bool
	var premiumExpires, lastPromptDate sql.NullTime
	var used int

	err := db.QueryRow(`
		SELECT is_premium, premium_expires, daily_used_prompts, last_prompt_date
		FROM users WHERE id = $1
	`, userID).Scan(&isPremium, &premiumExpires, &used, &lastPromptDate)
	if err != nil {
		return false, err
	}

	now := time.Now()

	// Проверяем истёк ли премиум
	if isPremium && premiumExpires.Valid && premiumExpires.Time.After(now) {
		return true, nil // премиум — без ограничений
	}

	// Если новая дата — сбрасываем счётчик
	if !lastPromptDate.Valid || lastPromptDate.Time.Format("2006-01-02") != now.Format("2006-01-02") {
		_, err = db.Exec(`UPDATE users SET daily_used_prompts = 1, last_prompt_date = $2 WHERE id = $1`, userID, now)
		return true, err
	}

	// Проверяем лимит
	if used >= FreeDailyLimit {
		return false, nil
	}

	// Увеличиваем счётчик
	_, err = db.Exec(`UPDATE users SET daily_used_prompts = daily_used_prompts + 1 WHERE id = $1`, userID)
	if err != nil {
		return false, err
	}

	return true, nil
}
