.PHONY: dev build run clean install-tools templ-generate help

# Variables
BINARY_NAME=main
BINARY_PATH=./tmp/$(BINARY_NAME)
TEMPL_BIN=~/go/bin/templ
AIR_BIN=~/go/bin/air

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

dev: ## Run development server with live reload (Air)
	@echo "Starting development server with live reload..."
	@$(AIR_BIN)

build: templ-generate ## Build production binary
	@echo "Building production binary..."
	@go build -o $(BINARY_PATH) .
	@echo "Build complete: $(BINARY_PATH)"

run: build ## Build and run server without live reload
	@echo "Starting server..."
	@$(BINARY_PATH) serve --http=127.0.0.1:8090

templ-generate: ## Generate Go code from templ templates
	@echo "Generating templ templates..."
	@$(TEMPL_BIN) generate

clean: ## Clean build artifacts and temporary files
	@echo "Cleaning build artifacts..."
	@rm -rf tmp/
	@rm -f web/templates/*_templ.go
	@go clean
	@echo "Clean complete"

install-tools: ## Install development tools (templ, air)
	@echo "Installing templ..."
	@go install github.com/a-h/templ/cmd/templ@latest
	@echo "Installing air..."
	@go install github.com/air-verse/air@latest
	@echo "Tools installed successfully"

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
