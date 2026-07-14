package memory

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/openai/openai-go/v3"
	log "github.com/sirupsen/logrus"
	"github.com/tianniu-ai/tianniu/pkg/agent/llm"
	"github.com/tianniu-ai/tianniu/pkg/agent/rag"
	"github.com/tianniu-ai/tianniu/pkg/shared"
)

type StrategyConfig struct {
	QuickSaveRounds          int     `yaml:"quick_save_rounds"`
	RegularSaveRounds        int     `yaml:"regular_save_rounds"`
	ForceSaveRounds          int     `yaml:"force_save_rounds"`
	MinTokenThreshold        int     `yaml:"min_token_threshold"`
	TopicSimilarityThreshold float32 `yaml:"topic_similarity_threshold"`
}

func DefaultStrategyConfig() StrategyConfig {
	return StrategyConfig{
		QuickSaveRounds:          5,
		RegularSaveRounds:        10,
		ForceSaveRounds:          20,
		MinTokenThreshold:        500,
		TopicSimilarityThreshold: 0.7,
	}
}

type LongTermMemoryManager struct {
	vectorStore      *rag.PGVectorStore
	embeddingService *rag.HTTPEmbeddingService
	rerankService    *rag.HTTPRerankService
	llmClient        openai.Client
	modelConf        shared.ModelConfig
	config           StrategyConfig

	messageBuffer      []shared.OpenAIMessage
	bufferMaxLength    int
	lastTopicEmbedding rag.Vector
	lastSaveRound      int
	accumulatedTokens  int
}

func NewLongTermMemoryManager(
	vectorStore *rag.PGVectorStore,
	embeddingService *rag.HTTPEmbeddingService,
	rerankService *rag.HTTPRerankService,
	modelConf shared.ModelConfig,
	configs ...StrategyConfig,
) *LongTermMemoryManager {
	config := DefaultStrategyConfig()
	if len(configs) > 0 {
		config = configs[0]
	}

	return &LongTermMemoryManager{
		vectorStore:      vectorStore,
		embeddingService: embeddingService,
		rerankService:    rerankService,
		llmClient:        llm.NewLLMClient(modelConf),
		modelConf:        modelConf,
		config:           config,
		messageBuffer:    make([]shared.OpenAIMessage, 0),
		bufferMaxLength:  50,
		lastSaveRound:    0,
	}
}

func (m *LongTermMemoryManager) ProcessConversation(ctx context.Context, userID, conversationID string, messages []shared.OpenAIMessage, currentRound int) error {
	m.messageBuffer = append(m.messageBuffer, messages...)

	if m.bufferMaxLength > 0 && len(m.messageBuffer) > m.bufferMaxLength {
		m.messageBuffer = m.messageBuffer[len(m.messageBuffer)-m.bufferMaxLength:]
	}

	tokens := m.countTokens(messages)
	m.accumulatedTokens += tokens

	shouldSave, reason := m.shouldSaveMemory(currentRound, messages)

	if !shouldSave {
		return nil
	}

	log.Infof("Saving long-term memory for user %s, conversation %s, reason: %s", userID, conversationID, reason)

	err := m.saveMemory(ctx, userID, conversationID, m.messageBuffer, currentRound)
	if err != nil {
		return fmt.Errorf("failed to save memory: %w", err)
	}

	m.lastSaveRound = currentRound
	m.accumulatedTokens = 0
	m.messageBuffer = make([]shared.OpenAIMessage, 0)

	return nil
}

func (m *LongTermMemoryManager) shouldSaveMemory(currentRound int, messages []shared.OpenAIMessage) (bool, string) {
	roundsSinceLastSave := currentRound - m.lastSaveRound

	if roundsSinceLastSave >= m.config.ForceSaveRounds {
		return true, "force save threshold reached"
	}

	isImportant := m.detectImportance(messages)
	if isImportant && roundsSinceLastSave >= m.config.QuickSaveRounds {
		return true, "detected important content"
	}

	if m.accumulatedTokens >= m.config.MinTokenThreshold && roundsSinceLastSave >= m.config.QuickSaveRounds {
		return true, "token threshold reached"
	}

	topicChanged, err := m.detectTopicChange(messages)
	if err != nil {
		log.Warnf("Failed to detect topic change: %v", err)
	} else if topicChanged && roundsSinceLastSave >= m.config.QuickSaveRounds {
		return true, "topic changed"
	}

	return false, ""
}

func (m *LongTermMemoryManager) detectImportance(messages []shared.OpenAIMessage) bool {
	importanceKeywords := []string{
		"决定", "方案", "结论", "问题", "解决", "答案", "总结",
		"规划", "设计", "架构", "实现", "部署", "配置",
		"bug", "error", "fix", "解决办法", "解决方案",
		"如何", "怎样", "为什么", "什么是", "定义", "说明",
	}

	for _, msg := range messages {
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if !ok {
			continue
		}

		for _, keyword := range importanceKeywords {
			if strings.Contains(strings.ToLower(*contentStr), strings.ToLower(keyword)) {
				return true
			}
		}
	}

	return false
}

