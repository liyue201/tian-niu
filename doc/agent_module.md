# TianNiu Backend System Design - Agent Module

## 1. Agent Manager

### 1.1 Responsibilities

- Agent lifecycle management (create, retrieve, destroy)
- Conversation session management
- Policy enforcement coordination
- Multi-tenancy support

### 1.2 Structure

```go
type Manager struct {
    repo         *repository.SQLStore
    modelConf    shared.ModelConfig
    client       openai.Client
    tools        []tool.Tool
    systemPrompt string
    mcpClients   []*mcp.Client
    policies     []context.Policy
    memory       memory.Memory
    skillManager *skill.Manager
    
    agents       map[string]*Agent  // key: conversationId
    sync.RWMutex
}
```

### 1.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewManager()` | Create a new agent manager |
| `GetAgent(userId, conversationId)` | Retrieve or create agent for conversation |
| `RemoveAgent(conversationId)` | Remove agent and cleanup |
| `loadSkillTools(userId)` | Load skills for user |

### 1.4 Agent Pool Management

```go
func (m *Manager) GetAgent(userId, conversationId string) *Agent {
    // 1. Check cache
    m.RLock()
    agent, ok := m.agents[conversationId]
    if ok {
        m.RUnlock()
        return agent
    }
    m.RUnlock()
    
    // 2. Create new agent with double-checked locking
    m.Lock()
    defer m.Unlock()
    
    // 3. Create context engine
    engine := context.NewContextEngine(m.memory, userId, conversationId, m.policies, m.repo)
    
    // 4. Load skills
    skillTools := m.loadSkillTools(userId)
    
    // 5. Create agent
    agent = NewAgent(m.modelConf, m.systemPrompt, m.tools, skillTools, m.mcpClients, engine)
    m.agents[conversationId] = agent
    
    return agent
}
```

## 2. Context Engine

### 2.1 Responsibilities

- Message management and token counting
- Policy application (truncate, etc.)
- System prompt building
- Turn management

### 2.2 Structure

```go
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
```

### 2.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewContextEngine()` | Create new context engine |
| `Init(systemPrompt, budget)` | Initialize engine |
| `CommitTurn(draft, usage)` | Commit turn and update memory |
| `BuildSystemPrompt()` | Build system prompt with memory |
| `ProcessMessages(messages)` | Process incoming messages |
| `applyPolicies()` | Apply context policies |

### 2.4 Turn Management

```go
func (e *Engine) CommitTurn(ctx context.Context, draft TurnDraft, usage Usage) error {
    // 1. Add messages to internal buffer
    for _, msg := range draft.NewMessages {
        e.messages = append(e.messages, messageWrap{Message: msg, Tokens: CountTokens(msg)})
    }
    
    // 2. Recount tokens
    e.recountTokens()
    
    // 3. Apply policies (truncate, etc.)
    if err := e.applyPolicies(ctx); err != nil {
        return err
    }
    
    // 4. Increment turn count
    e.turnCount++
    
    // 5. Update memory
    err := e.memory.Update(ctx, e.userId, e.conversationId, draft.NewMessages, e.turnCount)
    if err != nil {
        return err
    }
    
    return nil
}
```

### 2.5 System Prompt Building

```go
func (e *Engine) BuildSystemPrompt() string {
    replaceMap := make(map[string]string)
    
    if e.memory != nil {
        // Short-term memory (user preferences)
        replaceMap["{memory}"] = e.memory.GetShortTermMemory(e.userId, e.conversationId)
        
        // Working memory (current conversation)
        workingMessages := e.memory.GetWorkingMemory()
        replaceMap["{working_memory}"] = e.formatWorkingMemory(workingMessages)
        
        // Long-term memory (semantic retrieval)
        longTermMem, err := e.memory.GetLongTermMemory(ctx, e.userId, e.getCurrentUserQuery())
        if err != nil {
            log.Warnf("failed to get long-term memory: %v", err)
            longTermMem = ""
        }
        replaceMap["{long_term_memory}"] = longTermMem
    }
    
    // Replace placeholders in template
    prompt := e.systemPromptTemplate
    for k, v := range replaceMap {
        prompt = strings.ReplaceAll(prompt, k, v)
    }
    
    return prompt
}
```

