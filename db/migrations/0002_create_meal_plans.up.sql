-- Обновление таблицы пользователей (система баланса)
ALTER TABLE users ADD COLUMN IF NOT EXISTS balance INT DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS balance_reset_at TIMESTAMPTZ DEFAULT NOW();

-- Обновление таблицы рецептов (привязка к создателю)
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE recipes ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ DEFAULT NOW();

-- Создание новой таблицы избранного
CREATE TABLE IF NOT EXISTS favorites (
    id SERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    recipe_id UUID REFERENCES recipes(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_user_recipe UNIQUE (user_id, recipe_id)
);

-- Создание новой таблицы планов питания
CREATE TABLE IF NOT EXISTS meal_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    meal_plan JSONB NOT NULL,
    prompt TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Индексы для оптимизации новых таблиц
CREATE INDEX IF NOT EXISTS idx_meal_plans_user_id ON meal_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_recipes_user_id ON recipes(user_id);