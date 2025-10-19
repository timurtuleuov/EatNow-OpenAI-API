package main

import (
	"context"
	"fmt"
	
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func openaiApi() {
	apiKeys := []string{
		"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",
	}

	for i := 0; i < len(apiKeys); i++ { // Исправлено: i < len(apiKeys)
		fmt.Printf("Попытка с ключом %d\n", i+1)

		client := openai.NewClient(
			option.WithAPIKey(apiKeys[i]),
		)

		chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("How about these"),
			},
			Model: openai.ChatModelGPT4o,
		})

		if err != nil {
			fmt.Printf("Ошибка с ключом %d: %v\n", i+1, err)
			continue // Переходим к следующему ключу вместо остановки
		}

		fmt.Printf("Успех с ключом %d: %s\n", i+1, chatCompletion.Choices[0].Message.Content)
	}

	fmt.Println("Все итерации завершены")
}

// curl https://api.openai.com/v1/models \
//   -H "Authorization: Bearer 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"
