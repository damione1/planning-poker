PHONY: dev dev-build docker-up docker-down docker-logs docker-clean clean help

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Run development server with Docker (live reload)
	@echo "Starting development environment with Docker Compose..."
	@docker compose up

dev-build: ## Rebuild development Docker images
	@echo "Rebuilding development Docker images..."
	@docker compose build

docker-up: ## Start all services in background
	@echo "Starting services in detached mode..."
	@docker compose up -d

docker-down: ## Stop all services
	@echo "Stopping all services..."
	@docker compose down

docker-logs: ## Show logs from all services
	@docker compose logs -f

docker-clean: ## Stop services and remove volumes
	@echo "Stopping services and removing volumes..."
	@docker compose down -v

clean: ## Clean Docker artifacts and build cache
	@echo "Cleaning Docker artifacts..."
	@docker compose down -v --remove-orphans
	@docker system prune -f
	@echo "Clean complete"

tidy: ## Tidy go modules
	@echo "Tidying go modules..."
	@go mod tidy

test: ## Run tests
	@echo "Running tests..."
	@go test ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@$(TEMPL_BIN) fmt .
