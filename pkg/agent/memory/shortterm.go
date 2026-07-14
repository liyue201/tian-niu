package memory

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/tianniu-ai/tianniu/pkg/shared"
	"github.com/tianniu-ai/tianniu/pkg/storage"
)

type ShortTermMemory struct {
	storage       storage.Storage
	updater       *SmartMemoryUpdater
	buffer        map[string][]shared.OpenAIMessage
	bufferMutex   sync.Mutex
	lastFlushTime time.Time
	flushInterval time.Duration
}

func NewShortTermMemory(storage storage.Storage, updater *SmartMemoryUpdater) *ShortTermMemory {
	return &ShortTermMemory{
		storage:       storage,
		updater:       updater,
		buffer:        make(map[string][]shared.OpenAIMessage),
		flushInterval: 30 * time.Second,
	}
}

func (s *ShortTermMemory) Add(ctx context.Context, userId, conversationId string, messages []shared.OpenAIMessage) error {
	if len(messages) == 0 {
		return nil
	}

	key := s.cacheKey(userId, conversationId)
	s.bufferMutex.Lock()
	s.buffer[key] = append(s.buffer[key], messages...)

	if len(s.buffer[key]) > 20 {
		s.buffer[key] = s.buffer[key][len(s.buffer[key])-20:]
	}

	shouldFlush := time.Since(s.lastFlushTime) >= s.flushInterval
	s.bufferMutex.Unlock()

	if shouldFlush {
		return s.Flush(ctx, userId, conversationId)
	}

	return nil
}

func (s *ShortTermMemory) Flush(ctx context.Context, userId, conversationId string) error {
	key := s.cacheKey(userId, conversationId)

	s.bufferMutex.Lock()
	messages, ok := s.buffer[key]
	if !ok || len(messages) == 0 {
		s.bufferMutex.Unlock()
		return nil
	}
	delete(s.buffer, key)
	s.lastFlushTime = time.Now()
	s.bufferMutex.Unlock()

	content, err := s.getMemoryContent(ctx, userId, conversationId)
	if err != nil {
		return err
	}

	newMemory, err := s.updater.Update(ctx, content, messages)
	if err != nil {
		return err
	}

	return s.saveMemory(ctx, userId, conversationId, newMemory)
}

func (s *ShortTermMemory) FlushAll(ctx context.Context) error {
	s.bufferMutex.Lock()
	bufferCopy := make(map[string][]shared.OpenAIMessage)
	for k, v := range s.buffer {
		bufferCopy[k] = v
	}
	s.buffer = make(map[string][]shared.OpenAIMessage)
	s.lastFlushTime = time.Now()
	s.bufferMutex.Unlock()

	for key, messages := range bufferCopy {
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}
		userId, conversationId := parts[0], parts[1]

		if err := s.flushSingleConversation(ctx, userId, conversationId, messages); err != nil {
			log.Printf("Failed to flush conversation %s: %v", conversationId, err)
		}
	}
	return nil
}

func (s *ShortTermMemory) flushSingleConversation(ctx context.Context, userId, conversationId string, messages []shared.OpenAIMessage) error {
	content, err := s.getMemoryContent(ctx, userId, conversationId)
	if err != nil {
		return err
	}

	newMemory, err := s.updater.Update(ctx, content, messages)
	if err != nil {
		return err
	}

	return s.saveMemory(ctx, userId, conversationId, newMemory)
}

func (s *ShortTermMemory) GetMemory(userId, conversationId string) string {
	content, err := s.getMemoryContent(context.Background(), userId, conversationId)
	if err != nil {
		return ""
	}
	return content.String()
}

func (s *ShortTermMemory) GetMemoryContent(ctx context.Context, userId, conversationId string) (MemoryContent, error) {
	return s.getMemoryContent(ctx, userId, conversationId)
}

func (s *ShortTermMemory) cacheKey(userId, conversationId string) string {
	return userId + ":" + conversationId
}

func (s *ShortTermMemory) userGlobalMemoryKey(userId string) string {
	return "global_memory_" + userId
}

func (s *ShortTermMemory) conversationMemoryKey(conversationId string) string {
	return "conversation_memory_" + conversationId
}

func (s *ShortTermMemory) getMemoryContent(ctx context.Context, userId, conversationId string) (MemoryContent, error) {
	globalMemory, err := s.storage.Load(ctx, s.userGlobalMemoryKey(userId))
	if err != nil {
		return MemoryContent{}, err
	}
	conversationMemory, err := s.storage.Load(ctx, s.conversationMemoryKey(conversationId))
	if err != nil {
		return MemoryContent{}, err
	}
	return MemoryContent{
		GlobalMemory:       globalMemory,
		ConversationMemory: conversationMemory,
	}, nil
}

func (s *ShortTermMemory) saveMemory(ctx context.Context, userId, conversationId string, content MemoryContent) error {
	if content.GlobalMemory != "" {
		if err := s.storage.Store(ctx, s.userGlobalMemoryKey(userId), content.GlobalMemory); err != nil {
			return err
		}
	}
	if content.ConversationMemory != "" {
		if err := s.storage.Store(ctx, s.conversationMemoryKey(conversationId), content.ConversationMemory); err != nil {
			return err
		}
	}
	return nil
}
