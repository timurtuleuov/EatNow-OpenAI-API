package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"openai/db"
	handlers "openai/handlers"
	model "openai/models"
	"os"
	"strings"
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
	defaultFile := "configs/config.default.yaml"
	localFile := "configs/config.local.yaml"

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
	if err := watcher.Add("./configs"); err != nil {
		log.Printf("Ошибка watcher.Add: %v", err)
	}

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

func enrichWithTavily(tavilyKey, prompt string, isBrainrot bool) string {
	tavilyData, err := handlers.SearchTavily(tavilyKey, prompt, isBrainrot)
	if err != nil || len(tavilyData.Results) == 0 {
		return prompt
	}

	var contextInfo strings.Builder
	contextInfo.WriteString("\nНайденная информация в интернете:\n")
	for _, res := range tavilyData.Results {
		contextInfo.WriteString(fmt.Sprintf("- %s: %s\n", res.Title, res.Content))
	}

	return prompt + contextInfo.String()
}

func main() {
	InitConfig()
	// if err := godotenv.Load("dependencies.env"); err != nil {
	// 	log.Printf("Не удалось загрузить .env: %v", err)
	// }

	// ctx := context.Background()
	// imgGen, err := handlers.NewImageGenerator(ctx)
	// if err != nil {
	// 	log.Fatalf("Ошибка инициализации Gemini: %v", err)
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

	tavilyKey := viper.GetString("tavily_key")
	enableTavily := viper.GetBool("tavily_use")

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"}, // Для теста, потом замени на свой домен
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.Static("/images", "./images")

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
				DeviceID   string          `json:"device_id"`
				Prompt     string          `json:"prompt"`
				History    []model.Message `json:"history"`
				Image      string          `json:"image"`
				IsBrainrot bool            `json:"is_brainrot"`
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
			opName, refinedPrompt, err := handlers.DetectAIOperation(body.Prompt, body.History, hasImage, body.Image)

			if err != nil {
				logger.Error("ai_operation_detection_failed",
					"error", err,
					"user_email", email,
				)
				// Если детектор упал, по умолчанию считаем это обычной консультацией или генерацией
				opName = "CONSULT"
				refinedPrompt = body.Prompt
			}

			logger.Info("ai_operation_detected",
				"operation", opName,
				"refined_prompt", refinedPrompt,
				"user_email", email,
			)

			switch opName {
			case "GENERATE":

				if enableTavily {
					refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
				}

				recipe, err := handlers.GetRecipeByPrompt(refinedPrompt)
				isPremium := handlers.UserIsPremium(pool, email)
				logger.Info("premium_check", "email", email, "is_premium", isPremium)

				if isPremium {

					imagePrompt := "Сделай картинку блюда по рецепту, без надписей: " + refinedPrompt

					// Если включен экспериментальный режим brainrot
					if body.IsBrainrot {
						// Добавляем модификаторы промпта для генерации визуального безумия
						imagePrompt += viper.GetString("prompts.brainrot_image_prompt")

						logger.Info("brainrot_generation_triggered",
							"user_email", email,
							"operation", "GENERATE_IMAGE",
						)
					}
					imgURL, err := handlers.GenerateImage(imagePrompt)
					// imgURL, err := imgGen.GenerateGeminiImage(ctx, "Сделай картинку блюда по рецепту, без надписей:"+refinedPrompt)

					if err != nil {
						logger.Error("image_generation_failed",
							"error", err,
							"user_email", email,
						)
					} else {
						logger.Info("image_generation_success",
							"user_email", email,
						)
					}
					fileName, err := handlers.SaveImage(imgURL)
					logger.Info("save_image",
						"fileName", fileName,
						"user_email", email,
					)
					if err == nil {
						recipe.ImageURL = &fileName
					}

				}

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
					"operation": opName, "data": recipe,
				})
			case "CONSULT":
				if enableTavily {
					refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
				}

				consult, err := handlers.Consult(refinedPrompt)
				// println("ТЕЛО:", consult)
				if err != nil {
					log.Println("❌ Consult generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": opName, "data": consult,
				})
			case "CALORIES":
				if enableTavily {
					refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
				}

				calories, err := handlers.Calories(refinedPrompt, body.Image)
				// println("ТЕЛО:", calories)
				if err != nil {
					log.Println("❌ Consult generation error:", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": opName, "data": calories,
				})
			case "RECIPE_PHOTO":

				recipe, err := handlers.GetRecipeFromPhoto(refinedPrompt, body.Image)
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
					"operation": opName, "data": recipe,
				})

			default:
				// Если ИИ выдал что-то странное, просто консультируем
				consult, _ := handlers.Consult(refinedPrompt)
				c.JSON(http.StatusOK, gin.H{"operation": "CONSULT", "data": consult})
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

		protected.POST("/meal-plan", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			email := userEmail.(string)

			var body struct {
				Prompt     string          `json:"prompt"`
				History    []model.Message `json:"history"`
				IsBrainrot bool            `json:"is_brainrot"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if body.Prompt == "" {
				body.Prompt = "Составь план питания на неделю"
			}

			allowed, err := handlers.CanUsePrompt(pool, email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			refinedPrompt := body.Prompt
			if enableTavily {
				refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
			}

			mealPlan, err := handlers.GenerateMealPlan(refinedPrompt)
			if err != nil {
				logger.Error("meal_plan_generation_failed", "error", err, "user_email", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("meal_plan_generated", "user_email", email, "days", len(mealPlan.Days))
			c.JSON(http.StatusOK, gin.H{"operation": "MEAL_PLAN", "data": mealPlan})
		})

		favorites := protected.Group("/favorites")
		{
			favorites.POST("", handlers.AddToFavorites(pool))
			favorites.GET("", handlers.GetFavorites(pool))
			favorites.DELETE("/:id", handlers.RemoveFavorite(pool))
		}

		protected.POST("/substitute", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			email := userEmail.(string)

			var body struct {
				Ingredient string `json:"ingredient"`
				Reason     string `json:"reason"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if body.Ingredient == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing ingredient"})
				return
			}

			allowed, err := handlers.CanUsePrompt(pool, email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			result, err := handlers.GetSubstitutes(body.Ingredient, body.Reason)
			if err != nil {
				logger.Error("substitute_failed", "error", err, "user_email", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("substitute_generated", "user_email", email, "ingredient", body.Ingredient)
			c.JSON(http.StatusOK, gin.H{"operation": "SUBSTITUTE", "data": result})
		})

		protected.POST("/what-to-cook", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			email := userEmail.(string)

			var body struct {
				Ingredients []string `json:"ingredients"`
				Preferences string   `json:"preferences"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if len(body.Ingredients) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing ingredients"})
				return
			}

			allowed, err := handlers.CanUsePrompt(pool, email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			result, err := handlers.WhatToCook(body.Ingredients, body.Preferences)
			if err != nil {
				logger.Error("what_to_cook_failed", "error", err, "user_email", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("what_to_cook_generated", "user_email", email, "ingredients_count", len(body.Ingredients))
			c.JSON(http.StatusOK, gin.H{"operation": "WHAT_TO_COOK", "data": result})
		})

		protected.POST("/nutrition-log", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			email := userEmail.(string)

			var body struct {
				Meals []string `json:"meals"`
			}

			if err := c.ShouldBindJSON(&body); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if len(body.Meals) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing meals"})
				return
			}

			allowed, err := handlers.CanUsePrompt(pool, email)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"error": "daily prompt limit reached"})
				return
			}

			result, err := handlers.AnalyzeNutritionLog(body.Meals)
			if err != nil {
				logger.Error("nutrition_log_failed", "error", err, "user_email", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("nutrition_log_analyzed", "user_email", email, "meals", len(body.Meals))
			c.JSON(http.StatusOK, gin.H{"operation": "NUTRITION_LOG", "data": result})
		})

	}

	router.Run(":8080")
}
