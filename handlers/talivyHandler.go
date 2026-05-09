package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	model "openai/models" // Замени "eatnow" на название своего модуля из go.mod
)

// SearchTavily выполняет запрос к поисковому API
func SearchTavily(apiKey string, query string, isBrainrot bool) (*model.TavilyResponse, error) {
	url := "https://api.tavily.com/search"

	// Если включен brainrot, модифицируем поисковый запрос для поиска странного контента
	searchQuery := query
	if isBrainrot {
		searchQuery = "absurd weird cursed recipe " + query
	}

	reqBody := model.TavilySearchRequest{
		APIKey:      apiKey,
		Query:       searchQuery,
		SearchDepth: "basic",
		MaxResults:  3, // Ограничим количество для экономии токенов LLM
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Устанавливаем таймаут, чтобы поиск не вешал весь сервер
	client := &http.Client{Timeout: 15 * time.Second}

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("tavily request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tavily api returned status: %d", resp.StatusCode)
	}

	var tavilyResp model.TavilyResponse
	if err := json.NewDecoder(resp.Body).Decode(&tavilyResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tavilyResp, nil
}
