package model

import (
	"time"
)

type SkillMetadata struct {
	Emoji    string `json:"emoji" gorm:"type:text"`
	Author   string `json:"author"`
	Version  string `json:"version"`
	License  string `json:"license"`
	Category string `json:"category"`
	Homepage string `json:"homepage"`
}

type Skill struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex:idx_skill_name_user"`
	Description string `gorm:"type:text"`
	Homepage    string
	Metadata    SkillMetadata `gorm:"embedded"`
	Status      string        `gorm:"type:varchar(20);not null"`
	Type        string        `gorm:"type:varchar(20);not null"`
	UserID      string        `gorm:"uniqueIndex:idx_skill_name_user"`
	InstalledAt time.Time
	UpdatedAt   time.Time
	Path        string
	Content     string `gorm:"type:text"`
}
