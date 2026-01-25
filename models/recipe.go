package model

type Recipe struct {
	ID               int                    `json:"id"`
	Title            string                 `json:"title"`
	Description      *string                `json:"description,omitempty"`
	Servings         int                    `json:"servings"`
	TotalTimeMinutes int                    `json:"total_time_minutes"`
	Difficulty       *Difficulty            `json:"difficulty,omitempty"`
	Ingredients      []Ingredient           `json:"ingredients"`
	Steps            []StepModel            `json:"steps"`
	Nutrition        map[string]interface{} `json:"nutrition,omitempty"`
	Tags             []string               `json:"tags,omitempty"`
	ImageURL         *string                `json:"image_url,omitempty"`
	Source           *string                `json:"source,omitempty"`
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)
