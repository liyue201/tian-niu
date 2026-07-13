package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/libtnb/sqlite"
	"github.com/tianniu-ai/tianniu/pkg/model"
	"gorm.io/gorm"
)

var ErrDuplicateEntry = errors.New("duplicate entry")

type SQLStore struct {
	db *gorm.DB
}

func NewSQLStore(dsn string) (*SQLStore, error) {
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	err = db.AutoMigrate(&model.User{}, &model.Conversation{}, &model.ChatMessage{}, &model.KVData{}, &model.Skill{}, &model.McpServer{})
	if err != nil {
		return nil, err
	}
	return &SQLStore{db: db}, nil
}

// Transaction runs a function within a database transaction
func (r *SQLStore) Transaction(fn func(tx *gorm.DB) error) error {
	return r.db.Transaction(fn)
}

// Create inserts a new record into the database
func (r *SQLStore) Create(v interface{}) error {
	err := r.db.Create(v).Error
	if err != nil && isDuplicateEntryError(err) {
		return ErrDuplicateEntry
	}
	return err
}

// Delete removes a record from the database
func (r *SQLStore) Delete(v interface{}) error {
	return r.db.Delete(v).Error
}

// DeleteConversationWithMessages deletes a conversation and all its messages
func (r *SQLStore) DeleteConversationWithMessages(conversationID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("conversation_id = ?", conversationID).Delete(&model.ChatMessage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", conversationID).Delete(&model.Conversation{}).Error; err != nil {
			return err
		}
		return nil
	})
}

// KV storage methods

// Load retrieves a value by key from KV storage
func (r *SQLStore) Load(ctx context.Context, key string) (string, error) {
	var kv model.KVData
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&kv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return kv.Value, nil
}

// Store saves a key-value pair to KV storage
func (r *SQLStore) Store(ctx context.Context, key string, value string) error {
	kv := &model.KVData{Key: key, Value: value}
	return r.db.WithContext(ctx).Save(kv).Error
}

// DeleteKV removes a key-value pair from KV storage
func (r *SQLStore) DeleteKV(ctx context.Context, key string) error {
	return r.db.WithContext(ctx).Where("key = ?", key).Delete(&model.KVData{}).Error
}

// List returns all keys with the given prefix from KV storage
func (r *SQLStore) List(ctx context.Context, prefix string) ([]string, error) {
	var kvs []model.KVData
	err := r.db.WithContext(ctx).Where("key LIKE ?", prefix+"%").Find(&kvs).Error
	if err != nil {
		return nil, err
	}
	keys := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		keys = append(keys, kv.Key)
	}
	return keys, nil
}

func isDuplicateEntryError(err error) bool {
	s := err.Error()
	return strings.Contains(s, "UNIQUE constraint failed") || strings.Contains(s, "duplicate key")
}
