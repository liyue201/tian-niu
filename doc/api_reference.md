# TianNiu API Reference

## Base URL

All API endpoints are prefixed with `/api`.

## Authentication

Most endpoints require authentication via JWT token. Include the token in the Authorization header:

```http
Authorization: Bearer <your_jwt_token>
```

## Endpoints

### 1. Authentication

#### 1.1 Register User

**POST** `/api/user/register`

Register a new user.

**Request Body:**
```json
{
    "email": "string (required)",
    "password": "string (required)",
    "name": "string (optional)"
}
```

**Response:**
```json
{
    "id": "string",
    "email": "string",
    "name": "string",
    "created_at": "timestamp"
}
```

#### 1.2 Login

**POST** `/api/user/login`

Login and obtain JWT token.

**Request Body:**
```json
{
    "email": "string (required)",
    "password": "string (required)"
}
```

**Response:**
```json
{
    "access_token": "string",
    "token_type": "Bearer",
    "expires_in": 3600,
    "user": {
        "id": "string",
        "email": "string",
        "name": "string"
    }
}
```

### 2. Conversations

#### 2.1 Create Conversation

**POST** `/api/conversation`

Create a new conversation.

**Request Body:**
```json
{
    "title": "string (optional)"
}
```

**Response:**
```json
{
    "id": "string",
    "title": "string",
    "user_id": "string",
    "created_at": "timestamp",
    "updated_at": "timestamp"
}
```

#### 2.2 List Conversations

**GET** `/api/conversation`

List all conversations for the authenticated user.

**Query Parameters:**
- `limit`: Number of results per page (default: 20)
- `offset`: Offset for pagination (default: 0)

**Response:**
```json
{
    "conversations": [
        {
            "id": "string",
            "title": "string",
            "created_at": "timestamp",
            "updated_at": "timestamp"
        }
    ],
    "total": 100,
    "limit": 20,
    "offset": 0
}
```

#### 2.3 Get Conversation

**GET** `/api/conversation/:id`

Get detailed information about a conversation.

**Response:**
```json
{
    "id": "string",
    "title": "string",
    "user_id": "string",
    "messages": [...],
    "created_at": "timestamp",
    "updated_at": "timestamp"
}
```

#### 2.4 Update Conversation

**PATCH** `/api/conversation/:id`

Update conversation details (e.g., rename).

**Request Body:**
```json
{
    "title": "string"
}
```

**Response:**
```json
{
    "id": "string",
    "title": "string",
    "updated_at": "timestamp"
}
```

#### 2.5 Delete Conversation

**DELETE** `/api/conversation/:id`

Delete a conversation.

**Response:** `204 No Content`

### 3. Messages

#### 3.1 Send Message

**POST** `/api/conversation/:id/message`

Send a message to a conversation. Returns SSE stream for streaming responses.

**Request Body:**
```json
{
    "content": "string (required)",
    "role": "user"
}
```

**Response (SSE Stream):**
```
event: message
data: {"id": "string", "content": "partial response..."}

event: done
data: {"id": "string", "content": "complete response", "token_count": 0}
```

#### 3.2 List Messages

**GET** `/api/conversation/:id/message`

List messages in a conversation.

**Query Parameters:**
- `limit`: Number of results per page (default: 50)
- `offset`: Offset for pagination (default: 0)

**Response:**
```json
{
    "messages": [
        {
            "id": "string",
            "conversation_id": "string",
            "role": "user|assistant|system",
            "content": "string",
            "token_count": 0,
            "created_at": "timestamp"
        }
    ],
    "total": 100,
    "limit": 50,
    "offset": 0
}
```

#### 3.3 Delete Message

**DELETE** `/api/conversation/:id/message/:message_id`

Delete a specific message.

**Response:** `204 No Content`

### 4. Skills

#### 4.1 List Skills

**GET** `/api/skills`

List all available skills.

**Response:**
```json
{
    "skills": [
        {
            "id": "string",
            "name": "string",
            "description": "string",
            "version": "string",
            "enabled": true,
            "created_at": "timestamp"
        }
    ]
}
```

#### 4.2 Install Skill

**POST** `/api/skills/install`

Install a new skill from a ZIP archive.

