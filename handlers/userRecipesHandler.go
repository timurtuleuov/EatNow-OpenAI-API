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

type UserRecipeItem struct {
	ID               string                 `json:"id"`
	CreatedAt        time.Time              `json:"created_at"`
	Title            string                 `json:"title"`
	Description      *string                `json:"description,omitempty"`
	Servings         int                    `json:"servings"`
	TotalTimeMinutes int                    `json:"total_time_minutes"`
	Difficulty       *model.Difficulty      `json:"difficulty,omitempty"`
	Ingredients      []model.Ingredient     `json:"ingredients"`
	Steps            []model.StepModel      `json:"steps"`
	Nutrition        map[string]interface{} `json:"nutrition,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	ImageURL         *string                `json:"image_url,omitempty"`
	Source           *string                `json:"source,omitempty"`
}

func GetUserRecipes(db *pgxpool.Pool) gin.HandlerFunc {
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

		rows, err := db.Query(context.Background(),
			`SELECT id, recipe, created_at
			 FROM recipes
			 WHERE user_id = $1::uuid
			 ORDER BY created_at DESC`,
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load recipes"})
			return
		}
		defer rows.Close()

		recipes := []UserRecipeItem{}
		for rows.Next() {
			var id string
			var recipeBytes []byte
			var createdAt time.Time

			err := rows.Scan(&id, &recipeBytes, &createdAt)
			if err != nil {
				continue
			}

			var recipe model.Recipe
			if err := json.Unmarshal(recipeBytes, &recipe); err != nil {
				continue
			}

			item := UserRecipeItem{
				ID:               id,
				CreatedAt:        createdAt,
				Title:            recipe.Title,
				Description:      recipe.Description,
				Servings:         recipe.Servings,
				TotalTimeMinutes: recipe.TotalTimeMinutes,
				Difficulty:       recipe.Difficulty,
				Ingredients:      recipe.Ingredients,
				Steps:            recipe.Steps,
				Nutrition:        recipe.Nutrition,
				Tags:             recipe.Tags,
				ImageURL:         recipe.ImageURL,
				Source:           recipe.Source,
			}

			recipes = append(recipes, item)
		}

		if recipes == nil {
			recipes = []UserRecipeItem{}
		}

		c.JSON(http.StatusOK, gin.H{"recipes": recipes})
	}
}

func DeleteRecipe(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		recipeID := c.Param("id")
		if recipeID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing recipe id"})
			return
		}

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		tag, err := db.Exec(context.Background(),
			`DELETE FROM recipes WHERE id = $1::uuid AND user_id = $2::uuid`,
			recipeID, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete recipe"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found or access denied"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "deleted"})
	}
}
