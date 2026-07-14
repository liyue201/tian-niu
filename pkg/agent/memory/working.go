package memory

import (
	"github.com/tianniu-ai/tianniu/pkg/shared"
)

type WorkingMemory struct {
	messages  []shared.OpenAIMessage
	maxLength int
	turnCount int
}

func NewWorkingMemory(maxLength int) *WorkingMemory {
	return &WorkingMemory{
		messages:  make([]shared.OpenAIMessage, 0),
		maxLength: maxLength,
	}
}

func (w *WorkingMemory) Add(messages []shared.OpenAIMessage) error {
	w.messages = append(w.messages, messages...)

	if w.maxLength > 0 && len(w.messages) > w.maxLength {
		w.messages = w.messages[len(w.messages)-w.maxLength:]
	}

	w.turnCount += len(messages)
	return nil
}

func (w *WorkingMemory) GetMessages() []shared.OpenAIMessage {
	return w.messages
}

func (w *WorkingMemory) TurnCount() int {
	return w.turnCount
}

func (w *WorkingMemory) Clear() {
	w.messages = make([]shared.OpenAIMessage, 0)
	w.turnCount = 0
}
