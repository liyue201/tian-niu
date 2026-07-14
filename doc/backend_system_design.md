# TianNiu Backend System Design Document

## 1. System Overview

### 1.1 Purpose

TianNiu is an AI-powered coding assistant backend system that provides intelligent conversation capabilities with memory management, skill execution, and tool integration.

### 1.2 Core Features

| Feature | Description |
|---------|-------------|
| **Multi-level Memory** | Working, short-term, and long-term memory management |
| **Skill System** | Plugin-based skill execution framework |
| **Tool Integration** | Bash, file operations, and external API tools |
| **MCP Integration** | Model Context Protocol for extended capabilities |
| **LLM Orchestration** | Multi-model support with front/back model architecture |
| **Conversation Management** | Session-based conversation handling |

### 1.3 Design Principles

- **Modularity**: Clear separation of concerns between components
- **Scalability**: Stateless design for horizontal scaling
- **Extensibility**: Plugin-based architecture for skills and tools
- **Reliability**: Graceful error handling and fallback mechanisms

## 2. Architecture Design

### 2.1 High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Client Layer                                  │
│  [Web UI] [CLI] [API Clients]                                           │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           API Gateway                                   │
│  [REST API] [WebSocket] [Authentication] [Rate Limiting]               │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
          ┌─────────────────────────┼─────────────────────────┐
          ▼                         ▼                         ▼
┌───────────────────┐    ┌───────────────────┐    ┌───────────────────┐
│   Conversation    │    │      Skill        │    │       MCP         │
│      API          │    │      API          │    │      API          │
└───────────────────┘    └───────────────────┘    └───────────────────┘
          │                         │                         │
          ▼                         ▼                         ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                          Core Services                                  │
│  ┌──────────┐  ┌─────────────┐  ┌─────────────┐  ┌──────────────┐     │
│  │  Agent   │  │   Memory    │  │   Skill     │  │    LLM       │     │
│  │ Manager  │  │  Manager    │  │  Manager    │  │  Client      │     │
│  └──────────┘  └─────────────┘  └─────────────┘  └──────────────┘     │
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

### 2.2 Module Structure

```
pkg/
├── agent/                    # Core agent logic
│   ├── agent.go              # Agent implementation
│   ├── manager.go            # Agent lifecycle management
│   ├── context/              # Conversation context management
│   │   └── engine.go         # Context engine
│   ├── memory/               # Memory system
│   │   ├── memory.go         # Unified memory interface
│   │   ├── working.go        # Working memory
│   │   ├── shortterm.go      # Short-term memory
│   │   ├── longterm.go       # Long-term memory
│   │   └── update.go         # Memory updater
│   ├── llm/                  # LLM client
│   ├── skill/                # Skill system
│   ├── tool/                 # Tool implementations
│   └── mcp/                  # MCP integration
├── repository/               # Data access layer
├── server/                   # API server
├── shared/                   # Shared utilities
├── rag/                      # RAG components
│   ├── vector_store.go       # Vector database
│   ├── embedding.go          # Embedding service
│   └── rerank.go             # Rerank service
└── config/                   # Configuration management
```

## 3. Core Components

### 3.1 Agent Module

#### 3.1.1 Agent Manager

**Responsibilities**:
- Agent lifecycle management (create, retrieve, destroy)
- Conversation session management
- Policy enforcement

**Key Methods**:
```go
type Manager struct {
    agents map[string]*Agent  // key: conversationId
    memory memory.Memory
}

func (m *Manager) GetAgent(userId, conversationId string) *Agent
func (m *Manager) RemoveAgent(conversationId string)
```

#### 3.1.2 Context Engine

**Responsibilities**:
- Message management and token counting
- Policy application (truncate, etc.)
- System prompt building

**Key Methods**:
```go
type Engine struct {
    messages        []messageWrap
    memory          memory.Memory
    turnCount       int
}

func (e *Engine) CommitTurn(ctx context.Context, draft TurnDraft, usage Usage) error
func (e *Engine) BuildSystemPrompt() string
func (e *Engine) ProcessMessages(ctx context.Context, messages []shared.OpenAIMessage) error
```

### 3.2 Memory System

See [Memory System Design Document](memory_system_design.md) for detailed design.

### 3.3 Skill System

**Location**: `pkg/agent/skill/`

**Structure**:
```go
type Manager struct {
    store     SkillStore
    skillsDir string
    skills    map[string]*Skill
}

type Skill struct {
    ID          string
    Name        string
    Description string
    Tool        tool.Tool
}
```

**Key Features**:
- Skill discovery and loading
- Dynamic tool registration
- Skill installation/uninstallation

### 3.4 Tool System

**Location**: `pkg/agent/tool/`

**Supported Tools**:
| Tool | Description |
|------|-------------|
| **BashTool** | Execute shell commands |
| **FileTool** | File read/write operations |
| **MCPTool** | MCP integration |

### 3.5 MCP Integration

