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

type MealPlanRecord struct {
	ID        string         `json:"id"`
	MealPlan  model.MealPlan `json:"meal_plan"`
	Prompt    string         `json:"prompt,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}

func SaveMealPlan(db *pgxpool.Pool, email string, mealPlan *model.MealPlan, prompt string) error {
	ctx := context.Background()

	var userID string
	err := db.QueryRow(ctx,
		"SELECT id::text FROM users WHERE email = $1", email,
	).Scan(&userID)
	if err != nil {
		return err
	}

	planJSON, err := json.Marshal(mealPlan)
	if err != nil {
		return err
	}

	_, err = db.Exec(ctx, `
		INSERT INTO meal_plans (user_id, meal_plan, prompt)
		VALUES ($1::uuid, $2, $3)
	`, userID, planJSON, prompt)

	return err
}

func GetLatestMealPlan(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var id string
		var planBytes []byte
		var prompt string
		var createdAt time.Time

		err = db.QueryRow(context.Background(), `
			SELECT id, meal_plan, COALESCE(prompt, ''), created_at
			FROM meal_plans
			WHERE user_id = $1::uuid
			ORDER BY created_at DESC
			LIMIT 1
		`, userID).Scan(&id, &planBytes, &prompt, &createdAt)

		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "meal plan not found"})
			return
		}

		var mealPlan model.MealPlan
		if err := json.Unmarshal(planBytes, &mealPlan); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse meal plan"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"operation": "MEAL_PLAN",
			"data":      mealPlan,
			"prompt":    prompt,
			"created_at": createdAt,
		})
	}
}

func GetUserMealPlans(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		rows, err := db.Query(context.Background(), `
			SELECT id, meal_plan, COALESCE(prompt, ''), created_at
			FROM meal_plans
			WHERE user_id = $1::uuid
			ORDER BY created_at DESC
		`, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load meal plans"})
			return
		}
		defer rows.Close()

		plans := []MealPlanRecord{}
		for rows.Next() {
			var id string
			var planBytes []byte
			var prompt string
			var createdAt time.Time

			if err := rows.Scan(&id, &planBytes, &prompt, &createdAt); err != nil {
				continue
			}

			var mealPlan model.MealPlan
			if err := json.Unmarshal(planBytes, &mealPlan); err != nil {
				continue
			}

			plans = append(plans, MealPlanRecord{
				ID:        id,
				MealPlan:  mealPlan,
				Prompt:    prompt,
				CreatedAt: createdAt,
			})
		}

		if plans == nil {
			plans = []MealPlanRecord{}
		}

		c.JSON(http.StatusOK, gin.H{"meal_plans": plans})
	}
}
