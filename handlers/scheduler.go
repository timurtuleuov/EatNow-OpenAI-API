package handlers

import (
	"context"
	"log/slog"
	"openai/internal/logger"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/viper"
)

func StartBalanceScheduler(db *pgxpool.Pool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	slog.Info("balance_scheduler_started",
		"interval", interval.String(),
	)

	for range ticker.C {
		resetExpiredBalances(db)
	}
}

func resetExpiredBalances(db *pgxpool.Pool) {
	resetDays := viper.GetInt("balance.reset_days")
	freeMonthly := viper.GetInt("balance.free_monthly")
	premiumMonthly := viper.GetInt("balance.premium_monthly")

	tag, err := db.Exec(context.Background(), `
		UPDATE users
		SET balance = CASE
				WHEN is_premium AND (premium_expires IS NOT NULL AND premium_expires > NOW()) THEN $1
				ELSE $2
			END,
			balance_reset_at = NOW()
		WHERE balance_reset_at + ($3 || ' days')::INTERVAL < NOW()
	`, premiumMonthly, freeMonthly, resetDays)

	if err != nil {
		slog.Error("balance_scheduler_reset_failed",
			logger.KeyError, err,
		)
		return
	}

	if tag.RowsAffected() > 0 {
		slog.Info("balance_scheduler_reset_completed",
			"users_affected", tag.RowsAffected(),
		)
	}
}
