# LinkFlow Architecture Plan

> **Primary Goal**: Maximum Performance  
> **Stack**: Laravel (User API) + Go (Execution Engine)  
> **Author**: System Architect  
> **Created**: 2026-01-30

---

## 📊 Executive Summary

This document outlines a high-performance architecture where:
- **Laravel** handles user-facing API (auth, CRUD, validation)
- **Go** handles all heavy execution (workers, processing, real-time)
- **Performance gain**: 10-50x faster execution compared to pure PHP workers

---

## 🎯 Why This Architecture?

| Metric | Pure Laravel | Laravel + Go Engine |
|--------|--------------|---------------------|
| Requests/sec | ~500-1,000 | ~50,000-100,000 |
| Memory per worker | ~50MB | ~2-5MB |
| Cold start | ~100-200ms | ~1-5ms |
| Concurrent connections | ~1,000 | ~100,000+ |
| CPU-bound tasks | Slow (single-threaded) | Fast (multi-core) |

---

## 🏗️ Complete Architecture

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           LOAD BALANCER                                  │
│                        (Nginx / Cloudflare)                             │
└─────────────────────────────┬───────────────────────────────────────────┘
                              │
        ┌─────────────────────┴─────────────────────┐
        │                                           │
        ▼                                           ▼
┌───────────────────┐                    ┌─────────────────────┐
│   LARAVEL API     │                    │   GO REAL-TIME      │
│   (PHP-FPM)       │                    │   (WebSocket/SSE)   │
│                   │                    │                     │
│ • Authentication  │                    │ • Live updates      │
│ • User CRUD       │                    │ • Notifications     │
│ • Validation      │                    │ • Streaming         │
│ • Job Dispatch    │                    │ • Chat/Collab       │
│ • OAuth/Passport  │                    │                     │
└────────┬──────────┘                    └──────────┬──────────┘
         │                                          │
         │              ┌──────────────┐            │
         └──────────────►   REDIS      ◄────────────┘
                        │              │
                        │ • Queue      │
                        │ • Pub/Sub    │
                        │ • Cache      │
                        │ • Sessions   │
                        └──────┬───────┘
                               │
                               ▼
                    ┌─────────────────────┐
                    │   GO WORKER ENGINE  │
                    │                     │
                    │ • Worker Pool       │
                    │ • Task Executor     │
                    │ • Scheduler         │
                    │ • File Processing   │
                    │ • API Integrations  │
                    │ • Heavy Computation │
                    └──────────┬──────────┘
                               │
         ┌─────────────────────┼─────────────────────┐
         │                     │                     │
         ▼                     ▼                     ▼
┌─────────────┐      ┌─────────────────┐    ┌──────────────┐
│  PostgreSQL │      │  Object Storage │    │  External    │
│  (Primary)  │      │  (S3/MinIO)     │    │  APIs        │
└─────────────┘      └─────────────────┘    └──────────────┘
```

---

## 📁 Complete Project Structure

```
linkflow/
│
├── linkflow-api/                    # LARAVEL (Your Current Project)
│   ├── app/
│   │   ├── Http/
│   │   │   ├── Controllers/Api/     # ✅ User-facing API endpoints
│   │   │   ├── Middleware/          # ✅ Auth, Rate limiting
│   │   │   └── Requests/            # ✅ Validation
│   │   ├── Models/                  # ✅ Eloquent ORM
│   │   ├── Jobs/                    # 🔲 Job dispatchers (to Go)
│   │   │   ├── ProcessFileJob.php
│   │   │   ├── SendBulkEmailJob.php
│   │   │   └── GenerateReportJob.php
│   │   ├── Events/                  # 🔲 Real-time events
│   │   └── Services/                # ✅ Business logic
│   ├── config/
│   │   ├── queue.php                # 🔲 Redis queue config
│   │   └── engine.php               # 🔲 Go engine config
│   ├── routes/
│   │   └── api.php                  # ✅ API routes
│   └── database/                    # ✅ Migrations
│
├── linkflow-engine/                 # GO EXECUTION ENGINE (New)
│   ├── cmd/
│   │   ├── worker/                  # Worker process entry
│   │   │   └── main.go
│   │   ├── scheduler/               # Scheduled tasks entry
│   │   │   └── main.go
│   │   └── realtime/                # WebSocket server entry
│   │       └── main.go
│   │
│   ├── internal/                    # Private packages
│   │   ├── config/                  # Configuration
│   │   │   └── config.go
│   │   ├── queue/                   # Queue consumer
│   │   │   ├── consumer.go
│   │   │   └── redis.go
│   │   ├── worker/                  # Worker pool
│   │   │   ├── pool.go
│   │   │   └── worker.go
│   │   ├── executor/                # Task executors
│   │   │   ├── executor.go
│   │   │   ├── file_processor.go
│   │   │   ├── email_sender.go
│   │   │   └── report_generator.go
│   │   ├── storage/                 # Database & storage
│   │   │   ├── postgres.go
│   │   │   ├── redis.go
│   │   │   └── s3.go
│   │   └── realtime/                # WebSocket handling
│   │       ├── hub.go
│   │       └── client.go
│   │
│   ├── pkg/                         # Public packages
│   │   ├── protocol/                # Shared message format
│   │   │   └── message.go
│   │   └── logger/                  # Structured logging
│   │       └── logger.go
│   │
│   ├── migrations/                  # Go-specific migrations (if any)
│   ├── go.mod                       # Go module file
│   ├── go.sum                       # Dependencies lock
│   ├── Makefile                     # Build commands
│   └── Dockerfile                   # Container build
│
├── docker/                          # Docker configs
│   ├── nginx/
│   │   └── nginx.conf
│   ├── php/
│   │   └── Dockerfile
│   └── go/
│       └── Dockerfile
│
├── docker-compose.yml               # Full stack orchestration
├── docker-compose.dev.yml           # Development setup
└── Makefile                         # Root-level commands
```

---

## 🔄 Data Flow Diagrams

### Flow 1: Simple API Request (Laravel Only)

```
User Request → Nginx → Laravel → Database → Response
     │                    │
     │         ~50-100ms  │
     └────────────────────┘
