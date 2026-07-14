# TianNiu Backend System Design - Deployment

## 1. Overview

### 1.1 Deployment Architecture

```
┌──────────────────────────────────────────────────────────────────────────┐
│                           Kubernetes Cluster                            │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐  │
│  │   API Gateway    │    │   Application    │    │   Database       │  │
│  │   (Ingress/Nginx)│    │    (Pods)        │    │   (StatefulSet)  │  │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘  │
│         │                         │                         │          │
│         ▼                         ▼                         ▼          │
│  ┌──────────────────┐    ┌──────────────────┐    ┌──────────────────┐  │
│  │   Load Balancer  │    │   Service Mesh   │    │   Persistent     │  │
│  │                  │    │   (Istio)        │    │   Volume         │  │
│  └──────────────────┘    └──────────────────┘    └──────────────────┘  │
│                                                                         │
└──────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Deployment Environments

| Environment | Purpose | Configuration |
|-------------|---------|---------------|
| **Development** | Local development | Local database, debug enabled |
| **Staging** | Testing before production | Staging database, production-like |
| **Production** | Live environment | Production database, high availability |

## 2. Containerization

### 2.1 Dockerfile

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o tianniu ./tianniu/main.go

# Runtime stage
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/tianniu .
COPY --from=builder /app/config ./config

EXPOSE 8080

CMD ["./tianniu"]
```

### 2.2 Docker Compose

```yaml
version: '3.8'

services:
  tianniu:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://admin:password@postgres:5432/tianniu
      - REDIS_URL=redis://redis:6379/0
      - FRONT_MODEL_API_KEY=${FRONT_MODEL_API_KEY}
    depends_on:
      - postgres
      - redis
      - vector-db
    volumes:
      - ./skills:/app/skills
      - ./mcp_plugins:/app/mcp_plugins

  postgres:
    image: postgres:16
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=tianniu
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    volumes:
      - redis-data:/data
    ports:
      - "6379:6379"

  vector-db:
    image: pgvector/pgvector:pg16
    environment:
      - POSTGRES_USER=admin
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=memory_db
    volumes:
      - vector-data:/var/lib/postgresql/data
    ports:
      - "5433:5432"

volumes:
  postgres-data:
  redis-data:
  vector-data:
```

## 3. Kubernetes Deployment

### 3.1 Deployment Configuration

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tianniu
  labels:
    app: tianniu
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tianniu
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
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
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: database-url
        - name: REDIS_URL
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: redis-url
        - name: FRONT_MODEL_API_KEY
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: front-model-api-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### 3.2 Service Configuration

```yaml
apiVersion: v1
kind: Service
metadata:
  name: tianniu-service
spec:
  selector:
    app: tianniu
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
```

### 3.3 Ingress Configuration

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: tianniu-ingress
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - api.tianniu.example.com
    secretName: tianniu-tls
  rules:
  - host: api.tianniu.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: tianniu-service
            port:
              number: 80
```

### 3.4 StatefulSet for PostgreSQL

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:16
        ports:
        - containerPort: 5432
        env:
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: postgres-user
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tianniu-secrets
              key: postgres-password
        - name: POSTGRES_DB
          value: tianniu
        volumeMounts:
        - name: postgres-data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: postgres-data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 10Gi
```

### 3.5 Redis Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis
  labels:
    app: redis
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:7
        ports:
        - containerPort: 6379
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
        volumeMounts:
        - name: redis-data
          mountPath: /data
      volumes:
      - name: redis-data
        persistentVolumeClaim:
          claimName: redis-pvc
```

## 4. Secrets Management

### 4.1 Kubernetes Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tianniu-secrets
type: Opaque
data:
  database-url: cG9zdGdyZXM6Ly9hZG1pbi5wb3N0Z3Jlczo1NDMyL3RpYW5uaXU=
  redis-url: cmVkaXM6Ly9yZWRpczo2Mzc5LzA=
  front-model-api-key: dGVzdC1hcGkta2V5
  postgres-user: YWRtaW4=
  postgres-password: cGFzc3dvcmQ=
```

