.PHONY: help build run test clean install dev lint fmt vet demo test-db pm install-pm

# Default target
help:
	@echo "Shark Task Manager - Available commands:"
	@echo "  make install    - Install project dependencies"
	@echo "  make build      - Build the application"
	@echo "  make pm         - Build the PM CLI tool"
	@echo "  make install-pm - Install PM CLI to ~/go/bin"
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
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/shark-task-manager cmd/server/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/demo cmd/demo/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/test-db cmd/test-db/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/pm cmd/pm/main.go

# Build PM CLI tool
pm:
	@echo "Building PM CLI..."
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/pm cmd/pm/main.go
	@echo "PM CLI built: ./bin/pm"

# Install PM CLI to ~/go/bin
install-pm: pm
	@echo "Installing PM CLI to ~/go/bin..."
	@mkdir -p ~/go/bin
	@cp bin/pm ~/go/bin/pm
	@echo "PM CLI installed! Run 'pm --help' to get started."

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
	@echo "Running tests..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -v -coverprofile=coverage.out ./...
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