```

### Flow 2: Heavy Task (Laravel + Go)

```
User Request → Laravel → Dispatch Job → Redis Queue
     │             │           │
     │     ~10ms   │           │
     │             │           ▼
     │             │     Go Worker Pool
     │             │           │
     │             │    ~5-50ms (parallel)
     │             │           │
     │             │           ▼
     │             │     Task Complete
     │             │           │
     │             ◄───────────┘
     │                  │
     │      Optional: Real-time notification
     │                  │
     ◄──────────────────┘
```

### Flow 3: Real-Time Updates (Go WebSocket)

```
                    ┌─────────────────┐
                    │   Go WebSocket  │
User ◄──────────────│     Server      │
     WebSocket      │                 │
     Connection     └────────┬────────┘
                             │
                    Redis Pub/Sub
                             │
                    ┌────────┴────────┐
                    │                 │
              Go Worker          Laravel
              (publishes)        (publishes)
```

---

## 📋 What Goes Where

### Laravel Handles (User-Facing)

| Component | Why Laravel | Performance Impact |
|-----------|-------------|-------------------|
| **Authentication** | OAuth, Passport, Sessions | Low (cached) |
| **User CRUD** | Eloquent ORM, Validation | Low |
| **API Routing** | Clean syntax, middleware | Low |
| **Job Dispatching** | Queue facade, easy syntax | Negligible |
| **Database Migrations** | Schema management | N/A |
| **Admin Panel** | Rapid development | N/A |

### Go Handles (Heavy Execution)

| Component | Why Go | Performance Gain |
|-----------|--------|-----------------|
| **Worker Pool** | Goroutines, low memory | 10-50x |
| **File Processing** | Concurrent I/O | 20-100x |
| **PDF/Image Generation** | CPU-bound | 5-20x |
| **Bulk Email Sending** | Concurrent HTTP | 50-100x |
| **Report Generation** | Large data processing | 10-50x |
| **WebSocket Server** | 100k+ connections | 100x+ |
| **Scheduled Tasks** | Cron-like, precise | 5-10x |
| **External API Calls** | Concurrent requests | 20-50x |
| **Data Aggregation** | Stream processing | 10-50x |

---

## 📨 Communication Protocol

### Job Message Format (Laravel → Go)

```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "process_file",
  "queue": "high",
  "payload": {
    "workspace_id": 1,
    "user_id": 42,
    "file_path": "uploads/2026/01/document.pdf",
    "options": {
      "extract_text": true,
      "generate_thumbnail": true
    }
  },
  "attempts": 0,
  "max_attempts": 3,
  "timeout": 300,
  "created_at": "2026-01-30T10:00:00Z"
}
```

### Job Result Format (Go → Laravel)

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "result": {
    "text_content": "Extracted text...",
    "thumbnail_path": "thumbnails/doc_thumb.jpg",
    "pages": 15
  },
  "duration_ms": 245,
  "completed_at": "2026-01-30T10:00:01Z"
}
```

