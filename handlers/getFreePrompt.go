package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

// дать использование промпта
func GetFreePrompt(db *pgxpool.Pool, email string) (bool, error) {
	var used int
	FreeDailyLimit := viper.GetInt("server.free_daily_limit")

	err := db.QueryRow(context.Background(), `
		SELECT daily_used_prompts FROM users WHERE email=$1
	`, email).Scan(&used)

	if err != nil {
		return false, err
	}

	// Если еще есть промпты — бонус не нужен
	if used < FreeDailyLimit {
		return false, nil
	}

	cmdTag, err := db.Exec(context.Background(), `
		UPDATE users
		SET daily_used_prompts = GREATEST(daily_used_prompts - 1, 0)
		WHERE email = $1
	`, email)
	if err != nil {
		return false, err
	}

	return cmdTag.RowsAffected() > 0, nil
}

// выдать бонус
func GrantBonus(db *pgxpool.Pool, email, bonusType string, expiresIn time.Duration) error {
	_, err := db.Exec(context.Background(), `
        INSERT INTO user_bonuses (user_id, device_id, type, status, expires_at)
        SELECT id, device_id, $2, 'active', NOW() + $3 * INTERVAL '1 second'
        FROM users WHERE email = $1
    `, email, bonusType, int(expiresIn.Seconds()))
	return err
}

// отметить использование
func UseBonus(db *pgxpool.Pool, email string) (bool, error) {
	cmdTag, err := db.Exec(context.Background(), `
        UPDATE user_bonuses
        SET status = 'used', used_at = NOW()
        WHERE id = (
            SELECT b.id FROM user_bonuses b
            JOIN users u ON b.user_id = u.id
            WHERE u.email = $1 AND b.status = 'active'
              AND (b.expires_at IS NULL OR b.expires_at > NOW())
            ORDER BY b.issued_at ASC
            LIMIT 1
            FOR UPDATE SKIP LOCKED
        )
    `, email)
	return cmdTag.RowsAffected() > 0, err
}
