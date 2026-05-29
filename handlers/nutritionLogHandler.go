package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	model "openai/models"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

func AnalyzeNutritionLog(meals []string, dietaryContext string) (*model.NutritionLogResponse, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	prompt := "Вот что я съел за день:\n" + strings.Join(meals, "\n")

	systemMsg := viper.GetString("prompts.nutrition_log")
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMsg),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(1500),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("пустой ответ от модели")
	}

	raw := resp.Choices[0].Message.Content

	var result model.NutritionLogResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw:\n%s", err, raw)
	}

	return &result, nil
}