**Location**: `pkg/agent/mcp/`

**Structure**:
```go
type Manager struct {
    store McpStore
    clients []*Client
}

type Client struct {
    name    string
    process *os.Process
    port    int
}
```

**Key Features**:
- MCP server management
- Dynamic plugin loading
- Protocol handling

## 4. Database Design

### 4.1 PostgreSQL Tables

#### 4.1.1 conversations Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Primary key |
| user_id | VARCHAR(36) | User identifier |
| title | TEXT | Conversation title |
| created_at | TIMESTAMP | Creation time |
| updated_at | TIMESTAMP | Last update time |
| status | VARCHAR(20) | Status (active/closed) |

#### 4.1.2 messages Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Primary key |
| conversation_id | VARCHAR(36) | Foreign key |
| role | VARCHAR(20) | Message role (user/assistant/system) |
| content | TEXT | Message content |
| created_at | TIMESTAMP | Creation time |
| token_count | INTEGER | Token count |

#### 4.1.3 skills Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Primary key |
| name | VARCHAR(100) | Skill name |
| description | TEXT | Skill description |
| version | VARCHAR(20) | Version |
| installed_at | TIMESTAMP | Installation time |
| enabled | BOOLEAN | Enabled status |

#### 4.1.4 mcp_plugins Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Primary key |
| name | VARCHAR(100) | Plugin name |
| path | VARCHAR(255) | Plugin path |
| port | INTEGER | Listening port |
| status | VARCHAR(20) | Status (running/stopped) |

### 4.2 Vector Database Schema

#### memory_chunks Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Primary key |
| user_id | VARCHAR(36) | User identifier |
| conversation_id | VARCHAR(36) | Conversation ID |
| content | TEXT | Original content |
| summary | TEXT | Generated summary |
| embedding | vector(1536) | Vector embedding |
| round_number | INTEGER | Conversation round |
| created_at | TIMESTAMP | Creation time |

## 5. API Design

### 5.1 Conversation API

#### 5.1.1 Create Conversation

```http
POST /api/conversations
Content-Type: application/json

{
    "user_id": "string",
    "title": "string (optional)"
}

Response:
{
    "id": "string",
    "user_id": "string",
    "title": "string",
    "created_at": "timestamp"
}
```

#### 5.1.2 Get Conversation

```http
GET /api/conversations/{id}

Response:
{
    "id": "string",
    "user_id": "string",
    "title": "string",
    "messages": [...],
    "created_at": "timestamp",
    "updated_at": "timestamp"
}
```

#### 5.1.3 Send Message

```http
POST /api/conversations/{id}/messages
Content-Type: application/json

{
    "content": "string",
    "role": "user"
}

Response:
{
    "id": "string",
    "conversation_id": "string",
    "role": "assistant",
    "content": "string",
    "created_at": "timestamp"
}
```

#### 5.1.4 List Conversations

```http
GET /api/conversations?user_id={user_id}&limit=10&offset=0

Response:
{
    "conversations": [...],
    "total": 0
}
```

#### 5.1.5 Delete Conversation

```http
DELETE /api/conversations/{id}

Response: 204 No Content
```

### 5.2 Skill API

#### 5.2.1 List Skills

```http
GET /api/skills

Response:
{
    "skills": [
        {
            "id": "string",
            "name": "string",
            "description": "string",
            "version": "string",
            "enabled": true
        }
    ]
}
```

#### 5.2.2 Install Skill

```http
POST /api/skills
Content-Type: multipart/form-data

Body: skill.zip

Response:
{
    "id": "string",
    "name": "string",
    "status": "installed"
}
```

#### 5.2.3 Enable/Disable Skill

```http
PUT /api/skills/{id}/toggle

Response:
{
    "id": "string",
    "enabled": true
}
```

#### 5.2.4 Uninstall Skill

```http
DELETE /api/skills/{id}

Response: 204 No Content
```

### 5.3 MCP API

#### 5.3.1 List Plugins

```http
GET /api/mcp/plugins

Response:
{
    "plugins": [
        {
            "id": "string",
            "name": "string",
            "status": "running",
            "port": 0
        }
    ]
}
```

#### 5.3.2 Start Plugin

```http
POST /api/mcp/plugins/{id}/start

Response:
{
    "id": "string",
    "status": "running"
}
```

#### 5.3.3 Stop Plugin

```http
POST /api/mcp/plugins/{id}/stop

Response:
{
    "id": "string",
    "status": "stopped"
}
```

## 6. Configuration

### 6.1 Configuration Structure

```yaml
server:
  address: ":8080"
  timeout: 30s

database:
  host: localhost
  port: 5432
  user: admin
  password: password
  database: tianniu

redis:
  address: localhost:6379
  password: ""
  db: 0

llm_providers:
  front_model:
    api_base: https://api.openai.com/v1
    api_key: ${FRONT_MODEL_API_KEY}
    model: gpt-4o-mini
  back_model:
    api_base: http://localhost:8000/v1
    api_key: ""
    model: llama-3-70b

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
  rerank_service:
    api_key: ""
    base_url: http://localhost:8000/v1
    model: cross-encoder/ms-marco-MiniLM-L-6-v2

bash_tool:
  enabled: true
  working_dir: /workspace

mcp:
  plugins_dir: ./mcp_plugins
```

