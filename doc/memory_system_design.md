# Memory System Design Document

## 1. Overview

### 1.1 Purpose

The Memory System is a core component of the TianNiu AI Agent that provides multi-level memory management capabilities, enabling the agent to maintain context across conversations and retrieve relevant information from past interactions.

### 1.2 Design Goals

- **Multi-level Memory**: Support working memory, short-term memory, and long-term memory
- **Efficient Updates**: Minimize LLM calls through batch processing and intelligent triggering
- **Semantic Retrieval**: Enable similarity-based search for long-term memory
- **Scalability**: Support concurrent access and large-scale memory storage
- **Fault Tolerance**: Handle failures gracefully with proper error handling

## 2. Architecture Design

### 2.1 Three-Level Memory Hierarchy

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

### 2.2 Component Responsibilities

| Level | Responsibility | Storage | Update Frequency |
|-------|----------------|---------|------------------|
| **Working Memory** | Store current conversation messages for prompt building | In-memory | Real-time |
| **Short-term Memory** | Store user preferences and conversation-level memory | Database (cached) | Batch (30s / 20 messages) |
| **Long-term Memory** | Store conversation summaries for semantic retrieval | Vector DB | Hybrid strategy |

## 3. Component Details

### 3.1 WorkingMemory

**Location**: `pkg/agent/memory/working.go`

**Structure**:
```go
type WorkingMemory struct {
    messages  []shared.OpenAIMessage
    maxLength int
    turnCount int
}
```

**Key Features**:
- In-memory circular buffer with configurable max length (default: 100)
- Real-time message appending
- Automatic truncation when exceeding max length
- Thread-safe operations

### 3.2 ShortTermMemory

**Location**: `pkg/agent/memory/shortterm.go`

**Structure**:
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

**Key Features**:
- Per-conversation message buffering (max 20 messages per conversation)
- Periodic flush (every 30 seconds)
- Integration with SmartMemoryUpdater for LLM-based memory updates
- Global vs Conversation-level memory separation

### 3.3 LongTermMemoryManager

**Location**: `pkg/agent/memory/longterm.go`

**Structure**:
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
}
```

**Key Features**:
- Vector database integration (PostgreSQL + pgvector)
- Embedding-based semantic search
- Rerank service integration for improved retrieval accuracy
- Hybrid save strategy (see Section 3.4)

### 3.4 Save Strategy Configuration

```go
type StrategyConfig struct {
    QuickSaveRounds          int     // Minimum rounds before quick save (default: 5)
    RegularSaveRounds        int     // Regular save interval (default: 10)
    ForceSaveRounds          int     // Force save threshold (default: 20)
    MinTokenThreshold        int     // Minimum tokens before save (default: 500)
    TopicSimilarityThreshold float32 // Topic change threshold (default: 0.7)
}
```

**Save Triggers**:
1. **Force Save**: When `roundsSinceLastSave >= ForceSaveRounds`
2. **Important Content**: When important keywords detected AND `roundsSinceLastSave >= QuickSaveRounds`
3. **Token Threshold**: When `accumulatedTokens >= MinTokenThreshold` AND `roundsSinceLastSave >= QuickSaveRounds`
4. **Topic Change**: When cosine similarity < `TopicSimilarityThreshold` AND `roundsSinceLastSave >= QuickSaveRounds`

### 3.5 SmartMemoryUpdater

**Location**: `pkg/agent/memory/update.go`

**Structure**:
```go
type SmartMemoryUpdater struct {
    llmUpdater        *LLMMemoryUpdater
    batchSize         int           // default: 3
    maxBatchSize      int           // default: 10
    minUpdateInterval time.Duration // default: 60s
    messageBuffer     []shared.OpenAIMessage
    lastUpdateTime    time.Time
}
```

**Key Features**:
- Trivial message filtering (short greetings like "hello", "thank you")
- Batch processing to minimize LLM calls
- Time-based update triggering (minimum 60 seconds between updates)
- Memory format conversion (XML tags: `<global>` and `<workspace>`)

## 4. Data Flow

### 4.1 Memory Update Flow

```
User Message
    │
    ▼
MultiLevelMemory.Update()
    │
    ├─→ WorkingMemory.Add()
    │       │
    │       ▼
    │   [Store in in-memory buffer]
    │
    ├─→ ShortTermMemory.Add()
    │       │
    │       ▼
    │   [Buffer messages (max 20 per conv)]
    │       │
    │       ├─→ [Every 30s] → Flush() → SmartMemoryUpdater.Update() → Database
    │       │
    │       └─→ [Every message] → SmartMemoryUpdater.Update()
    │                   │
    │                   └─→ [Batch size >= 3 OR Time >= 60s] → LLM Call
    │
    └─→ LongTermMemory.ProcessConversation()
            │
            ▼
        [Buffer messages (max 50)]
            │
            ├─→ [Check save triggers]
            │       │
            │       ├─→ Force Save?
            │       ├─→ Important Content?
            │       ├─→ Token Threshold?
            │       └─→ Topic Changed?
            │
            └─→ [Save triggered] → Generate Summary → Embed → Vector DB