### 4.2 Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `DATABASE_URL` | PostgreSQL connection string | Yes |
| `REDIS_URL` | Redis connection string | Yes |
| `FRONT_MODEL_API_KEY` | Front LLM API key | Yes |
| `BACK_MODEL_API_KEY` | Back LLM API key | No |
| `SKILLS_DIR` | Skills directory | No |
| `MCP_PLUGINS_DIR` | MCP plugins directory | No |

## 5. CI/CD Pipeline

### 5.1 Pipeline Stages

```
Code Commit
    │
    ▼
Lint → Test → Build → Publish → Deploy
    │         │         │         │
    ▼         ▼         ▼         ▼
golangci-lint  go test  docker build  k8s apply
```

### 5.2 GitHub Actions Workflow

```yaml
name: CI/CD

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Run linter
      run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest && golangci-lint run

  test:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_USER: admin
          POSTGRES_PASSWORD: password
          POSTGRES_DB: test_db
        ports:
          - 5432:5432
      redis:
        image: redis:7
        ports:
          - 6379:6379
    steps:
    - uses: actions/checkout@v4
    - name: Run tests
      run: go test ./...

  build:
    needs: [lint, test]
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Build Docker image
      run: docker build -t tianniu:${{ github.sha }} .
    - name: Push to registry
      run: docker push tianniu:${{ github.sha }}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Deploy to Kubernetes
      run: kubectl apply -f k8s/deployment.yaml
```

## 6. Monitoring Integration

### 6.1 Prometheus Service Monitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: tianniu-monitor
  labels:
    release: prometheus
spec:
  selector:
    matchLabels:
      app: tianniu
  endpoints:
  - port: 8080
    path: /metrics
    interval: 30s
```

### 6.2 Grafana Dashboard

```json
{
  "title": "TianNiu Dashboard",
  "panels": [
    {
      "type": "graph",
      "title": "HTTP Requests",
      "targets": ["rate(http_request_count[5m])"]
    },
    {
      "type": "graph",
      "title": "Request Duration",
      "targets": ["histogram_quantile(0.95, sum(rate(http_request_duration_bucket[5m])) by (le))"]
    },
    {
      "type": "singlestat",
      "title": "Active Agents",
      "targets": ["agent_count"]
    },
    {
      "type": "graph",
      "title": "LLM Calls",
      "targets": ["rate(llm_call_count[5m])"]
    }
  ]
}
```

## 7. Scaling Strategies

### 7.1 Horizontal Pod Autoscaling

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tianniu-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tianniu
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 7.2 Database Scaling

- **Read Replicas**: Add PostgreSQL read replicas for high read traffic
- **Connection Pooling**: Use PgBouncer for connection pooling
- **Sharding**: Consider sharding for very large datasets

## 8. Backup and Restore

### 8.1 Automated Backups

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16
            command: ["/bin/sh", "-c", "pg_dump -h postgres -U admin tianniu > /backup/backup_$(date +%Y%m%d_%H%M%S).sql"]
            volumeMounts:
            - name: backup-volume
              mountPath: /backup
          volumes:
          - name: backup-volume
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

### 8.2 Restore Process

```bash
# Copy backup to pod
kubectl cp backup_20240101.sql postgres-0:/tmp/

# Restore
kubectl exec -it postgres-0 -- psql -U admin -d tianniu -f /tmp/backup_20240101.sql
```

## 9. Environment Configuration

### 9.1 Development Configuration

```yaml
server:
  address: ":8080"
  debug: true

database:
  host: localhost
  port: 5432

llm_providers:
  front_model:
    api_key: ${FRONT_MODEL_API_KEY}
```

### 9.2 Production Configuration

```yaml
server:
  address: ":8080"
  debug: false
  timeout: 30s

database:
  host: postgres-service
  port: 5432
  pool_size: 20

llm_providers:
  front_model:
    api_key: ${FRONT_MODEL_API_KEY}
```

## 10. Security

### 10.1 Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: tianniu-network-policy
spec:
  podSelector:
    matchLabels:
      app: tianniu
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - ipBlock:
        cidr: 10.0.0.0/8
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

### 10.2 TLS Configuration

- Use Let's Encrypt for SSL certificates
- Configure TLS termination at the ingress level
- Enable HTTP/2 for better performance