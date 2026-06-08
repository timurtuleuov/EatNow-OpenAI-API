package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	model "openai/models"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

func Detective(prompt, dietaryContext, style string, history []model.Message) (*model.DetectiveResponse, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	systemMsg := applyStyle("detect_dish", style)
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: buildMessages(systemMsg, history, openai.UserMessage(prompt)),
		MaxCompletionTokens: openai.Int(1000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := cleanJSON(resp.Choices[0].Message.Content)

	var result model.DetectiveResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		slog.Warn("detective_json_fallback",
			"error", err,
			"raw", raw,
		)
		result = model.DetectiveResponse{
			Message:    raw,
			Questions:  []string{},
			Hypothesis: "",
			Confidence: 0,
			EnoughInfo: false,
		}
	}

	return &result, nil
}
