package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	model "openai/models"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"

	"github.com/spf13/viper"
)

// 🍳 GetRecipeByPrompt — основная функция, обращается к GPT и возвращает структуру рецепта.
func GetRecipeByPrompt(prompt, dietaryContext string) (*model.Recipe, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	systemMsg := viper.GetString("prompts.generate_recipe")
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMsg),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(2000),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("DeepSeek request failed: %w", err)
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
func Consult(prompt, dietaryContext string) (*model.Consult, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	systemMsg := viper.GetString("prompts.consult")
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemMsg),
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
func GetRecipeFromPhoto(prompt, base64Image, dietaryContext string) (*model.Recipe, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)
	imageUrl := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)

	systemMsg := viper.GetString("prompts.recipe_from_photo")
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{

			openai.SystemMessage(systemMsg),
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
func Calories(prompt, base64Image, dietaryContext string) (*model.Calories, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	imageUrl := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)

	systemMsg := viper.GetString("prompts.calories_estimation")
	if dietaryContext != "" {
		systemMsg = dietaryContext + "\n\n" + systemMsg
	}

	params := openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{

			openai.SystemMessage(systemMsg),
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
func DetectAIOperation(prompt string, history []model.Message, hasImage bool, base64Image string) (string, string, error) {
	var apiKey = viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return "", "", fmt.Errorf("environment variable DEEPSEEK_API_KEY not set")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	// 1. Выбираем базовый системный промпт
	systemPromptKey := "prompts.detect_operation"
	if hasImage {
		systemPromptKey = "prompts.detect_operation_image"
	}

	// 2. Формируем массив сообщений
	var messages []openai.ChatCompletionMessageParamUnion
	messages = append(messages, openai.SystemMessage(viper.GetString(systemPromptKey)))

	// Добавляем историю (History)
	for _, msg := range history {
		if msg.Role == "user" {
			messages = append(messages, openai.UserMessage(msg.Content))
		} else {
			messages = append(messages, openai.AssistantMessage(msg.Content))
		}
	}

	// 3. Формируем текущее сообщение пользователя (с картинкой или без)
	if hasImage && base64Image != "" {
		imageUrl := fmt.Sprintf("data:image/jpeg;base64,%s", base64Image)
		messages = append(messages, openai.UserMessage(
			[]openai.ChatCompletionContentPartUnionParam{
				openai.TextContentPart(prompt),
				openai.ImageContentPart(
					openai.ChatCompletionContentPartImageImageURLParam{
						URL: imageUrl,
					}),
			},
		))
	} else {
		messages = append(messages, openai.UserMessage(prompt))
	}

	// 4. Запрос к модели
	params := openai.ChatCompletionNewParams{
		Model:    viper.GetString("deepseek.model"),
		Messages: messages,

		MaxCompletionTokens: openai.Int(100),
	}

	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return "", "", fmt.Errorf("OpenAI request failed: %w", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return "", "", fmt.Errorf("пустой ответ от модели")
	}

	rawAnswer := resp.Choices[0].Message.Content

	// 5. Парсим ответ формата "ОПЕРАЦИЯ|УТОЧНЕННЫЙ_ПРОМПТ"
	parts := strings.SplitN(rawAnswer, "|", 2)

	operation := strings.TrimSpace(parts[0])
	refinedPrompt := prompt

	if len(parts) > 1 {
		refinedPrompt = strings.TrimSpace(parts[1])
	}

	return operation, refinedPrompt, nil
}

func GenerateImage(prompt string) (string, error) {
	apiKey := viper.GetString("openai.api_key")
	if apiKey == "" {
		return "", fmt.Errorf("OPENAI_API_KEY not set")
	}

	client := openai.NewClient(option.WithAPIKey(apiKey))

	resp, err := client.Images.Generate(context.TODO(), openai.ImageGenerateParams{
		Model:   "gpt-image-1.5",
		Prompt:  prompt,
		Size:    openai.ImageGenerateParamsSize1024x1024,
		Quality: openai.ImageGenerateParamsQualityMedium,
	})
	if err != nil {
		return "", err
	}

	// 👉 GPT модель возвращает base64
	b64 := resp.Data[0].B64JSON

	return b64, nil
}

func SaveImage(base64Str string) (string, error) {
	dir := "./images"

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return "", err
	}

	// Генерируем чистое имя файла
	filename := uuid.New().String() + ".png"

	// Полный путь используем ТОЛЬКО для записи на диск
	filePath := filepath.Join(dir, filename)

	data, err := base64.StdEncoding.DecodeString(base64Str)
	if err != nil {
		return "", err
	}

	// Сохраняем по полному пути, но возвращаем только filename
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return "", err
	}

	return filename, nil // <-- Возвращаем только имя "uuid.png"
}
