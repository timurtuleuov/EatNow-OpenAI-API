package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	model "openai/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SaveNutritionLog(db *pgxpool.Pool, email string, result *model.NutritionLogResponse) (string, error) {
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx,
		"SELECT id::text FROM users WHERE email = $1", email,
	).Scan(&userID)
	if err != nil {
		return "", err
	}

	mealsJSON, err := json.Marshal(result.Meals)
	if err != nil {
		return "", err
	}

	totalJSON, err := json.Marshal(result.Total)
	if err != nil {
		return "", err
	}

	var logID string
	err = db.QueryRow(ctx, `
		INSERT INTO nutrition_logs (user_id, meals, total, water_gl, health_score, analysis, tips)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id, log_date)
		DO UPDATE SET
			meals = EXCLUDED.meals,
			total = EXCLUDED.total,
			water_gl = EXCLUDED.water_gl,
			health_score = EXCLUDED.health_score,
			analysis = EXCLUDED.analysis,
			tips = EXCLUDED.tips,
			updated_at = NOW()
		RETURNING id
	`, userID, mealsJSON, totalJSON, result.WaterGL, result.HealthScore, result.Analysis, result.Tips).Scan(&logID)

	if err != nil {
		return "", err
	}

	return logID, nil
}

func GetNutritionLogs(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		date := c.Query("date")

		rows, err := db.Query(context.Background(), `
			SELECT id, log_date, meals, total, water_gl, health_score, analysis, tips, created_at, updated_at
			FROM nutrition_logs
			WHERE user_id = $1::uuid
			AND ($2 = '' OR log_date = $2::date)
			ORDER BY log_date DESC
		`, userID, date)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load nutrition logs"})
			return
		}
		defer rows.Close()

		logs := []model.NutritionLog{}
		for rows.Next() {
			var log model.NutritionLog
			var mealsBytes, totalBytes []byte
			var logDate time.Time

			if err := rows.Scan(&log.ID, &logDate, &mealsBytes, &totalBytes,
				&log.WaterGL, &log.HealthScore, &log.Analysis, &log.Tips,
				&log.CreatedAt, &log.UpdatedAt); err != nil {
				continue
			}

			log.LogDate = logDate.Format("2006-01-02")

			json.Unmarshal(mealsBytes, &log.Meals)
			json.Unmarshal(totalBytes, &log.Total)

			logs = append(logs, log)
		}

		if logs == nil {
			logs = []model.NutritionLog{}
		}

		c.JSON(http.StatusOK, gin.H{"logs": logs})
	}
}

func GetNutritionLogToday(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var log model.NutritionLog
		var mealsBytes, totalBytes []byte
		var logDate time.Time

		err = db.QueryRow(context.Background(), `
			SELECT id, log_date, meals, total, water_gl, health_score, analysis, tips, created_at, updated_at
			FROM nutrition_logs
			WHERE user_id = $1::uuid AND log_date = CURRENT_DATE
			LIMIT 1
		`, userID).Scan(&log.ID, &logDate, &mealsBytes, &totalBytes,
			&log.WaterGL, &log.HealthScore, &log.Analysis, &log.Tips,
			&log.CreatedAt, &log.UpdatedAt)

		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"log": nil,
				"date": time.Now().Format("2006-01-02"),
			})
			return
		}

		log.LogDate = logDate.Format("2006-01-02")
		json.Unmarshal(mealsBytes, &log.Meals)
		json.Unmarshal(totalBytes, &log.Total)

		c.JSON(http.StatusOK, gin.H{"log": log})
	}
}

func DeleteNutritionLog(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")
		logID := c.Param("id")

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		tag, err := db.Exec(context.Background(),
			`DELETE FROM nutrition_logs WHERE id = $1::uuid AND user_id = $2::uuid`,
			logID, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete log"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "log not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}

func GetNutritionStats(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")
		period := c.DefaultQuery("period", "week")

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var interval string
		switch period {
		case "month":
			interval = "30 days"
		default:
			interval = "7 days"
		}

		var stats model.NutritionStats
		stats.Period = period

		err = db.QueryRow(context.Background(), `
			SELECT
				COALESCE(AVG((total->>'calories')::numeric), 0) as avg_calories,
				COALESCE(AVG((total->>'protein_g')::numeric), 0) as avg_protein,
				COALESCE(AVG((total->>'fat_g')::numeric), 0) as avg_fat,
				COALESCE(AVG((total->>'carbs_g')::numeric), 0) as avg_carbs,
				COALESCE(AVG(health_score), 0) as avg_health,
				COUNT(*) as log_count
			FROM nutrition_logs
			WHERE user_id = $1::uuid
				AND log_date >= CURRENT_DATE - $2::interval
		`, userID, interval).Scan(
			&stats.AverageCalories,
			&stats.AverageProtein,
			&stats.AverageFat,
			&stats.AverageCarbs,
			&stats.AverageHealthScore,
			&stats.LogCount,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to compute stats"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"stats": stats})
	}
}
