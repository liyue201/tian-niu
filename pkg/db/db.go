package db

import (
	"github.com/libtnb/sqlite"
	"gorm.io/gorm"
)

type User struct {
	UserID       string `gorm:"primaryKey"`
	Username     string `gorm:"unique;not null"`
	PasswordHash string `gorm:"not null"`
	Email        string
	CreatedAt    int64
}

type Conversation struct {
	ConversationID string `gorm:"primaryKey"`
	UserID         string `gorm:"index"`
	Title          string
	CreatedAt      int64
}

type ChatMessage struct {
	MessageID       string `gorm:"primaryKey"`
	UserID          string `gorm:"index"`
	ConversationID  string `gorm:"index"`
	ParentMessageID string

	Query    string // User's original question
	Response string // Model's final output

	Rounds string // All LLM requests between user question and tool loop end, stored as JSON

	Model string // Model used
	Usage string

	CreatedAt int64
}

func InitDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&User{}, &Conversation{}, &ChatMessage{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
