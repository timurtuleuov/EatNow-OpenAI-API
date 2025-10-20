package model

import "time"

type User struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Password         string    `json:"password"`
	Email            string    `json:"email"`
	DeviceID         string    `json:"device_id"`
	Platform         string    `json:"platform"`
	IsPremium        bool      `json:"is_premium" db:"is_premium"`
	PremiumExpires   time.Time `json:"premium_expires" db:"premium_expires"`
	DailyUsedPrompts int       `json:"daily_used_prompts" db:"daily_used_prompts"`
	LastPromptDate   time.Time `json:"last_prompt_date" db:"last_prompt_date"`
	CreatedAt        time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
}
