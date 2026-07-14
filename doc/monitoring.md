# TianNiu Backend System Design - Monitoring & Observability

## 1. Overview

### 1.1 Monitoring Strategy

- **Metrics**: Track system performance and usage
- **Logging**: Record events and errors for debugging
- **Tracing**: Follow requests across services
- **Alerting**: Notify on critical conditions

### 1.2 Monitoring Stack

```
┌──────────────────────────────────────────────────────────────────────────┐
│                        Observability Layer                             │
│                                                                         │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────┐   │
│  │ Prometheus  │    │   Loki      │    │ Jaeger      │    │ Alert   │   │
│  │ (Metrics)   │    │  (Logging)  │    │ (Tracing)   │    │ Manager │   │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────┘   │
│         │                  │                  │                  │      │
│         └──────────────────┴──────────────────┴──────────────────┘      │
│                              │                                         │
│                              ▼                                         │
│                      ┌─────────────┐                                   │
│                      │  Grafana    │                                   │
│                      │ (Dashboard) │                                   │
│                      └─────────────┘                                   │
│                                                                         │
└──────────────────────────────────────────────────────────────────────────┘
```

## 2. Metrics

### 2.1 Metric Categories

| Category | Metrics |
|----------|---------|
| **HTTP** | Request count, duration, status codes |
| **Memory** | Memory updates, retrievals, cache hits |
| **LLM** | Call count, latency, token usage |
| **Skill** | Execution count, success rate |
| **Database** | Query count, latency, connection pool |
| **System** | CPU, memory, disk, network |

### 2.2 HTTP Metrics

```go
var (
    httpRequestCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_request_count",
            Help: "Total HTTP requests",
        },
        []string{"endpoint", "method", "status"},
    )
    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "http_request_duration_seconds",
            Help:    "HTTP request duration",
            Buckets: prometheus.DefBuckets,
        },
        []string{"endpoint", "method"},
    )
)
```

### 2.3 Memory Metrics

```go
var (
    memoryUpdateCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "memory_update_count",
            Help: "Memory update operations",
        },
        []string{"level"},
    )
    memoryRetrievalCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "memory_retrieval_count",
            Help: "Memory retrieval operations",
        },
        []string{"level", "success"},
    )
    memoryCacheHitCount = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "memory_cache_hit_count",
            Help: "Memory cache hits",
        },
    )
)
```

### 2.4 LLM Metrics

```go
var (
    llmCallCount = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_call_count",
            Help: "LLM API calls",
        },
        []string{"model", "success"},
    )
    llmTokenUsage = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "llm_token_usage",
            Help: "LLM token usage",
        },
        []string{"model", "type"},
    )
)
```

### 2.5 Custom Metric Collector

```go
func init() {
    prometheus.MustRegister(httpRequestCount)
    prometheus.MustRegister(httpRequestDuration)
    prometheus.MustRegister(memoryUpdateCount)
    prometheus.MustRegister(memoryRetrievalCount)
    prometheus.MustRegister(llmCallCount)
    prometheus.MustRegister(llmTokenUsage)
}
```

## 3. Logging

### 3.1 Log Structure

```json
{
    "timestamp": "2024-01-01T12:00:00Z",
    "level": "INFO",
    "service": "tianniu",
    "component": "agent",
    "request_id": "abc-123",
    "user_id": "user-456",
    "conversation_id": "conv-789",
    "message": "Processing conversation",
    "details": {
        "turn_count": 5,
        "memory_updated": true
    }
}
```

### 3.2 Log Levels

| Level | Usage |
|-------|-------|
| **DEBUG** | Detailed debug information |
| **INFO** | General operational information |
| **WARN** | Warning conditions |
| **ERROR** | Error conditions |
| **FATAL** | Critical errors causing shutdown |

### 3.3 Structured Logging

```go
import (
    "github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
    log.SetFormatter(&logrus.JSONFormatter{})
    log.SetLevel(logrus.InfoLevel)
}

func exampleFunction(ctx context.Context) {
    log.WithFields(logrus.Fields{
        "request_id":      ctx.Value("request_id"),
        "user_id":         ctx.Value("user_id"),
        "conversation_id": ctx.Value("conversation_id"),
    }).Info("Processing request")
}
```

### 3.4 Request Logging Middleware

```go
func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // Extract request ID
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        
        // Create logger with context
        logger := log.WithFields(logrus.Fields{
            "request_id": requestID,
            "method":     r.Method,
            "path":       r.URL.Path,
        })
        
        // Wrap response writer to capture status code
        ww := &responseWriter{ResponseWriter: w}
        
        // Process request
        next.ServeHTTP(ww, r)
        
        // Log request completion
        logger.WithFields(logrus.Fields{
            "status":   ww.status,
            "duration": time.Since(start).String(),
        }).Info("Request completed")
    })
}
```

