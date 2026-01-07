package main

import (
	"fmt"
	"log"
	"net/http"
	"openai/db"
	handlers "openai/handlers"
	"time"

	"github.com/gin-contrib/cors"
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
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Для теста, потом замени на свой домен
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))
	auth := router.Group("/auth")
	{
		auth.POST("/register", func(c *gin.Context) {
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

			if ok {
				token, err := handlers.GenerateJWT(body.Email)
				refreshToken, _ := handlers.GenerateRefreshToken()

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
					return
				}

				err = handlers.SaveRefreshToken(pool, body.Email, refreshToken)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save session"})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"access_token":  token,
					"refresh_token": refreshToken,
				})
			}
		})
		auth.POST("/login", func(c *gin.Context) {
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			log.Printf("🚀 Попытка входа:")
			log.Printf("   Email: [%s]", body.Email)
			log.Printf("   Password length: %d", len(body.Password)) // Пароли лучше не логировать целиком

			ok, err := handlers.CheckUserExistsAndAuth(pool, body.Email, body.Password)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
				return
			}

			if ok {
				token, err := handlers.GenerateJWT(body.Email)
				refreshToken, _ := handlers.GenerateRefreshToken()

				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
					return
				}

				err = handlers.SaveRefreshToken(pool, body.Email, refreshToken)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save session"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"access_token":  token,
					"refresh_token": refreshToken,
				})
			}
		})

		auth.POST("/refresh", func(c *gin.Context) {

			var body struct {
				Email        string `json:"email"`
				RefreshToken string `json:"refresh_token"`
			}
			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			err := handlers.VerifyRefreshToken(pool, body.Email, body.RefreshToken)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
				return
			}

			newAccessToken, _ := handlers.GenerateJWT(body.Email)

			c.JSON(http.StatusOK, gin.H{
				"access_token": newAccessToken,
			})
		})
	}

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
	protected := router.Group("/api")
	protected.Use(handlers.AuthMiddleware())
	{
		protected.GET("/home", func(c *gin.Context) {
			c.JSON(http.StatusAccepted, gin.H{"msg": "Hello"})
			return
		})

		protected.POST("/recipe", func(c *gin.Context) {
			userEmail, _ := c.Get("email")

			var body struct {
				DeviceID string `json:"device_id"`
				Prompt   string `json:"prompt"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				log.Println("❌ JSON bind error:", err)
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if body.Prompt == "" {
				log.Printf("❌ Missing fields: prompt='%s'\n", body.Prompt)
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
				return
			}

			allowed, err := handlers.CanUsePrompt(pool, userEmail.(string))
			if err != nil {
				log.Println("❌ DB error in CanUsePrompt:", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {
				log.Println("❌ Daily limit reached for", userEmail.(string))
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			start := time.Now()
			recipe, err := handlers.GetRecipeByPrompt(body.Prompt)
			if err != nil {
				log.Println("❌ Recipe generation error:", err)
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

		protected.POST("/recipe/get-free", func(c *gin.Context) {
			userEmail, _ := c.Get("email")

			// Проверка, что device_id и prompt не пустые
			if userEmail.(string) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
				return
			}

			//bonus works 7 days
			if err := handlers.GrantBonus(pool, userEmail.(string), "reward_ad", 168*time.Hour); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "bonus granted"})
		})

		protected.GET("/user/prompts-count", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			print(userEmail.(string))
			if userEmail.(string) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing required fields"})
				return
			}

			userFreePromptsCount, err := handlers.GetUserFreePromptsCount(pool, userEmail.(string))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"userFreePromptsCount": userFreePromptsCount})

		})
	}

	router.Run(":8080")
}
