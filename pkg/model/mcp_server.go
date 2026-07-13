package model

import (
	"time"
)

type McpServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Url     string            `json:"url,omitempty"`
}

type McpServer struct {
	ID          string `gorm:"primaryKey"`
	Name        string `gorm:"uniqueIndex:idx_mcp_name_user"`
	Description string `gorm:"type:text"`
	Status      string `gorm:"type:varchar(20);not null"`
	Type        string `gorm:"type:varchar(20);not null"`
	UserID      string `gorm:"uniqueIndex:idx_mcp_name_user"`
	Config      string `gorm:"type:text"`
	InstalledAt time.Time
	UpdatedAt   time.Time
}
