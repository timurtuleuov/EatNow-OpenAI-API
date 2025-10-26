package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitTables(pool *pgxpool.Pool) error {
	ctx := context.Background()

	script := `
	CREATE EXTENSION IF NOT EXISTS "pgcrypto";

	-- 1. Создаём таблицу пользователей
	CREATE TABLE IF NOT EXISTS users (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		name VARCHAR(100) NOT NULL,
		password VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		device_id VARCHAR(255),
		platform VARCHAR(50),
		is_premium BOOLEAN DEFAULT FALSE,
		premium_expires TIMESTAMPTZ,
		daily_used_prompts INT DEFAULT 0,
		last_prompt_date DATE,
		created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
		updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
	);

	-- 2. Создаём таблицу логов (prompts)
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
		created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
	);

	CREATE TABLE IF NOT EXISTS user_bonuses (
		id SERIAL PRIMARY KEY,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		device_id VARCHAR(255) NOT NULL,
		type VARCHAR(50) NOT NULL,              -- например: 'ad', 'referral', 'gift'
		status VARCHAR(20) NOT NULL DEFAULT 'active',  -- 'active' | 'used' | 'expired'
		issued_at TIMESTAMPTZ DEFAULT NOW(),
		used_at TIMESTAMPTZ,
		expires_at TIMESTAMPTZ,                
		meta JSONB DEFAULT '{}'::jsonb          
	);
	`

	_, err := pool.Exec(ctx, script)
	if err != nil {
		return fmt.Errorf("failed to init tables: %w", err)
	}
	return nil
}
