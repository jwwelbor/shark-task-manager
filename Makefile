.PHONY: help build run test clean install dev lint fmt vet demo test-db test-e2e shark install-shark

# Default target
help:
	@echo "Shark Task Manager - Available commands:"
	@echo "  make install    - Install project dependencies"
	@echo "  make build      - Build the application"
	@echo "  make shark         - Build the Shark CLI tool"
	@echo "  make install-shark - Install Shark CLI to ~/go/bin"
	@echo "  make run        - Run the application"
	@echo "  make dev        - Run in development mode with auto-reload"
	@echo "  make demo       - Run interactive demo (creates sample data)"
	@echo "  make test-db    - Run database integration tests"
	@echo "  make test       - Run tests"
	@echo "  make lint       - Run linter"
	@echo "  make fmt        - Format code"
	@echo "  make vet        - Run go vet"
	@echo "  make clean      - Clean build artifacts"

# Install dependencies
install:
	@echo "Installing dependencies..."
	@export PATH=$$PATH:$$HOME/go/bin && go mod download
	@export PATH=$$PATH:$$HOME/go/bin && go mod tidy

# Build the application
build:
	@echo "Building application..."
	@export PATH=$$PATH:$$HOME/go/bin && go build -tags "fts5" -o bin/shark-task-manager cmd/server/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -tags "fts5" -o bin/demo cmd/demo/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -tags "fts5" -o bin/test-db cmd/test-db/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -tags "fts5" -o bin/shark cmd/shark/main.go

# Build Shark CLI tool
shark:
	@echo "Building Shark CLI..."
	@export PATH=$$PATH:$$HOME/go/bin && go build -tags "fts5" -o bin/shark cmd/shark/main.go
	@echo "Shark CLI built: ./bin/shark"

# Install Shark CLI to ~/go/bin
install-shark: shark
	@echo "Installing Shark CLI to ~/go/bin..."
	@mkdir -p ~/go/bin
	@cp bin/shark ~/go/bin/shark
	@echo "Shark CLI installed! Run 'shark --help' to get started."

# Run the application
run: build
	@echo "Starting Shark Task Manager..."
	@./bin/shark-task-manager

# Development mode (requires air for hot reload)
dev:
	@if ! command -v air > /dev/null; then \
		echo "Installing air for hot reload..."; \
		export PATH=$$PATH:$$HOME/go/bin && go install github.com/air-verse/air@latest; \
	fi
	@export PATH=$$PATH:$$HOME/go/bin && air

# Run tests
test:
	@echo "Cleaning test database..."
	@rm -f internal/repository/test-shark-tasks.db*
	@rm -f /tmp/shark-test-tasks.db*
	@echo "Running tests..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -tags "fts5" -v -p=1 -parallel=1 ./...

# Run tests with coverage
test-coverage:
	@echo "Cleaning test database..."
	@rm -f internal/repository/test-shark-tasks.db*
	@rm -f /tmp/shark-test-tasks.db*
	@echo "Running tests with coverage..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -tags "fts5" -v -p=1 -parallel=1 -coverprofile=coverage.out ./...
	@export PATH=$$PATH:$$HOME/go/bin && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint:
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.55.2; \
	fi
	@export PATH=$$PATH:$$HOME/go/bin && golangci-lint run

# Format code
format:
	@echo "Formatting code..."
	@export PATH=$$PATH:$$HOME/go/bin && go fmt ./...

fmt:
	@echo "Formatting code..."
	@export PATH=$$PATH:$$HOME/go/bin && go fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	@export PATH=$$PATH:$$HOME/go/bin && go vet ./...

# Run demo (creates sample data)
demo: build
	@echo "Running database demo..."
	@./bin/demo

# Run database integration tests
test-db: build
	@echo "Running database integration tests..."
	@./bin/test-db

# Run E2E shell tests
test-e2e: shark
	@echo "Running E2E shell tests..."
	@bash test/e2e/test_enhanced_status.sh

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f *.db *.db-shm *.db-wal
	@rm -f internal/repository/*.db internal/repository/*.db-shm internal/repository/*.db-wal
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# Initialize database
db-init:
	@echo "Database will be initialized on first run"

# Database migrations (placeholder for future use)
db-migrate:
	@echo "Database migrations will be implemented here"
