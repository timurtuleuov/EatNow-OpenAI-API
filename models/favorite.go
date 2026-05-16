package model

import "time"

type Favorite struct {
	ID        int       `json:"id"`
	UserID    string    `json:"user_id"`
	Recipe    Recipe    `json:"recipe"`
	CreatedAt time.Time `json:"created_at"`
}