### Real-Time Event Format (Go → Client)

```json
{
  "event": "job.completed",
  "channel": "workspace.1",
  "data": {
    "job_id": "550e8400-e29b-41d4-a716-446655440000",
    "type": "process_file",
    "result": { ... }
  },
  "timestamp": "2026-01-30T10:00:01Z"
}
```

---

## 🚀 Implementation Phases

### Phase 1: Foundation (Week 1-2)

**Laravel Side:**
- [ ] Configure Redis queue driver
- [ ] Create base Job class for Go engine
- [ ] Create first job: `ProcessFileJob`
- [ ] Add job status tracking table

**Go Side:**
- [ ] Learn Go basics (see Learning Path below)
- [ ] Set up Go project structure
- [ ] Create Redis queue consumer
- [ ] Create basic worker pool

### Phase 2: Core Workers (Week 3-4)

**Go Side:**
- [ ] File processing executor
- [ ] Database connection (same DB as Laravel)
- [ ] Job result callback to Laravel
- [ ] Error handling & retry logic
- [ ] Logging & monitoring

**Laravel Side:**
- [ ] Job status webhook endpoint
- [ ] Real-time event broadcasting setup

### Phase 3: Real-Time (Week 5-6)

**Go Side:**
- [ ] WebSocket server
- [ ] Redis Pub/Sub integration
- [ ] Connection management (100k+ support)
- [ ] Heartbeat & reconnection

**Laravel Side:**
- [ ] Event dispatching to Go WebSocket
- [ ] Client-side WebSocket integration

### Phase 4: Production Ready (Week 7-8)

- [ ] Docker containerization
- [ ] Health checks & monitoring
- [ ] Graceful shutdown
- [ ] Rate limiting in Go
- [ ] Load testing & optimization
- [ ] Documentation

---

## 📚 Go Learning Path (For Laravel Developer)

### Week 1: Go Fundamentals

```go
// Day 1-2: Basic syntax
package main

import "fmt"

func main() {
    // Variables
    name := "LinkFlow"
    count := 42
    
    // Control flow
    if count > 0 {
        fmt.Println(name)
    }
    
    // Loops
    for i := 0; i < 10; i++ {
        fmt.Println(i)
    }
}
```

**Resources:**
- https://go.dev/tour/ (Official Tour - 2 hours)
- https://gobyexample.com/ (Examples)

### Week 2: Go for Backend

```go
// Structs (like PHP classes)
type Job struct {
    ID      string
    Type    string
    Payload map[string]interface{}
}

// Methods
func (j *Job) Process() error {
    // Process the job
    return nil
}

// Interfaces
type Executor interface {
    Execute(job *Job) error
}

// Goroutines (concurrency)
go func() {
    // This runs concurrently
}()

// Channels (communication)
jobs := make(chan Job, 100)
jobs <- newJob  // Send
job := <-jobs   // Receive
```

### Week 3: Practical Go

```go
// HTTP Server
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("OK"))
})
http.ListenAndServe(":8080", nil)

// JSON handling
json.Marshal(data)    // Encode
json.Unmarshal(bytes, &data)  // Decode

// Database
db.Query("SELECT * FROM jobs WHERE status = $1", "pending")

// Redis
rdb.LPush(ctx, "queue:jobs", jobJSON)
```

---

## 🔧 Performance Optimizations

### Go Engine Optimizations

```go
// 1. Worker Pool with optimal size
poolSize := runtime.NumCPU() * 2

// 2. Connection pooling
db.SetMaxOpenConns(100)
db.SetMaxIdleConns(10)

// 3. Batch processing
batchSize := 100
jobs := fetchJobs(batchSize)
for _, job := range jobs {
    go process(job)
}

// 4. Memory pooling
var bufferPool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 4096)
    },
}

// 5. Efficient JSON parsing
jsoniter.ConfigFastest.Unmarshal(data, &job)
```

### Redis Optimizations

```go
// 1. Pipeline commands
pipe := rdb.Pipeline()
pipe.LPush(ctx, "queue", job1)
pipe.LPush(ctx, "queue", job2)
pipe.Exec(ctx)

// 2. Batch pop
rdb.BRPop(ctx, 0, "queue:high", "queue:default", "queue:low")

// 3. Pub/Sub for real-time
pubsub := rdb.Subscribe(ctx, "events")
ch := pubsub.Channel()
for msg := range ch {
    broadcast(msg.Payload)
}
```

### Database Optimizations

