package memory

import (
	"context"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/openai/openai-go/v3"
	"github.com/tianniu-ai/tianniu/pkg/agent/llm"
	"github.com/tianniu-ai/tianniu/pkg/shared"
)

type MemoryUpdater interface {
	Update(ctx context.Context, oldMemory MemoryContent, newMessages []shared.OpenAIMessage) (MemoryContent, error)
	Flush(ctx context.Context, oldMemory MemoryContent) (MemoryContent, error)
}

type SmartMemoryUpdater struct {
	llmUpdater        *LLMMemoryUpdater
	batchSize         int
	maxBatchSize      int
	minUpdateInterval time.Duration
	messageBuffer     []shared.OpenAIMessage
	lastUpdateTime    time.Time
	bufferMutex       sync.Mutex
	updateMutex       sync.Mutex
}

func NewSmartMemoryUpdater(modelConf shared.ModelConfig) *SmartMemoryUpdater {
	return &SmartMemoryUpdater{
		llmUpdater:        NewLLMMemoryUpdater(modelConf),
		batchSize:         3,
		maxBatchSize:      10,
		minUpdateInterval: 60 * time.Second,
		lastUpdateTime:    time.Now(),
	}
}

func (u *SmartMemoryUpdater) Update(ctx context.Context, oldMemory MemoryContent, newMessages []shared.OpenAIMessage) (MemoryContent, error) {
	if len(newMessages) == 0 {
		return oldMemory, nil
	}

	filteredMessages := u.filterTrivialMessages(newMessages)
	if len(filteredMessages) == 0 {
		return oldMemory, nil
	}

	u.bufferMutex.Lock()
	u.messageBuffer = append(u.messageBuffer, filteredMessages...)

	shouldUpdate := u.shouldTriggerUpdate()

	if shouldUpdate {
		messagesToProcess := u.messageBuffer
		u.messageBuffer = nil
		u.bufferMutex.Unlock()

		return u.processBatch(ctx, oldMemory, messagesToProcess)
	}

	u.bufferMutex.Unlock()

	return oldMemory, nil
}

func (u *SmartMemoryUpdater) Flush(ctx context.Context, oldMemory MemoryContent) (MemoryContent, error) {
	u.bufferMutex.Lock()
	messages := u.messageBuffer
	u.messageBuffer = nil
	u.bufferMutex.Unlock()

	if len(messages) == 0 {
		return oldMemory, nil
	}

	return u.processBatch(ctx, oldMemory, messages)
}

func (u *SmartMemoryUpdater) filterTrivialMessages(messages []shared.OpenAIMessage) []shared.OpenAIMessage {
	trivialKeywords := []string{"你好", "嗨", "谢谢", "再见", "好的", "知道了", "嗯", "哦", "啊"}
	var filtered []shared.OpenAIMessage

	for _, msg := range messages {
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if !ok {
			continue
		}

		content := strings.ToLower(*contentStr)

		if len(content) < 10 {
			isTrivial := false
			for _, keyword := range trivialKeywords {
				if strings.Contains(content, keyword) {
					isTrivial = true
					break
				}
			}
			if isTrivial {
				continue
			}
		}

		filtered = append(filtered, msg)
	}

	return filtered
}

func (u *SmartMemoryUpdater) shouldTriggerUpdate() bool {
	if len(u.messageBuffer) >= u.batchSize {
		return true
	}

	if len(u.messageBuffer) >= u.maxBatchSize {
		return true
	}

	u.updateMutex.Lock()
	defer u.updateMutex.Unlock()

	if time.Since(u.lastUpdateTime) >= u.minUpdateInterval && len(u.messageBuffer) > 0 {
		return true
	}

	return false
}

func (u *SmartMemoryUpdater) processBatch(ctx context.Context, oldMemory MemoryContent, messages []shared.OpenAIMessage) (MemoryContent, error) {
	result, err := u.llmUpdater.Update(ctx, oldMemory, messages)
	if err != nil {
		return oldMemory, err
	}

	u.updateMutex.Lock()
	u.lastUpdateTime = time.Now()
	u.updateMutex.Unlock()

	return result, nil
}

type LLMMemoryUpdater struct {
	client    openai.Client
	modelConf shared.ModelConfig
}

func NewLLMMemoryUpdater(modelConf shared.ModelConfig) *LLMMemoryUpdater {
	return &LLMMemoryUpdater{
		client:    llm.NewLLMClient(modelConf),
		modelConf: modelConf,
	}
}

func (u *LLMMemoryUpdater) Update(ctx context.Context, oldMemory MemoryContent, newMessages []shared.OpenAIMessage) (MemoryContent, error) {
	if len(newMessages) == 0 {
		return oldMemory, nil
	}

	var b strings.Builder
	for _, msg := range newMessages {
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if !ok {
			continue
		}

		b.WriteString(shared.GetRoleName(msg))
		b.WriteString(": ")
		b.WriteString(*contentStr)
		b.WriteString("\n")
	}

	prompt := updateMemoryPrompt
	prompt = strings.ReplaceAll(prompt, "{current_memory}", oldMemory.String())
	prompt = strings.ReplaceAll(prompt, "{new_messages}", b.String())

	request := openai.ChatCompletionNewParams{
		Model: u.modelConf.Model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	resp, err := u.client.Chat.Completions.New(ctx, request)
	if err != nil {
		log.Printf("failed to update memory through llm: %v", err)
		return oldMemory, err
	}
	if len(resp.Choices) == 0 {
		log.Printf("no choices returned, resp: %s", resp.RawJSON())
		return oldMemory, nil
	}

	respContent := resp.Choices[0].Message.Content
	newMemory := MemoryContent{}
	newMemory.GlobalMemory = extractXMLTag(respContent, "global")
	newMemory.ConversationMemory = extractXMLTag(respContent, "workspace")

	return newMemory, nil
}

func extractXMLTag(content, tagName string) string {
	pattern := regexp.MustCompile(`<` + regexp.QuoteMeta(tagName) + `>([\s\S]*?)</` + regexp.QuoteMeta(tagName) + `>`)
	matches := pattern.FindStringSubmatch(content)
	if len(matches) < 2 {
		return ""
	}
	return strings.TrimSpace(matches[1])
}

const updateMemoryPrompt = `You are a memory management system for an AI coding assistant. Update the memory based on new messages.

Current Memory:
{current_memory}

New Messages:
{new_messages}

Output Format:
<global>
[Global memory content in Markdown]
</global>

<workspace>
[Workspace memory content in Markdown]
</workspace>

Guidelines:
- Keep content concise but informative
- Use ## for sections, - for bullet points, **bold** for emphasis
- Only update what's necessary
- Preserve existing important information`
