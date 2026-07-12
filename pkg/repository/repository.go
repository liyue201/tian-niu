package repository

import (
	"context"

	"github.com/tianniu-ai/tianniu/pkg/model"
	"gorm.io/gorm"
)

// Store defines the interface for repository operations
type Store interface {
	Create(v interface{}) error
	Delete(v interface{}) error
	DeleteConversationWithMessages(conversationID string) error
	GetConversationByID(id string) (*model.Conversation, error)
	GetUserConversations(userID string) ([]*model.Conversation, error)
	UpdateConversationTitle(conversation *model.Conversation) error
	SaveKVData(ctx context.Context, key, value string) error
	GetKVData(ctx context.Context, key string) (string, error)
	GetConversationMessages(conversationID string, limit int) ([]*model.ChatMessage, error)
	GetUserByUsername(username string) (*model.User, error)
	// KV storage interface methods
	Load(ctx context.Context, key string) (string, error)
	Store(ctx context.Context, key string, value string) error
	DeleteKV(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// Transactional defines the interface for transaction support
type Transactional interface {
	Transaction(fn func(tx *gorm.DB) error) error
}

// KVStore defines the interface for KV storage operations
type KVStore interface {
	Load(ctx context.Context, key string) (string, error)
	Store(ctx context.Context, key string, value string) error
	DeleteKV(ctx context.Context, key string) error
	List(ctx context.Context, prefix string) ([]string, error)
}

// UserRepository defines the interface for user operations
type UserRepository interface {
	GetUserByUsername(username string) (*model.User, error)
}

// ConversationRepository defines the interface for conversation operations
type ConversationRepository interface {
	GetConversationByID(id string) (*model.Conversation, error)
	GetUserConversations(userID string) ([]*model.Conversation, error)
	UpdateConversationTitle(conversation *model.Conversation) error
	DeleteConversationWithMessages(conversationID string) error
	GetConversationMessages(conversationID string, limit int) ([]*model.ChatMessage, error)
}
