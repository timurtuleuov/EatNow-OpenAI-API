package main

import (
	"context"
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

	conn, err := db.Connect()
	if err != nil {
		log.Fatalf("DB conenction error: %v", err)
	}
	defer conn.Close(context.Background())

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
			Prompt string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
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
