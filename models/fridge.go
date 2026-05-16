package model

type FridgeRecipe struct {
	Title       string `json:"title"`
	Match       int    `json:"match_percent"`
	Description string `json:"description"`
	TotalTime   int    `json:"total_time_minutes"`
	Difficulty  string `json:"difficulty"`
	Missing     []string `json:"missing_ingredients"`
}

type WhatToCookResponse struct {
	Recipes []FridgeRecipe `json:"recipes"`
	Tip     string         `json:"tip"`
}
