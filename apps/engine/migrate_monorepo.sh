#!/bin/bash
set -e

# Target Monorepo Directory
TARGET_DIR="../lnkflow"
CURRENT_DIR=$(pwd)

echo "Starting migration to $TARGET_DIR..."

# 1. Ensure Target Directory Structure
echo "Creating directory structure..."
mkdir -p "$TARGET_DIR/apps"
mkdir -p "$TARGET_DIR/infra"

# 2. Move API
echo "Moving linkflow-api..."
if [ -d "../linkflow-api" ]; then
    if [ -d "$TARGET_DIR/apps/api" ]; then
        echo "Target api directory already exists. Skipping move."
    else
        mv ../linkflow-api "$TARGET_DIR/apps/api"
        echo "linkflow-api moved."
    fi
else
    echo "Warning: ../linkflow-api not found."
fi

# 3. Copy Infrastructure
echo "Copying infrastructure..."
if [ -d "../infrastructure" ]; then
    # Copy contents to infra
    cp -R ../infrastructure/* "$TARGET_DIR/infra/" 2>/dev/null || echo "No files in infrastructure to copy."
    echo "Infrastructure copied."
else
    echo "Warning: ../infrastructure not found."
fi

# 4. Create docker-compose.yml
echo "Creating docker-compose.yml..."
cat <<EOF > "$TARGET_DIR/docker-compose.yml"
version: '3.8'

services:
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

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: linkflow
      POSTGRES_PASSWORD: secret_password
      POSTGRES_DB: linkflow_db
    volumes:
      - postgres_data:/var/lib/postgresql/data
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
EOF

# 5. Create Makefile
echo "Creating Makefile..."
cat <<EOF > "$TARGET_DIR/Makefile"
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
EOF

# 6. Move Engine (Self)
echo "Moving go-engine..."
# We assume we are in go-engine.
if [ -d "$TARGET_DIR/apps/engine" ]; then
    echo "Target engine directory already exists. Skipping move."
else
    # To execute this clean, we move the folder.
    mv "$CURRENT_DIR" "$TARGET_DIR/apps/engine"
    echo "go-engine moved to apps/engine."
fi

echo "Migration Successfully Completed!"
echo "Please open '$TARGET_DIR' (resolved as full path) in your editor."
