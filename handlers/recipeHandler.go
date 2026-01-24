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
1. Output ONLY raw JSON - no code fences, no explanations, no additional text.
2. Use English field names, but ALL content (values) and units of measurement MUST be in the user's language.
3. ADAPTATION: If the user provides imaginary or nonsensical ingredients, transform them into the most plausible real-world culinary equivalents.
4. REALISM: Keep recipes practical and grounded. Use 3-8 ingredients.
5. STEPS: Cooking steps must be highly detailed, realistic, and logically ordered. Minimum 4 steps.
6. NEVER use markdown formatting.
7. Ensure valid JSON syntax.

### EXAMPLE OUTPUT (User asked in Russian):
{
  "id": 1,
  "title": "Классическая Карбонара",
  "description": "Традиционная итальянская паста с нежным соусом",
  "servings": 2,
  "total_time_minutes": 20,
  "difficulty": "medium",
  "ingredients": [
    {"id": "1", "name": "спагетти", "quantity": 200, "unit": "г", "optional": false},
    {"id": "2", "name": "яйца", "quantity": 2, "unit": "шт", "optional": false},
    {"id": "3", "name": "бекон или панчетта", "quantity": 100, "unit": "г", "optional": false}
  ],
  "steps": [
    {"order": 1, "description": "Поставьте кастрюлю с 2 литрами воды на сильный огонь, доведите до кипения и добавьте щепотку соли. Опустите спагетти и варите до состояния аль-денте (на 1-2 минуты меньше, чем указано на упаковке).", "duration_seconds": 600, "ingredients_used": ["1"]},
    {"order": 2, "description": "Пока варится паста, нарежьте бекон мелкими кубиками. Разогрейте сковороду на среднем огне и обжаривайте бекон 5-7 минут до золотистой корочки, чтобы вытопился жир.", "duration_seconds": 420, "ingredients_used": ["3"]},
    {"order": 3, "description": "В отдельной миске взбейте два яйца вилкой до однородности. Добавьте немного тертого сыра (если есть) и молотый черный перец. Тщательно перемешайте.", "duration_seconds": 120, "ingredients_used": ["2"]},
    {"order": 4, "description": "Откиньте пасту на дуршлаг, сохранив немного воды от варки. Переложите пасту в сковороду к бекону, снимите с огня и быстро влейте яичную смесь, интенсивно помешивая, чтобы яйца превратились в кремовый соус, а не в омлет.", "duration_seconds": 180, "ingredients_used": ["1", "2", "3"]}
  ],
  "nutrition": {"calories": 450, "protein": "15г", "fat": "22г", "carbs": "48г"},
  "tags": ["паста", "итальянская кухня", "ужин"],
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
	// fmt.Println("🧾 RAW RESPONSE:", resp)

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

	return &recipe, nil
}

// тип операции консультация
func Consult(prompt string) (*string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
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

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("модель не сгенерировала контент (пустой ответ)")
	}

	answer := resp.Choices[0].Message.Content

	return &answer, nil
}
