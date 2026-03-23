package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/spf13/viper"
	"google.golang.org/genai"
)

type ImageGenerator struct {
	client *genai.Client
}

func NewImageGenerator(ctx context.Context) (*ImageGenerator, error) {
	// Инициализируем один раз при старте
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: viper.GetString("openai.gemini_api_key"),
	})
	if err != nil {
		return nil, err
	}
	return &ImageGenerator{client: client}, nil
}

func (ig *ImageGenerator) GenerateGeminiImage(ctx context.Context, prompt string) (string, error) {
	dir := "./images"

	// Используем переданный ctx, чтобы запрос можно было отменить
	result, err := ig.client.Models.GenerateContent(
		ctx,
		"gemini-3.1-flash-image-preview",
		genai.Text(prompt),
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	if len(result.Candidates) == 0 {
		return "", fmt.Errorf("no candidates returned")
	}

	filename := uuid.New().String() + ".png"
	filePath := filepath.Join(dir, filename)

	for _, part := range result.Candidates[0].Content.Parts {
		if part.InlineData != nil {
			imageBytes := part.InlineData.Data
			if err := os.WriteFile(filePath, imageBytes, 0644); err != nil {
				return "", fmt.Errorf("failed to save file: %w", err)
			}
			return filename, nil
		}
	}

	return "", fmt.Errorf("no image data in response")
}
