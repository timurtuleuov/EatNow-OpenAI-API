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

// 🧠 systemPrompt — это инструкция для модели.
// Модель должна сгенерировать рецепт строго в JSON-формате по указанной структуре.
var systemPrompt = `
You are a professional chef and nutrition expert.
Generate complete cooking recipes in STRICT JSON format only.

### JSON SCHEMA:
{
  "id": 1,
  "title": "string",
  "description": "string (optional)",
  "servings": 2,
  "total_time_minutes": 30,
  "difficulty": "easy|medium|hard",
  "ingredients": [
    {
      "id": "1", "name": "string", "quantity": 100, "unit": "string", 
      "prepared": "string (optional)", "optional": false
    }
  ],
  "steps": [
    {
      "order": 1, "description": "string", 
      "duration_seconds": 60, "ingredients_used": ["1"]
    }
  ],
  "nutrition": {
    "calories": 250, "protein": "10g", "fat": "5g", "carbs": "40g"
  },
  "tags": ["tag1", "tag2"],
  "image_url": "",
  "source": ""
}

### CRITICAL RULES:
1. Output ONLY raw JSON - no code fences, no explanations, no additional text
2. Use English field names but content in user's language
3. All required fields must have meaningful values
4. Keep it realistic: 3-8 ingredients, 3-6 steps
5. If unsure, make reasonable assumptions
6. NEVER use markdown formatting
7. Ensure valid JSON syntax

### EXAMPLE OUTPUT:
{
  "id": 1,
  "title": "Spaghetti Carbonara",
  "description": "Classic Italian pasta dish",
  "servings": 2,
  "total_time_minutes": 20,
  "difficulty": "medium",
  "ingredients": [
    {"id": "1", "name": "spaghetti", "quantity": 200, "unit": "g", "optional": false},
    {"id": "2", "name": "eggs", "quantity": 2, "unit": "pieces", "optional": false},
    {"id": "3", "name": "bacon", "quantity": 100, "unit": "g", "optional": false}
  ],
  "steps": [
    {"order": 1, "description": "Cook spaghetti in boiling salted water"},
    {"order": 2, "description": "Fry bacon until crispy"},
    {"order": 3, "description": "Mix eggs with cheese and combine with pasta"}
  ],
  "nutrition": {"calories": 450},
  "tags": ["pasta", "italian"],
  "source": ""
}
`

// 🍳 GetRecipeByPrompt — основная функция, обращается к GPT и возвращает структуру рецепта.
func GetRecipeByPrompt(prompt string) (*model.Recipe, error) {
	// 1️⃣ Берём ключ API из окружения
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("environment variable OPENAI_API_KEY not set")
	}

	// 2️⃣ Создаём клиента OpenAI
	client := openai.NewClient(option.WithAPIKey(apiKey))

	// 3️⃣ Готовим параметры запроса
	params := openai.ChatCompletionNewParams{
		Model: "gpt-4o-mini", // ✅ gpt-5-mini — быстрая и дешёвая модель
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(systemPrompt),
			openai.UserMessage(prompt),
		},
		MaxCompletionTokens: openai.Int(2000), // ✅ корректный тип — *int64
	}

	// 4️⃣ Отправляем запрос
	resp, err := client.Chat.Completions.New(context.Background(), params)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	fmt.Println("🧾 RAW RESPONSE:", resp)

	// 5️⃣ Проверяем ответ
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	// 6️⃣ Извлекаем текст
	raw := resp.Choices[0].Message.Content // ✅ теперь это просто *string

	// 7️⃣ Парсим JSON
	var recipe model.Recipe
	if err := json.Unmarshal([]byte(raw), &recipe); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw output:\n%s", err, raw)
	}

	// 8️⃣ Подстраховка
	if recipe.ID == 0 {
		recipe.ID = 1
	}

	return &recipe, nil
}
