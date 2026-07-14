# TianNiu Backend System Design - Architecture

## 1. High-Level Architecture

### 1.1 System Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Client Layer                                  │
│  [Web UI] [CLI] [API Clients] [Mobile Apps]                            │
└──────────────────────────────────────────────────────────────────────────┘
                                    │ HTTP/gRPC/WebSocket
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           API Gateway                                   │
│  [Authentication] [Rate Limiting] [Request Routing] [Logging]          │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Conversation    │    │      Skill        │    │       MCP         │
│      API          │    │      API          │    │      API          │
│  [REST/WebSocket] │    │     [REST]        │    │     [REST]        │
└───────────────────┘    └───────────────────┘    └───────────────────┘
          │                         │                         │
          ▼                         ▼                         ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          Core Services                                  │
│                                                                         │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                        Agent Manager                            │   │
│  │  [Agent Pool] [Session Management] [Policy Enforcement]         │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│                              ▼                                          │
│  ┌──────────────────────────────────────────────────────────────────┐   │
│  │                      Context Engine                              │   │
│  │  [Message Management] [Token Counting] [Prompt Building]         │   │
│  │  [Policy Application] [Turn Management]                          │   │
│  └──────────────────────────────────────────────────────────────────┘   │
│                              │                                          │
│          ┌───────────────────┼───────────────────┐                      │
│          ▼                   ▼                   ▼                      │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐                │
│  │   Memory    │    │   Skill     │    │    LLM       │                │
│  │  Manager    │    │  Manager    │    │  Client      │                │
│  └─────────────┘    └─────────────┘    └──────────────┘                │
│          │                   │                   │                      │
│          ▼                   ▼                   ▼                      │
│  ┌─────────────┐    ┌─────────────┐    ┌──────────────┐                │
│  │  Tool       │    │    MCP      │    │   RAG        │                │
│  │  Registry   │    │  Manager    │    │  Components  │                │
│  └─────────────┘    └─────────────┘    └──────────────┘                │
│                                                                         │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   PostgreSQL      │    │      Redis        │    │    Vector DB      │
│  (Conversation    │    │   (Cache/Queue)   │    │   (Long-term     │
│    History)       │    │                   │    │    Memory)        │
└───────────────────┘    └───────────────────┘    └───────────────────┘
```

### 1.2 Layered Architecture

| Layer | Components | Responsibility |
|-------|------------|----------------|
| **Presentation** | API Gateway | Request handling, authentication, routing |
| **Application** | Conversation API, Skill API, MCP API | Business logic, request processing |
| **Domain** | Agent Manager, Context Engine | Core business logic |
| **Service** | Memory Manager, Skill Manager, LLM Client | Domain services |
| **Infrastructure** | Database, Redis, Vector DB | Data storage and caching |

## 2. Component Interaction Flow

### 2.1 Message Processing Flow

```
User Message
    │
    ▼
API Gateway → Authentication → Rate Limiting
    │
    ▼
Conversation API → Agent Manager.GetAgent()
    │
    ▼
Context Engine.ProcessMessages()
    │
    ├─→ Memory.Update()
    │       ├─→ WorkingMemory.Add()
    │       ├─→ ShortTermMemory.Add()
    │       └─→ LongTermMemory.ProcessConversation()
    │
    ├─→ BuildSystemPrompt()
    │       ├─→ GetShortTermMemory()
    │       ├─→ GetWorkingMemory()
    │       └─→ GetLongTermMemory()
    │
    ├─→ LLMClient.ChatCompletion()
    │
    └─→ Tool/Skill Execution (if needed)
            ├─→ SkillManager.Execute()
            └─→ Tool.Execute()
    │
    ▼
Context Engine.CommitTurn()
    │
    ▼
Database.SaveMessage()
    │
    ▼
Response to Client
```

### 2.2 Skill Execution Flow

```
LLM Response with <function_call>
    │
    ▼
SkillManager.Resolve(skill_id)
    │
    ▼
