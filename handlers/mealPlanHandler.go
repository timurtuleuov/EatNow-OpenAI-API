package handlers

import (
	"context"
	"encoding/json"
	"fmt"

	model "openai/models"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

func GenerateMealPlan(prompt string) (*model.MealPlan, error) {
	apiKey := viper.GetString("openai.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("openai.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(viper.GetString("prompts.meal_plan")),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(4000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := resp.Choices[0].Message.Content

	var mealPlan model.MealPlan
	if err := json.Unmarshal([]byte(raw), &mealPlan); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	return &mealPlan, nil
}
