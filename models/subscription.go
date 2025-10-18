package model

import "time"

type Subscription struct {
	ID            int64     `json:"id" db:"id" gorm:"primaryKey;autoIncrement"`
	UserID        int64     `json:"user_id" db:"user_id" gorm:"not null;index"`
	Platform      string    `json:"platform" db:"platform" gorm:"type:varchar(20);not null;index"` // android | ios
	PurchaseToken string    `json:"purchase_token" db:"purchase_token" gorm:"type:text;unique;not null"`
	ProductID     string    `json:"product_id" db:"product_id" gorm:"type:varchar(100);not null"`
	ExpiresAt     time.Time `json:"expires_at" db:"expires_at" gorm:"type:timestamptz;not null;index"`
	IsActive      bool      `json:"is_active" db:"is_active" gorm:"default:true;not null"`
	Provider      string    `json:"provider" db:"provider" gorm:"type:varchar(50);not null;index"` // google_play | app_store
	CreatedAt     time.Time `json:"created_at" db:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at" gorm:"autoUpdateTime"`
}
