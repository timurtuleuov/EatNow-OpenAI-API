package handlers

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// дать использование промпта
func GetFreePrompt(db *pgxpool.Pool, deviceID string) (bool, error) {
	var used int
	err := db.QueryRow(context.Background(), `
		SELECT daily_used_prompts FROM users WHERE device_id=$1
	`, deviceID).Scan(&used)

	if err != nil {
		return false, err
	}

	// Если еще есть промпты — бонус не нужен
	if used < 5 {
		return false, nil
	}

	cmdTag, err := db.Exec(context.Background(), `
		UPDATE users
		SET daily_used_prompts = GREATEST(daily_used_prompts - 1, 0)
		WHERE device_id = $1
	`, deviceID)
	if err != nil {
		return false, err
	}

	return cmdTag.RowsAffected() > 0, nil
}

// выдать бонус
func GrantBonus(db *pgxpool.Pool, deviceID, bonusType string, expiresIn time.Duration) error {
	_, err := db.Exec(context.Background(), `
		INSERT INTO user_bonuses (device_id, type, status, expires_at)
		VALUES ($1, $2, 'active', NOW() + $3 * INTERVAL '1 second')
	`, deviceID, bonusType, int(expiresIn.Seconds()))
	return err
}

// отметить использование
func UseBonus(db *pgxpool.Pool, deviceID string) (bool, error) {
	cmdTag, err := db.Exec(context.Background(), `
        UPDATE user_bonuses
        SET status = 'used', used_at = NOW()
        WHERE id = (
            SELECT id FROM user_bonuses
            WHERE device_id = $1 AND status = 'active'
              AND (expires_at IS NULL OR expires_at > NOW())
            ORDER BY issued_at ASC
            LIMIT 1
			FOR UPDATE SKIP LOCKED
        )
    `, deviceID)

	// true если бонус реально был найден и использован
	return cmdTag.RowsAffected() > 0, err
}
