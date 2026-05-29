package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	model "openai/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func GetProfile(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")

		var profile model.DietaryProfile
		var isPremium bool
		var expiresAt *string

		err := db.QueryRow(context.Background(), `
			SELECT
				COALESCE(diet_type, ''),
				COALESCE(allergies, '{}'),
				COALESCE(excluded_ingredients, '{}'),
				COALESCE(cuisine_preferences, '{}'),
				COALESCE(daily_calorie_goal, 0),
				COALESCE(daily_protein_goal, 0),
				COALESCE(daily_fat_goal, 0),
				COALESCE(daily_carbs_goal, 0),
				is_premium,
				COALESCE(premium_expires::text, '')
			FROM users WHERE email = $1
		`, email).Scan(
			&profile.DietType,
			&profile.Allergies,
			&profile.ExcludedIngredients,
			&profile.CuisinePreferences,
			&profile.DailyCalorieGoal,
			&profile.DailyProteinGoal,
			&profile.DailyFatGoal,
			&profile.DailyCarbsGoal,
			&isPremium,
			&expiresAt,
		)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"email":      email,
			"is_premium": isPremium,
			"profile":    profile,
		})
	}
}

func UpdateProfile(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		email, _ := c.Get("email")

		var body model.DietaryProfile
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		_, err := db.Exec(context.Background(), `
			UPDATE users SET
				diet_type = $1,
				allergies = $2,
				excluded_ingredients = $3,
				cuisine_preferences = $4,
				daily_calorie_goal = $5,
				daily_protein_goal = $6,
				daily_fat_goal = $7,
				daily_carbs_goal = $8
			WHERE email = $9
		`,
			body.DietType,
			body.Allergies,
			body.ExcludedIngredients,
			body.CuisinePreferences,
			body.DailyCalorieGoal,
			body.DailyProteinGoal,
			body.DailyFatGoal,
			body.DailyCarbsGoal,
			email,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "updated", "profile": body})
	}
}

func BuildDietaryContext(db *pgxpool.Pool, email string) string {
	var profile model.DietaryProfile

	err := db.QueryRow(context.Background(), `
		SELECT
			COALESCE(diet_type, ''),
			COALESCE(allergies, '{}'),
			COALESCE(excluded_ingredients, '{}'),
			COALESCE(cuisine_preferences, '{}')
		FROM users WHERE email = $1
	`, email).Scan(
		&profile.DietType,
		&profile.Allergies,
		&profile.ExcludedIngredients,
		&profile.CuisinePreferences,
	)
	if err != nil {
		return ""
	}

	var parts []string
	if profile.DietType != "" {
		parts = append(parts, fmt.Sprintf("Diet type: %s", profile.DietType))
	}
	if len(profile.Allergies) > 0 {
		parts = append(parts, fmt.Sprintf("Allergies: %s", strings.Join(profile.Allergies, ", ")))
	}
	if len(profile.ExcludedIngredients) > 0 {
		parts = append(parts, fmt.Sprintf("Excluded ingredients: %s", strings.Join(profile.ExcludedIngredients, ", ")))
	}
	if len(profile.CuisinePreferences) > 0 {
		parts = append(parts, fmt.Sprintf("Preferred cuisines: %s", strings.Join(profile.CuisinePreferences, ", ")))
	}

	if len(parts) == 0 {
		return ""
	}

	return "User dietary preferences: " + strings.Join(parts, ". ") + "."
}
