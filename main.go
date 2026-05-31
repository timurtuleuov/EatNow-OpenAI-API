// @title What2Eat API
// @version 1.3.1
// @description AI-powered culinary assistant API. Generate recipes, analyze nutrition, create meal plans, and more.
// @termsOfService https://eatnow.app/terms

// @contactName What2Eat Support
// @contactEmail support@eatnow.app

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

// @host localhost:8080
// @BasePath
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"openai/db"
	handlers "openai/handlers"
	logpkg "openai/internal/logger"
	model "openai/models"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	_ "openai/docs"
)

var APP_VERSION = "1.3.1"

const swaggerHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>What2Eat API - Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    SwaggerUIBundle({ url: "/swagger.json", dom_id: "#swagger-ui" })
  </script>
</body>
</html>`

func InitConfig() {
	// 1. Указываем файлы жестко, чтобы не путаться в именах
	defaultFile := "configs/config.default.yaml"
	localFile := "configs/config.local.yaml"

	// Функция для загрузки (слоеный пирог)
	load := func() {
		viper.SetConfigFile(defaultFile)
		if err := viper.ReadInConfig(); err != nil {
			slog.Error("config_default_load_failed",
				"error", err,
			)
		}

		// Накладываем локальный
		viper.SetConfigFile(localFile)
		if err := viper.MergeInConfig(); err != nil {
			// Не страшно, если локального нет
		}
		slog.Info("config_reloaded")
	}

	// 2. Первый запуск
	load()

	// 3. Создаем СВОЙ наблюдатель (Watcher)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		slog.Error("watcher_create_failed",
			"error", err,
		)
		os.Exit(1)
	}
	// Мы не закрываем его (defer), так как он должен жить всё время работы сервера
	if err := watcher.Add("./configs"); err != nil {
		slog.Error("watcher_add_failed",
			"error", err,
		)
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
					slog.Info("config_file_changed",
						"file", event.Name,
					)
					load() // Перезагружаем данные в Viper
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				slog.Error("watcher_error",
					"error", err,
				)
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

	if viper.GetString("deepseek.api_key") == "" {
		slog.Error("config_validation_failed", "key", "deepseek.api_key")
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	switch viper.GetString("logging.level") {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	isProd := viper.GetBool("server.is_prod")
	logger := logpkg.Init(logLevel, APP_VERSION, isProd)

	pool, databaseURL, err := db.Connect()
	if err != nil {
		logger.Error("db_connect_failed",
			logpkg.KeyError, err,
		)
		os.Exit(1)
	}
	defer pool.Close()

	db.RunMigrations(databaseURL)
	logger.Info("db_connected")

	interval, _ := time.ParseDuration(viper.GetString("scheduler.balance_reset_interval"))
	go handlers.StartBalanceScheduler(pool, interval)

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

	router.GET("/swagger.json", func(c *gin.Context) {
		c.File("./docs/swagger.json")
	})
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})
	router.GET("/swagger/index.html", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(swaggerHTML))
	})

	router.GET("/reset-password", handlers.ResetPasswordPage(pool))

	auth := router.Group("/api/auth")
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

		auth.POST("/forgot-password", handlers.ForgotPassword(pool))
		auth.POST("/reset-password", handlers.ResetPassword(pool))
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

		protected.GET("/me/profile", handlers.GetProfile(pool))
		protected.PUT("/me/profile", handlers.UpdateProfile(pool))

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
				"user", email,
				"has_image", body.Image != "",
				"version", APP_VERSION,
			)

			if err := c.ShouldBindJSON(&body); err != nil {
				logger.Error("request_parse_failed",
					"error", err,
					"user", email,
				)
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
				return
			}

			if body.Prompt == "" {
				logger.Warn("validation_failed",
					"reason", "empty_prompt",
					"user", email,
				)
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
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
					"user", email,
				)
				// Если детектор упал, по умолчанию считаем это обычной консультацией или генерацией
				opName = "CONSULT"
				refinedPrompt = body.Prompt
			}

			logger.Info("ai_operation_detected",
				"op", opName,
				"user", email,
			)

			switch opName {
			case "GENERATE":

				if enableTavily {
					refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
				}

				if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.generate")); err != nil {
					logger.Warn("insufficient_balance",
						"user", email,
						"op", "GENERATE",
						"error", err,
					)
					c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}

				dietaryCtx := handlers.BuildDietaryContext(pool, email)
				recipe, err := handlers.GetRecipeByPrompt(refinedPrompt, dietaryCtx)
				isPremium := handlers.UserIsPremium(pool, email)
				logger.Info("premium_check",
					"user", email,
					"is_premium", isPremium,
				)

				if isPremium {

					imagePrompt := "Сделай картинку блюда по рецепту, без надписей: " + refinedPrompt

					// Если включен экспериментальный режим brainrot
					if body.IsBrainrot {
						// Добавляем модификаторы промпта для генерации визуального безумия
						imagePrompt += viper.GetString("prompts.brainrot_image_prompt")

						logger.Info("brainrot_generation_triggered",
							"user", email,
							"op", "GENERATE_IMAGE",
						)
					}

					if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.generate_image")); err != nil {
						logger.Warn("image_skipped_low_balance",
							"user", email,
							"error", err,
						)
					} else {
						imgURL, err := handlers.GenerateImage(imagePrompt)

						if err != nil {
							logger.Error("image_generation_failed",
								"error", err,
								"user", email,
							)
						} else {
							logger.Info("image_generation_success",
								"user", email,
							)
						}
						fileName, err := handlers.SaveImage(imgURL)
						logger.Info("save_image",
							"fileName", fileName,
							"user", email,
						)
						if err == nil {
							recipe.ImageURL = &fileName
						}
					}

				}

				if err != nil {
					logger.Error("recipe_gen_error",
						logpkg.KeyError, err,
						logpkg.KeyOp, "GENERATE",
						logpkg.KeyUser, email,
					)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				durMs := time.Since(start).Milliseconds()
				logEntry := handlers.PromptLog{
					UserID:     &email,
					DeviceID:   body.DeviceID,
					Prompt:     body.Prompt,
					Response:   recipe,
					TokensUsed: 0,
					Model:      viper.GetString("openai.model"),
					DurationMs: int(durMs),
					Success:    true,
					AppVersion: APP_VERSION,
					Language:   "ru",
					Country:    "KZ",
				}
				recipeDBID, _ := handlers.LogPrompt(pool, logEntry)

				if recipeDBID != "" {
					recipe.ID = recipeDBID
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": opName, "data": recipe,
				})
			case "CONSULT":
				if enableTavily {
					refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
				}

				if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.consult")); err != nil {
					logger.Warn("insufficient_balance",
						"user", email,
						"op", "CONSULT",
						"error", err,
					)
					c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}

				dietaryCtx := handlers.BuildDietaryContext(pool, email)
				consult, err := handlers.Consult(refinedPrompt, dietaryCtx)
				if err != nil {
					logger.Error("consult_gen_error",
						logpkg.KeyError, err,
						logpkg.KeyOp, "CONSULT",
						logpkg.KeyUser, email,
					)
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

				if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.calories")); err != nil {
					logger.Warn("insufficient_balance",
						"user", email,
						"op", "CALORIES",
						"error", err,
					)
					c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}

				dietaryCtx := handlers.BuildDietaryContext(pool, email)
				calories, err := handlers.Calories(refinedPrompt, body.Image, dietaryCtx)
				if err != nil {
					logger.Error("calories_gen_error",
						logpkg.KeyError, err,
						logpkg.KeyOp, "CALORIES",
						logpkg.KeyUser, email,
					)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": opName, "data": calories,
				})
			case "RECIPE_PHOTO":

				if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.recipe_photo")); err != nil {
					logger.Warn("insufficient_balance",
						"user", email,
						"op", "RECIPE_PHOTO",
						"error", err,
					)
					c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
					return
				}

				dietaryCtx := handlers.BuildDietaryContext(pool, email)
				recipe, err := handlers.GetRecipeFromPhoto(refinedPrompt, body.Image, dietaryCtx)
				if err != nil {
					logger.Error("recipe_photo_gen_error",
						logpkg.KeyError, err,
						logpkg.KeyOp, "RECIPE_PHOTO",
						logpkg.KeyUser, email,
					)
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}

				durMs := time.Since(start).Milliseconds()
				logEntry := handlers.PromptLog{
					UserID:     &email,
					DeviceID:   body.DeviceID,
					Prompt:     body.Prompt,
					Response:   recipe,
					TokensUsed: 0,
					Model:      viper.GetString("openai.model"),
					DurationMs: int(durMs),
					Success:    true,
					AppVersion: APP_VERSION,
					Language:   "ru",
					Country:    "KZ",
				}
				recipeDBID, _ := handlers.LogPrompt(pool, logEntry)

				if recipeDBID != "" {
					recipe.ID = recipeDBID
				}

				c.JSON(http.StatusOK, gin.H{
					"operation": opName, "data": recipe,
				})

			default:
				// Если ИИ выдал что-то странное, просто консультируем
				dietaryCtx := handlers.BuildDietaryContext(pool, email)
				consult, _ := handlers.Consult(refinedPrompt, dietaryCtx)
				c.JSON(http.StatusOK, gin.H{"operation": "CONSULT", "data": consult})
			}

			duration := time.Since(start).Milliseconds()
			if err != nil {
				// 3. Логируем ошибку генерации с контекстом
				logger.Error("ai_generation_failed",
					"op", opName,
					"error", err,
					"duration_ms", duration,
					"user", email,
				)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to process ai request"})
				return
			}

			logger.Info("recipe_processed_success",
				"op", opName,
				"duration_ms", duration,
				"user", email,
				"device", body.DeviceID,
			)
		})

		protected.POST("/recipe/get-free", func(c *gin.Context) {
			userEmail, _ := c.Get("email")

			if userEmail.(string) == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing device_id or prompt"})
				return
			}

			if err := handlers.GrantBonus(pool, userEmail.(string), "reward_ad", viper.GetInt("balance.ad_reward")); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "bonus granted"})
		})

		protected.GET("/user/prompts-count", func(c *gin.Context) {
			userEmail, _ := c.Get("email")
			logger.Debug("user_prompts_count_request",
				logpkg.KeyUser, userEmail.(string),
			)
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

			if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.meal_plan")); err != nil {
				logger.Warn("insufficient_balance",
					"user", email,
					"op", "MEAL_PLAN",
					"error", err,
				)
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}

			refinedPrompt := body.Prompt
			if enableTavily {
				refinedPrompt = enrichWithTavily(tavilyKey, body.Prompt, body.IsBrainrot)
			}

			dietaryCtx := handlers.BuildDietaryContext(pool, email)
			mealPlan, err := handlers.GenerateMealPlan(refinedPrompt, dietaryCtx)
			if err != nil {
				logger.Error("meal_plan_generation_failed", "error", err, "user", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			if err := handlers.SaveMealPlan(pool, email, mealPlan, body.Prompt); err != nil {
				logger.Error("save_meal_plan_failed", "error", err, "user", email)
			}

			logger.Info("meal_plan_generated", "user", email, "days", len(mealPlan.Days))
			c.JSON(http.StatusOK, gin.H{"operation": "MEAL_PLAN", "data": mealPlan})
		})

		protected.GET("/meal-plan", handlers.GetLatestMealPlan(pool))
		protected.GET("/meal-plans", handlers.GetUserMealPlans(pool))

		favorites := protected.Group("/favorites")
		{
			favorites.POST("", handlers.AddToFavorites(pool))
			favorites.GET("", handlers.GetFavorites(pool))
			favorites.DELETE("/:id", handlers.RemoveFavorite(pool))
		}

		protected.GET("/recipes", handlers.GetUserRecipes(pool))
		protected.GET("/recipes/:id/export", handlers.ExportRecipe(pool))
		protected.DELETE("/recipes/:id", handlers.DeleteRecipe(pool))

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

			if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.substitute")); err != nil {
				logger.Warn("insufficient_balance",
					"user", email,
					"op", "SUBSTITUTE",
					"error", err,
				)
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}

			dietaryCtx := handlers.BuildDietaryContext(pool, email)
			result, err := handlers.GetSubstitutes(body.Ingredient, body.Reason, dietaryCtx)
			if err != nil {
				logger.Error("substitute_failed", "error", err, "user", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("substitute_generated", "user", email, "ingredient", body.Ingredient)
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

			if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.what_to_cook")); err != nil {
				logger.Warn("insufficient_balance",
					"user", email,
					"op", "WHAT_TO_COOK",
					"error", err,
				)
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}

			dietaryCtx := handlers.BuildDietaryContext(pool, email)
			result, err := handlers.WhatToCook(body.Ingredients, body.Preferences, dietaryCtx)
			if err != nil {
				logger.Error("what_to_cook_failed", "error", err, "user", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logger.Info("what_to_cook_generated", "user", email, "ingredients_count", len(body.Ingredients))
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

			if err := handlers.CheckBalance(pool, email, viper.GetInt("pricing.nutrition_log")); err != nil {
				logger.Warn("insufficient_balance",
					"user", email,
					"op", "NUTRITION_LOG",
					"error", err,
				)
				c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
				return
			}

			dietaryCtx := handlers.BuildDietaryContext(pool, email)
			result, err := handlers.AnalyzeNutritionLog(body.Meals, dietaryCtx)
			if err != nil {
				logger.Error("nutrition_log_failed", "error", err, "user", email)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			logID, err := handlers.SaveNutritionLog(pool, email, result)
			if err != nil {
				logger.Error("nutrition_log_save_failed", "error", err, "user", email)
			}

			logger.Info("nutrition_log_analyzed", "user", email, "meals", len(body.Meals))
			c.JSON(http.StatusOK, gin.H{
				"operation": "NUTRITION_LOG",
				"data":      result,
				"log_id":    logID,
			})
		})

		protected.GET("/nutrition-logs", handlers.GetNutritionLogs(pool))
		protected.GET("/nutrition-logs/today", handlers.GetNutritionLogToday(pool))
		protected.GET("/nutrition-logs/stats", handlers.GetNutritionStats(pool))
		protected.DELETE("/nutrition-logs/:id", handlers.DeleteNutritionLog(pool))

	}

	router.Run(":8080")
}
