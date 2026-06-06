package model

type RateResponse struct {
	Rating   int      `json:"rating"`
	Review   string   `json:"review"`
	Rational string   `json:"rational"`
	MemeTags []string `json:"meme_tags"`
}