func (m *LongTermMemoryManager) detectTopicChange(messages []shared.OpenAIMessage) (bool, error) {
	if m.lastTopicEmbedding == nil {
		return false, nil
	}

	content := m.messagesToString(messages)
	if len(content) < 10 {
		return false, nil
	}

	log.Infof("embedding message: %v", content)
	currentEmbedding, err := m.embeddingService.Embed(context.Background(), content)
	if err != nil {
		log.Errorf("failed to embed message: %v", err)
		return false, fmt.Errorf("failed to embed message: %w", err)
	}

	similarity := cosineSimilarity(m.lastTopicEmbedding, currentEmbedding)
	log.Debugf("Topic similarity: %f, threshold: %f", similarity, m.config.TopicSimilarityThreshold)

	if similarity < m.config.TopicSimilarityThreshold {
		m.lastTopicEmbedding = currentEmbedding
		return true, nil
	}

	return false, nil
}

func cosineSimilarity(a, b rag.Vector) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, magA, magB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}

	if magA == 0 || magB == 0 {
		return 0
	}

	return dotProduct / (float32(magnitude(magA)) * float32(magnitude(magB)))
}

func magnitude(x float32) float64 {
	return math.Sqrt(float64(x))
}

func (m *LongTermMemoryManager) countTokens(messages []shared.OpenAIMessage) int {
	total := 0
	for _, msg := range messages {
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if ok {
			total += len([]rune(*contentStr)) / 2
		}
	}
	return total
}

func (m *LongTermMemoryManager) saveMemory(ctx context.Context, userID, conversationID string, messages []shared.OpenAIMessage, currentRound int) error {
	summary, err := m.generateSummary(ctx, messages)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	content := m.messagesToString(messages)

	embedding, err := m.embeddingService.Embed(ctx, summary)
	if err != nil {
		return fmt.Errorf("failed to embed summary: %w", err)
	}

	if m.lastTopicEmbedding == nil {
		m.lastTopicEmbedding = embedding
	}

	err = m.vectorStore.InsertMemory(ctx, userID, conversationID, content, summary, embedding, currentRound)
	if err != nil {
		return fmt.Errorf("failed to insert memory: %w", err)
	}

	return nil
}

func (m *LongTermMemoryManager) RetrieveMemory(ctx context.Context, userID, query string, limit int) ([]rag.MemoryMatch, error) {
	queryEmbedding, err := m.embeddingService.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}

	candidates, err := m.vectorStore.SearchMemory(ctx, userID, queryEmbedding, limit*2)
	if err != nil {
		return nil, fmt.Errorf("failed to search memory: %w", err)
	}

	if len(candidates) == 0 {
		return candidates, nil
	}

	reranked, err := m.rerankService.Rerank(ctx, query, candidates)
	if err != nil {
		return nil, fmt.Errorf("failed to rerank: %w", err)
	}

	if len(reranked) > limit {
		reranked = reranked[:limit]
	}

	return reranked, nil
}

func (m *LongTermMemoryManager) generateSummary(ctx context.Context, messages []shared.OpenAIMessage) (string, error) {
	var b strings.Builder
	for _, msg := range messages {
		roleName := shared.GetRoleName(msg)
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if !ok {
			continue
		}
		b.WriteString(roleName)
		b.WriteString(": ")
		b.WriteString(*contentStr)
		b.WriteString("\n")
	}

	prompt := `Summarize the following conversation history concisely. Focus on key topics discussed, decisions made, and important information exchanged. Keep it under 200 words.

Conversation:
` + b.String()

	resp, err := m.llmClient.Chat.Completions.New(ctx,
		openai.ChatCompletionNewParams{
			Model: m.modelConf.Model,
			Messages: []shared.OpenAIMessage{
				openai.UserMessage(prompt),
			},
		},
	)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned")
	}

	return resp.Choices[0].Message.Content, nil
}

func (m *LongTermMemoryManager) messagesToString(messages []shared.OpenAIMessage) string {
	var b strings.Builder
	for _, msg := range messages {
		roleName := shared.GetRoleName(msg)
		contentAny := msg.GetContent().AsAny()
		contentStr, ok := contentAny.(*string)
		if !ok {
			continue
		}
		b.WriteString(roleName)
		b.WriteString(": ")
		b.WriteString(*contentStr)
		b.WriteString("\n")
	}
	return b.String()
}

func (m *LongTermMemoryManager) GetMemoryPrompt(ctx context.Context, userID, query string) (string, error) {
	matches, err := m.RetrieveMemory(ctx, userID, query, 5)
	if err != nil {
		return "", err
	}

	if len(matches) == 0 {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("### Long-Term Memory\n")
	b.WriteString("Relevant information from past conversations:\n\n")

	for i, match := range matches {
		b.WriteString(fmt.Sprintf("%d. [Round %d] %s\n", i+1, match.RoundNumber, match.Summary))
	}

	return b.String(), nil
}
