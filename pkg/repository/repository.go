package repository

import (
	"context"

	"github.com/tianniu-ai/tianniu/pkg/model"
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
}
