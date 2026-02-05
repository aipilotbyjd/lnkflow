# LinkFlow Service Catalog

This document describes all services in the LinkFlow platform, their responsibilities, and configuration.

## Overview

LinkFlow consists of a Laravel API (control plane) and 8 Go microservices (execution engine).

---

## API (Laravel)

### Web Service
| Property | Value |
|----------|-------|
| **Path** | `apps/api` |
| **Port** | 8000 (HTTP) |
| **Technology** | Laravel 12, PHP 8.2+ |
| **Dependencies** | PostgreSQL, Redis |
| **Health Endpoint** | `/api/health` |

**Responsibilities:**
- REST API for workflow management
- User authentication (Laravel Passport)
- Workflow CRUD operations
- Job queue for background tasks

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `DB_CONNECTION` | Database driver | `sqlite` |
| `DB_HOST` | Database host | `127.0.0.1` |
| `DB_PORT` | Database port | `3306` |
| `REDIS_HOST` | Redis host | `127.0.0.1` |
| `REDIS_PORT` | Redis port | `6379` |
| `LINKFLOW_ENGINE_SECRET` | Shared secret for engine callbacks | - |

### Queue Worker
| Property | Value |
|----------|-------|
| **Path** | `apps/api` |
| **Container** | `linkflow-queue` |
| **Dependencies** | PostgreSQL, Redis, API |

**Responsibilities:**
- Processes background jobs from Redis queues
- Handles workflow execution tasks
- Priority queue support: `workflows-high`, `workflows-default`, `workflows-low`, `default`

### Scheduler
| Property | Value |
|----------|-------|
| **Path** | `apps/api` |
| **Container** | `linkflow-scheduler` |
| **Profile** | `full` (optional) |

**Responsibilities:**
- Runs Laravel scheduled tasks
- Cron-style job scheduling

---

## Engine Microservices (Go)

### Frontend Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/frontend` |
| **Port** | 8080 (HTTP), 9090 (gRPC) |
| **Container** | `linkflow-frontend` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- API Gateway for the execution engine
- Request routing to internal services
- Authentication & authorization (JWT validation)
- Rate limiting and load balancing

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | `8080` |
| `GRPC_PORT` | gRPC server port | `9090` |
| `HISTORY_ADDR` | History service address | `linkflow-history:7234` |
| `MATCHING_ADDR` | Matching service address | `linkflow-matching:7235` |
| `VISIBILITY_ADDR` | Visibility service address | `linkflow-visibility:7237` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | **Required** |
| `RATE_LIMIT_REQUESTS` | Requests per window | `1000` |
| `RATE_LIMIT_WINDOW` | Rate limit window | `60s` |

---

### History Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/history` |
| **Port** | 8081 (HTTP), 7234 (gRPC) |
| **Container** | `linkflow-history` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Event sourcing store for workflow state
- Event store (append-only log)
- Mutable state (current execution state)
- Sharding for scale
- Replay capability

**Dependencies:** PostgreSQL

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | `7234` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `SHARD_COUNT` | Number of shards | `16` |
| `MATCHING_ADDR` | Matching service address | `linkflow-matching:7235` |
| `TIMER_ADDR` | Timer service address | `linkflow-timer:7238` |

---

### Matching Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/matching` |
| **Port** | 8082 (HTTP), 7235 (gRPC) |
| **Container** | `linkflow-matching` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Task queue management
- Long polling for workers
- Priority handling
- Load distribution and worker dispatching

**Dependencies:** Redis

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | `7235` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `REDIS_URL` | Redis connection string | - |
| `PARTITION_COUNT` | Number of partitions | `4` |
| `TASK_QUEUE_SYNC_INTERVAL` | Task queue sync interval | `1s` |
| `LONG_POLL_TIMEOUT` | Long poll timeout | `60s` |

---

### Worker Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/worker` |
| **Port** | 8083 (HTTP) |
| **Container** | `linkflow-worker` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Executes workflow nodes (HTTP, AI, Email, Database)
- Retry logic with circuit breakers
- Connection pooling
- Credential resolution
- Reports execution results to Laravel via callback

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | `8080` |
| `MATCHING_ADDR` | Matching service address | `linkflow-matching:7235` |
| `HISTORY_ADDR` | History service address | `linkflow-history:7234` |
| `TASK_QUEUE` | Queues to poll | `workflows-high,workflows-default,workflows-low,default` |
| `NUM_WORKERS` | Number of worker goroutines | `4` |
| `POLL_INTERVAL` | Poll interval | `1s` |
| `CALLBACK_URL` | Laravel callback URL | `http://linkflow-api:8000/api/v1/jobs/callback` |
| `CALLBACK_SECRET` | Callback authentication secret | **Required** |

---

### Timer Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/timer` |
| **Port** | 8084 (HTTP), 7238 (gRPC) |
| **Container** | `linkflow-timer` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Scheduled workflows (cron)
- Delay nodes (wait X minutes)
- Timeout enforcement
- Retry delays

**Dependencies:** Redis, PostgreSQL

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | `7238` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `REDIS_URL` | Redis connection string | - |
| `HISTORY_ADDR` | History service address | `linkflow-history:7234` |
| `SCAN_INTERVAL` | Timer scan interval | `1s` |
| `BATCH_SIZE` | Batch size for timer processing | `100` |

---

