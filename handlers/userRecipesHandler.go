package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
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

func ExportRecipe(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)
		recipeID := c.Param("id")

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var recipeBytes []byte
		err = db.QueryRow(context.Background(),
			`SELECT recipe FROM recipes WHERE id = $1::uuid AND user_id = $2::uuid`,
			recipeID, userID,
		).Scan(&recipeBytes)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "recipe not found or access denied"})
			return
		}

		var recipe model.Recipe
		if err := json.Unmarshal(recipeBytes, &recipe); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse recipe"})
			return
		}

		var b strings.Builder
		fmt.Fprintf(&b, "%s\n", recipe.Title)
		if recipe.Description != nil {
			fmt.Fprintf(&b, "%s\n", *recipe.Description)
		}
		fmt.Fprintf(&b, "Порции: %d | Время: %d мин", recipe.Servings, recipe.TotalTimeMinutes)
		if recipe.Difficulty != nil {
			fmt.Fprintf(&b, " | Сложность: %s", *recipe.Difficulty)
		}
		b.WriteString("\n\nИнгредиенты:\n")
		for _, ing := range recipe.Ingredients {
			b.WriteString("- ")
			if ing.Quantity != nil {
				fmt.Fprintf(&b, "%g ", *ing.Quantity)
			}
			if ing.Unit != nil {
				fmt.Fprintf(&b, "%s ", *ing.Unit)
			}
			b.WriteString(ing.Name)
			if ing.Prepared != nil {
				fmt.Fprintf(&b, " (%s)", *ing.Prepared)
			}
			if ing.Optional {
				b.WriteString(" (по желанию)")
			}
			b.WriteString("\n")
		}

		b.WriteString("\nПриготовление:\n")
		for _, step := range recipe.Steps {
			fmt.Fprintf(&b, "%d. %s\n", step.Order, step.Description)
			if step.DurationSeconds != nil {
				fmt.Fprintf(&b, "   ⏱ %d сек\n", *step.DurationSeconds)
			}
			if step.Tip != nil {
				fmt.Fprintf(&b, "   💡 %s\n", *step.Tip)
			}
		}

		if len(recipe.Nutrition) > 0 {
			b.WriteString("\nПищевая ценность:\n")
			if cal, ok := recipe.Nutrition["calories"]; ok {
				fmt.Fprintf(&b, "  Калории: %v\n", cal)
			}
			if prot, ok := recipe.Nutrition["protein"]; ok {
				fmt.Fprintf(&b, "  Белки: %v\n", prot)
			}
			if fat, ok := recipe.Nutrition["fat"]; ok {
				fmt.Fprintf(&b, "  Жиры: %v\n", fat)
			}
			if carbs, ok := recipe.Nutrition["carbs"]; ok {
				fmt.Fprintf(&b, "  Углеводы: %v\n", carbs)
			}
		}

		if len(recipe.Tags) > 0 {
			fmt.Fprintf(&b, "\nТеги: %s\n", strings.Join(recipe.Tags, ", "))
		}

		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(b.String()))
	}
}
