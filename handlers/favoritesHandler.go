package handlers

import (
	"context"
	"net/http"
	"strconv"

	model "openai/models"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func AddToFavorites(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		var body struct {
			Recipe model.Recipe `json:"recipe"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		var favID int
		err = db.QueryRow(context.Background(),
			`INSERT INTO favorites (user_id, recipe) VALUES ($1, $2) RETURNING id`,
			userID, body.Recipe,
		).Scan(&favID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save favorite"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"id": favID, "status": "saved"})
	}
}

func GetFavorites(db *pgxpool.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		userEmail, _ := c.Get("email")
		email := userEmail.(string)

		var userID string
		err := db.QueryRow(context.Background(),
			"SELECT id FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		rows, err := db.Query(context.Background(),
			`SELECT id, recipe, created_at FROM favorites WHERE user_id = $1 ORDER BY created_at DESC`,
			userID,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load favorites"})
			return
		}
		defer rows.Close()

		favorites := []model.Favorite{}
		for rows.Next() {
			var fav model.Favorite
			if err := rows.Scan(&fav.ID, &fav.Recipe, &fav.CreatedAt); err != nil {
				continue
			}
			favorites = append(favorites, fav)
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
			"SELECT id FROM users WHERE email = $1", email,
		).Scan(&userID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		tag, err := db.Exec(context.Background(),
			`DELETE FROM favorites WHERE id = $1 AND user_id = $2`,
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

		c.JSON(http.StatusOK, gin.H{"status": "removed"})
	}
}
