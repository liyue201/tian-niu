# TianNiu Backend System Design - API Design

## 1. Overview

### 1.1 API Structure

| API Group | Base Path | Description |
|-----------|-----------|-------------|
| **Conversation API** | `/api/conversations` | Conversation management |
| **Message API** | `/api/conversations/{id}/messages` | Message operations |
| **Skill API** | `/api/skills` | Skill management |
| **MCP API** | `/api/mcp` | MCP plugin management |
| **User API** | `/api/users` | User management |
| **Health API** | `/health` | Health checks |

### 1.2 API Design Principles

- **RESTful**: Follow REST design principles
- **Versioning**: API versioning via URL
- **Error Handling**: Consistent error response format
- **Authentication**: API key or JWT authentication
- **Rate Limiting**: Protect against abuse
- **Idempotency**: Make requests idempotent where possible

### 1.3 Key Design Decisions

| Decision | Rationale |
|----------|-----------|
| **SSE for streaming** | Real-time message delivery without WebSocket overhead |
| **JWT Authentication** | Stateless authentication for scalability |
| **JSON request/response** | Standard format for easy integration |
| **Consistent error format** | Simplify client error handling |

## 2. Authentication Design

### 2.1 Authentication Flow

```
Client → Login → JWT Token → API Requests → Validate Token → Response
```

### 2.2 Security Considerations

- **Token Expiration**: Short-lived access tokens (1 hour)
- **Token Refresh**: Refresh token mechanism for long sessions
- **Secure Storage**: HTTPS only, HttpOnly cookies for refresh tokens
- **Rate Limiting**: Prevent brute force attacks on login endpoint

## 3. API Architecture

### 3.1 Layered Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           API Gateway                                   │
│  [Authentication] [Rate Limiting] [Request Validation] [Logging]       │
├──────────────────────────────────────────────────────────────────────────┤
│                           API Handlers                                  │
│  [ConversationHandler] [SkillHandler] [MCPHandler] [UserHandler]       │
├──────────────────────────────────────────────────────────────────────────┤
│                           Services                                      │
│  [ConversationService] [SkillService] [MCPManager] [UserService]       │
├──────────────────────────────────────────────────────────────────────────┤
│                           Repositories                                  │
│  [ConversationRepo] [SkillRepo] [MCPRepo] [UserRepo]                   │
├──────────────────────────────────────────────────────────────────────────┤
│                           Databases                                     │
│  [PostgreSQL] [Redis] [VectorDB]                                        │
└──────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Request Processing Flow

```
HTTP Request
    │
    ├─→ Authentication Middleware
    │       └─→ Validate JWT token
    │
    ├─→ Rate Limiting Middleware
    │       └─→ Check request count
    │
    ├─→ Request Validation
    │       └─→ Validate request body/schema
    │
    ├─→ Handler
    │       └─→ Business logic
    │
    ├─→ Service
    │       └─→ Domain logic
    │
    ├─→ Repository
    │       └─→ Data access
    │
    └─→ Response
            └─→ Serialize and return
```

## 4. Error Handling Design

### 4.1 Error Response Format

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

### 4.2 Error Code Classification

| Category | Codes | HTTP Status |
|----------|-------|-------------|
| **Client Errors** | INVALID_REQUEST, UNAUTHORIZED, FORBIDDEN, NOT_FOUND, CONFLICT | 4xx |
| **Server Errors** | INTERNAL_ERROR, SERVICE_UNAVAILABLE | 5xx |

### 4.3 Error Handling Strategy

1. **Validation Errors**: Return 400 with detailed field errors
2. **Authentication Errors**: Return 401 with challenge
3. **Authorization Errors**: Return 403 without revealing details
4. **Resource Not Found**: Return 404 with resource identifier
5. **Server Errors**: Return 500 without stack trace, log internally

## 5. Rate Limiting Design

### 5.1 Rate Limit Configuration

| Endpoint | Limit | Window |
|----------|-------|--------|
| `/api/user/login` | 10 requests | 1 minute |
| `/api/conversation` | 100 requests | 1 minute |
| `/api/conversation/:id/message` | 50 messages | 1 minute |
| `/api/skills/install` | 5 requests | 1 hour |

### 5.2 Rate Limit Headers

```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1699999999
Retry-After: 60
```

## 6. Versioning Strategy

### 6.1 URL-Based Versioning

```
/api/v1/conversations
/api/v2/conversations
```

### 6.2 Deprecation Policy

- **Phase 1**: Mark deprecated endpoints with `X-API-Deprecated` header
- **Phase 2**: Add `X-API-Deprecation-Date` header with removal date
- **Phase 3**: Remove deprecated endpoints after grace period

## 7. Security Design

### 7.1 HTTP Security Headers

| Header | Value | Purpose |
|--------|-------|---------|
| `X-Content-Type-Options` | nosniff | Prevent MIME sniffing |
| `X-Frame-Options` | DENY | Prevent clickjacking |
| `X-XSS-Protection` | 1; mode=block | Enable XSS protection |
| `Strict-Transport-Security` | max-age=31536000 | Enforce HTTPS |
| `Content-Security-Policy` | default-src 'self' | Mitigate XSS |

### 7.2 Input Sanitization

- **Parameter validation**: Validate all inputs against schema
- **Path sanitization**: Prevent path traversal attacks
- **SQL injection**: Use prepared statements
- **XSS prevention**: Escape user-generated content

## 8. API Reference

For detailed endpoint documentation, see [API Reference](api_reference.md).

## 9. Future Enhancements

- **GraphQL API**: Alternative to REST for flexible queries
- **gRPC API**: High-performance internal communication
- **Webhook Support**: Event-driven integrations
- **Real-time Updates**: WebSocket for push notifications