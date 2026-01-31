.PHONY: setup start stop restart logs ps test clean

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