### 2.6 Policy Application

```go
func (e *Engine) applyPolicies(ctx context.Context) error {
    for _, policy := range e.policies {
        result, err := policy.Apply(ctx, PolicyContext{
            Messages:       e.messages,
            ContextTokens:  e.contextTokens,
            ContextWindow:  e.contextWindow,
        })
        if err != nil {
            return fmt.Errorf("apply policy %s: %w", policy.Name(), err)
        }
        e.messages = result.Messages
        e.recountTokens()
    }
    return nil
}
```

## 3. Agent Core

### 3.1 Responsibilities

- LLM interaction orchestration
- Tool/Skill execution coordination
- Response generation
- Error handling

### 3.2 Structure

```go
type Agent struct {
    modelConf    shared.ModelConfig
    systemPrompt string
    tools        []tool.Tool
    mcpClients   []*mcp.Client
    engine       *context.Engine
    client       openai.Client
}
```

### 3.3 Key Methods

| Method | Description |
|--------|-------------|
| `NewAgent()` | Create new agent instance |
| `Chat(ctx, messages)` | Process chat messages |
| `ExecuteTool(toolCall)` | Execute tool/skill |
| `generateResponse()` | Generate LLM response |

### 3.4 Message Processing Flow

```
Agent.Chat(ctx, messages)
    │
    ├─→ Engine.ProcessMessages(messages)
    │
    ├─→ Engine.BuildSystemPrompt()
    │
    ├─→ LLMClient.ChatCompletion()
    │       │
    │       ├─→ [Tool call detected] → ExecuteTool() → Recurse
    │       │
    │       └─→ [Direct response] → Return
    │
    └─→ Engine.CommitTurn()
```

## 4. Policy System

### 4.1 Policy Interface

```go
type Policy interface {
    Name() string
    Apply(ctx context.Context, ctx PolicyContext) (PolicyResult, error)
}
```

### 4.2 Built-in Policies

| Policy | Description |
|--------|-------------|
| **TruncatePolicy** | Truncate messages when token limit exceeded |
| **FilterPolicy** | Filter out irrelevant messages |
| **SummarizePolicy** | Summarize long conversations |

### 4.3 Policy Context

```go
type PolicyContext struct {
    Messages       []messageWrap
    ContextTokens  int
    ContextWindow  int
    TurnCount      int
}

type PolicyResult struct {
    Messages      []messageWrap
    ShouldContinue bool
}
```

## 5. Token Management

### 5.1 Token Counting

```go
func CountTokens(msg shared.OpenAIMessage) int {
    contentAny := msg.GetContent().AsAny()
    contentStr, ok := contentAny.(*string)
    if !ok {
        return 0
    }
    // Approximate token count (1 token ≈ 4 chars for English)
    return len([]rune(*contentStr)) / 4
}
```

### 5.2 Context Window Management

```go
func (e *Engine) recountTokens() {
    e.contextTokens = 0
    for _, msg := range e.messages {
        e.contextTokens += msg.Tokens
    }
}
```

## 6. Error Handling

### 6.1 Error Types

| Error | Description |
|-------|-------------|
| `ErrPolicyFailed` | Policy application failed |
| `ErrMemoryUpdateFailed` | Memory update failed |
| `ErrLLMCallFailed` | LLM API call failed |
| `ErrToolExecutionFailed` | Tool execution failed |

### 6.2 Error Handling Strategy

- **Retry**: Retry transient errors (LLM timeouts)
- **Fallback**: Use fallback LLM on failure
- **Graceful Degradation**: Continue with reduced functionality
- **Logging**: Detailed error logging for debugging