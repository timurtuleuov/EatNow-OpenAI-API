package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitPromptsTable(pool *pgxpool.Pool) error {
	ctx := context.Background()

	query := `
	CREATE TABLE IF NOT EXISTS prompts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		device_id VARCHAR(255),
		prompt TEXT NOT NULL,
		response JSONB,
		tokens_used INT DEFAULT 0,
		model VARCHAR(100),
		duration_ms INT,
		success BOOLEAN DEFAULT TRUE,
		error_message TEXT,
		app_version VARCHAR(20),
		language VARCHAR(10),
		country VARCHAR(50),
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
	);`

	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create prompts table: %w", err)
	}
	return nil
}
