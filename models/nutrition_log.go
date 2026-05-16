package model

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
	WaterGL    float64      `json:"water_gl"`
	HealthScore int        `json:"health_score"`
	Analysis   string      `json:"analysis"`
	Tips       []string    `json:"tips"`
}