**Request:** `multipart/form-data`
- `file`: ZIP file containing skill

**Response:**
```json
{
    "id": "string",
    "name": "string",
    "version": "string",
    "status": "installed"
}
```

#### 4.3 Enable Skill

**POST** `/api/skills/:id/enable`

Enable a skill.

**Response:**
```json
{
    "id": "string",
    "enabled": true
}
```

#### 4.4 Disable Skill

**POST** `/api/skills/:id/disable`

Disable a skill.

**Response:**
```json
{
    "id": "string",
    "enabled": false
}
```

#### 4.5 Uninstall Skill

**POST** `/api/skills/:id/uninstall`

Uninstall a skill.

**Response:** `204 No Content`

### 5. MCP Servers

#### 5.1 List MCP Servers

**GET** `/api/mcps`

List all configured MCP servers.

**Response:**
```json
{
    "servers": [
        {
            "id": "string",
            "name": "string",
            "type": "stdio|http",
            "status": "running|stopped|error",
            "created_at": "timestamp"
        }
    ]
}
```

#### 5.2 Install MCP Server

**POST** `/api/mcps/install`

Install a new MCP server.

**Request Body:**
```json
{
    "name": "string",
    "type": "stdio|http",
    "command": "string (for stdio)",
    "args": ["string"],
    "url": "string (for http)",
    "headers": {"key": "value"}
}
```

**Response:**
```json
{
    "id": "string",
    "name": "string",
    "status": "installed"
}
```

#### 5.3 Enable MCP Server

**POST** `/api/mcps/:id/enable`

Enable an MCP server.

**Response:**
```json
{
    "id": "string",
    "status": "running"
}
```

#### 5.4 Disable MCP Server

**POST** `/api/mcps/:id/disable`

Disable an MCP server.

**Response:**
```json
{
    "id": "string",
    "status": "stopped"
}
```

#### 5.5 Uninstall MCP Server

**POST** `/api/mcps/:id/uninstall`

Uninstall an MCP server.

**Response:** `204 No Content`

### 6. Health Check

#### 6.1 Health Status

**GET** `/health`

Check the health status of the service.

**Response:**
```json
{
    "status": "healthy",
    "timestamp": "timestamp",
    "services": {
        "database": "healthy",
        "redis": "healthy",
        "vector_db": "healthy"
    }
}
```

#### 6.2 Metrics

**GET** `/metrics`

Get Prometheus metrics.

**Response:** Plain text metrics output.

## Error Responses

All error responses follow this format:

```json
{
    "error": {
        "code": "string",
        "message": "string",
        "details": "string (optional)",
        "timestamp": "timestamp"
    }
}
```

### Error Codes

| Code | HTTP Status | Description |
|------|-------------|-------------|
| `INVALID_REQUEST` | 400 | Invalid request parameters |
| `UNAUTHORIZED` | 401 | Authentication required |
| `FORBIDDEN` | 403 | Insufficient permissions |
| `NOT_FOUND` | 404 | Resource not found |
| `CONFLICT` | 409 | Resource conflict |
| `INTERNAL_ERROR` | 500 | Internal server error |
| `SERVICE_UNAVAILABLE` | 503 | Service unavailable |

## Rate Limiting

All API endpoints are rate-limited. The following headers are returned with each response:

- `X-RateLimit-Limit`: Maximum requests per minute
- `X-RateLimit-Remaining`: Remaining requests in current window
- `X-RateLimit-Reset`: Unix timestamp when the window resets

## Versioning

The API is versioned via the URL path. Current version is v1:

```
/api/v1/conversation
```

## WebSocket API

For real-time communication, use the WebSocket endpoint:

```
ws://localhost:8080/api/ws
```

### WebSocket Message Types

| Type | Description |
|------|-------------|
| `subscribe` | Subscribe to conversation events |
| `message` | New message received |
| `typing` | Agent is typing |
| `error` | Error occurred |

### Example WebSocket Usage

```javascript
const ws = new WebSocket('ws://localhost:8080/api/ws');

ws.onopen = () => {
    ws.send(JSON.stringify({
        type: 'subscribe',
        conversationId: 'conv-123'
    }));
};

ws.onmessage = (event) => {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```