package handlers

import model "openai/models"

// Inline request types for Swagger documentation
type (
	RegisterRequest struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Platform string `json:"platform,omitempty"`
		DeviceID string `json:"device_id,omitempty"`
	}

	LoginRequest struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	RefreshRequest struct {
		Email        string `json:"email"`
		RefreshToken string `json:"refresh_token"`
	}

	RecipeRequest struct {
		DeviceID   string          `json:"device_id"`
		Prompt     string          `json:"prompt"`
		History    []model.Message `json:"history,omitempty"`
		Image      string          `json:"image,omitempty"`
		IsBrainrot bool            `json:"is_brainrot,omitempty"`
	}

	MealPlanRequest struct {
		Prompt     string          `json:"prompt"`
		History    []model.Message `json:"history,omitempty"`
		IsBrainrot bool            `json:"is_brainrot,omitempty"`
	}

	SubstituteRequest struct {
		Ingredient string `json:"ingredient"`
		Reason     string `json:"reason,omitempty"`
	}

	WhatToCookRequest struct {
		Ingredients []string `json:"ingredients"`
		Preferences string   `json:"preferences,omitempty"`
	}

	NutritionLogRequest struct {
		Meals []string `json:"meals"`
	}

	FavoriteRequest struct {
		RecipeID string `json:"recipe_id"`
	}
)

// Auth endpoints

// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "Registration details"
// @Success 200 {object} object{access_token=string,refresh_token=string}
// @Failure 400 {object} object{error=string}
// @Router /auth/register [post]
func _auth_register() {}

// @Summary Login
// @Description Authenticate with email and password
// @Tags auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} object{access_token=string,refresh_token=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /auth/login [post]
func _auth_login() {}

// @Summary Refresh access token
// @Description Get a new access token using a refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "Refresh token details"
// @Success 200 {object} object{access_token=string}
// @Failure 400 {object} object{error=string}
// @Failure 401 {object} object{error=string}
// @Router /auth/refresh [post]
func _auth_refresh() {}

// User endpoints

// @Summary Home
// @Description Simple health check
// @Tags user
// @Produce json
// @Success 202 {object} object{msg=string}
// @Router /api/home [get]
// @Security BearerAuth
func _api_home() {}

// @Summary Get current user
// @Description Returns email and premium status
// @Tags user
// @Produce json
// @Success 200 {object} object{email=string,is_premium=bool}
// @Failure 404 {object} object{error=string}
// @Router /api/me [get]
// @Security BearerAuth
func _api_me() {}

// @Summary Get profile with dietary preferences
// @Description Returns full profile including dietary preferences and nutrition goals
// @Tags user
// @Produce json
// @Success 200 {object} object{email=string,is_premium=bool,profile=model.DietaryProfile}
// @Failure 404 {object} object{error=string}
// @Router /api/me/profile [get]
// @Security BearerAuth
func _api_me_profile_get() {}

// @Summary Update dietary preferences
// @Description Update dietary profile, allergies, excluded ingredients, cuisine preferences, and nutrition goals
// @Tags user
// @Accept json
// @Produce json
// @Param request body model.DietaryProfile true "Dietary preferences"
// @Success 200 {object} object{status=string,profile=model.DietaryProfile}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/me/profile [put]
// @Security BearerAuth
func _api_me_profile_put() {}

// Recipe endpoints

// @Summary Generate recipe or get AI response
// @Description Main AI endpoint. Auto-detects operation: GENERATE (text recipe), CONSULT (culinary advice), CALORIES (nutrition estimation), RECIPE_PHOTO (recipe from photo). Saves recipe to history on GENERATE and RECIPE_PHOTO.
// @Tags recipes
// @Accept json
// @Produce json
// @Param request body RecipeRequest true "Recipe generation request"
// @Success 200 {object} object{operation=string,data=model.Recipe}
// @Failure 400 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/recipe [post]
// @Security BearerAuth
func _api_recipe_post() {}

// @Summary Get free bonus
// @Description Grant bonus tokens for watching an ad
// @Tags recipes
// @Produce json
// @Success 200 {object} object{status=string}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/recipe/get-free [post]
// @Security BearerAuth
func _api_recipe_get_free() {}

// @Summary Get user's prompt count / balance
// @Description Returns the current balance of the authenticated user
// @Tags user
// @Produce json
// @Success 200 {object} object{userFreePromptsCount=int}
// @Failure 400 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/user/prompts-count [get]
// @Security BearerAuth
func _api_user_prompts_count() {}

// Payments

// @Summary Verify Google Play purchase
// @Description Verify a Google Play subscription purchase and activate premium
// @Tags payments
// @Accept json
// @Produce json
// @Param request body GoogleVerifyRequest true "Google Play purchase verification"
// @Success 200 {object} object{status=string,message=string,expires_at=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/payments/verify-google [post]
// @Security BearerAuth
func _api_payments_verify_google() {}

// Meal Plan endpoints

// @Summary Generate weekly meal plan
// @Description Generate a 7-day meal plan with breakfast, lunch, dinner, and snack for each day
// @Tags meal-plans
// @Accept json
// @Produce json
// @Param request body MealPlanRequest true "Meal plan request"
// @Success 200 {object} object{operation=string,data=model.MealPlan}
// @Failure 400 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/meal-plan [post]
// @Security BearerAuth
func _api_meal_plan_post() {}

// @Summary Get latest meal plan
// @Description Get the most recently generated meal plan
// @Tags meal-plans
// @Produce json
// @Success 200 {object} object{operation=string,data=model.MealPlan,prompt=string,created_at=string}
// @Failure 404 {object} object{error=string}
// @Router /api/meal-plan [get]
// @Security BearerAuth
func _api_meal_plan_get() {}

