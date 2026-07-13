package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/tianniu-ai/tianniu/pkg/shared"
	"github.com/tianniu-ai/tianniu/pkg/storage"
)

type Memory interface {
	String(userId, conversationId string) string
	Update(ctx context.Context, userId, conversationId string, newMessages []shared.OpenAIMessage) error
}

type MultiLevelMemory struct {
	storage storage.Storage

	updater MemoryUpdater
}

func NewMultiLevelMemory(storage storage.Storage, u MemoryUpdater) *MultiLevelMemory {
	m := &MultiLevelMemory{
		storage: storage,
		updater: u,
	}
	return m
}

func (m *MultiLevelMemory) String(userId string, conversationId string) string {
	content, err := m.getMemoryContent(context.Background(), userId, conversationId)
	if err != nil {
		return ""
	}
	return content.String()
}

func (m *MultiLevelMemory) userGlobalMemoryKey(userId string) string {
	return fmt.Sprintf("global_memory_%s", userId)
}

func (m *MultiLevelMemory) conversationMemoryKey(conversationId string) string {
	return fmt.Sprintf("conversation_memory_%s", conversationId)
}

func (m *MultiLevelMemory) getMemoryContent(ctx context.Context, userId string, conversationId string) (MemoryContent, error) {
	globalMemory, err := m.storage.Load(ctx, m.userGlobalMemoryKey(userId))
	if err != nil {
		return MemoryContent{}, err
	}
	conversationMemory, err := m.storage.Load(ctx, m.conversationMemoryKey(conversationId))
	if err != nil {
		return MemoryContent{}, err
	}
	return MemoryContent{
		GlobalMemory:       globalMemory,
		ConversationMemory: conversationMemory,
	}, nil
}

func (m *MultiLevelMemory) Update(ctx context.Context, userId string, conversationId string, newMessages []shared.OpenAIMessage) error {
	if len(newMessages) == 0 {
		return nil
	}

	content, err := m.getMemoryContent(ctx, userId, conversationId)
	if err != nil {
		return err
	}

	newMemory, err := m.updater.Update(ctx, content, newMessages)
	if err != nil {
		return err
	}

	if err := m.storage.Store(ctx, m.userGlobalMemoryKey(userId), newMemory.GlobalMemory); err != nil {
		return err
	}
	if err := m.storage.Store(ctx, m.conversationMemoryKey(conversationId), newMemory.ConversationMemory); err != nil {
		return err
	}

	return nil
}

type MemoryContent struct {
	GlobalMemory       string `json:"global_memory,omitempty"`
	ConversationMemory string `json:"conversation_memory,omitempty"`
}

func (m *MemoryContent) String() string {
	prompt := memoryPromptTemplate
	prompt = strings.ReplaceAll(prompt, "{global_memory}", m.GlobalMemory)
	prompt = strings.ReplaceAll(prompt, "{conversation_memory}", m.ConversationMemory)
	return prompt
}

const memoryPromptTemplate = `### Global Memory
Here is the memory about the user among all conversations:
{global_memory}

### Conversation Memory
The memory of the current conversation is:
{conversation_memory}
`
