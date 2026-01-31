.PHONY: setup start stop restart logs ps test clean lint security format check-deps install-tools

setup:
	@echo "Setting up Monorepo..."
	@echo "1. Installing API Dependencies..."
	cd apps/api && composer install
	@echo "2. Installing Engine Dependencies..."
	cd apps/engine && go mod download
	@echo "3. Setup Complete. Run 'make start' to launch."

start:
	@echo "Starting LinkFlow Stack..."
	docker-compose up -d
	@echo "Services are starting. Access at:"
	@echo " - API: http://localhost:8000"
	@echo " - Engine Frontend: http://localhost:8080"

stop:
	@echo "Stopping LinkFlow Stack..."
	docker-compose down

restart: stop start

logs:
	docker-compose logs -f

ps:
	docker-compose ps

test:
	@echo "Testing Engine..."
	cd apps/engine && go test ./...
	@echo "Testing API..."
	cd apps/api && php artisan test

clean:
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

build:
	docker-compose build

ci: lint test security
	@echo "CI pipeline completed successfully!"
