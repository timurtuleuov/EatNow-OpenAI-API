package model

type DetectiveResponse struct {
	Message    string   `json:"message"`
	Questions  []string `json:"questions"`
	Hypothesis string   `json:"hypothesis"`
	Confidence int      `json:"confidence"`
	EnoughInfo bool     `json:"enough_info"`
}
