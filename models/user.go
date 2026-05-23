package model

import "time"

type User struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	Password       string     `json:"-"`
	Email          string     `json:"email"`
	DeviceID       string     `json:"device_id"`
	Platform       string     `json:"platform"`
	IsPremium      bool       `json:"is_premium"`
	PremiumExpires *time.Time `json:"premium_expires"`
	Balance        int        `json:"balance"`
	BalanceResetAt time.Time  `json:"balance_reset_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
