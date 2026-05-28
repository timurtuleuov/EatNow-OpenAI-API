CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Таблица пользователей
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

-- 2. Таблица рецептов
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe JSONB
);

-- 3. Таблица логов (prompts)
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

-- 4. Таблица бонусов
CREATE TABLE IF NOT EXISTS user_bonuses (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    device_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    issued_at TIMESTAMPTZ DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    meta JSONB DEFAULT '{}'::jsonb
);

-- 5. Таблица refresh токенов
CREATE TABLE IF NOT EXISTS refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_email VARCHAR(255) UNIQUE NOT NULL,
    token TEXT NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 6. Таблица с платежами
CREATE TABLE IF NOT EXISTS payments (
    id SERIAL PRIMARY KEY,
    user_email TEXT NOT NULL REFERENCES users(email),
    subscription_type TEXT NOT NULL,
    amount NUMERIC(10, 2) NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    platform TEXT NOT NULL,
    status TEXT NOT NULL,
    transaction_id TEXT UNIQUE NOT NULL,
    purchase_token TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,
    is_auto_renewing BOOLEAN DEFAULT TRUE
);