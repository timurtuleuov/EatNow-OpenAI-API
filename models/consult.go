package model

type Consult struct {
	Text        string   `json:"text"`
	Suggestions []string `json:"suggestions"`
	Tip         *string  `json:"tip,omitempty"`
}
