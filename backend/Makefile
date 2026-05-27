.PHONY: help run build docker-up docker-down docker-logs test tidy lint clean

# Variables
APP_NAME   := bongcloud-chess
BUILD_DIR  := ./bin
VERSION    ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS    := -s -w -X main.version=$(VERSION)

#  Help
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

#  Local development
run: ## Run the server locally (requires .env file)
	@go run ./cmd/server/...

build: ## Build the binary to ./bin/
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server
	@echo "✓ Built $(BUILD_DIR)/$(APP_NAME)"

tidy: ## Tidy and verify Go modules
	go mod tidy
	go mod verify

lint: ## Run golangci-lint (must be installed separately)
	golangci-lint run ./...

test: ## Run all tests
	go test -race -count=1 ./...

test-cover: ## Run tests with coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

clean: ## Remove build artefacts
	rm -rf $(BUILD_DIR) coverage.out coverage.html

# Docker
docker-up: ## Start all services (detached)
	docker compose up -d --build

docker-down: ## Stop all services and remove volumes
	docker compose down -v

docker-logs: ## Tail application logs
	docker compose logs -f app

docker-ps: ## Show running containers
	docker compose ps