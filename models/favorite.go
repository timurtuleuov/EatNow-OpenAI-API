package model

import "time"

type Favorite struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	RecipeID  string    `json:"recipe_id"` // Соответствует типу INT в новой таблице
	Recipe    Recipe    `json:"recipe"`    // Объект рецепта, собираемый через JOIN
	CreatedAt time.Time `json:"created_at"`
}
