package model

import "time"

type MealEntry struct {
	Name        string  `json:"name"`
	Calories    float64 `json:"calories"`
	Protein     float64 `json:"protein_g"`
	Fat         float64 `json:"fat_g"`
	Carbs       float64 `json:"carbs_g"`
}

type NutritionLogResponse struct {
	Meals      []MealEntry `json:"meals"`
	Total      MealEntry   `json:"total"`
	WaterGL    float64     `json:"water_gl"`
	HealthScore int        `json:"health_score"`
	Analysis   string      `json:"analysis"`
	Tips       []string    `json:"tips"`
}

type NutritionLog struct {
	ID          string                `json:"id"`
	UserID      string                `json:"user_id,omitempty"`
	LogDate     string                `json:"log_date"`
	Meals       []MealEntry           `json:"meals"`
	Total       MealEntry             `json:"total"`
	WaterGL     float64               `json:"water_gl"`
	HealthScore int                   `json:"health_score"`
	Analysis    string                `json:"analysis"`
	Tips        []string              `json:"tips"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

type NutritionStats struct {
	Period          string     `json:"period"`
	AverageCalories float64    `json:"average_calories"`
	AverageProtein  float64    `json:"average_protein"`
	AverageFat      float64    `json:"average_fat"`
	AverageCarbs    float64    `json:"average_carbs"`
	AverageHealthScore float64 `json:"average_health_score"`
	LogCount        int        `json:"log_count"`
}
