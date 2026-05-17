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
	ALTER TABLE recipes ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
	ALTER TABLE recipes ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();
	-- 2. Отдельная таблица для рецептов
	CREATE TABLE IF NOT EXISTS recipes (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id UUID REFERENCES users(id) ON DELETE CASCADE,
	recipe JSONB,
	created_at TIMESTAMPTZ DEFAULT NOW()
	);

	

	-- 3. Создаём таблицу логов (prompts)
	CREATE TABLE IF NOT EXISTS prompts (
		id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		device_id VARCHAR(255),
		prompt TEXT NOT NULL,
		recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
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

	-- 4. Создаём таблицу бонусов
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

	-- 5. Создаём таблицу refresh токенов
	CREATE TABLE IF NOT EXISTS refresh_tokens (
		id SERIAL PRIMARY KEY,
		user_email VARCHAR(255) UNIQUE NOT NULL,
		token TEXT NOT NULL,
		expires_at TIMESTAMP NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	-- 6. Создаем таблицу избранных рецептов
	CREATE TABLE IF NOT EXISTS favorites (
		id SERIAL PRIMARY KEY,
		user_id UUID REFERENCES users(id) ON DELETE CASCADE,
		recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE, 
		created_at TIMESTAMPTZ DEFAULT NOW(),
		CONSTRAINT unique_user_recipe UNIQUE (user_id, recipe_id)
	);

	-- Миграция для существующей таблицы favorites (была колонка recipe JSONB, стала recipe_id UUID)
	ALTER TABLE prompts DROP CONSTRAINT IF EXISTS fk_prompts_recipe;
	ALTER TABLE favorites ADD COLUMN IF NOT EXISTS recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE;
	ALTER TABLE favorites DROP CONSTRAINT IF EXISTS unique_user_recipe;
	ALTER TABLE favorites ADD CONSTRAINT unique_user_recipe UNIQUE (user_id, recipe_id);

	-- 7. Создаем таблицу с платежами
	CREATE TABLE IF NOT EXISTS payments (
		id SERIAL PRIMARY KEY,
		user_email TEXT NOT NULL REFERENCES users(email), 
		subscription_type TEXT NOT NULL,                  -- premium_subscription_monthly
		amount NUMERIC(10, 2) NOT NULL,                   -- 2.99
		currency TEXT NOT NULL DEFAULT 'USD',             -- USD, KZT и т.д.
		platform TEXT NOT NULL,                           -- android или ios
		
		-- Статус платежа
		status TEXT NOT NULL,                             -- completed, expired, refunded
		
		-- Данные от Google/Apple
		transaction_id TEXT UNIQUE NOT NULL,              -- Уникальный ID транзакции от Google
		purchase_token TEXT NOT NULL,                     -- Нужен для проверки/рефреша
		
		-- Даты
		created_at TIMESTAMP DEFAULT NOW(),               -- Когда совершена покупка
		expires_at TIMESTAMP NOT NULL,                    -- Когда подписка закончится
		
		-- Мой вариант полезного поля:
		is_auto_renewing BOOLEAN DEFAULT TRUE             -- Продлевается ли подписка автоматически
	);
	`

	_, err := pool.Exec(ctx, script)
	if err != nil {
		return fmt.Errorf("failed to init tables: %w", err)
	}
	return nil
}
