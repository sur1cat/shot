package models

type LinkTag struct {
	ID     uint `json:"id" gorm:"primaryKey"`
	LinkID uint `json:"link_id" gorm:"index;not null"`
	TagID  uint `json:"tag_id" gorm:"index;not null"`
}
