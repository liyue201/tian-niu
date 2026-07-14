package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Vector = []float32

type MemoryChunk struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         string    `gorm:"type:text;not null;index"`
	ConversationID string    `gorm:"type:text;not null;index"`
	Content        string    `gorm:"type:text;not null"`
	Summary        string    `gorm:"type:text;not null"`
	Embedding      string    `gorm:"type:vector(1536)"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	RoundNumber    int       `gorm:"not null"`
}

func (MemoryChunk) TableName() string {
	return "long_term_memory"
}

type PGVectorStore struct {
	db        *gorm.DB
	dimension int
}

type Config struct {
	Host      string
	Port      int
	User      string
	Password  string
	Database  string
	Dimension int
}

func NewPGVectorStore(config Config) (*PGVectorStore, error) {
	if config.Dimension == 0 {
		config.Dimension = 1536
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		config.Host, config.Port, config.User, config.Password, config.Database,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &PGVectorStore{
		db:        db,
		dimension: config.Dimension,
	}

	if err := store.initTable(); err != nil {
		return nil, fmt.Errorf("failed to initialize table: %w", err)
	}

	return store, nil
}

func (s *PGVectorStore) initTable() error {
	if err := s.db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("failed to create vector extension: %w", err)
	}

	if err := s.db.AutoMigrate(&MemoryChunk{}); err != nil {
		return fmt.Errorf("failed to migrate table: %w", err)
	}

	indexSQL := `
		CREATE INDEX IF NOT EXISTS idx_long_term_memory_embedding
		ON long_term_memory
		USING ivfflat (embedding vector_cosine_ops)
		WITH (lists = 100)
	`
	if err := s.db.Exec(indexSQL).Error; err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

func (s *PGVectorStore) InsertMemory(ctx context.Context, userID, conversationID string, content, summary string, embedding Vector, roundNumber int) error {
	if len(embedding) != s.dimension {
		return fmt.Errorf("vector dimension mismatch: expected %d, got %d", s.dimension, len(embedding))
	}

	chunk := &MemoryChunk{
		UserID:         userID,
		ConversationID: conversationID,
		Content:        content,
		Summary:        summary,
		Embedding:      vectorToPGVector(embedding),
		RoundNumber:    roundNumber,
	}

	return s.db.WithContext(ctx).Create(chunk).Error
}

func (s *PGVectorStore) SearchMemory(ctx context.Context, userID string, queryVector Vector, limit int) ([]MemoryMatch, error) {
	if len(queryVector) != s.dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: expected %d, got %d", s.dimension, len(queryVector))
	}

	vectorStr := vectorToPGVector(queryVector)

	var results []struct {
		ID             int
		UserID         string
		ConversationID string
		Content        string
		Summary        string
		RoundNumber    int
		Score          float32
	}

	query := `
		SELECT id, user_id, conversation_id, content, summary, round_number,
		       1 - (embedding <=> ?) as score
		FROM long_term_memory
		WHERE user_id = ?
		ORDER BY embedding <=> ?
		LIMIT ?
	`

	err := s.db.WithContext(ctx).Raw(query, vectorStr, userID, vectorStr, limit).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	memoryMatches := make([]MemoryMatch, len(results))
	for i, r := range results {
		memoryMatches[i] = MemoryMatch{
			Content:        r.Content,
			Summary:        r.Summary,
			ConversationID: r.ConversationID,
			RoundNumber:    r.RoundNumber,
			Score:          r.Score,
		}
	}

	return memoryMatches, nil
}

func (s *PGVectorStore) GetUserMemoryCount(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&MemoryChunk{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (s *PGVectorStore) DeleteByUserID(ctx context.Context, userID string) error {
	return s.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&MemoryChunk{}).Error
}

func (s *PGVectorStore) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func vectorToPGVector(v Vector) string {
	if len(v) == 0 {
		return "[]"
	}

	strValues := make([]string, len(v))
	for i, val := range v {
		strValues[i] = fmt.Sprintf("%f", val)
	}

	return "[" + strings.Join(strValues, ",") + "]"
}

type MemoryMatch struct {
	Content        string
	Summary        string
	ConversationID string
	RoundNumber    int
	Score          float32
}
