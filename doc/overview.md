# TianNiu Backend System Design - Overview

## 1. System Overview

### 1.1 Purpose

TianNiu is an AI-powered chat assistant backend system that provides intelligent conversation capabilities with memory management, skill execution, and tool integration.

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

### 1.4 Module Structure

```
pkg/
├── agent/                    # Core agent logic
│   ├── agent.go              # Agent implementation
│   ├── manager.go            # Agent lifecycle management
│   ├── context/              # Conversation context management
│   ├── memory/               # Memory system
│   ├── llm/                  # LLM client
│   ├── skill/                # Skill system
│   ├── tool/                 # Tool implementations
│   └── mcp/                  # MCP integration
├── repository/               # Data access layer
├── server/                   # API server
├── shared/                   # Shared utilities
└── rag/                      # RAG components
```

### 1.5 Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Client Layer                                  │
│  [Web UI] [CLI] [API Clients]                                           │
└──────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────────────┐
│                           API Gateway                                   │
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
└───────────────────┘    └───────────────────┘    └───────────────────┘
```

### 1.6 Key Components Summary

| Component | Package | Responsibility |
|-----------|---------|----------------|
| **Agent Manager** | `pkg/agent/` | Agent lifecycle management |
| **Context Engine** | `pkg/agent/context/` | Conversation context management |
| **Memory System** | `pkg/agent/memory/` | Multi-level memory management |
| **Skill System** | `pkg/agent/skill/` | Plugin-based skill execution |
| **Tool System** | `pkg/agent/tool/` | Built-in tool implementations |
| **MCP System** | `pkg/agent/mcp/` | MCP plugin integration |
| **LLM Client** | `pkg/agent/llm/` | LLM API client |
| **RAG Components** | `pkg/rag/` | Vector database and embedding |

## 2. Design Documents Index

| Document | Description |
|----------|-------------|
| `overview.md` | System overview and design principles |
| `architecture.md` | High-level architecture design |
| `agent_module.md` | Agent and context engine design |
| `memory_system.md` | Multi-level memory system design |
| `skill_system.md` | Skill system design |
| `tool_system.md` | Tool system design |
| `mcp_system.md` | MCP integration design |
| `database_design.md` | Database schema design |
| `api_design.md` | REST API design |
| `deployment.md` | Deployment architecture |
| `security.md` | Security considerations |
| `monitoring.md` | Monitoring and observability |