### 6.2 Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `FRONT_MODEL_API_KEY` | Front model API key | Yes |
| `BACK_MODEL_API_KEY` | Back model API key | No |
| `DATABASE_URL` | Database connection string | Yes |
| `REDIS_URL` | Redis connection string | Yes |
| `SKILLS_DIR` | Skills directory path | No |

## 7. Deployment Architecture

### 7.1 Containerized Deployment

```yaml
# docker-compose.yml
version: '3.8'

services:
  tianniu:
    image: tianniu:latest
    ports:
      - "8080:8080"
    environment:
      - FRONT_MODEL_API_KEY=${FRONT_MODEL_API_KEY}
      - DATABASE_URL=postgres://admin:password@postgres:5432/tianniu
      - REDIS_URL=redis://redis:6379/0
    depends_on:
      - postgres
      - redis
      - vector-db

  postgres:
    image: postgres:16
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=tianniu
    volumes:
      - postgres-data:/var/lib/postgresql/data

  redis:
    image: redis:7
    volumes:
      - redis-data:/data

  vector-db:
    image: pgvector/pgvector:pg16
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=memory_db
    volumes:
      - vector-data:/var/lib/postgresql/data

volumes:
  postgres-data:
  redis-data:
  vector-data:
```

### 7.2 Kubernetes Deployment

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tianniu
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tianniu
  template:
    metadata:
      labels:
        app: tianniu
    spec:
      containers:
      - name: tianniu
        image: tianniu:latest
        ports:
        - containerPort: 8080
        env:
        - name: FRONT_MODEL_API_KEY
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: front-model-api-key
        - name: DATABASE_URL
          value: postgres://admin:password@postgres:5432/tianniu
        - name: REDIS_URL
          value: redis://redis:6379/0
```

## 8. Security Considerations

### 8.1 Authentication

- API key authentication for server-to-server communication
- OAuth 2.0 for user authentication
- JWT tokens for session management

### 8.2 Authorization

- Role-based access control (RBAC)
- User-specific data isolation
- Skill execution permissions

### 8.3 Data Protection

- Encryption at rest (database)
- TLS encryption in transit
- Sensitive data masking in logs

### 8.4 Input Validation

- Strict input validation for all API endpoints
- Sanitization of user inputs
- Rate limiting to prevent abuse

## 9. Monitoring & Observability

### 9.1 Metrics

| Metric | Description |
|--------|-------------|
| `http_request_count` | Total HTTP requests |
| `http_request_duration` | Request duration histogram |
| `agent_count` | Active agent count |
| `memory_update_count` | Memory update operations |
| `llm_call_count` | LLM API calls |
| `skill_execution_count` | Skill executions |
| `error_count` | Error count by type |

### 9.2 Logging

- Structured logging (JSON format)
- Request tracing with correlation IDs
- Error logging with stack traces
- Performance logging for slow operations

### 9.3 Tracing

- Distributed tracing with OpenTelemetry
- Trace spans for LLM calls, database queries, and external API calls
- Sampling for high-volume endpoints

## 10. Error Handling

### 10.1 Error Types

| Error Code | Description | HTTP Status |
|------------|-------------|-------------|
| `CONV_NOT_FOUND` | Conversation not found | 404 |
| `MSG_INVALID` | Invalid message format | 400 |
| `SKILL_NOT_FOUND` | Skill not found | 404 |
| `SKILL_DISABLED` | Skill is disabled | 403 |
| `LLM_ERROR` | LLM service error | 503 |
| `MEMORY_ERROR` | Memory operation error | 500 |

### 10.2 Error Response Format

```json
{
    "error": {
        "code": "string",
        "message": "string",
        "details": "string (optional)"
    }
}
```

## 11. Performance Optimization

### 11.1 Caching Strategies

- Redis caching for frequent queries
- Memory caching for conversation context
- Result caching for repeated queries

### 11.2 Asynchronous Processing

- Background workers for memory updates
- Async task queues for skill execution
- Event-driven architecture for non-critical operations

### 11.3 Database Optimization

- Indexing for frequently queried columns
- Connection pooling
- Read replicas for high-traffic scenarios

## 12. Future Enhancements

1. **Multi-tenancy Support**: Multi-tenant architecture for SaaS deployment
2. **Real-time Collaboration**: WebSocket-based real-time conversation
3. **Model Fine-tuning**: Built-in fine-tuning capabilities
4. **Analytics Dashboard**: Built-in analytics and reporting
5. **Mobile SDK**: Mobile application SDK
6. **Voice Support**: Voice-to-text and text-to-voice capabilities