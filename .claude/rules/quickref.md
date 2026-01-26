# Quick Reference

Common commands for building, testing, and using Shark Task Manager.

## Build Commands

```bash
make build              # Build all binaries (shark-task-manager, shark CLI, demo, test-db)
make shark             # Build only the Shark CLI tool
make install-shark     # Install Shark CLI to ~/go/bin
```

## Testing

```bash
make test              # Run all tests with verbose output
make test-coverage     # Run tests with HTML coverage report (coverage.html)
make test-db           # Run specific database integration tests
```

## Code Quality

```bash
make fmt               # Format code with gofmt
make vet               # Run go vet for static analysis
make lint              # Run golangci-lint (auto-installs if needed)
```

## Common Shark Commands

### Task Management
```bash
# List tasks
./bin/shark task list
./bin/shark task list E04        # Filter by epic
./bin/shark task list E04 F01    # Filter by epic and feature

# Get next available task
./bin/shark task next
./bin/shark task next --agent=backend           # Standard agent type
./bin/shark task next --agent=architect         # Custom agent type

# Get task details
./bin/shark task get E07-F20-001

# Task lifecycle
./bin/shark task start E07-F20-001
./bin/shark task complete E07-F20-001 --notes="Implementation done"
./bin/shark task approve E07-F20-001
./bin/shark task block E07-F20-001 --reason="Waiting on API"
./bin/shark task unblock E07-F20-001
```

### Feature Management
```bash
# Create feature (positional syntax recommended)
./bin/shark feature create E07 "Feature Title"

# Create feature with custom file path
./bin/shark feature create E07 "Feature Title" --file="docs/custom/path.md"

# List features
./bin/shark feature list
./bin/shark feature list E07     # Filter by epic

# Get feature details
./bin/shark feature get E07-F01
./bin/shark feature get F01      # Short format also works
```

### Epic Management
```bash
# Create epic
./bin/shark epic create --title="Epic Title"

# Create epic with custom file path
./bin/shark epic create --title="Epic Title" --file="docs/custom/epic.md"

# List epics
./bin/shark epic list

# Get epic details
./bin/shark epic get E07
```

### Task Creation
```bash
# Positional syntax (recommended)
./bin/shark task create E07 F01 "Task Title"                    # 3-arg format
./bin/shark task create E07-F01 "Task Title"                    # 2-arg format

# With additional options (standard agent type)
./bin/shark task create E07 F01 "Task Title" --agent=backend --priority=5

# With custom agent type
./bin/shark task create E07 F01 "Task Title" --agent=architect --priority=3
./bin/shark task create E07 F01 "Task Title" --agent=business-analyst --priority=4

# With custom file path
./bin/shark task create E07 F01 "Task Title" --file="docs/custom/task.md"

# Legacy flag syntax (still supported)
./bin/shark task create --epic=E07 --feature=F01 --title="Task Title"
```

## Synchronization

```bash
# Preview sync changes
./bin/shark sync --dry-run

# Sync filesystem to database (file wins)
./bin/shark sync --strategy=file-wins

# Sync database to filesystem (database wins)
./bin/shark sync --strategy=database-wins
```

## Configuration & Initialization

```bash
# View configuration
./bin/shark config get <key>

# Set configuration
./bin/shark config set <key> <value>

# Update Shark configuration with workflow profiles
./bin/shark init update                           # Add missing fields
./bin/shark init update --workflow=basic          # Apply basic workflow
./bin/shark init update --workflow=advanced       # Apply advanced workflow
./bin/shark init update --workflow=advanced --dry-run  # Preview changes
./bin/shark init update --workflow=basic --force  # Force overwrite
```

## Cloud Database (Turso)

```bash
# Initialize cloud database
./bin/shark cloud init --url="libsql://..." --auth-token="..." --non-interactive

# Check cloud status
./bin/shark cloud status
./bin/shark cloud status --json
```

## Key Format Notes

- **Case insensitive**: `E07`, `e07`, `E07-user-management` all work
- **Short format**: `E07-F20-001` (recommended) or `T-E07-F20-001` (traditional)
- **Slugged keys**: `E07-F20-001-task-name` also works
- **Feature keys**: `E07-F01`, `F01`, `E07-F01-feature-name`, `F01-feature-name`
