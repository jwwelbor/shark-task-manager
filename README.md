# Shark Task Manager

A task management system built with Go and SQLite, featuring both an HTTP API and a powerful CLI tool for AI-driven development workflows.

## Prerequisites

- Go 1.23.4 or later
- SQLite3
- Make

## Project Structure

```
.
├── cmd/
│   └── server/          # Application entry point
│       └── main.go
├── internal/
│   ├── db/              # Database initialization and setup
│   ├── handlers/        # HTTP request handlers
│   └── models/          # Data models
├── migrations/          # Database migrations
├── Makefile            # Development commands
└── README.md
```

## Getting Started

### Install Dependencies

```bash
make install
```

### Build the Application

```bash
make build
```

### Run the Application

```bash
make run
```

The server will start on `http://localhost:8080`

### Development Mode (Hot Reload)

For development with automatic reloading on file changes:

```bash
make dev
```

This will install `air` if not already installed and run the application with hot reload enabled.

## Available Make Commands

### CLI Tools
- `make pm` - Build the PM CLI tool
- `make install-pm` - Install PM CLI to ~/go/bin ⭐

### Application
- `make help` - Show all available commands
- `make install` - Install project dependencies
- `make build` - Build the application binary
- `make run` - Build and run the application
- `make dev` - Run in development mode with hot reload

### Testing
- `make demo` - Run interactive demo with sample data ⭐
- `make test-db` - Run database integration tests
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report

### Code Quality
- `make fmt` - Format code using gofmt
- `make vet` - Run go vet for code analysis
- `make lint` - Run golangci-lint (installs if needed)
- `make clean` - Remove build artifacts and databases

## Testing

### Interactive Demo

See the database in action with sample data:

```bash
make demo
```

This creates an epic, feature, and tasks, then demonstrates:
- CRUD operations
- Progress calculations
- Query filtering
- Status updates with history tracking

### Integration Tests

Run comprehensive database tests:

```bash
make test-db
```

Tests include:
- Epic/Feature/Task CRUD operations
- Atomic status updates
- Progress calculations
- Cascade deletes
- All constraints and validations

See [TESTING.md](docs/TESTING.md) for detailed testing guide.

## PM CLI - Task Management

The `pm` (Project Manager) CLI provides a powerful interface for managing tasks, epics, and features:

```bash
# Install the CLI
make install-pm

# Quick start
pm --help
pm task --help
pm epic list
pm feature list --epic=E04
```

See [CLI Documentation](docs/CLI.md) for complete command reference.

### Documentation

- [Complete Documentation Index](docs/DOCUMENTATION_INDEX.md) - Find all documentation
- [Epic & Feature Query Guide](docs/EPIC_FEATURE_QUERIES.md) - Query epics and features with progress
- [Quick Reference](docs/EPIC_FEATURE_QUICK_REFERENCE.md) - Fast command lookup
- [Examples](docs/EPIC_FEATURE_EXAMPLES.md) - Real-world usage scenarios

## API Endpoints

- `GET /` - API welcome message
- `GET /health` - Health check endpoint (includes database status)

## Development

### Database

The application uses SQLite for data persistence with a complete schema:

**Tables:**
- `epics` - Top-level project organization units
- `features` - Mid-level components within epics
- `tasks` - Atomic work units within features
- `task_history` - Audit trail of task status changes

**Features:**
- Foreign key constraints with CASCADE DELETE
- Auto-update triggers for timestamps
- 10+ indexes for query performance
- WAL mode for better concurrency
- Comprehensive validation at application layer

The database file (`shark-tasks.db`) is automatically created on first run.

See [internal/db/README.md](internal/db/README.md) for detailed schema documentation.

### Code Formatting

Before committing, format your code:

```bash
make fmt
make vet
```

### Testing

Run the test suite:

```bash
make test
```

Generate coverage report:

```bash
make test-coverage
```

## Environment Setup

Go is installed in `~/go/bin`. Make sure your PATH includes this directory:

```bash
export PATH=$PATH:$HOME/go/bin
```

This is automatically added to your `~/.bashrc` and `~/.profile`.

## License

See LICENSE file for details.
