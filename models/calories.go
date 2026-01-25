package model

type Calories struct {
	FoodName         string   `json:"food_name"`
	EstimatedWeightG int      `json:"estimated_weight_g"`
	Calories         int      `json:"calories"`
	Protein          float64  `json:"protein"`
	Fat              float64  `json:"fat"`
	Carbs            float64  `json:"carbs"`
	Analysis         string   `json:"analysis"`
	HealthRating     int      `json:"health_rating"`
	Suggestions      []string `json:"suggestions"`
}
