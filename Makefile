.PHONY: help setup dev-backend dev-frontend test test-backend test-frontend build fmt vet check clean \
	docker-build docker-test docker-up docker-down docker-logs docker-clean

help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[1m%-16s\033[0m %s\n", $$1, $$2}'

setup: ## Install frontend dependencies and verify the Go module
	cd frontend && npm install
	cd backend && go mod tidy

dev-backend: ## Run the Go API on :8080
	cd backend && go run ./cmd/server

dev-frontend: ## Run the Vite dev server on :5173
	cd frontend && npm run dev

test: test-backend test-frontend ## Run every test and check

test-backend: ## Run Go tests with the race detector
	cd backend && go test -race ./...

test-frontend: ## Type-check the frontend
	cd frontend && npm run typecheck

build: ## Build the API binary and the frontend bundle
	cd backend && go build -o bin/server ./cmd/server
	cd frontend && npm run build

fmt: ## Format Go sources
	cd backend && gofmt -w .

vet: ## Run go vet
	cd backend && go vet ./...

check: fmt vet test ## Format, vet and test

clean: ## Remove build output
	rm -rf backend/bin frontend/dist

## --- Docker ------------------------------------------------------------------
## These need no local Go or Node: the toolchains live in the build stages.

docker-build: ## Build both images
	docker compose build

docker-test: ## Run the Go suite and the frontend type-check inside Docker
	docker build --target test ./backend
	docker build --target build ./frontend

docker-up: ## Start the stack, then open http://localhost:8080
	docker compose up --build -d
	@echo "finance_app is on http://localhost:$${WEB_PORT:-8080}"

docker-down: ## Stop the stack
	docker compose down

docker-logs: ## Follow logs from both containers
	docker compose logs -f

docker-clean: ## Stop the stack and remove its images and volumes
	docker compose down --rmi local --volumes --remove-orphans
