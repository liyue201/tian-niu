package context

import (
	"context"
	"fmt"
	"strings"

	"github.com/openai/openai-go/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tianniu-ai/tianniu/pkg/agent/memory"
	"github.com/tianniu-ai/tianniu/pkg/repository"
	"github.com/tianniu-ai/tianniu/pkg/shared"
)

type messageWrap struct {
	Message shared.OpenAIMessage
	Tokens  int
}

type Engine struct {
	memory               memory.Memory
	userId               string
	conversationId       string
	currentUserMsg       shared.OpenAIMessage
	repo                 *repository.SQLStore
	systemPromptTemplate string
	messages             []messageWrap
	policies             []Policy
	onPolicyEvent        func(policyName string, running bool, err error)
	contextTokens        int
	contextWindow        int
	turnCount            int
}

type TokenBudget struct {
	ContextWindow int
}

type Usage struct {
	PromptTokens int
}

type TurnDraft struct {
	NewMessages []shared.OpenAIMessage
}

func NewContextEngine(memory memory.Memory, userId string, conversationId string, policies []Policy, repo *repository.SQLStore) *Engine {
	return &Engine{
		memory:         memory,
		userId:         userId,
		conversationId: conversationId,
		repo:           repo,
		policies:       policies,
		messages:       make([]messageWrap, 0),
		contextWindow:  200000,
		turnCount:      0,
	}
}

func (c *Engine) Init(systemPrompt string, budget TokenBudget) {
	c.systemPromptTemplate = systemPrompt
	if budget.ContextWindow > 0 {
		c.contextWindow = budget.ContextWindow
	}

	historyMsgs, err := c.repo.GetConversationMessages(c.conversationId, 10)
	if err != nil {
		log.Errorf("load conversation messages: %v", err)
		return
	}
	if len(historyMsgs) == 0 {
		return
	}
	c.turnCount = len(historyMsgs)
	msgs := buildHistory(historyMsgs, historyMsgs[0].ID)

	for i := range msgs {
		msg := msgs[i]
		c.messages = append(c.messages, messageWrap{Message: msg, Tokens: CountTokens(msg)})
	}
}

func (c *Engine) BuildRequestMessages() []shared.OpenAIMessage {
	result := make([]shared.OpenAIMessage, 0, len(c.messages)+1)
	if c.systemPromptTemplate != "" {
		result = append(result, openai.SystemMessage(c.BuildSystemPrompt()))
	}
	for i := range c.messages {
		result = append(result, c.messages[i].Message)
	}
	return result
}

func (c *Engine) StartTurn(userMsg shared.OpenAIMessage) TurnDraft {
	c.currentUserMsg = userMsg
	return TurnDraft{
		NewMessages: []shared.OpenAIMessage{userMsg},
	}
}

func (c *Engine) CommitTurn(ctx context.Context, draft TurnDraft, usage Usage) error {
	for i := range draft.NewMessages {
		msg := draft.NewMessages[i]
		c.messages = append(c.messages, messageWrap{Message: msg, Tokens: CountTokens(msg)})
	}

	c.recountTokens()

	if err := c.applyPolicies(ctx); err != nil {
		return err
	}

	c.turnCount++

	err := c.memory.Update(ctx, c.userId, c.conversationId, draft.NewMessages, c.turnCount)
	if err != nil {
		return err
	}

	return nil
}

func (c *Engine) AbortTurn(_ TurnDraft) {
}

func (c *Engine) FlushMemory(ctx context.Context) error {
	if multiLevelMem, ok := c.memory.(interface {
		Flush(ctx context.Context) error
	}); ok {
		return multiLevelMem.Flush(ctx)
	}
	return nil
}

func (c *Engine) GetContextUsage() float64 {
	if c.contextWindow <= 0 {
		return 0
	}
	return float64(c.contextTokens) / float64(c.contextWindow)
}

func (c *Engine) recountTokens() {
	totalTokens := 0
	for i := range c.messages {
		totalTokens += c.messages[i].Tokens
	}
	c.contextTokens = totalTokens
}

func (c *Engine) applyPolicies(ctx context.Context) error {
	ctx = context.WithValue(ctx, "conversationId", c.conversationId)
	for _, policy := range c.policies {
		if !policy.ShouldApply(ctx, c) {
			continue
		}
		if c.onPolicyEvent != nil {
			c.onPolicyEvent(policy.Name(), true, nil)
		}
		result, err := policy.Apply(ctx, c)
		if c.onPolicyEvent != nil {
			c.onPolicyEvent(policy.Name(), false, err)
		}
		if err != nil {
			return fmt.Errorf("apply policy %s: %w", policy.Name(), err)
		}
		c.messages = result.Messages
		c.recountTokens()
	}
	return nil
}

func (c *Engine) SetPolicyEventHook(hook func(policyName string, running bool, err error)) {
	c.onPolicyEvent = hook
}

func (c *Engine) BuildSystemPrompt() string {
	replaceMap := make(map[string]string)

	if c.memory != nil {
		replaceMap["{memory}"] = c.memory.GetShortTermMemory(c.userId, c.conversationId)

		workingMessages := c.memory.GetWorkingMemory()
		replaceMap["{working_memory}"] = c.formatWorkingMemory(workingMessages)

		longTermMem, err := c.memory.GetLongTermMemory(context.Background(), c.userId, c.getCurrentUserQuery())
		if err != nil {
			log.Warnf("failed to get long-term memory prompt: %v", err)
			longTermMem = ""
		}
		replaceMap["{long_term_memory}"] = longTermMem
	} else {
		replaceMap["{memory}"] = ""
		replaceMap["{working_memory}"] = ""
		replaceMap["{long_term_memory}"] = ""
	}

	prompt := c.systemPromptTemplate
	for k, v := range replaceMap {
		prompt = strings.ReplaceAll(prompt, k, v)
	}
	return prompt
}

func (c *Engine) formatWorkingMemory(messages []shared.OpenAIMessage) string {
	if len(messages) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("### Conversation History\n")
	for _, msg := range messages {
		role := shared.GetRoleName(msg)
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if ok && contentStr != nil {
			b.WriteString(role)
			b.WriteString(": ")
			b.WriteString(*contentStr)
			b.WriteString("\n")
		}
	}
	return b.String()
}

func (c *Engine) getCurrentUserQuery() string {
	msg := c.currentUserMsg
	contentAny := msg.GetContent().AsAny()
	if contentStr, ok := contentAny.(*string); ok {
		return *contentStr
	}

	return ""
}

func (c *Engine) Reset() {
	c.messages = make([]messageWrap, 0)
	c.contextTokens = 0
}
