package model

import "time"

type DietaryProfile struct {
	DietType           string   `json:"diet_type"`
	Allergies          []string `json:"allergies"`
	ExcludedIngredients []string `json:"excluded_ingredients"`
	CuisinePreferences []string `json:"cuisine_preferences"`
	DailyCalorieGoal   int      `json:"daily_calorie_goal"`
	DailyProteinGoal   float64  `json:"daily_protein_goal"`
	DailyFatGoal       float64  `json:"daily_fat_goal"`
	DailyCarbsGoal     float64  `json:"daily_carbs_goal"`
}

type User struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Password       string     `json:"-"`
	Email          string     `json:"email"`
	DeviceID       string     `json:"device_id"`
	Platform       string     `json:"platform"`
	IsPremium      bool       `json:"is_premium"`
	PremiumExpires *time.Time `json:"premium_expires"`
	Balance        int        `json:"balance"`
	BalanceResetAt time.Time  `json:"balance_reset_at"`
	DietaryProfile
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
