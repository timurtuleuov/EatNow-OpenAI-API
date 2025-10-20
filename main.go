package main

import (
	"log"
	"net/http"
	"openai/db"
	handlers "openai/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load("dependencies.env"); err != nil {
		log.Printf("Не удалось загрузить .env: %v", err)
	}

	pool, err := db.Connect()
	if err != nil {
		log.Fatalf("DB connection error: %v", err)
	}
	defer pool.Close()

	log.Println("Succesfully connected to the DB")

	router := gin.Default()
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	})
	router.POST("/recipe", func(c *gin.Context) {
		var body struct {
			UserID string `json:"user"`
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		// Проверка, что user_id и prompt не пустые
		if body.UserID == "" || body.Prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing user_id or prompt"})
			return
		}

		allowed, err := handlers.CanUsePrompt(pool, body.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
			return
		}

		recipe, err := handlers.GetRecipeByPrompt(body.Prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, recipe)
	})
	router.Run("localhost:8080")
}
