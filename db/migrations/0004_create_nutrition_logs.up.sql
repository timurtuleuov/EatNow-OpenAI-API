CREATE TABLE IF NOT EXISTS nutrition_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    log_date DATE NOT NULL DEFAULT CURRENT_DATE,
    meals JSONB NOT NULL DEFAULT '[]',
    total JSONB NOT NULL DEFAULT '{}',
    water_gl REAL DEFAULT 0,
    health_score INT DEFAULT 0,
    analysis TEXT DEFAULT '',
    tips TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_user_date UNIQUE (user_id, log_date)
);

CREATE INDEX IF NOT EXISTS idx_nutrition_logs_user_date ON nutrition_logs(user_id, log_date DESC);
