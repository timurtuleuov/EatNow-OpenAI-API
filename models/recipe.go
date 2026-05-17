package model

import (
	"encoding/json"
	"fmt"
)

type Recipe struct {
	ID               string                 `json:"id"`
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

func (r *Recipe) UnmarshalJSON(data []byte) error {
	type Alias Recipe
	aux := &struct {
		ID json.RawMessage `json:"id"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if len(aux.ID) == 0 || string(aux.ID) == "null" {
		return nil
	}

	var s string
	if err := json.Unmarshal(aux.ID, &s); err == nil {
		r.ID = s
		return nil
	}

	var n int
	if err := json.Unmarshal(aux.ID, &n); err == nil {
		r.ID = fmt.Sprintf("%d", n)
		return nil
	}

	var f float64
	if err := json.Unmarshal(aux.ID, &f); err == nil {
		r.ID = fmt.Sprintf("%.0f", f)
		return nil
	}

	return nil
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)
