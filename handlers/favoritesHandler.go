package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"openai/internal/logger"
	"strconv"

	model "openai/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AddToFavorites(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		// 🟢 ИСПРАВЛЕНО: Теперь ждем строку (UUID), а не int
		var body struct {
			RecipeID string `json:"recipe_id" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON, recipe_id (string UUID) is required"})
			return
		}

		// Получаем UUID пользователя по email
		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var favID int
		err = db.QueryRow(context.Background(),
			`INSERT INTO favorites (user_id, recipe_id) 
             VALUES ($1::uuid, $2::uuid) 
             ON CONFLICT (user_id, recipe_id) DO UPDATE SET recipe_id = EXCLUDED.recipe_id
             RETURNING id`,
			userID, body.RecipeID,
		).Scan(&favID)

		if err != nil {
			slog.Error("favorite_insert_failed",
				logger.KeyError, err,
				logger.KeyUser, email,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save favorite"})
			return
		}

		go func() {
			if recipe := fetchAndBuildRecipe(db, body.RecipeID); recipe != nil {
				UpdateUserMemory(db, email, recipe)
			}
		}()

		c.JSON(http.StatusCreated, gin.H{"id": favID, "status": "saved"})
	}
}

func fetchAndBuildRecipe(db *pgxpool.Pool, recipeID string) *model.Recipe {
	recipe, err := RecipeFromDB(db, recipeID)
	if err != nil {
		slog.Error("fetch_recipe_for_memory_failed", "recipe_id", recipeID, "error", err)
	}
	return recipe
}

func GetFavorites(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		// Получаем UUID пользователя
		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		rows, err := db.Query(context.Background(),
			`SELECT 
                f.id, 
                f.user_id::text,
                f.recipe_id::text,
                f.created_at,
                r.recipe
             FROM favorites f
             INNER JOIN recipes r ON f.recipe_id = r.id 
             WHERE f.user_id = $1::uuid
             ORDER BY f.created_at DESC`,
			userID,
		)
		if err != nil {
			slog.Error("favorites_query_failed",
				logger.KeyError, err,
				logger.KeyUser, email,
			)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load favorites"})
			return
		}
		defer rows.Close()

		favorites := []model.Favorite{}
		for rows.Next() {
			var fav model.Favorite
			var recipeBytes []byte

			err := rows.Scan(
				&fav.ID,
				&fav.UserID,
				&fav.RecipeID,
				&fav.CreatedAt,
				&recipeBytes,
			)
			if err != nil {
				slog.Error("favorite_scan_failed",
					logger.KeyError, err,
					logger.KeyUser, email,
				)
				continue
			}

			if err := json.Unmarshal(recipeBytes, &fav.Recipe); err != nil {
				slog.Error("favorite_unmarshal_failed",
					logger.KeyError, err,
					logger.KeyUser, email,
				)
				continue
			}

			// 🟢 ТЕПЕРЬ ОШИБКИ НЕТ: Оба поля имеют тип string
			fav.Recipe.ID = fav.RecipeID

			favorites = append(favorites, fav)
		}

		if favorites == nil {
			favorites = []model.Favorite{}
		}

		c.JSON(http.StatusOK, gin.H{"favorites": favorites})
	}
}

func RemoveFavorite(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		favIDStr := c.Param("id")
		favID, err := strconv.Atoi(favIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid favorite id"})
			return
		}

		var userID string
		err = db.QueryRow(context.Background(),
			"SELECT id::text FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		// Получаем recipe_id до удаления
		var recipeID string
		err = db.QueryRow(context.Background(),
			"SELECT recipe_id::text FROM favorites WHERE id = $1 AND user_id = $2::uuid",
			favID, userID,
		).Scan(&recipeID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found"})
			return
		}

		tag, err := db.Exec(context.Background(),
			`DELETE FROM favorites WHERE id = $1 AND user_id = $2::uuid`,
			favID, userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove favorite"})
			return
		}
		if tag.RowsAffected() == 0 {
			c.JSON(http.StatusNotFound, gin.H{"error": "favorite not found"})
			return
		}

		// Обновляем память — пользователь убрал из избранного
		go func() {
			if recipe := fetchAndBuildRecipe(db, recipeID); recipe != nil {
				recipe.Title = recipe.Title + " (user removed from favorites)"
				UpdateUserMemory(db, email, recipe)
			}
		}()

		c.JSON(http.StatusOK, gin.H{"status": "removed"})
	}
}
