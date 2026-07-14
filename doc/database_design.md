# TianNiu Backend System Design - Database Design

## 1. Overview

### 1.1 Database Components

| Component | Database | Purpose |
|-----------|----------|---------|
| **Primary Database** | PostgreSQL | Conversation history, user data, skill metadata |
| **Cache** | Redis | Session management, short-term caching |
| **Vector Database** | PostgreSQL + pgvector | Long-term memory storage and retrieval |

### 1.2 ER Diagram

```
┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│  users       │      │conversations │      │   messages   │
├──────────────┤      ├──────────────┤      ├──────────────┤
│ id (PK)      │◄─────│ id (PK)      │◄─────│ id (PK)      │
│ name         │      │ user_id (FK) │      │ conv_id (FK) │
│ email        │      │ title        │      │ role         │
│ created_at   │      │ created_at   │      │ content      │
└──────────────┘      │ updated_at   │      │ created_at   │
                      └──────────────┘      └──────────────┘

┌──────────────┐      ┌──────────────┐      ┌──────────────┐
│   skills     │      │ user_skills  │      │  mcp_plugins │
├──────────────┤      ├──────────────┤      ├──────────────┤
│ id (PK)      │◄─────│ user_id (FK) │      │ id (PK)      │
│ name         │      │ skill_id(FK) │      │ name         │
│ description  │      │ enabled      │      │ path         │
│ version      │      └──────────────┘      │ port         │
│ enabled      │                           │ status       │
└──────────────┘                           └──────────────┘

┌─────────────────────┐
│  memory_chunks      │
├─────────────────────┤
│ id (PK)             │
│ user_id             │
│ conversation_id     │
│ content             │
│ summary             │
│ embedding (vector)  │
│ round_number        │
│ created_at          │
└─────────────────────┘
```

## 2. PostgreSQL Schema

### 2.1 Users Table

```sql
CREATE TABLE users (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255),
    role VARCHAR(20) DEFAULT 'user',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
```

### 2.2 Conversations Table

```sql
CREATE TABLE conversations (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_conversations_user_id ON conversations(user_id);
CREATE INDEX idx_conversations_status ON conversations(status);
CREATE INDEX idx_conversations_updated_at ON conversations(updated_at);
```

### 2.3 Messages Table

```sql
CREATE TABLE messages (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id VARCHAR(36) NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER DEFAULT 0,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_role ON messages(role);
CREATE INDEX idx_messages_created_at ON messages(created_at);
```

### 2.4 Skills Table

```sql
CREATE TABLE skills (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    version VARCHAR(20) DEFAULT '1.0.0',
    path VARCHAR(255),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_skills_name ON skills(name);
CREATE INDEX idx_skills_enabled ON skills(enabled);
```

### 2.5 User Skills Table

```sql
CREATE TABLE user_skills (
    user_id VARCHAR(36) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    skill_id VARCHAR(36) NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    PRIMARY KEY (user_id, skill_id)
);

CREATE INDEX idx_user_skills_user_id ON user_skills(user_id);
CREATE INDEX idx_user_skills_skill_id ON user_skills(skill_id);
```

### 2.6 MCP Plugins Table

```sql
CREATE TABLE mcp_plugins (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    path VARCHAR(255) NOT NULL,
    port INTEGER,
    status VARCHAR(20) DEFAULT 'stopped',
    last_started_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_mcp_plugins_name ON mcp_plugins(name);
CREATE INDEX idx_mcp_plugins_status ON mcp_plugins(status);
```

### 2.7 Memory Chunks Table (Vector DB)

```sql
-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE memory_chunks (
    id VARCHAR(36) PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(36) NOT NULL,
    conversation_id VARCHAR(36),
    content TEXT NOT NULL,
    summary TEXT,
    embedding vector(1536) NOT NULL,
    round_number INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_memory_chunks_user_id ON memory_chunks(user_id);
CREATE INDEX idx_memory_chunks_conversation_id ON memory_chunks(conversation_id);
CREATE INDEX idx_memory_chunks_embedding ON memory_chunks USING hnsw(embedding vector_cosine_ops);
```

## 3. Repository Layer

### 3.1 Repository Interface

```go
type Store interface {
    // Conversation operations
    CreateConversation(ctx context.Context, conv *Conversation) error
    GetConversation(ctx context.Context, id string) (*Conversation, error)
    ListConversations(ctx context.Context, userId string, limit, offset int) ([]*Conversation, error)
    UpdateConversation(ctx context.Context, conv *Conversation) error
    DeleteConversation(ctx context.Context, id string) error
    
    // Message operations
    CreateMessage(ctx context.Context, msg *Message) error
    GetMessages(ctx context.Context, conversationId string, limit, offset int) ([]*Message, error)
    DeleteMessages(ctx context.Context, conversationId string) error
    
    // Skill operations
    CreateSkill(ctx context.Context, skill *Skill) error
    GetSkill(ctx context.Context, id string) (*Skill, error)
    ListSkills(ctx context.Context) ([]*Skill, error)
    UpdateSkill(ctx context.Context, skill *Skill) error
    DeleteSkill(ctx context.Context, id string) error
    
    // User-Skill operations
    AddUserSkill(ctx context.Context, userId, skillId string) error
    RemoveUserSkill(ctx context.Context, userId, skillId string) error
    GetUserSkills(ctx context.Context, userId string) ([]*Skill, error)
}
```

### 3.2 SQL Store Implementation

```go
type SQLStore struct {
    db *sql.DB
}

func NewSQLStore(db *sql.DB) *SQLStore {
    return &SQLStore{db: db}
}
```

## 4. Redis Schema

