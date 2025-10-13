.PHONY: dev dev-build docker-up docker-down docker-logs docker-clean clean help prod-build package release version

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Run development server with Docker (live reload)
	@echo "Starting development environment with Docker Compose..."
	@echo "  - DEV_MODE=true (Secure cookies disabled for localhost)"
	@echo "  - WS_ALLOWED_ORIGINS=localhost:*,127.0.0.1:*,host.docker.internal:*"
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

lint: ## Run golangci-lint (install from https://golangci-lint.run/usage/install/)
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@$(TEMPL_BIN) fmt .

prod-build: ## Build optimized production binary for Linux AMD64
	@echo "Building production binary..."
	@./build/build.sh

package: ## Package application for deployment (requires VERSION=x.y.z)
	@echo "Packaging application..."
	@./build/package.sh

release: prod-build package ## Build and package for release (requires VERSION=x.y.z)
	@echo "Release complete!"
	@echo "Package: dist/planning-poker-v$(VERSION).tar.gz"

version: ## Show version information
	@if [ -f dist/planning-poker ]; then \
		./dist/planning-poker --version 2>/dev/null || echo "Binary exists but version info not available"; \
	else \
		echo "No production binary found. Run 'make prod-build' first."; \
	fi

clean-dist: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	@rm -rf dist/
	@echo "Build artifacts cleaned"
