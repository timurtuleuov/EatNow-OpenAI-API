package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"openai/internal/logger"
	"strings"
	"time"

	model "openai/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/spf13/viper"
)

type StructuredMemory struct {
	Summary             string   `json:"summary"`
	LikedCuisines       []string `json:"liked_cuisines"`
	FrequentIngredients []string `json:"frequent_ingredients"`
	DietStyle           string   `json:"diet_style"`
	SkillLevel          string   `json:"skill_level"`
	RecentDishes        []string `json:"recent_dishes"`
}

func BuildUserMemory(db *pgxpool.Pool, email string) string {
	var parts []string

	userID, err := getUserID(db, email)
	if err != nil {
		return ""
	}

	structured, err := getStructuredMemory(db, userID)
	if err == nil && structured != nil {
		parts = append(parts, formatStructuredMemory(structured))
	} else {
		summary, _ := getMemorySummary(db, userID)
		if summary != "" {
			parts = append(parts, "User memory: "+summary)
		}
	}

	recent, err := getRecentRecipes(db, email, 3)
	if err == nil && len(recent) > 0 {
		parts = append(parts, "Recently cooked: "+strings.Join(recent, ", "))
	}

	if len(parts) == 0 {
		return ""
	}

	return "\n--- USER PROFILE ---\n" + strings.Join(parts, "\n")
}

func formatStructuredMemory(m *StructuredMemory) string {
	var lines []string

	if len(m.LikedCuisines) > 0 {
		lines = append(lines, "Likes cuisine: "+strings.Join(m.LikedCuisines, ", "))
	}
	if len(m.FrequentIngredients) > 0 {
		lines = append(lines, "Often cooks with: "+strings.Join(m.FrequentIngredients, ", "))
	}
	if m.DietStyle != "" {
		lines = append(lines, "Diet style: "+m.DietStyle)
	}
	if m.SkillLevel != "" {
		lines = append(lines, "Skill level: "+m.SkillLevel)
	}
	if len(m.RecentDishes) > 0 {
		lines = append(lines, "Recent dishes: "+strings.Join(m.RecentDishes, ", "))
	}
	if m.Summary != "" {
		lines = append(lines, "Memory: "+m.Summary)
	}

	return strings.Join(lines, "\n")
}

func getUserID(db *pgxpool.Pool, email string) (string, error) {
	var userID string
	err := db.QueryRow(context.Background(),
		"SELECT id::text FROM users WHERE email = $1", email,
	).Scan(&userID)
	return userID, err
}

func getMemorySummary(db *pgxpool.Pool, userID string) (string, error) {
	var summary string
	err := db.QueryRow(context.Background(),
		"SELECT COALESCE(summary, '') FROM user_memories WHERE user_id = $1::uuid",
		userID,
	).Scan(&summary)
	if err != nil {
		return "", err
	}
	return summary, nil
}

func getStructuredMemory(db *pgxpool.Pool, userID string) (*StructuredMemory, error) {
	var raw []byte
	err := db.QueryRow(context.Background(),
		"SELECT COALESCE(structured, '{}'::jsonb) FROM user_memories WHERE user_id = $1::uuid",
		userID,
	).Scan(&raw)
	if err != nil {
		return nil, err
	}
	// Проверяем что это не пустой объект
	var m StructuredMemory
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	if m.Summary == "" && len(m.LikedCuisines) == 0 {
		return nil, nil
	}
	return &m, nil
}

func getRecentRecipes(db *pgxpool.Pool, email string, limit int) ([]string, error) {
	rows, err := db.Query(context.Background(), `
		SELECT r.recipe->>'title'
		FROM recipes r
		JOIN users u ON r.user_id = u.id
		WHERE u.email = $1 AND r.recipe->>'title' IS NOT NULL
		ORDER BY r.created_at DESC
		LIMIT $2
	`, email, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			continue
		}
		titles = append(titles, title)
	}
	return titles, nil
}

func RecipeFromDB(db *pgxpool.Pool, recipeID string) (*model.Recipe, error) {
	var recipeJSON []byte
	err := db.QueryRow(context.Background(),
		"SELECT recipe FROM recipes WHERE id = $1::uuid", recipeID,
	).Scan(&recipeJSON)
	if err != nil {
		return nil, err
	}
	var recipe model.Recipe
	if err := json.Unmarshal(recipeJSON, &recipe); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w\nRaw: %s", err, string(recipeJSON))
	}
	return &recipe, nil
}

func UpdateUserMemory(db *pgxpool.Pool, email string, recipe *model.Recipe) {
	updateUserMemoryWithAction(db, email, recipe, "")
}

func updateUserMemoryWithAction(db *pgxpool.Pool, email string, recipe *model.Recipe, action string) {
	if recipe == nil || recipe.Title == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, err := getUserID(db, email)
	if err != nil {
		slog.Error("memory_update_user_not_found", "error", err, "user", email)
		return
	}

	// Текущая структура памяти
	currentStructured := &StructuredMemory{}
	raw, _ := func() ([]byte, error) {
		var raw []byte
		err := db.QueryRow(ctx,
			"SELECT COALESCE(structured, '{}'::jsonb) FROM user_memories WHERE user_id = $1::uuid",
			userID,
		).Scan(&raw)
		return raw, err
	}()
	if raw != nil {
		json.Unmarshal(raw, currentStructured)
	}

	apiKey := viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	currentJSON, _ := json.Marshal(currentStructured)
	tags := strings.Join(recipe.Tags, ", ")

	actionHint := ""
	if action == "favorite" {
		actionHint = " (user LOVED this recipe and saved it to favorites)"
	} else if action == "unfavorite" {
		actionHint = " (user REMOVED this recipe from favorites — they didn't like it enough)"
	}

	msg := fmt.Sprintf(
		`Current memory (JSON): %s
New recipe: "%s"%s%s

Update the structured memory JSON. Return ONLY valid JSON with these fields:
{
  "summary": "1-2 sentence summary in user's language",
  "liked_cuisines": ["cuisine names"],
  "frequent_ingredients": ["ingredients commonly used"],
  "diet_style": "diet preference or empty string",
  "skill_level": "beginner|intermediate|advanced",
  "recent_dishes": ["last 5 dish names with newest first"]
}

Rules:
- Keep liked_cuisines up to 3 entries
- Keep frequent_ingredients up to 5 entries
- recent_dishes: max 5, most recent first
- If action is "unfavorite", consider removing from liked lists
- summary must be in the user's language`,
		string(currentJSON),
		recipe.Title,
		mapBool(tags != "", ", Tags: "+tags),
		actionHint,
	)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a user profiling assistant. Update structured memory JSON based on new activity."),
			openai.UserMessage(msg),
		},
		MaxCompletionTokens: openai.Int(300),
	})
	if err != nil {
		slog.Error("memory_update_ai_failed", "error", err, "user", email)
		return
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return
	}

	raw = []byte(strings.TrimSpace(resp.Choices[0].Message.Content))
	var updated StructuredMemory
	if err := json.Unmarshal(raw, &updated); err != nil {
		slog.Error("memory_parse_failed", "error", err, "user", email)
		return
	}

	_, err = db.Exec(ctx, `
		INSERT INTO user_memories (user_id, summary, structured, updated_at)
		VALUES ($1::uuid, $2, $3, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			summary = $2,
			structured = $3,
			updated_at = NOW()
	`, userID, updated.Summary, raw)
	if err != nil {
		slog.Error("memory_update_save_failed", "error", err, "user", email)
		return
	}

	slog.Info("memory_updated",
		logger.KeyUser, email,
		"recipe", recipe.Title,
		"action", action,
	)
}

func mapBool(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}
