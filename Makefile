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
