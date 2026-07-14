package memory

import (
	"context"
	"strings"

	"github.com/tianniu-ai/tianniu/pkg/shared"
	"github.com/tianniu-ai/tianniu/pkg/storage"
)

type Memory interface {
	GetShortTermMemory(userId, conversationId string) string
	GetWorkingMemory() []shared.OpenAIMessage
	Update(ctx context.Context, userId, conversationId string, newMessages []shared.OpenAIMessage, turnCount int) error
	GetLongTermMemory(ctx context.Context, userId, query string) (string, error)
	Flush(ctx context.Context, userId, conversationId string) error
}

type MultiLevelMemory struct {
	workingMemory   *WorkingMemory
	shortTermMemory *ShortTermMemory
	longTermMemory  LongTermMemoryProvider
}

type LongTermMemoryProvider interface {
	ProcessConversation(ctx context.Context, userId, conversationId string, messages []shared.OpenAIMessage, turnCount int) error
	GetMemoryPrompt(ctx context.Context, userId, query string) (string, error)
}

func NewMultiLevelMemory(storage storage.Storage, updater *SmartMemoryUpdater, longTermMemory LongTermMemoryProvider) *MultiLevelMemory {
	return &MultiLevelMemory{
		workingMemory:   NewWorkingMemory(100),
		shortTermMemory: NewShortTermMemory(storage, updater),
		longTermMemory:  longTermMemory,
	}
}

func (m *MultiLevelMemory) GetShortTermMemory(userId string, conversationId string) string {
	shortTermContent := m.shortTermMemory.GetMemory(userId, conversationId)
	return shortTermContent
}

func (m *MultiLevelMemory) Update(ctx context.Context, userId, conversationId string, newMessages []shared.OpenAIMessage, turnCount int) error {
	if err := m.workingMemory.Add(newMessages); err != nil {
		return err
	}

	if err := m.shortTermMemory.Add(ctx, userId, conversationId, newMessages); err != nil {
		return err
	}

	if m.longTermMemory != nil {
		if err := m.longTermMemory.ProcessConversation(ctx, userId, conversationId, newMessages, turnCount); err != nil {
			return err
		}
	}

	return nil
}

func (m *MultiLevelMemory) GetLongTermMemory(ctx context.Context, userId, query string) (string, error) {
	if m.longTermMemory != nil {
		return m.longTermMemory.GetMemoryPrompt(ctx, userId, query)
	}
	return "", nil
}

func (m *MultiLevelMemory) Flush(ctx context.Context, userId, conversationId string) error {
	return m.shortTermMemory.Flush(ctx, userId, conversationId)
}

func (m *MultiLevelMemory) GetWorkingMemory() []shared.OpenAIMessage {
	return m.workingMemory.GetMessages()
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
{conversation_memory}`
