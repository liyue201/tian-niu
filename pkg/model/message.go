package model

import "gorm.io/gorm"

type ChatMessage struct {
	ID              string `gorm:"primaryKey"`
	UserID          string `gorm:"index"`
	ConversationID  string `gorm:"index"`
	ParentMessageID string

	Query    string // User's original question
	Response string // Model's final output

	Rounds string // All LLM requests between user question and tool loop end, stored as JSON

	Model string // Model used
	Usage string

	CreatedAt int64
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
