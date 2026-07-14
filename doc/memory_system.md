# TianNiu Backend System Design - Memory System

## 1. Overview

### 1.1 Purpose

The Memory System provides multi-level memory management capabilities for the TianNiu AI Agent, enabling context retention across conversations and intelligent retrieval of relevant information.

### 1.2 Three-Level Memory Hierarchy

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        MultiLevelMemory                               │
│                    (Unified Memory Interface)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────────┐  │
│  │ WorkingMemory   │    │ ShortTermMemory │    │ LongTermMemory      │  │
│  │ (当前对话上下文)  │    │  (用户偏好/会话)   │    │  (对话摘要/语义检索) │  │
│  ├─────────────────┤    ├─────────────────┤    ├─────────────────────┤  │
│  │ • 内存存储       │    │ • 数据库存储     │    │ • 向量数据库存储    │  │
│  │ • 实时更新       │    │ • 批量缓冲(30s)  │    │ • 混合策略触发     │  │
│  │ • 最多100条消息  │    │ • 消息过滤       │    │ • 最多50条缓冲     │  │
│  └─────────────────┘    └─────────────────┘    └─────────────────────┘  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.3 Memory Interface

```go
type Memory interface {
    // GetShortTermMemory retrieves user preferences and conversation memory
    GetShortTermMemory(userId, conversationId string) string
    
    // GetWorkingMemory retrieves current conversation messages
    GetWorkingMemory() []shared.OpenAIMessage
    
    // Update processes new messages and updates all memory levels
    Update(ctx context.Context, userId, conversationId string, 
           newMessages []shared.OpenAIMessage, turnCount int) error
    
    // GetLongTermMemory retrieves relevant historical summaries
    GetLongTermMemory(ctx context.Context, userId, query string) (string, error)
    
    // Flush writes buffered short-term memory to storage
    Flush(ctx context.Context, userId, conversationId string) error
}
```

## 2. Working Memory

### 2.1 Responsibilities

- Store current conversation messages for immediate context
- Support prompt building with recent conversation history
- Thread-safe operations

### 2.2 Structure

```go
type WorkingMemory struct {
    messages  []shared.OpenAIMessage
    maxLength int
    turnCount int
    sync.Mutex
}
```

### 2.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewWorkingMemory(maxLength)` | Create new working memory |
| `Add(messages)` | Add messages to buffer |
| `GetMessages()` | Retrieve all messages |
| `Clear()` | Clear all messages |
| `GetTurnCount()` | Get current turn count |

### 2.4 Implementation Details

```go
func (w *WorkingMemory) Add(messages []shared.OpenAIMessage) error {
    w.Lock()
    defer w.Unlock()
    
    w.messages = append(w.messages, messages...)
    
    if w.maxLength > 0 && len(w.messages) > w.maxLength {
        w.messages = w.messages[len(w.messages)-w.maxLength:]
    }
    
    w.turnCount += len(messages)
    return nil
}
```

## 3. Short-Term Memory

### 3.1 Responsibilities

- Store user preferences and conversation-level memory
- Batch processing to minimize LLM calls
- Periodic flush to database

### 3.2 Structure

```go
type ShortTermMemory struct {
    storage       storage.Storage
    updater       *SmartMemoryUpdater
    buffer        map[string][]shared.OpenAIMessage  // key: userId:conversationId
    bufferMutex   sync.Mutex
    lastFlushTime time.Time
    flushInterval time.Duration  // default: 30s
}
```

### 3.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewShortTermMemory()` | Create new short-term memory |
| `Add(userId, conversationId, messages)` | Add messages to buffer |
| `Flush(userId, conversationId)` | Flush buffer to storage |
| `FlushAll()` | Flush all buffers |
| `GetMemory(userId, conversationId)` | Retrieve memory content |

### 3.4 Buffering Strategy

- **Per-conversation buffer**: Max 20 messages per conversation
- **Flush interval**: Every 30 seconds
- **Memory format**: XML with `<global>` and `<workspace>` tags

