# 天牛

A lightweight AI chat agent built with Go, featuring streaming message processing, tool calls, MCP integration, and multi-threaded conversations.

## Features

- ✅ **Smart Conversations**: Fluent AI model interaction with streaming responses
- ✅ **Multi-thread Management**: Create, rename, and delete conversation threads
- ✅ **Streaming Messages**: Real-time message delivery via Server-Sent Events (SSE)
- ✅ **Tool Calls**: AI can invoke external tools to fetch information
- ✅ **MCP Integration**: Connect to any MCP-compatible tool server (stdio / HTTP) to extend agent capabilities
- ✅ **Reasoning Panel**: Display the AI's thinking process (DeepSeek-R1, QwQ, etc.)
- ✅ **Bash Tool**: Execute shell commands with built-in security restrictions (dangerous pattern blocking, timeout, output limits, env filtering)
- ✅ **Memory System**: Multi-level memory management (global + workspace) for context retention
- ✅ **Markdown Rendering**: Full markdown support (GFM) for AI responses and tool results
- ✅ **JWT Authentication**: User registration, login, and token-based access control
- ✅ **Context Management**: Automatic message summarization and content offloading to manage context window
- [ ] **File Processing**: Support for file upload and analysis (RAG)
- [ ] **Message Editing**: Edit and recall sent messages
- [ ] **Message Reply**: Quote specific messages for contextual responses
- [ ] **Message Search**: Search through message history
- [ ] **Conversation Export**: Export chat history in JSON/Markdown format
- [ ] **Parameter Tuning**: Customize temperature, max tokens, etc.
- [ ] **Web Search**: Real-time internet search capabilities
- [ ] **User Preferences**: Store personalized settings and role presets
- [ ] **API Usage Stats**: Track API calls and token consumption per user
- [ ] **Skill System**: Skill store, management (install/uninstall), and custom skill creation
- [ ] **Multimodal Support**: Image generation (text-to-image) and image understanding (vision)
- [ ] **Voice Capabilities**: Speech-to-text and text-to-speech
- [ ] **Social Sharing**: Share chat history and collaborate with others
- [ ] **Content Creation**: Document generation, code writing, and table processing
- [ ] **Productivity Tools**: Calendar integration, to-do lists, and reminders
- [ ] **Role Store**: Discover and use pre-built AI personas/characters
- [ ] **Favorites & Notes**: Save and organize important conversations
- [ ] **Translation**: Multi-language translation support
- [ ] **AI Summarization**: Automatic long text summarization
- [ ] **Third-party Integration**: Browser extensions, mobile apps, and external services

## Tech Stack

### Backend
- Go 1.26.4 + Gin
- OpenAI Go SDK v3
- MCP Go SDK v1
- GORM + SQLite
- Redis for memory storage
- JWT authentication (golang-jwt/v5)

## Quick Start

### Local Development

1. **Configure the backend**

```bash
cp config.example.yaml config.yaml
```

Edit `config.yaml` with your LLM provider and tool settings:


2. **(Optional) Configure MCP servers**

Edit `mcp-server.json` to connect external tool servers:

3. **Start the backend**

```bash
go run ./tianniu/main.go
```
The server runs on `http://localhost:8080`.

## Frontend Integration
Refer to the workspace repository: https://github.com/tianniu-ai/tianniu-workspace

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_PATH` | SQLite database file path | `test.db` |
| `LEVELDB_PATH` | LevelDB storage path for memory and offloaded content | `leveldb_data` |
| `JWT_SECRET` | Secret key for JWT token signing (≥16 bytes) | `tian-niu-dev-secret-change-in-production` |
| `GIN_MODE` | Gin run mode (`debug`/`release`) | `debug` |

### MCP Server Configuration

MCP (Model Context Protocol) allows the agent to use external tool servers. Configure in `mcp-server.json`:

**Stdio transport** (local process):

```json
{
  "filesystem": {
    "command": "npx",
    "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/dir"]
  }
}
```

**HTTP transport** (remote server):

```json
{
  "remote-api": {
    "type": "http",
    "url": "http://localhost:3001/mcp",
    "headers": {
      "Authorization": "Bearer token"
    }
  }
}
```

MCP tools are automatically discovered and registered as agent tools at startup. Tool names are prefixed as `babyagent_mcp__<server>__<tool>` to avoid conflicts.

## Supported Models

- OpenAI: gpt-4o, gpt-4o-mini, gpt-4-turbo, gpt-5.2
- DeepSeek: deepseek-chat, deepseek-reasoner (with reasoning output)
- Zhipu AI: GLM-5.2, GLM-4, GLM-4.6V
- Qwen: QwQ, Qwen3
- Any model compatible with the OpenAI API format

## API Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/user/register` | No | Register a new user |
| POST | `/api/user/login` | No | Login and get JWT token |
| POST | `/api/conversation` | Yes | Create a conversation |
| GET | `/api/conversation` | Yes | List user's conversations |
| PATCH | `/api/conversation/:id` | Yes | Rename a conversation |
| DELETE | `/api/conversation/:id` | Yes | Delete a conversation |
| POST | `/api/conversation/:id/message` | Yes | Send a message (SSE stream) |
| GET | `/api/conversation/:id/message` | Yes | List conversation messages |

## Preview

![Effect](./doc/tianniu.png)

## License

MIT License