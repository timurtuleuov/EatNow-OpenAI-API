package main

import (
	"net/http"
	handlers "openai/handlers"

	"github.com/gin-gonic/gin"
)

func main() {
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
	router.GET("/recipe/:prompt", func(c *gin.Context) {
		recipe, err := handlers.GetRecipeByPrompt(c)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, recipe)
	})
	router.Run("localhost:8080")
}