## 4. Long-Term Memory

### 4.1 Responsibilities

- Store conversation summaries for semantic retrieval
- Manage vector embeddings
- Implement intelligent save triggers

### 4.2 Structure

```go
type LongTermMemoryManager struct {
    vectorStore      *rag.PGVectorStore
    embeddingService *rag.HTTPEmbeddingService
    rerankService    *rag.HTTPRerankService
    llmClient        openai.Client
    config           StrategyConfig
    
    messageBuffer     []shared.OpenAIMessage  // max 50 messages
    lastTopicEmbedding rag.Vector
    lastSaveRound      int
    accumulatedTokens  int
    sync.Mutex
}
```

### 4.3 Save Strategy Configuration

```go
type StrategyConfig struct {
    QuickSaveRounds          int     // Default: 5
    RegularSaveRounds        int     // Default: 10
    ForceSaveRounds          int     // Default: 20
    MinTokenThreshold        int     // Default: 500
    TopicSimilarityThreshold float32 // Default: 0.7
    BufferMaxLength          int     // Default: 50
}
```

### 4.4 Save Triggers

| Trigger | Condition |
|---------|-----------|
| **Force Save** | `roundsSinceLastSave >= ForceSaveRounds` |
| **Important Content** | Important keywords detected AND `rounds >= QuickSaveRounds` |
| **Token Threshold** | `accumulatedTokens >= MinTokenThreshold` AND `rounds >= QuickSaveRounds` |
| **Topic Change** | cosine similarity < `TopicSimilarityThreshold` AND `rounds >= QuickSaveRounds` |

### 4.5 Memory Retrieval Flow

```
GetLongTermMemory(ctx, userId, query)
    │
    ├─→ 1. Embed query text
    │       └─→ embeddingService.Embed(query)
    │
    ├─→ 2. Vector search
    │       └─→ vectorStore.Search(embedding, topK=10)
    │
    ├─→ 3. Rerank results
    │       └─→ rerankService.Rerank(query, results, topN=5)
    │
    └─→ 4. Format and return
            └─→ Join summaries with relevance scores
```

## 5. Smart Memory Updater

### 5.1 Responsibilities

- Filter trivial messages
- Batch processing to minimize LLM calls
- Time-based update triggering

### 5.2 Structure

```go
type SmartMemoryUpdater struct {
    llmUpdater        *LLMMemoryUpdater
    batchSize         int           // Default: 3
    maxBatchSize      int           // Default: 10
    minUpdateInterval time.Duration // Default: 60s
    messageBuffer     []shared.OpenAIMessage
    lastUpdateTime    time.Time
}
```

### 5.3 Message Filtering

```go
func (u *SmartMemoryUpdater) filterMessages(messages []shared.OpenAIMessage) []shared.OpenAIMessage {
    trivialKeywords := []string{"你好", "嗨", "谢谢", "再见", "好的", "知道了", "嗯", "哦", "啊"}
    var filtered []shared.OpenAIMessage
    
    for _, msg := range messages {
        content := extractContent(msg)
        if len(content) < 10 {
            isTrivial := false
            for _, kw := range trivialKeywords {
                if strings.Contains(content, kw) {
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
```

### 5.4 Batch Update Logic

```go
func (u *SmartMemoryUpdater) Update(ctx context.Context, oldMemory MemoryContent, 
                                     newMessages []shared.OpenAIMessage) (MemoryContent, error) {
    // 1. Filter trivial messages
    filtered := u.filterMessages(newMessages)
    
    // 2. Add to buffer
    u.messageBuffer = append(u.messageBuffer, filtered...)
    if len(u.messageBuffer) > u.maxBatchSize {
        u.messageBuffer = u.messageBuffer[len(u.messageBuffer)-u.maxBatchSize:]
    }
    
    // 3. Check conditions
    now := time.Now()
    hasEnoughMessages := len(u.messageBuffer) >= u.batchSize
    hasEnoughTime := now.Sub(u.lastUpdateTime) >= u.minUpdateInterval
    
    if !hasEnoughMessages && !hasEnoughTime {
        return oldMemory, nil
    }
    
    // 4. Update with LLM
    result, err := u.llmUpdater.Update(ctx, oldMemory, u.messageBuffer)
    if err != nil {
        return oldMemory, err
    }
    
    // 5. Reset
    u.messageBuffer = make([]shared.OpenAIMessage, 0)
    u.lastUpdateTime = now
    
    return result, nil
}
```

