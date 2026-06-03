package model

import "time"

type UserMemory struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Summary   string    `json:"summary"`
	UpdatedAt time.Time `json:"updated_at"`
}