// @Summary Get all meal plans
// @Description Get all meal plans for the authenticated user, ordered by creation date DESC
// @Tags meal-plans
// @Produce json
// @Success 200 {object} object{meal_plans=[]MealPlanRecord}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/meal-plans [get]
// @Security BearerAuth
func _api_meal_plans_get() {}

// Favorites endpoints

// @Summary Add recipe to favorites
// @Description Save a recipe to the user's favorites list
// @Tags favorites
// @Accept json
// @Produce json
// @Param request body FavoriteRequest true "Recipe ID to favorite"
// @Success 201 {object} object{id=int,status=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/favorites [post]
// @Security BearerAuth
func _api_favorites_post() {}

// @Summary Get favorites
// @Description Get all favorited recipes for the authenticated user
// @Tags favorites
// @Produce json
// @Success 200 {object} object{favorites=[]model.Favorite}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/favorites [get]
// @Security BearerAuth
func _api_favorites_get() {}

// @Summary Remove favorite
// @Description Remove a recipe from favorites by favorite record ID
// @Tags favorites
// @Produce json
// @Param id path int true "Favorite record ID"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/favorites/{id} [delete]
// @Security BearerAuth
func _api_favorites_delete() {}

// User Recipes

// @Summary Get user's recipes
// @Description Get all previously generated recipes for the authenticated user
// @Tags recipes
// @Produce json
// @Success 200 {object} object{recipes=[]UserRecipeItem}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/recipes [get]
// @Security BearerAuth
func _api_recipes_get() {}

// @Summary Delete a recipe
// @Description Delete a generated recipe by its UUID
// @Tags recipes
// @Produce json
// @Param id path string true "Recipe UUID"
// @Success 200 {object} object{status=string}
// @Failure 400 {object} object{error=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/recipes/{id} [delete]
// @Security BearerAuth
func _api_recipes_delete() {}

// Substitute

// @Summary Find ingredient substitutes
// @Description Find alternative ingredients for a given ingredient, optionally with a reason
// @Tags substitute
// @Accept json
// @Produce json
// @Param request body SubstituteRequest true "Ingredient substitution request"
// @Success 200 {object} object{operation=string,data=model.SubstituteResponse}
// @Failure 400 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/substitute [post]
// @Security BearerAuth
func _api_substitute() {}

// What to Cook / Fridge

// @Summary Find recipes from available ingredients
// @Description Suggest recipes based on ingredients the user has available
// @Tags what-to-cook
// @Accept json
// @Produce json
// @Param request body WhatToCookRequest true "Available ingredients"
// @Success 200 {object} object{operation=string,data=model.WhatToCookResponse}
// @Failure 400 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/what-to-cook [post]
// @Security BearerAuth
func _api_what_to_cook() {}

// Nutrition Log

// @Summary Analyze daily nutrition
// @Description Analyze a list of meals consumed during the day, returns nutrition breakdown and health score. Automatically saves to the user's nutrition log.
// @Tags nutrition
// @Accept json
// @Produce json
// @Param request body NutritionLogRequest true "Meals consumed"
// @Success 200 {object} object{operation=string,data=model.NutritionLogResponse,log_id=string}
// @Failure 400 {object} object{error=string}
// @Failure 403 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/nutrition-log [post]
// @Security BearerAuth
func _api_nutrition_log_post() {}

// @Summary Get nutrition logs
// @Description Get nutrition logs, optionally filtered by date
// @Tags nutrition
// @Produce json
// @Param date query string false "Filter by date (YYYY-MM-DD)"
// @Success 200 {object} object{logs=[]model.NutritionLog}
// @Failure 404 {object} object{error=string}
// @Router /api/nutrition-logs [get]
// @Security BearerAuth
func _api_nutrition_logs_get() {}

// @Summary Get today's nutrition log
// @Description Get the nutrition log for the current date, or null if none exists
// @Tags nutrition
// @Produce json
// @Success 200 {object} object{log=model.NutritionLog,date=string}
// @Router /api/nutrition-logs/today [get]
// @Security BearerAuth
func _api_nutrition_logs_today() {}

// @Summary Get nutrition statistics
// @Description Get aggregated nutrition statistics for the last week or month
// @Tags nutrition
// @Produce json
// @Param period query string false "Period: week or month" Enums(week, month)
// @Success 200 {object} object{stats=model.NutritionStats}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/nutrition-logs/stats [get]
// @Security BearerAuth
func _api_nutrition_logs_stats() {}

// @Summary Delete nutrition log
// @Description Delete a nutrition log entry by its UUID
// @Tags nutrition
// @Produce json
// @Param id path string true "Nutrition log UUID"
// @Success 200 {object} object{status=string}
// @Failure 404 {object} object{error=string}
// @Failure 500 {object} object{error=string}
// @Router /api/nutrition-logs/{id} [delete]
// @Security BearerAuth
func _api_nutrition_logs_delete() {}

// @Summary Forgot password
// @Description Send a password reset link to the user's email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{email=string} true "User email"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Router /auth/forgot-password [post]
func _auth_forgot_password() {}

// @Summary Reset password
// @Description Reset password using a token received by email
// @Tags auth
// @Accept json
// @Produce json
// @Param request body object{token=string,new_password=string} true "Reset token and new password"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} object{error=string}
// @Router /auth/reset-password [post]
func _auth_reset_password() {}

// @Summary Export recipe as text
// @Description Export a recipe as plain text format
// @Tags recipes
// @Produce text/plain
// @Param id path string true "Recipe UUID"
// @Success 200 {string} string "Plain text recipe"
// @Failure 404 {object} object{error=string}
// @Router /api/recipes/{id}/export [get]
// @Security BearerAuth
func _api_recipes_export() {}
