package model

import "gorm.io/gorm"

type Conversation struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	Title     string
	CreatedAt int64
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