### Visibility Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/visibility` |
| **Port** | 8085 (HTTP), 7237 (gRPC) |
| **Container** | `linkflow-visibility` |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Full-text search for workflows
- Filter by status, time, attributes
- Pagination
- Real-time indexing

**Dependencies:** PostgreSQL, Elasticsearch (optional)

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | `7237` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `DATABASE_URL` | PostgreSQL connection string | - |
| `ELASTICSEARCH_URL` | Elasticsearch URL (optional) | - |

---

### Control Plane Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/control-plane` |
| **Port** | 8087 (HTTP), 7239 (gRPC) |
| **Container** | `linkflow-control-plane` |
| **Profile** | `control` (optional) |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Configuration distribution
- Service discovery
- Multi-region federation
- Health monitoring

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `GRPC_PORT` | gRPC server port | `7239` |
| `HTTP_PORT` | HTTP server port | `8080` |
| `REDIS_URL` | Redis connection string | - |
| `CELL_ID` | Cell identifier | `cell-1` |
| `REGION` | Region identifier | `local` |

---

### Edge Service
| Property | Value |
|----------|-------|
| **Path** | `apps/engine/cmd/edge` |
| **Port** | 8086 (HTTP) |
| **Container** | `linkflow-edge` |
| **Profile** | `edge` (optional) |
| **Health Endpoint** | `/health` |

**Responsibilities:**
- Low-latency execution close to data
- Offline capability
- WASM runtime
- Sync to cloud

**Environment Variables:**
| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP server port | `8080` |
| `FRONTEND_ADDR` | Frontend service address | `linkflow-frontend:9090` |
| `SYNC_INTERVAL` | Cloud sync interval | `30s` |

---

## Infrastructure

### PostgreSQL
| Property | Value |
|----------|-------|
| **Image** | `postgres:16-alpine` |
| **Port** | 5432 |
| **Container** | `linkflow-postgres` |
| **Health Check** | `pg_isready` |

**Configuration:**
| Setting | Value |
|---------|-------|
| `shared_buffers` | 256MB |
| `max_connections` | 200 |
| `work_mem` | 16MB |
| `maintenance_work_mem` | 128MB |
| `effective_cache_size` | 512MB |

### Redis
| Property | Value |
|----------|-------|
| **Image** | `redis:7-alpine` |
| **Port** | 6379 |
| **Container** | `linkflow-redis` |
| **Health Check** | `redis-cli ping` |

**Configuration:**
| Setting | Value |
|---------|-------|
| `appendonly` | yes |
| `appendfsync` | everysec |
| `maxmemory` | 256mb |
| `maxmemory-policy` | allkeys-lru |

---

## Common Environment Variables

All Go engine services share these environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://user:pass@host:5432/db` |
| `REDIS_URL` | Redis connection string | `redis://linkflow-redis:6379` |
| `LOG_LEVEL` | Logging level | `info` |
| `LOG_FORMAT` | Log format | `json` |

---

## Service Dependencies

```
┌─────────────┐     ┌─────────────┐
│   Laravel   │────▶│  Frontend   │
│     API     │     │   Gateway   │
└─────────────┘     └──────┬──────┘
       │                   │
       │           ┌───────┴───────┐
       │           ▼               ▼
       │    ┌──────────┐    ┌──────────┐
       │    │  History │    │ Matching │
       │    └──────────┘    └──────────┘
       │           │               │
       │           │        ┌──────┴──────┐
       │           │        ▼             ▼
       │           │   ┌────────┐   ┌─────────┐
       │           │   │ Worker │   │  Timer  │
       │           │   └────────┘   └─────────┘
       │           │
       ▼           ▼
┌─────────────┬─────────────┐
│  PostgreSQL │    Redis    │
└─────────────┴─────────────┘
```

## Port Summary

| Service | HTTP Port | gRPC Port | Container Name |
|---------|-----------|-----------|----------------|
| Laravel API | 8000 | - | `linkflow-api` |
| Frontend | 8080 | 9090 | `linkflow-frontend` |
| History | 8081 | 7234 | `linkflow-history` |
| Matching | 8082 | 7235 | `linkflow-matching` |
| Worker | 8083 | - | `linkflow-worker` |
| Timer | 8084 | 7238 | `linkflow-timer` |
| Visibility | 8085 | 7237 | `linkflow-visibility` |
| Edge | 8086 | - | `linkflow-edge` |
| Control Plane | 8087 | 7239 | `linkflow-control-plane` |
| PostgreSQL | 5432 | - | `linkflow-postgres` |
| Redis | 6379 | - | `linkflow-redis` |

---

## Security Configuration

### Required Secrets

| Secret | Used By | Description |
|--------|---------|-------------|
| `JWT_SECRET` | Frontend | JWT token signing (min 32 chars) |
| `LINKFLOW_SECRET` | Worker, Laravel | Callback authentication between engine and API |
| `POSTGRES_PASSWORD` | All services | Database authentication |

### Production Checklist

- [ ] `JWT_SECRET` is a cryptographically random 32+ character string
- [ ] `LINKFLOW_SECRET` is a cryptographically random 32+ character string
- [ ] `POSTGRES_PASSWORD` is changed from default
- [ ] Database SSL is enabled (`sslmode=require` or `verify-full`)
- [ ] All services are behind a reverse proxy with TLS
- [ ] Network policies restrict inter-service communication
- [ ] Secrets are managed via a secret management solution

Generate secrets using:
```bash
openssl rand -base64 32
```
