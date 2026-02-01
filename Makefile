.PHONY: help setup start stop restart logs ps test clean lint security format check-deps install-tools build ci validate-deps validate-docker

validate-docker:
	@if ! command -v docker >/dev/null 2>&1; then \
		echo "❌ Docker is not installed"; \
		exit 1; \
	fi
	@if ! docker info >/dev/null 2>&1; then \
		echo "❌ Docker daemon is not running"; \
		exit 1; \
	fi
	@echo "✅ Docker is ready"

validate-deps: validate-docker
	@echo "Checking system dependencies..."
	@missing=0; \
	for dep in git curl docker-compose make go php composer; do \
		if ! command -v $$dep >/dev/null 2>&1; then \
			echo "❌ Missing dependency: $$dep"; \
			missing=1; \
		else \
			echo "✅ $$dep"; \
		fi; \
	done; \
	if command -v wget >/dev/null 2>&1; then \
		echo "✅ wget"; \
	else \
		echo "⚠️  wget not found (optional)"; \
	fi; \
	if [ "$$missing" = "1" ]; then \
		echo "Install missing dependencies and try again"; \
		exit 1; \
	fi

setup: validate-deps
	@echo "Setting up Monorepo..."
	@echo "1. Installing API Dependencies..."
	cd apps/api && composer install
	@echo "2. Installing Engine Dependencies..."
	cd apps/engine && go mod download
	@echo "3. Setup Complete. Run 'make start' to launch."

start: validate-docker
	@echo "Starting LinkFlow Stack..."
	docker-compose up -d || (echo "❌ Failed to start services" && exit 1)
	@echo "Services are starting. Access at:"
	@echo " - API: http://localhost:8000"
	@echo " - Engine Frontend: http://localhost:8080"

stop: validate-docker
	@echo "Stopping LinkFlow Stack..."
	docker-compose down

restart: stop start

logs: validate-docker
	docker-compose logs -f

ps: validate-docker
	docker-compose ps

test: validate-docker
	@echo "Testing Engine..."
	cd apps/engine && go test ./...
	@echo "Testing API..."
	cd apps/api && php artisan test

clean: validate-docker
	docker-compose down -v
	@echo "Data volumes removed."

# Quality Assurance Tools
lint:
	@echo "Running linters..."
	cd apps/api && ./vendor/bin/pint --test

format:
	@echo "Formatting code..."
	cd apps/engine && gofmt -w .
	cd apps/api && ./vendor/bin/pint

security:
	@echo "Running security scans..."
	cd apps/api && composer audit --no-dev || true
	echo "✅ Security scans completed!"

check-deps:
	@echo "Checking for outdated dependencies..."
	cd apps/api && composer outdated
	cd apps/engine && go list -u -m all

install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/sonatype-nexus-community/nancy@latest
	brew install pre-commit semgrep
	pre-commit install
	npm install -g @commitlint/cli @commitlint/config-conventional

# Development helpers
dev-api:
	cd apps/api && composer dev

dev-engine:
	cd apps/engine && air

build: validate-docker
	docker-compose build

ci: validate-docker lint test security
	@echo "CI pipeline completed successfully!"

help:
	@echo "LinkFlow Monorepo Makefile"
	@echo "=========================="
	@echo "setup     - Install dependencies and prepare environment"
	@echo "start     - Start all services"
	@echo "stop      - Stop all services"
	@echo "restart   - Restart all services"
	@echo "logs      - Stream service logs"
	@echo "ps        - Show service status"
	@echo "test      - Run all tests"
	@echo "clean     - Remove containers and volumes"
	@echo "lint      - Run code linters"
	@echo "format    - Format code"
	@echo "security  - Run security scans"
	@echo "check-deps - Check for outdated dependencies"
	@echo "build     - Build Docker images"
	@echo "ci        - Run CI pipeline (lint + test + security)"
	@echo "help      - Show this help message"
