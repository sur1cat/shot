package models

import (
	"time"
)

type ClickStat struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	LinkID      uint      `json:"link_id"`
	ClickedAt   time.Time `json:"clicked_at"`
	ReferrerURL string    `json:"referrer_url"`
	UserAgent   string    `json:"user_agent"`
	IPAddress   string    `json:"ip_address"`
}