### 4.1 Key Structure

| Key Pattern | Description | TTL |
|-------------|-------------|-----|
| `user:{userId}:sessions` | User active sessions | 1 hour |
| `conv:{convId}:context` | Conversation context | 5 minutes |
| `memory:{userId}:shortterm` | Short-term memory cache | 5 minutes |
| `rate_limit:{userId}` | Rate limiting counter | 1 minute |
| `embedding:{hash}` | Embedding cache | 24 hours |

### 4.2 Session Management

```go
func (r *RedisCache) SetSession(ctx context.Context, userId, sessionId string, data interface{}) error {
    key := fmt.Sprintf("user:%s:sessions:%s", userId, sessionId)
    value, err := json.Marshal(data)
    if err != nil {
        return err
    }
    return r.client.Set(ctx, key, value, time.Hour).Err()
}

func (r *RedisCache) GetSession(ctx context.Context, userId, sessionId string) (interface{}, error) {
    key := fmt.Sprintf("user:%s:sessions:%s", userId, sessionId)
    value, err := r.client.Get(ctx, key).Result()
    if err != nil {
        return nil, err
    }
    var data interface{}
    return data, json.Unmarshal([]byte(value), &data)
}
```

### 4.3 Rate Limiting

```go
func (r *RedisCache) CheckRateLimit(ctx context.Context, userId string, limit int) (bool, error) {
    key := fmt.Sprintf("rate_limit:%s", userId)
    count, err := r.client.Incr(ctx, key).Result()
    if err != nil {
        return false, err
    }
    if count == 1 {
        r.client.Expire(ctx, key, time.Minute)
    }
    return count <= int64(limit), nil
}
```

## 5. Vector Database Operations

### 5.1 Vector Store Interface

```go
type VectorStore interface {
    Insert(ctx context.Context, userId, conversationId string, 
           content, summary string, embedding Vector, roundNumber int) error
    Search(ctx context.Context, userId string, queryEmbedding Vector, topK int) ([]MemoryChunk, error)
    Delete(ctx context.Context, userId, conversationId string) error
}
```

### 5.2 PGVector Implementation

```go
type PGVectorStore struct {
    db *sql.DB
}

func (p *PGVectorStore) Search(ctx context.Context, userId string, 
                               queryEmbedding Vector, topK int) ([]MemoryChunk, error) {
    query := `
        SELECT id, user_id, conversation_id, content, summary, round_number, created_at
        FROM memory_chunks
        WHERE user_id = $1
        ORDER BY embedding <-> $2
        LIMIT $3
    `
    
    rows, err := p.db.QueryContext(ctx, query, userId, queryEmbedding, topK)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var chunks []MemoryChunk
    for rows.Next() {
        var chunk MemoryChunk
        if err := rows.Scan(&chunk.ID, &chunk.UserID, &chunk.ConversationID,
                           &chunk.Content, &chunk.Summary, &chunk.RoundNumber, &chunk.CreatedAt); err != nil {
            return nil, err
        }
        chunks = append(chunks, chunk)
    }
    
    return chunks, nil
}
```

## 6. Data Access Patterns

### 6.1 Conversation Flow

```
Create Conversation
    │
    ├─→ Begin transaction
    │
    ├─→ INSERT INTO conversations
    │
    └─→ Commit

Add Message
    │
    ├─→ Begin transaction
    │
    ├─→ INSERT INTO messages
    │
    ├─→ UPDATE conversations SET updated_at = NOW()
    │
    └─→ Commit

Get Conversation with Messages
    │
    ├─→ SELECT * FROM conversations WHERE id = $1
    │
    └─→ SELECT * FROM messages WHERE conversation_id = $1 ORDER BY created_at
```

### 6.2 Memory Flow

```
Save Memory Chunk
    │
    ├─→ Generate summary (LLM)
    │
    ├─→ Generate embedding
    │
    └─→ INSERT INTO memory_chunks

Retrieve Memory
    │
    ├─→ Generate query embedding
    │
    ├─→ SELECT * FROM memory_chunks WHERE user_id = $1 ORDER BY embedding <-> $2 LIMIT $3
    │
    └─→ Rerank results
```

## 7. Database Configuration

### 7.1 Connection Pooling

```yaml
database:
  host: localhost
  port: 5432
  user: admin
  password: password
  database: tianniu
  pool_size: 20
  max_idle_conns: 10
  conn_max_lifetime: 30m
```

### 7.2 Redis Configuration

```yaml
redis:
  address: localhost:6379
  password: ""
  db: 0
  pool_size: 10
```

## 8. Migration Strategy

### 8.1 Migration Files

```
migrations/
├── 001_create_users.sql
├── 002_create_conversations.sql
├── 003_create_messages.sql
├── 004_create_skills.sql
├── 005_create_user_skills.sql
├── 006_create_mcp_plugins.sql
└── 007_create_memory_chunks.sql
```

### 8.2 Migration Execution

```go
func RunMigrations(db *sql.DB) error {
    migrator, err := migrate.NewWithDatabaseInstance(
        "file://migrations",
        "postgres",
        migrate.PostgresInstance(db),
    )
    if err != nil {
        return err
    }
    
    return migrator.Up()
}
```

## 9. Backup Strategy

### 9.1 Automated Backups

- **Daily backups**: Full database backup at 2 AM
- **Incremental backups**: Every hour
- **Retention**: 7 days locally, 30 days in cloud storage

### 9.2 Backup Commands

```bash
# Full backup
pg_dump -U admin -d tianniu -f backup_$(date +%Y%m%d).sql

# Restore
psql -U admin -d tianniu -f backup_20240101.sql
```