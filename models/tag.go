package models

import (
	"time"
)

type Tag struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"unique;not null"`
	CreatedAt time.Time `json:"created_at"`
	LinkTags  []LinkTag `json:"-" gorm:"foreignKey:TagID"`
}
