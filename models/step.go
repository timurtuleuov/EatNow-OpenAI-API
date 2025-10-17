package model

type StepModel struct {
	Order           int       `json:"order"`
	Title           *string   `json:"title,omitempty"`
	Description     string    `json:"description"`
	DurationSeconds *int      `json:"duration_seconds,omitempty"`
	TemperatureC    *int      `json:"temperature_c,omitempty"`
	IngredientsUsed []string  `json:"ingredients_used,omitempty"`
	MediaURL        *string   `json:"media_url,omitempty"`
	Tip             *string   `json:"tip,omitempty"`
}