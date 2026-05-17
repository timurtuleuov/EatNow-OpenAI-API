package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PromptLog struct {
	UserID     *string // может быть nil, если гость
	DeviceID   string
	Prompt     string
	Response   interface{} // JSON-ответ (любой тип)
	TokensUsed int
	Model      string
	DurationMs int
	Success    bool
	ErrorMsg   string
	AppVersion string
	Language   string
	Country    string
}

func resolveUserID(db *pgxpool.Pool, userID *string) (*string, error) {
	if userID == nil || *userID == "" {
		return nil, nil
	}

	if strings.Contains(*userID, "@") {
		var uuid string
		err := db.QueryRow(context.Background(),
			"SELECT id FROM users WHERE email = $1", *userID,
		).Scan(&uuid)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve user email to UUID: %w", err)
		}
		return &uuid, nil
	}

	return userID, nil
}

func LogPrompt(db *pgxpool.Pool, log PromptLog) error {
	ctx := context.Background()

	var recipeId string

	respJSON, err := json.Marshal(log.Response)
	if err != nil {
		respJSON = []byte("{}")
	}

	userUUID, err := resolveUserID(db, log.UserID)
	if err != nil {
		return fmt.Errorf("failed to resolve user ID: %w", err)
	}

	err = db.QueryRow(ctx, `
	INSERT INTO recipes (
			user_id, recipe
		) VALUES ($1, $2)
		 RETURNING id
	`, userUUID, respJSON,
	).Scan(&recipeId)

	if err != nil {
		fmt.Printf("[LogPrompt] Error inserting recipe: %v\n", err)
		return fmt.Errorf("failed to insert recipe: %w", err)
	}

	_, err = db.Exec(ctx, `
		INSERT INTO prompts (
			user_id, device_id, prompt, recipe_id, tokens_used, model,
			duration_ms, success, error_message, app_version,
			language, country, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
	`,
		userUUID,
		log.DeviceID,
		log.Prompt,
		recipeId,
		log.TokensUsed,
		log.Model,
		log.DurationMs,
		log.Success,
		log.ErrorMsg,
		log.AppVersion,
		log.Language,
		log.Country,
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to log prompt: %w", err)
	}
	return nil
}
