.PHONY: all help build build-frontend build-backend dev dev-backend dev-frontend test test-backend test-frontend lint lint-backend lint-frontend docker-build clean

APP_NAME ?= quizme
BIN_DIR ?= bin
DOCKER_IMAGE ?= $(APP_NAME):latest

all: build

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-16s %s\n", $$1, $$2}'

build-frontend: ## Build React frontend production assets
	@echo "==> Building frontend assets..."
	cd frontend && npm install && npm run build

build-backend: ## Build Go monolith binary (embedding frontend/dist)
	@echo "==> Building Go binary..."
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(APP_NAME) cmd/server/main.go

build: build-frontend build-backend ## Build both frontend and Go monolith binary

dev-backend: ## Run Go backend in development mode
	go run cmd/server/main.go

dev-frontend: ## Run Vite frontend dev server with HMR
	cd frontend && npm run dev

dev: ## Run backend (:8080) and frontend (:5173) dev servers concurrently
	@echo "==> Starting backend and frontend dev servers (press Ctrl+C to stop both)..."
	@trap 'kill 0' SIGINT SIGTERM EXIT; \
		go run cmd/server/main.go & \
		(cd frontend && npm run dev) & \
		wait

test-backend: ## Run backend Go tests
	@echo "==> Running backend tests..."
	go test -v ./...

test-frontend: ## Check frontend lint and build
	@echo "==> Checking frontend..."
	cd frontend && npm run lint && npm run build

test: test-backend test-frontend ## Run backend tests and frontend checks

lint-backend: ## Check Go formatting and vet
	@echo "==> Checking Go formatting and vet..."
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files:" && gofmt -l . && exit 1)
	go vet ./...

lint-frontend: ## Run oxlint on frontend
	@echo "==> Linting frontend..."
	cd frontend && npm run lint

lint: lint-backend lint-frontend ## Run all linters (Go and frontend)

docker-build: ## Build monolith Docker image
	@echo "==> Building Docker image $(DOCKER_IMAGE)..."
	docker build --network=host --load -t $(DOCKER_IMAGE) .

clean: ## Remove build artifacts
	@echo "==> Cleaning artifacts..."
	rm -rf $(BIN_DIR)
