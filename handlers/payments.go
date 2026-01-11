package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GoogleVerifyRequest struct {
	Email         string `json:"email"`
	ProductID     string `json:"product_id"`
	PurchaseToken string `json:"purchase_token"`
	TransactionID string `json:"transaction_id"`
	Platform      string `json:"platform"`
}

func VerifyGooglePurchase(db *pgxpool.Pool) gin.HandlerFunc {
	// Конфигурация тарифов (легко вынести в конфиг или БД)
	planDuration := map[string]struct {
		Months int
		Years  int
		Price  float64
	}{
		"premium_subscription_monthly": {Months: 1, Years: 0, Price: 2.99},
		"premium_subscription_yearly":  {Months: 0, Years: 1, Price: 29.99},
	}

	return func(c *gin.Context) {
		var req GoogleVerifyRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		// 1. Проверяем, существует ли такой тариф
		plan, exists := planDuration[req.ProductID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown product ID"})
			return
		}

		ctx := context.Background()
		tx, err := db.Begin(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Transaction start failed"})
			return
		}
		defer tx.Rollback(ctx)

		// 2. Получаем текущую дату окончания премиума пользователя (если есть)
		var currentExpires *time.Time
		err = tx.QueryRow(ctx, "SELECT premium_expires FROM users WHERE email = $1", req.Email).Scan(&currentExpires)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}

		// 3. Рассчитываем новую дату окончания
		// Если премиум еще активен, прибавляем к нему. Если нет — к текущему моменту.
		baseDate := time.Now()
		if currentExpires != nil && currentExpires.After(time.Now()) {
			baseDate = *currentExpires
		}
		newExpiresAt := baseDate.AddDate(plan.Years, plan.Months, 0)

		// 4. Логируем платеж
		_, err = tx.Exec(ctx, `
			INSERT INTO payments (
				user_email, subscription_type, amount, platform, 
				status, transaction_id, purchase_token, expires_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (transaction_id) DO NOTHING`,
			req.Email, req.ProductID, plan.Price, req.Platform,
			"completed", req.TransactionID, req.PurchaseToken, newExpiresAt,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save payment record"})
			return
		}

		// 5. Активируем премиум
		_, err = tx.Exec(ctx, `
			UPDATE users 
			SET is_premium = true, 
			    premium_expires = $1, 
			    updated_at = NOW() 
			WHERE email = $2`,
			newExpiresAt, req.Email,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to activate premium"})
			return
		}

		if err := tx.Commit(ctx); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Commit failed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":     "success",
			"message":    "Premium updated",
			"expires_at": newExpiresAt.Format(time.RFC3339),
		})
	}
}