```go
// 1. Prepared statements
stmt, _ := db.Prepare("UPDATE jobs SET status = $1 WHERE id = $2")
defer stmt.Close()
stmt.Exec("completed", jobID)

// 2. Bulk inserts
tx, _ := db.Begin()
stmt, _ := tx.Prepare(pq.CopyIn("results", "job_id", "data"))
for _, r := range results {
    stmt.Exec(r.JobID, r.Data)
}
stmt.Exec()
tx.Commit()

// 3. Read replicas
readDB := connectToReplica()
writeDB := connectToPrimary()
```

---

## 🐳 Docker Setup

### docker-compose.yml

```yaml
version: '3.8'

services:
  # Laravel API
  api:
    build:
      context: ./linkflow-api
      dockerfile: ../docker/php/Dockerfile
    volumes:
      - ./linkflow-api:/var/www/html
    depends_on:
      - postgres
      - redis
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - QUEUE_CONNECTION=redis

  # Nginx
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./docker/nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./linkflow-api/public:/var/www/html/public
    depends_on:
      - api
      - go-realtime

  # Go Worker Engine
  go-worker:
    build:
      context: ./linkflow-engine
      dockerfile: Dockerfile
    command: ./worker
    deploy:
      replicas: 4  # Scale workers
    depends_on:
      - postgres
      - redis
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - WORKER_CONCURRENCY=100

  # Go Scheduler
  go-scheduler:
    build:
      context: ./linkflow-engine
      dockerfile: Dockerfile
    command: ./scheduler
    depends_on:
      - redis

  # Go Real-Time Server
  go-realtime:
    build:
      context: ./linkflow-engine
      dockerfile: Dockerfile
    command: ./realtime
    ports:
      - "8080:8080"
    depends_on:
      - redis

  # PostgreSQL
  postgres:
    image: postgres:16-alpine
    volumes:
      - postgres_data:/var/lib/postgresql/data
    environment:
      - POSTGRES_DB=linkflow
      - POSTGRES_USER=linkflow
      - POSTGRES_PASSWORD=secret

  # Redis
  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

---

## 📊 Monitoring & Observability

### Metrics to Track

| Metric | Tool | Alert Threshold |
|--------|------|-----------------|
| Jobs/second | Prometheus | < 100/s |
| Job latency (p99) | Prometheus | > 5s |
| Queue depth | Prometheus | > 10,000 |
| Worker memory | Prometheus | > 500MB |
| Error rate | Prometheus | > 1% |
| WebSocket connections | Prometheus | - |

### Health Check Endpoints

```
GET /health/api      → Laravel API health
GET /health/worker   → Go worker health
GET /health/realtime → Go WebSocket health
GET /metrics         → Prometheus metrics
```

---

## ✅ Complete Checklist

### Laravel (linkflow-api)

- [x] User authentication (OAuth/Passport)
- [x] User CRUD
- [x] Workspace management
- [x] Subscription/Plans
- [x] Invitations
- [ ] Redis queue configuration
- [ ] Job dispatcher classes
- [ ] Job status tracking
- [ ] Webhook for job results
- [ ] Event broadcasting

### Go Engine (linkflow-engine)

- [ ] Project structure setup
- [ ] Configuration management
- [ ] Redis queue consumer
- [ ] Worker pool implementation
- [ ] Database connection
- [ ] Task executors
- [ ] Error handling & retries
- [ ] Logging (structured)
- [ ] Health check endpoint
- [ ] Graceful shutdown
- [ ] WebSocket server
- [ ] Redis Pub/Sub
- [ ] Metrics endpoint

### Infrastructure

- [ ] Docker setup
- [ ] Nginx configuration
- [ ] Redis configuration
- [ ] PostgreSQL optimization
- [ ] CI/CD pipeline
- [ ] Monitoring setup
- [ ] Load testing

---

## 🎯 Performance Targets

| Metric | Target |
|--------|--------|
| API Response (p50) | < 50ms |
| API Response (p99) | < 200ms |
| Job Processing (p50) | < 100ms |
| Job Processing (p99) | < 1s |
| Jobs/second | 10,000+ |
| WebSocket Connections | 100,000+ |
| Memory per worker | < 10MB |
| CPU utilization | < 70% |

---

## 📞 Next Steps

1. **Start Go Learning** - Complete go.dev/tour (2 hours)
2. **Create linkflow-engine/** - Basic Go project structure
3. **Configure Laravel Redis** - Queue driver setup
4. **Build First Worker** - Simple job consumer
5. **End-to-End Test** - Laravel → Redis → Go → DB

---

*This document should be updated as the project evolves.*
