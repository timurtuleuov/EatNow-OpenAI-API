package model

type SubstituteSuggestion struct {
	Ingredient string `json:"ingredient"`
	Reason     string `json:"reason"`
	Ratio      string `json:"ratio"`
}

type SubstituteResponse struct {
	Original     string                  `json:"original"`
	Reason       string                  `json:"reason,omitempty"`
	Substitutes  []SubstituteSuggestion  `json:"substitutes"`
	Tip          string                  `json:"tip"`
}
