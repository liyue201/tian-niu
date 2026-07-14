# TianNiu Backend System Design - Security

## 1. Overview

### 1.1 Security Principles

- **Defense in Depth**: Multiple layers of security controls
- **Least Privilege**: Minimal permissions required for operations
- **Zero Trust**: Verify every request regardless of source
- **Secure by Default**: Security enabled by default

### 1.2 Security Layers

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Application Layer                               │
│  [Authentication] [Authorization] [Input Validation] [Encryption]      │
├──────────────────────────────────────────────────────────────────────────┤
│                        Network Layer                                   │
│  [Firewalls] [Network Segmentation] [TLS] [Rate Limiting]             │
├──────────────────────────────────────────────────────────────────────────┤
│                        Data Layer                                      │
│  [Encryption at Rest] [Access Control] [Backup Encryption]            │
├──────────────────────────────────────────────────────────────────────────┤
│                        Infrastructure Layer                            │
│  [Secure Config] [Patch Management] [Audit Logs]                      │
└──────────────────────────────────────────────────────────────────────────┘
```

## 2. Authentication

### 2.1 API Key Authentication

```go
func (a *APIKeyAuth) Authenticate(ctx context.Context, token string) (*User, error) {
    user, err := a.store.GetUserByAPIKey(ctx, token)
    if err != nil {
        return nil, errors.New("invalid API key")
    }
    return user, nil
}
```

### 2.2 JWT Authentication

```go
type JWTAuthenticator struct {
    secret []byte
}

func (j *JWTAuthenticator) GenerateToken(userID string) (string, error) {
    claims := jwt.MapClaims{
        "user_id": userID,
        "exp":     time.Now().Add(time.Hour * 24).Unix(),
    }
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(j.secret)
}

func (j *JWTAuthenticator) ValidateToken(tokenString string) (string, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return j.secret, nil
    })
    if err != nil {
        return "", err
    }
    claims, ok := token.Claims.(jwt.MapClaims)
    if !ok || !token.Valid {
        return "", errors.New("invalid token")
    }
    return claims["user_id"].(string), nil
}
```

### 2.3 OAuth 2.0

```go
type OAuthProvider interface {
    GetAuthURL(state string) string
    ExchangeCode(code string) (*Token, error)
    GetUserInfo(token *Token) (*User, error)
}
```

## 3. Authorization

### 3.1 Role-Based Access Control (RBAC)

```go
type Role string

const (
    RoleAdmin Role = "admin"
    RoleUser  Role = "user"
    RoleGuest Role = "guest"
)

type Permission struct {
    Resource string
    Action   string
}

var rolePermissions = map[Role][]Permission{
    RoleAdmin: {
        {"conversations", "create"},
        {"conversations", "read"},
        {"conversations", "update"},
        {"conversations", "delete"},
        {"skills", "manage"},
        {"users", "manage"},
    },
    RoleUser: {
        {"conversations", "create"},
        {"conversations", "read"},
        {"conversations", "update"},
        {"conversations", "delete"},
        {"skills", "use"},
    },
    RoleGuest: {
        {"conversations", "read"},
    },
}

func (r Role) HasPermission(resource, action string) bool {
    permissions := rolePermissions[r]
    for _, p := range permissions {
        if p.Resource == resource && p.Action == action {
            return true
        }
    }
    return false
}
```

### 3.2 Resource-Level Authorization

```go
func (s *ConversationService) GetConversation(ctx context.Context, userID, conversationID string) (*Conversation, error) {
    conv, err := s.store.GetConversation(ctx, conversationID)
    if err != nil {
        return nil, err
    }
    
    // Check ownership
    if conv.UserID != userID {
        return nil, errors.New("forbidden: not owner")
    }
    
    return conv, nil
}
```

## 4. Input Validation

### 4.1 Request Validation

```go
type CreateConversationRequest struct {
    Title    string `json:"title"`
    UserID   string `json:"user_id" validate:"required,uuid"`
}

