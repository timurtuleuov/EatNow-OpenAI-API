package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"openai/db"
	handlers "openai/handlers"
	"os"

	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	// "github.com/joho/godotenv"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var APP_VERSION = "1.2.0"

func InitConfig() {
	// 1. Указываем файлы жестко, чтобы не путаться в именах
	defaultFile := "config.default.yaml"
	localFile := "config.local.yaml"

	// Функция для загрузки (слоеный пирог)
	load := func() {
		viper.SetConfigFile(defaultFile)
		if err := viper.ReadInConfig(); err != nil {
			log.Printf("Ошибка дефолта: %v", err)
		}

		// Накладываем локальный
		viper.SetConfigFile(localFile)
		if err := viper.MergeInConfig(); err != nil {
			// Не страшно, если локального нет
		}
		fmt.Println("✅ Конфиг успешно обновлен в памяти")
	}

	// 2. Первый запуск
	load()

	// 3. Создаем СВОЙ наблюдатель (Watcher)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	// Мы не закрываем его (defer), так как он должен жить всё время работы сервера

	// 4. Добавляем файлы в список слежки
	watcher.Add(defaultFile)
	watcher.Add(localFile)

	// 5. Запускаем фоновый процесс обработки событий
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				// Проверяем, было ли это изменение (Write)
				if event.Op&fsnotify.Write == fsnotify.Write {
					fmt.Printf("изменение в файле: %s\n", event.Name)
					load() // Перезагружаем данные в Viper
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("ошибка watcher:", err)
			}
		}
	}()
}

func main() {
	InitConfig()
	// if err := godotenv.Load("dependencies.env"); err != nil {
	// 	log.Printf("Не удалось загрузить .env: %v", err)
	// }

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	slog.SetDefault(logger)

	pool, err := db.Connect()
	if err != nil {
		logger.Error("DB connection error", "message", err, "version", APP_VERSION)
	}
	defer pool.Close()

	if err := db.InitTables(pool); err != nil {
		logger.Error("Failed to initialize users table", "message", err, "version", APP_VERSION)
	}
	logger.Info("Succesfully connected to the DB", "version", APP_VERSION)

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

			// log.Printf("🚀 Попытка входа:")
			// log.Printf("   Email: [%s]", body.Email)
			// log.Printf("   Password length: %d", len(body.Password)) // Пароли лучше не логировать целиком

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

		protected.GET("/me", handlers.GetMe(pool))

		protected.POST("/recipe", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			email := userEmail.(string)

			var body struct {
				DeviceID string `json:"device_id"`
				Prompt   string `json:"prompt"`
				Image    string `json:"image"`
			}

			logger.Info("new_recipe_request",
				"user_email", email,
				"has_image", body.Image != "",
				"version", APP_VERSION,
			)

			if err := c.ShouldBindJSON(&body); err != nil {
				logger.Error("request_parse_failed",
					"error", err,
					"user_email", email,
				)
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if body.Prompt == "" {
				logger.Warn("validation_failed",
					"reason", "empty_prompt",
					"user_email", email,
				)
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
				return
			}

			allowed, err := handlers.CanUsePrompt(pool, userEmail.(string))
			if err != nil {
				logger.Error("db_error_check_limits",
					"error", err,
					"user_email", email,
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {

				logger.Warn("limit_reached", "user_email", email)
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			start := time.Now()
			// Сделаю тестовый возврат готового рецепта чтобы
			// recipe := models.MockRecipes()[0]
			// time.Sleep(time.Duration(20) * time.Second)
			// TODO: не забудь
			opName := "unknown"

			hasImage := body.Image != ""
			operation, err := handlers.DetectAIOperation(body.Prompt, hasImage)

			if operation != nil {
				opName = *operation
			}

			logger.Info("ai_operation_detected",
				"operation", opName,
				"user_email", email,
			)

			switch opName {
			case "GENERATE":
				recipe, err := handlers.GetRecipeByPrompt(body.Prompt)
				// println("ТЕЛО:", recipe)
				if err != nil {
					log.Println("❌ Recipe generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// log := handlers.PromptLog{
				// 	DeviceID:   body.DeviceID,
				// 	Prompt:     body.Prompt,
				// 	Response:   recipe,
				// 	Model:      "gpt-4o-mini",
				// 	DurationMs: int(duration),
				// 	Success:    err == nil,
				// 	ErrorMsg:   fmt.Sprintf("%v", err),
				// 	AppVersion: "1.1.0",
				// 	Language:   "ru",
				// 	Country:    "KZ",
				// }
				// _ = handlers.LogPrompt(pool, log)

				c.JSON(http.StatusOK, gin.H{
					"operation": operation, "data": recipe,
				})
			case "CONSULT":
				consult, err := handlers.Consult(body.Prompt)
				// println("ТЕЛО:", consult)
				if err != nil {
					log.Println("❌ Consult generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": operation, "data": consult,
				})
			case "CALORIES":
				calories, err := handlers.Calories(body.Prompt, body.Image)
				// println("ТЕЛО:", calories)
				if err != nil {
					log.Println("❌ Consult generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": operation, "data": calories,
				})
			case "RECIPE_PHOTO":
				recipe, err := handlers.GetRecipeFromPhoto(body.Prompt, body.Image)
				// println("ТЕЛО:", recipe)
				if err != nil {
					log.Println("❌ Recipe generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				// log := handlers.PromptLog{
				// 	DeviceID:   body.DeviceID,
				// 	Prompt:     body.Prompt,
				// 	Response:   recipe,
				// 	Model:      "gpt-4o-mini",
				// 	DurationMs: int(duration),
				// 	Success:    err == nil,
				// 	ErrorMsg:   fmt.Sprintf("%v", err),
				// 	AppVersion: "1.1.0",
				// 	Language:   "ru",
				// 	Country:    "KZ",
				// }
				// _ = handlers.LogPrompt(pool, log)

				c.JSON(http.StatusOK, gin.H{
					"operation": operation, "data": recipe,
				})
			}

			duration := time.Since(start).Milliseconds()
			if err != nil {
				// 3. Логируем ошибку генерации с контекстом
				logger.Error("ai_generation_failed",
					"operation", opName,
					"error", err,
					"duration_ms", duration,
					"user_email", email,
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process ai request"})
				return
			}

			logger.Info("recipe_processed_success",
				"operation", opName,
				"duration_ms", duration,
				"user_email", email,
				"device_id", body.DeviceID,
				"country", "KZ",
			)
		})

		protected.POST("/recipe/get-free", func(c *gin.Context) {
			userEmail, _ := c.Get("email")

			if userEmail.(string) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
				return
			}

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

		//payments
		payments := protected.Group("/payments")
		{
			payments.POST("/verify-google", handlers.VerifyGooglePurchase(pool))
		}

	}

	router.Run(":8080")
}
