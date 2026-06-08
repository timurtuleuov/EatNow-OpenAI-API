package model

import (
	"encoding/json"
	"strings"
)

type RateResponse struct {
	Rating   int      `json:"rating"`
	Review   string   `json:"review"`
	Rational string   `json:"rational"`
	MemeTags []string `json:"meme_tags"`
}

func (r *RateResponse) UnmarshalJSON(data []byte) error {
	type Alias RateResponse
	aux := &struct {
		MemeTags json.RawMessage `json:"meme_tags"`
		*Alias
	}{Alias: (*Alias)(r)}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if len(aux.MemeTags) > 0 && aux.MemeTags[0] == '"' {
		var s string
		if err := json.Unmarshal(aux.MemeTags, &s); err == nil {
			r.MemeTags = strings.Split(s, ",")
			for i := range r.MemeTags {
				r.MemeTags[i] = strings.TrimSpace(r.MemeTags[i])
			}
		}
	} else if len(aux.MemeTags) > 0 {
		json.Unmarshal(aux.MemeTags, &r.MemeTags)
	}
	return nil
}