## 6. Vector Database Integration

### 6.1 Vector Store Interface

```go
type VectorStore interface {
    Insert(ctx context.Context, userId, conversationId string, 
           content, summary string, embedding rag.Vector, roundNumber int) error
    Search(ctx context.Context, userId string, queryEmbedding rag.Vector, topK int) ([]MemoryChunk, error)
    Delete(ctx context.Context, userId, conversationId string) error
}
```

### 6.2 Memory Chunk Structure

```go
type MemoryChunk struct {
    ID             string
    UserID         string
    ConversationID string
    Content        string
    Summary        string
    Embedding      rag.Vector
    RoundNumber    int
    CreatedAt      time.Time
}
```

## 7. RAG Components

### 7.1 Embedding Service

```go
type HTTPEmbeddingService struct {
    client     *resty.Client
    apiKey     string
    baseURL    string
    model      string
    dimensions int
}

func (e *HTTPEmbeddingService) Embed(ctx context.Context, text string) (rag.Vector, error)
```

### 7.2 Rerank Service

```go
type HTTPRerankService struct {
    client  *resty.Client
    apiKey  string
    baseURL string
    model   string
}

func (r *HTTPRerankService) Rerank(ctx context.Context, query string, 
                                   documents []string, topN int) ([]RerankResult, error)
```

## 8. Memory Update Flow

```
MultiLevelMemory.Update(ctx, userId, conversationId, messages, turnCount)
    │
    ├─→ WorkingMemory.Add(messages)
    │       │
    │       ▼
    │   [Store in circular buffer, max 100 messages]
    │
    ├─→ ShortTermMemory.Add(userId, conversationId, messages)
    │       │
    │       ▼
    │   [Buffer, max 20 per conversation]
    │       │
    │       ├─→ [Every 30s] → Flush() → SmartMemoryUpdater → Database
    │       │
    │       └─→ [Every message] → SmartMemoryUpdater.Update()
    │                   │
    │                   └─→ [Batch >= 3 OR Time >= 60s] → LLM Call
    │
    └─→ LongTermMemory.ProcessConversation(userId, conversationId, messages, turnCount)
            │
            ▼
        [Buffer messages, max 50]
            │
            ├─→ CheckSaveTriggers()
            │       ├─→ Force Save?
            │       ├─→ Important Content?
            │       ├─→ Token Threshold?
            │       └─→ Topic Changed?
            │
            └─→ [Save] → GenerateSummary() → Embed() → VectorDB.Insert()
```

## 9. Configuration

### 9.1 Default Configuration

| Parameter | Default | Description |
|-----------|---------|-------------|
| `working_memory_max_length` | 100 | Max messages in working memory |
| `short_term_buffer_max` | 20 | Max messages per conversation buffer |
| `short_term_flush_interval` | 30s | Flush interval |
| `long_term_buffer_max` | 50 | Max messages in long-term buffer |
| `quick_save_rounds` | 5 | Minimum rounds for quick save |
| `force_save_rounds` | 20 | Force save threshold |
| `min_token_threshold` | 500 | Minimum tokens before save |
| `topic_similarity_threshold` | 0.7 | Topic change threshold |

### 9.2 YAML Configuration

```yaml
memory:
  working_memory:
    max_length: 100
  
  short_term:
    buffer_max: 20
    flush_interval: 30s
  
  long_term:
    enabled: true
    buffer_max: 50
    strategy:
      quick_save_rounds: 5
      regular_save_rounds: 10
      force_save_rounds: 20
      min_token_threshold: 500
      topic_similarity_threshold: 0.7
```