func (h *ConversationHandler) CreateConversation(w http.ResponseWriter, r *http.Request) {
    var req CreateConversationRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }
    
    if err := validator.New().Struct(req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Proceed with creation
}
```

### 4.2 Sanitization

```go
func sanitizePath(path string) (string, error) {
    // Remove null bytes
    path = strings.ReplaceAll(path, "\x00", "")
    
    // Normalize path
    path = filepath.Clean(path)
    
    // Check for path traversal
    if strings.Contains(path, "..") {
        return "", errors.New("invalid path")
    }
    
    return path, nil
}
```

## 5. Data Protection

### 5.1 Encryption at Rest

- **Database**: PostgreSQL Transparent Data Encryption (TDE)
- **Secrets**: Kubernetes Secrets encrypted at rest
- **Files**: Encrypt sensitive files before storage

### 5.2 Encryption in Transit

```go
// TLS Configuration
tlsConfig := &tls.Config{
    MinVersion:               tls.VersionTLS12,
    CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
    PreferServerCipherSuites: true,
    CipherSuites: []uint16{
        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
        tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
        tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
    },
}
```

### 5.3 Sensitive Data Handling

```go
func maskEmail(email string) string {
    atIndex := strings.Index(email, "@")
    if atIndex == -1 {
        return email
    }
    prefix := email[:atIndex]
    domain := email[atIndex:]
    if len(prefix) <= 2 {
        return prefix + "***" + domain
    }
    return prefix[:2] + "***" + domain
}
```

## 6. Secure Configuration

### 6.1 Environment Variables

```go
func loadConfig() (*Config, error) {
    config := &Config{
        DatabaseURL:     getEnv("DATABASE_URL", ""),
        RedisURL:        getEnv("REDIS_URL", ""),
        FrontModelAPIKey: getEnv("FRONT_MODEL_API_KEY", ""),
    }
    
    // Validate required variables
    if config.DatabaseURL == "" {
        return nil, errors.New("DATABASE_URL is required")
    }
    
    return config, nil
}

func getEnv(key, defaultValue string) string {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value
}
```

### 6.2 Secret Management

- **Kubernetes Secrets**: Store sensitive configuration
- **Vault Integration**: For enterprise-grade secret management
- **Never commit secrets**: Use .gitignore for configuration files

## 7. Rate Limiting

### 7.1 Rate Limiter Implementation

```go
type RateLimiter struct {
    store  *RedisCache
    prefix string
}

func (r *RateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
    redisKey := fmt.Sprintf("%s:%s", r.prefix, key)
    
    count, err := r.store.Incr(ctx, redisKey)
    if err != nil {
        return false, err
    }
    
    if count == 1 {
        r.store.Expire(ctx, redisKey, window)
    }
    
    return count <= int64(limit), nil
}
```

### 7.2 Rate Limit Configuration

```yaml
rate_limiting:
  enabled: true
  limits:
    requests_per_minute: 100
    requests_per_hour: 1000
    messages_per_conversation: 1000
```

## 8. Security Headers

### 8.1 HTTP Security Headers

```go
func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        w.Header().Set("Content-Security-Policy", "default-src 'self'")
        w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
        next.ServeHTTP(w, r)
    })
}
```

## 9. Audit Logging

### 9.1 Audit Log Structure

```go
type AuditLog struct {
    ID        string    `json:"id"`
    UserID    string    `json:"user_id"`
    Action    string    `json:"action"`
    Resource  string    `json:"resource"`
    ResourceID string    `json:"resource_id"`
    Timestamp time.Time `json:"timestamp"`
    IPAddress string    `json:"ip_address"`
    UserAgent string    `json:"user_agent"`
    Success   bool      `json:"success"`
    Error     string    `json:"error,omitempty"`
}

func (s *AuditService) Log(ctx context.Context, logEntry AuditLog) error {
    return s.store.CreateAuditLog(ctx, logEntry)
}
```

### 9.2 Sensitive Actions to Log

| Action | Resource |
|--------|----------|
| `create` | conversation |
| `delete` | conversation |
| `login` | user |
| `logout` | user |
| `install` | skill |
| `uninstall` | skill |
| `start` | mcp_plugin |
| `stop` | mcp_plugin |

## 10. Vulnerability Management

### 10.1 Dependency Scanning

```yaml
# .github/workflows/vulnerability-scan.yml
name: Vulnerability Scan
on: [push]
jobs:
  scan:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Run trivy scan
      uses: aquasecurity/trivy-action@master
      with:
        scan-type: 'fs'
        scan-ref: '.'
        format: 'table'
        exit-code: '1'
        severity: 'CRITICAL,HIGH'
```

### 10.2 Security Testing

- **Penetration Testing**: Regular security assessments
- **Static Code Analysis**: golangci-lint with security rules
- **Dynamic Testing**: OWASP ZAP for API security

## 11. Incident Response

### 11.1 Incident Response Plan

1. **Detection**: Monitor logs and alerts for suspicious activity
2. **Containment**: Isolate affected systems
3. **Eradication**: Remove malicious code/data
4. **Recovery**: Restore systems to normal operation
5. **Lessons Learned**: Document and improve

### 11.2 Alerting

```yaml
alerting:
  enabled: true
  thresholds:
    error_rate: 0.1
    request_timeout: 5s
    unauthorized_attempts: 10
  channels:
    - slack
    - email
```

## 12. Compliance

### 12.1 Data Privacy

- **GDPR**: EU data protection compliance
- **CCPA**: California consumer privacy compliance
- **Data Retention**: Define retention policies for user data

### 12.2 Compliance Checklist

- [ ] Data encryption at rest and in transit
- [ ] Access control and audit logging
- [ ] Regular security assessments
- [ ] Incident response plan
- [ ] Data breach notification process
- [ ] Privacy policy and terms of service