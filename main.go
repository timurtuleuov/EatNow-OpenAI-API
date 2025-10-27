package main

import (
	"fmt"
	"log"
	"net/http"
	"openai/db"
	handlers "openai/handlers"
	"time"

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

	if err := db.InitTables(pool); err != nil {
		log.Fatalf("Failed to initialize users table: %v", err)
	}

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

	router.GET("/home", func(c *gin.Context) {
		c.JSON(http.StatusAccepted, gin.H{"msg": "Hello"})
		return
	})

	router.POST("/recipe", func(c *gin.Context) {
		var body struct {
			DeviceID string `json:"device_id"`
			Prompt   string `json:"prompt"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		// Проверка, что device_id и prompt не пустые
		if body.DeviceID == "" || body.Prompt == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
			return
		}

		allowed, err := handlers.CanUsePrompt(pool, body.DeviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}
		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
			return
		}
		// handlers.GetFreePrompt(pool, body.DeviceID)
		start := time.Now()
		recipe, err := handlers.GetRecipeByPrompt(body.Prompt)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})

			return
		}

		duration := time.Since(start).Milliseconds()

		log := handlers.PromptLog{
			DeviceID:   body.DeviceID,
			Prompt:     body.Prompt,
			Response:   recipe,
			Model:      "gpt-4o-mini",
			DurationMs: int(duration),
			Success:    err == nil,
			ErrorMsg:   fmt.Sprintf("%v", err),
			AppVersion: "1.0.0",
			Language:   "ru",
			Country:    "KZ",
		}
		_ = handlers.LogPrompt(pool, log)

		c.JSON(http.StatusOK, recipe)
	})

	router.POST("/recipe/get-free", func(c *gin.Context) {
		var body struct {
			DeviceID string `json:"device_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		// Проверка, что device_id и prompt не пустые
		if body.DeviceID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
			return
		}

		//bonus works 7 days
		if err := handlers.GrantBonus(pool, body.DeviceID, "reward_ad", 168*time.Hour); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "bonus granted"})
	})

	router.POST("/auth/register", func(c *gin.Context) {
		var body struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			Password string `json:"password"`
			Platform string `json:"platform"`
			DeviceID string `json:"device_id"`
		}

		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		if body.Username == "" || body.Email == "" || body.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
			return
		}

		ok, err := handlers.CreateUser(pool, body.Username, body.Email, body.Password, body.Platform, body.DeviceID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "user not created"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "user registered successfully"})
	})

	router.POST("/auth/login", func(c *gin.Context) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
			DeviceID string `json:"device_id"`
		}

		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		ok, err := handlers.CheckUserExistsAndAuth(pool, body.Email, body.DeviceID, body.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}

		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "login successful"})
	})

	router.Run(":8080")
}
