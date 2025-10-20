package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitUserTable(pool *pgxpool.Pool) error {
	ctx := context.Background()

	tableScript := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		device_id VARCHAR(255),
		platform VARCHAR(50),
		is_premium BOOLEAN DEFAULT FALSE,
		premium_expires TIMESTAMP WITH TIME ZONE,
		daily_used_prompts INT DEFAULT 0,
		last_prompt_date DATE,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
		updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
	);
	`

	_, err := pool.Exec(ctx, tableScript)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	return nil
}
