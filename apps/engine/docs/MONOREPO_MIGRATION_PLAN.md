# LinkFlow Monorepo Migration Plan

This guide outlines the steps to consolidate `go-engine`, `linkflow-api`, and `infrastructure` into a single monorepo.

## 1. Directory Structure

The goal is to move from:
```text
~/Herd/
  ├── go-engine/
  ├── linkflow-api/
  └── infrastructure/
```

To:
```text
~/Herd/linkflow/
  ├── apps/
  │   ├── api/       (was linkflow-api)
  │   └── engine/    (was go-engine)
  ├── infra/         (was infrastructure)
  └── docker-compose.yml
```

## 2. Migration Steps

### Step 1: Create Root Directory
```bash
mkdir -p ~/Herd/linkflow/apps
mkdir -p ~/Herd/linkflow/infra
cd ~/Herd/linkflow
git init
```

### Step 2: Move Projects
*Warning: Close your IDEs/Editors before doing this to avoid file locks.*

```bash
# Move API
mv ~/Herd/linkflow-api ~/Herd/linkflow/apps/api

# Move Engine
mv ~/Herd/go-engine ~/Herd/linkflow/apps/engine

# Move Infrastructure content
# We copy contents, not the folder itself, to fit the new naming
cp -r ~/Herd/infrastructure/* ~/Herd/linkflow/infra/
```

### Step 3: Create Master Docker Compose
Create `~/Herd/linkflow/docker-compose.yml` to orchestrate both services.

```yaml
version: '3.8'

services:
  # ----------------------------------------------------------------
  # Laravel API
  # ----------------------------------------------------------------
  api:
    build: 
      context: ./apps/api
      dockerfile: Dockerfile
    ports:
      - "8000:8000"
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    networks:
      - linkflow-net

  # ----------------------------------------------------------------
  # Go Engine (Control Plane)
  # ----------------------------------------------------------------
  engine:
    build: 
      context: ./apps/engine
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    environment:
      - POSTGRES_HOST=postgres
      - API_ENDPOINT=http://api:8000
    depends_on:
      - postgres
    networks:
      - linkflow-net

  # ----------------------------------------------------------------
  # Shared Infrastructure
  # ----------------------------------------------------------------
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: linkflow
      POSTGRES_PASSWORD: secret_password
      POSTGRES_DB: linkflow_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
      # Mount initialization scripts from both apps if needed
      - ./infra/init:/docker-entrypoint-initdb.d
    networks:
      - linkflow-net

  redis:
    image: redis:7-alpine
    networks:
      - linkflow-net

networks:
  linkflow-net:
    driver: bridge

volumes:
  postgres_data:
```

### Step 4: Create Global Makefile
Create `~/Herd/linkflow/Makefile` for easy management.

```makefile
.PHONY: setup start stop test

setup:
	@echo "Setting up Monorepo..."
	cd apps/api && composer install
	cd apps/engine && go mod download

start:
	docker-compose up -d

stop:
	docker-compose down

test:
	@echo "Testing Engine..."
	cd apps/engine && go test ./...
	@echo "Testing API..."
	cd apps/api && php artisan test
```

## 3. Post-Migration Checklist

- [ ] **Update Git Ignores:** Ensure the root `.gitignore` excludes `vendor/`, `node_modules/`, and build/binaries for all languages.
- [ ] **Fix Imports:** Dockerfiles in `apps/api` and `apps/engine` used to look for context in `.`. They are now in subfolders, so `docker-compose.yml` paths need to check `context: ./apps/api`.
- [ ] **Shared Secrets:** Create a single `.env` at the root and pass it to all containers so you manage secrets in one place.