Skill.Validate()
    │
    ├─→ [Valid] → Skill.Execute()
    │       │
    │       ├─→ Tool.Execute()
    │       ├─→ MCP.Call()
    │       │
    │       ▼
    │   Return Result
    │
    └─→ [Invalid] → Return Error
    │
    ▼
Context Engine.ProcessToolResult()
    │
    ▼
LLMClient.ChatCompletion() (with tool result)
```

## 3. Key Design Patterns

### 3.1 Factory Pattern

Used for creating Agent instances with different configurations:

```go
func NewAgent(modelConf shared.ModelConfig, systemPrompt string, 
              tools []tool.Tool, mcpClients []*mcp.Client, engine *context.Engine) *Agent
```

### 3.2 Strategy Pattern

Used for memory save strategies:

```go
type StrategyConfig struct {
    QuickSaveRounds          int
    ForceSaveRounds          int
    MinTokenThreshold        int
    TopicSimilarityThreshold float32
}
```

### 3.3 Observer Pattern

Used for policy events:

```go
func (e *Engine) SetPolicyEventHook(hook func(policyName string, running bool, err error))
```

### 3.4 Builder Pattern

Used for constructing complex prompts:

```go
func (e *Engine) BuildSystemPrompt() string
```

## 4. Data Flow Diagrams

### 4.1 Memory Update Flow

```
User Message
    │
    ▼
MultiLevelMemory.Update()
    │
    ├─→ WorkingMemory.Add()         [In-memory, real-time]
    │
    ├─→ ShortTermMemory.Add()       [Buffered, 30s flush]
    │       │
    │       ▼
    │   SmartMemoryUpdater.Update() [Batch: 3 msg OR 60s]
    │       │
    │       ▼
    │   Database.Store()
    │
    └─→ LongTermMemory.ProcessConversation() [Hybrid strategy]
            │
            ├─→ [Check triggers]
            │       ├─→ Force Save (20 rounds)
            │       ├─→ Important Content (5 rounds)
            │       ├─→ Token Threshold (500 tokens)
            │       └─→ Topic Change (similarity < 0.7)
            │
            └─→ [Save] → GenerateSummary() → Embed() → VectorDB.Insert()
```

### 4.2 Prompt Construction Flow

```
BuildSystemPrompt()
    │
    ├─→ {memory}        ← GetShortTermMemory()
    │
    ├─→ {working_memory} ← GetWorkingMemory()
    │
    └─→ {long_term_memory} ← GetLongTermMemory(query)
            │
            ├─→ Embed(query)
            ├─→ VectorDB.Search()
            ├─→ Rerank()
            │
            └─→ FormatResults()
```

## 5. Scalability Considerations

### 5.1 Horizontal Scaling

- **Stateless Services**: API Gateway, Agent Manager can be scaled horizontally
- **Shared Cache**: Redis for session management and caching
- **Database Sharding**: PostgreSQL read replicas for high availability

### 5.2 Load Balancing

- Round-robin or least-connections algorithm
- Session affinity for WebSocket connections
- Health checks for service discovery

### 5.3 Caching Strategy

| Cache Level | Cache Type | TTL |
|-------------|------------|-----|
| Short-term memory | Redis | 5 minutes |
| Conversation history | Redis | 1 hour |
| Skill metadata | Memory | Application lifetime |
| Embedding cache | Redis | 24 hours |

## 6. Fault Tolerance

### 6.1 Retry Mechanism

- LLM API calls: 3 retries with exponential backoff
- Database operations: 2 retries
- External service calls: Circuit breaker pattern

### 6.2 Fallback Strategies

- Primary LLM failure → Fallback to secondary LLM
- Vector DB failure → Skip long-term memory retrieval
- Skill execution failure → Return error gracefully

### 6.3 Graceful Degradation

- High load → Reduce memory update frequency
- Database overload → Use cached responses
- LLM rate limit → Queue requests or return error