package handlers

import (
	"context"
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

func BuildUserMemory(db *pgxpool.Pool, email string) string {
	var parts []string

	summary, err := getMemorySummary(db, email)
	if err == nil && summary != "" {
		parts = append(parts, "User memory: "+summary)
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

func getMemorySummary(db *pgxpool.Pool, email string) (string, error) {
	var userID string
	err := db.QueryRow(context.Background(),
		"SELECT id::text FROM users WHERE email = $1", email,
	).Scan(&userID)
	if err != nil {
		return "", err
	}

	var summary string
	err = db.QueryRow(context.Background(),
		"SELECT COALESCE(summary, '') FROM user_memories WHERE user_id = $1::uuid",
		userID,
	).Scan(&summary)
	if err != nil {
		return "", err
	}
	return summary, nil
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

func UpdateUserMemory(db *pgxpool.Pool, email string, recipe *model.Recipe) {
	if recipe == nil || recipe.Title == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var userID string
	err := db.QueryRow(ctx,
		"SELECT id::text FROM users WHERE email = $1", email,
	).Scan(&userID)
	if err != nil {
		slog.Error("memory_update_user_not_found", "error", err, "user", email)
		return
	}

	currentSummary, _ := getMemorySummary(db, email)

	apiKey := viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.deepseek.com/"),
	)

	tags := strings.Join(recipe.Tags, ", ")

	msg := fmt.Sprintf(
		`Current memory: "%s"
New recipe: "%s"%s

Update the memory (1-2 sentences in the user's language).
Focus on: cuisine preferences, diet type, frequently used ingredients, cooking style.
Keep it concise. Output ONLY the new summary text.`,
		currentSummary,
		recipe.Title,
		mapBool(tags != "", ", Tags: "+tags),
	)

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: viper.GetString("deepseek.model"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage("You are a user profiling assistant. Update a short memory summary based on new activity."),
			openai.UserMessage(msg),
		},
		MaxCompletionTokens: openai.Int(200),
	})
	if err != nil {
		slog.Error("memory_update_ai_failed", "error", err, "user", email)
		return
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		return
	}

	newSummary := strings.TrimSpace(resp.Choices[0].Message.Content)
	if newSummary == "" || newSummary == currentSummary {
		return
	}

	_, err = db.Exec(ctx, `
		INSERT INTO user_memories (user_id, summary, updated_at)
		VALUES ($1::uuid, $2, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			summary = $2,
			updated_at = NOW()
	`, userID, newSummary)
	if err != nil {
		slog.Error("memory_update_save_failed", "error", err, "user", email)
		return
	}

	slog.Info("memory_updated",
		logger.KeyUser, email,
		"recipe", recipe.Title,
	)
}

func mapBool(cond bool, s string) string {
	if cond {
		return s
	}
	return ""
}