```

### 4.2 Prompt Building Flow

```
BuildSystemPrompt()
    │
    ├─→ GetShortTermMemory()
    │       │
    │       ▼
    │   [Retrieve from Database]
    │       │
    │       └─→ {memory}
    │
    ├─→ GetWorkingMemory()
    │       │
    │       ▼
    │   [Retrieve from in-memory buffer]
    │       │
    │       └─→ {working_memory}
    │
    └─→ GetLongTermMemory(query)
            │
            ▼
        [Semantic Search]
            │
            ├─→ Embed Query
            ├─→ Vector Search (top 10)
            ├─→ Rerank (top 5)
            │
            └─→ {long_term_memory}
```

## 5. API Specification

### 5.1 Memory Interface

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

### 5.2 LongTermMemoryProvider Interface

```go
type LongTermMemoryProvider interface {
    ProcessConversation(ctx context.Context, userId, conversationId string, 
                        messages []shared.OpenAIMessage, turnCount int) error
    GetMemoryPrompt(ctx context.Context, userId, query string) (string, error)
}
```

## 6. Configuration

### 6.1 Configuration Structure

```yaml
long_term_memory:
  enabled: true
  vector_db:
    host: localhost
    port: 5432
    user: admin
    password: password
    database: memory_db
    dimension: 1536
  embedding_service:
    api_key: ""
    base_url: http://localhost:8000/v1
    model: text-embedding-ada-002
    dimensions: 1536
  rerank_service:
    api_key: ""
    base_url: http://localhost:8000/v1
    model: cross-encoder/ms-marco-MiniLM-L-6-v2
  strategy:
    quick_save_rounds: 5
    regular_save_rounds: 10
    force_save_rounds: 20
    min_token_threshold: 500
    topic_similarity_threshold: 0.7
```

### 6.2 Default Values

| Parameter | Default | Description |
|-----------|---------|-------------|
| `quick_save_rounds` | 5 | Minimum rounds for quick save triggers |
| `regular_save_rounds` | 10 | Regular save interval |
| `force_save_rounds` | 20 | Force save threshold |
| `min_token_threshold` | 500 | Minimum tokens before save |
| `topic_similarity_threshold` | 0.7 | Topic change detection threshold |
| `working_memory_max_length` | 100 | Max messages in working memory |
| `short_term_buffer_max` | 20 | Max messages per conversation buffer |
| `long_term_buffer_max` | 50 | Max messages in long-term buffer |
| `flush_interval` | 30s | Short-term memory flush interval |
| `min_update_interval` | 60s | Minimum interval between LLM updates |

## 7. Deployment Considerations

### 7.1 Dependencies

| Component | Required | Purpose |
|-----------|----------|---------|
| PostgreSQL + pgvector | Required for long-term memory | Vector storage and similarity search |
| Embedding Service | Required for long-term memory | Text-to-vector conversion |
| Rerank Service | Optional | Improve retrieval accuracy |
| LLM Service | Required for memory updates and summarization | Core AI capabilities |

### 7.2 Scalability

- **Horizontal Scaling**: The memory system is stateless and can be scaled horizontally
- **Database Optimization**: Use read replicas for high-traffic retrieval scenarios
- **Caching**: Implement Redis caching for frequently accessed short-term memory

### 7.3 Fault Tolerance

- **Buffer Persistence**: Consider persisting message buffers to disk for crash recovery
- **Retry Mechanism**: Implement retry logic for external service calls (embedding, rerank)
- **Circuit Breaker**: Add circuit breaker pattern to prevent cascading failures

### 7.4 Security

- **Data Encryption**: Encrypt sensitive memory data at rest and in transit
- **Access Control**: Implement role-based access control for memory retrieval
- **Audit Logs**: Maintain logs for memory access and modifications

## 8. Monitoring & Observability

### 8.1 Metrics

| Metric | Description |
|--------|-------------|
| `memory_update_count` | Total number of memory updates |
| `llm_call_count` | Number of LLM calls for memory updates |
| `embedding_call_count` | Number of embedding service calls |
| `memory_retrieval_count` | Number of memory retrieval operations |
| `avg_memory_size` | Average memory size per user |
| `save_trigger_distribution` | Distribution of save triggers (force/important/token/topic) |

### 8.2 Logging

- **Info**: Memory save events with reason
- **Warn**: Failed memory operations
- **Debug**: Detailed flow information

## 9. Future Enhancements

1. **Memory Compression**: Implement compression for large memory content
2. **Memory Expiration**: Add TTL for outdated memory entries
3. **Multi-modal Memory**: Support for image and audio memory
4. **Memory Editing**: Allow users to edit or delete memory entries
5. **Distributed Memory**: Support for distributed vector databases