## 4. Tracing

### 4.1 Distributed Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("tianniu")

func processConversation(ctx context.Context, userID, conversationID string) error {
    ctx, span := tracer.Start(ctx, "processConversation")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("user_id", userID),
        attribute.String("conversation_id", conversationID),
    )
    
    // ...
    
    return nil
}
```

### 4.2 Trace Context Propagation

```go
func tracingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        
        // Extract trace context from headers
        propagator := otel.GetTextMapPropagator()
        ctx = propagator.Extract(ctx, propagation.HeaderCarrier(r.Header))
        
        // Create span
        ctx, span := tracer.Start(ctx, r.URL.Path)
        defer span.End()
        
        // Inject trace context into response
        propagator.Inject(ctx, propagation.HeaderCarrier(w.Header()))
        
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## 5. Alerting

### 5.1 Alert Rules

```yaml
groups:
- name: tianniu-alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_request_count{status=~"5.."}[5m]) / rate(http_request_count[5m]) > 0.1
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value }}% for the last 5 minutes"

  - alert: HighLatency
    expr: histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (le)) > 5
    for: 1m
    labels:
      severity: warning
    annotations:
      summary: "High request latency"
      description: "95th percentile latency is {{ $value }}s"

  - alert: MemoryUpdateFailure
    expr: rate(memory_retrieval_count{success="false"}[5m]) > 0.1
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Memory update failures"
      description: "{{ $value }} memory updates failed in the last 5 minutes"

  - alert: LLMServiceUnavailable
    expr: llm_call_count{success="false"} / llm_call_count > 0.5
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "LLM service unavailable"
      description: "50% of LLM calls are failing"
```

### 5.2 Alert Channels

| Channel | Description |
|---------|-------------|
| **Slack** | Real-time alerts to channel |
| **Email** | Critical alerts to team |
| **PagerDuty** | On-call notifications |
| **Webhook** | Custom integration |

## 6. Dashboard

### 6.1 Grafana Dashboard Structure

| Section | Panels |
|---------|--------|
| **Overview** | Request rate, error rate, latency |
| **Memory** | Memory updates, retrievals, cache hits |
| **LLM** | Call count, token usage, latency |
| **Skills** | Execution count, success rate |
| **System** | CPU, memory, disk I/O |

### 6.2 Example Panel Configuration

```json
{
    "id": 1,
    "title": "HTTP Request Rate",
    "type": "graph",
    "targets": [
        {
            "expr": "rate(http_request_count[5m])",
            "legendFormat": "{{endpoint}}"
        }
    ],
    "yAxes": [
        {
            "label": "Requests/sec",
            "format": "short"
        }
    ]
}
```

## 7. Health Checks

### 7.1 Health Check Endpoint

```go
func healthHandler(w http.ResponseWriter, r *http.Request) {
    status := "healthy"
    services := map[string]string{
        "database":   "healthy",
        "redis":      "healthy",
        "vector_db":  "healthy",
        "llm":        "healthy",
    }
    
    // Check database connection
    if err := checkDatabase(); err != nil {
        services["database"] = "unhealthy"
        status = "unhealthy"
    }
    
    // Check redis connection
    if err := checkRedis(); err != nil {
        services["redis"] = "unhealthy"
        status = "unhealthy"
    }
    
    response := map[string]interface{}{
        "status":    status,
        "timestamp": time.Now().UTC().Format(time.RFC3339),
        "services":  services,
    }
    
    w.Header().Set("Content-Type", "application/json")
    if status == "unhealthy" {
        w.WriteHeader(http.StatusServiceUnavailable)
    } else {
        w.WriteHeader(http.StatusOK)
    }
    json.NewEncoder(w).Encode(response)
}
```

### 7.2 Kubernetes Health Checks

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5

readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 5
```

## 8. Best Practices

### 8.1 Metric Naming Conventions

- Use snake_case for metric names
- Include units in names (e.g., `duration_seconds`)
- Use labels for dimensions (e.g., `endpoint`, `status`)

### 8.2 Logging Best Practices

- Always include context (request ID, user ID)
- Avoid logging sensitive information
- Use structured logging for easier parsing
- Set appropriate log levels

### 8.3 Tracing Best Practices

- Instrument critical paths
- Use meaningful span names
- Add relevant attributes to spans
- Propagate trace context across services

### 8.4 Alerting Best Practices

- Set appropriate thresholds
- Use `for` clause to avoid flapping
- Include actionable information in annotations
- Group related alerts