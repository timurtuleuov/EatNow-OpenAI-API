package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"openai/internal/logger"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

func CheckBalance(db *pgxpool.Pool, email string, cost int) error {
	var balance int
	var balanceResetAt time.Time
	var isPremium bool
	var premiumExpires *time.Time

	err := db.QueryRow(context.Background(), `
		SELECT balance, balance_reset_at, is_premium, premium_expires
		FROM users WHERE email = $1
	`, email).Scan(&balance, &balanceResetAt, &isPremium, &premiumExpires)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	resetDays := viper.GetInt("balance.reset_days")
	now := time.Now()

	if balanceResetAt.AddDate(0, 0, resetDays).Before(now) {
		freeMonthly := viper.GetInt("balance.free_monthly")
		premiumMonthly := viper.GetInt("balance.premium_monthly")

		if isPremium && premiumExpires != nil && premiumExpires.After(now) {
			balance = premiumMonthly
		} else {
			balance = freeMonthly
		}

		_, err = db.Exec(context.Background(), `
			UPDATE users
			SET balance = $1, balance_reset_at = NOW()
			WHERE email = $2
		`, balance, email)
		if err != nil {
			return fmt.Errorf("failed to reset balance: %w", err)
		}

		slog.Info("balance_reset",
			logger.KeyUser, email,
			"new_balance", balance,
			"is_premium", isPremium,
		)
	}

	if balance < cost {
		return fmt.Errorf("insufficient balance: need %d, have %d", cost, balance)
	}

	tag, err := db.Exec(context.Background(), `
		UPDATE users
		SET balance = balance - $1
		WHERE email = $2 AND balance >= $1
	`, cost, email)
	if err != nil {
		return fmt.Errorf("failed to deduct balance: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("insufficient balance: need %d", cost)
	}

	return nil
}
