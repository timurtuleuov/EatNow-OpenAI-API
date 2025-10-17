package model

type Ingredient struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Quantity     *float64 `json:"quantity,omitempty"`
	Unit         *string  `json:"unit,omitempty"`
	Prepared     *string  `json:"prepared,omitempty"`
	Optional     bool     `json:"optional"`
	OriginalText *string  `json:"original_text,omitempty"`
}