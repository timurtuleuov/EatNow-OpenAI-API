package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	model "openai/models"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

var apiKey = os.Getenv("OPENAI_API_KEY")

// 🍳 GetRecipeByPrompt — основная функция, обращается к GPT и возвращает структуру рецепта.
func GetRecipeByPrompt(prompt string) (*model.Recipe, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(2000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	// fmt.Println("🧾 RAW RESPONSE:", resp)

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := resp.Choices[0].Message.Content

	var recipe model.Recipe
	if err := json.Unmarshal([]byte(raw), &recipe); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	return &recipe, nil
}

// тип операции консультация
func Consult(prompt string) (*model.Consult, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPromptConsult),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(2000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := resp.Choices[0].Message.Content

	var consult model.Consult
	if err := json.Unmarshal([]byte(raw), &consult); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	return &consult, nil
}

// рецепт с фото
func GetRecipeFromPhoto(prompt, base64Image string) (*model.Recipe, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}
	imageUrl := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{

			openai.SystemMessage(systemPromptRecipeFromPhoto),
			openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(prompt),
					openai.ImageContentPart(
						openai.ChatCompletionContentPartImageImageURLParam{
							URL: imageUrl,
						}),
				},
			),
		},

		MaxCompletionTokens: openai.Int(2000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := resp.Choices[0].Message.Content

	var recipe model.Recipe
	if err := json.Unmarshal([]byte(raw), &recipe); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	return &recipe, nil
}

// рецепт с фото
func Calories(prompt, base64Image string) (*model.Calories, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	imageUrl := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini",
		Messages: []openai.ChatCompletionMessageParamUnion{

			openai.SystemMessage(systemPromptCalories),
			openai.UserMessage(
				[]openai.ChatCompletionContentPartUnionParam{
					openai.TextContentPart(prompt),
					openai.ImageContentPart(
						openai.ChatCompletionContentPartImageImageURLParam{
							URL: imageUrl,
						}),
				},
			),
		},

		MaxCompletionTokens: openai.Int(2000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)

	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	raw := resp.Choices[0].Message.Content

	var calorie model.Calories

	if err := json.Unmarshal([]byte(raw), &calorie); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	return &calorie, nil
}

// Определение AI операции. Варианты: GENERATE, CALORIES, RECIPE_PHOTO, CONSULT
func DetectAIOperation(prompt string, hasImage bool) (*string, error) {

	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))
	if !hasImage {
		params := openai.ChatCompletionNewParams{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPromptDetectOp),
				openai.UserMessage(prompt),
			},
			MaxCompletionTokens: openai.Int(2000),
		}

		resp, err := client.Chat.Completions.New(context.Background(), params)
		if err != nil {
			return nil, fmt.Errorf("OpenAI request failed: %w", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
			return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
		}

		answer := resp.Choices[0].Message.Content

		return &answer, nil
	} else {
		params := openai.ChatCompletionNewParams{
			Model: "gpt-4o-mini",
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.SystemMessage(systemPromptDetectOpWithImage),
				openai.UserMessage(prompt),
			},
			MaxCompletionTokens: openai.Int(2000),
		}

		resp, err := client.Chat.Completions.New(context.Background(), params)
		if err != nil {
			return nil, fmt.Errorf("OpenAI request failed: %w", err)
		}

		if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
			return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
		}

		answer := resp.Choices[0].Message.Content

		return &answer, nil

	}
}
