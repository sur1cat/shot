package models

import (
	"time"
)

type Link struct {
	ID          uint        `json:"id" gorm:"primaryKey"`
	UserID      uint        `json:"user_id" gorm:"not null"`
	OriginalURL string      `json:"original_url" gorm:"not null"`
	ShortCode   string      `json:"short_code" gorm:"unique;not null"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   *time.Time  `json:"expires_at"`
	ClickCount  int         `json:"click_count" gorm:"default:0"`
	ClickStats  []ClickStat `json:"click_stats,omitempty" gorm:"foreignKey:LinkID"`
}
