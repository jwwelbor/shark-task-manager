.PHONY: help build run test clean install dev lint fmt vet demo test-db test-e2e shark install-shark

# Default target
help:
	@echo "Shark Task Manager - Available commands:"
	@echo "  make install    - Install project dependencies"
	@echo "  make build      - Build the application"
	@echo "  make shark         - Build the Shark CLI tool"
	@echo "  make install-shark - Install Shark CLI (detects existing location or ~/go/bin)"
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

# Build-time variables
BUILD_DATE := $(shell date -u '+%Y-%m-%d')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS := -X main.BuildDate=$(BUILD_DATE) -X main.GitCommit=$(GIT_COMMIT)

# Build the application
build:
	@echo "Building application..."
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/shark-task-manager ./cmd/server/
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/demo cmd/demo/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -o bin/test-db cmd/test-db/main.go
	@export PATH=$$PATH:$$HOME/go/bin && go build -ldflags "$(LDFLAGS)" -o bin/shark cmd/shark/main.go

# Build Shark CLI tool
shark:
	@echo "Building Shark CLI..."
	@export PATH=$$PATH:$$HOME/go/bin && go build -ldflags "$(LDFLAGS)" -o bin/shark cmd/shark/main.go
	@echo "Shark CLI built: ./bin/shark"

# Install Shark CLI (finds and updates all installed copies, or defaults to ~/go/bin)
install-shark: shark
	@FOUND=0; \
	for LOC in $$(which -a shark 2>/dev/null | sort -u); do \
		INSTALL_DIR=$$(dirname "$$LOC"); \
		FOUND=1; \
		echo "Updating $$LOC..."; \
		if [ -w "$$INSTALL_DIR" ]; then \
			cp bin/shark "$$LOC"; \
		else \
			echo "Error: Insufficient permissions to write to $$INSTALL_DIR. Please run 'sudo make install-shark' to update '$$LOC'."; \
			exit 1; \
		fi; \
		echo "  Updated."; \
	done; \
	if [ "$$FOUND" = "0" ]; then \
		echo "No existing shark found, installing to ~/go/bin..."; \
		mkdir -p ~/go/bin; \
		cp bin/shark ~/go/bin/shark; \
		echo "Shark CLI installed to ~/go/bin/shark"; \
	fi

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
# Packages run in parallel by default (Go's default -p=GOMAXPROCS).
# Tests within each package that call t.Parallel() also run concurrently.
# Repository tests that need DB isolation should use test.NewIsolatedTestDB(t)
# instead of the shared test.GetTestDB() singleton.
test:
	@echo "Cleaning test database..."
	@rm -f internal/repository/test-shark-tasks.db*
	@rm -f /tmp/shark-test-tasks.db*
	@echo "Running tests..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Cleaning test database..."
	@rm -f internal/repository/test-shark-tasks.db*
	@rm -f /tmp/shark-test-tasks.db*
	@echo "Running tests with coverage..."
	@export PATH=$$PATH:$$HOME/go/bin && go test -v -coverprofile=coverage.out ./...
	@export PATH=$$PATH:$$HOME/go/bin && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Lint code
lint:
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.9.0; \
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
