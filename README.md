# Shark Task Manager

A task management system built with Go and SQLite, featuring both an HTTP API and a powerful CLI tool for AI-driven development workflows.

## Key Features

- **Hierarchical Task Organization**: Organize work into Epics → Features → Tasks with auto-generated keys
- **AI-Driven Workflows**: Built-in support for multiple agent types with dependency-aware task selection
- **Flexible Organization**: Organize with custom folder base paths (`--path` flag) or specific file paths (`--filename` flag)
- **Bidirectional Sync**: Synchronize markdown files with SQLite database with conflict resolution
- **Progress Tracking**: Automatic progress calculation from task completion to features and epics
- **Audit Trail**: Complete history of all status changes with timestamps and agent tracking
- **Dependency Management**: Express task dependencies and get warnings about blocking work
- **JSON Output**: Machine-readable JSON output for all commands (with `--json` flag)

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
- `make shark` - Build the Shark CLI tool
- `make install-shark` - Install Shark CLI to ~/go/bin ⭐

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

## Installation

Shark CLI is available for macOS, Linux, and Windows through multiple installation methods.

### Quick Install

**macOS** (Homebrew):
```bash
brew install jwwelbor/shark/shark
```

**Windows** (Scoop):
```powershell
scoop bucket add shark https://github.com/jwwelbor/scoop-shark
scoop install shark
```

**Linux/macOS** (Manual):
```bash
curl -fsSL https://github.com/jwwelbor/shark-task-manager/releases/latest/download/shark_$(uname -s)_$(uname -m).tar.gz -o shark.tar.gz
tar -xzf shark.tar.gz
sudo mv shark /usr/local/bin/
```

**Verify Installation**:
```bash
shark --version
```

### Detailed Installation Instructions

#### macOS

**Option 1: Homebrew (Recommended)**

1. Add the Shark tap:
   ```bash
   brew tap jwwelbor/shark
   ```

2. Install Shark:
   ```bash
   brew install shark
   ```

3. Verify installation:
   ```bash
   shark --version
   ```

**Option 2: Manual Installation**

1. Download the latest release for your architecture:
   - Intel Macs: `shark_*_darwin_amd64.tar.gz`
   - Apple Silicon (M1/M2/M3): `shark_*_darwin_arm64.tar.gz`

2. Extract and install:
   ```bash
   tar -xzf shark_*.tar.gz
   sudo mv shark /usr/local/bin/
   ```

3. Verify installation:
   ```bash
   shark --version
   ```

#### Linux

**Option 1: Manual Installation (All Distributions)**

1. Download the latest release for your architecture:
   - AMD64/x86_64: `shark_*_linux_amd64.tar.gz`
   - ARM64/aarch64: `shark_*_linux_arm64.tar.gz`

2. Download and verify checksums (recommended):
   ```bash
   # Download binary and checksums
   wget https://github.com/jwwelbor/shark-task-manager/releases/latest/download/shark_*_linux_amd64.tar.gz
   wget https://github.com/jwwelbor/shark-task-manager/releases/latest/download/checksums.txt

   # Verify integrity
   sha256sum -c checksums.txt --ignore-missing
   ```

3. Extract and install:
   ```bash
   tar -xzf shark_*.tar.gz
   sudo mv shark /usr/local/bin/
   ```

4. Verify installation:
   ```bash
   shark --version
   ```

**Option 2: From Source**

1. Install Go 1.23 or later
2. Clone and build:
   ```bash
   git clone https://github.com/jwwelbor/shark-task-manager.git
   cd shark-task-manager
   make install-shark
   ```

#### Windows

**Option 1: Scoop (Recommended)**

1. Add the Shark bucket:
   ```powershell
   scoop bucket add shark https://github.com/jwwelbor/scoop-shark
   ```

2. Install Shark:
   ```powershell
   scoop install shark
   ```

3. Verify installation:
   ```powershell
   shark --version
   ```

**Option 2: Manual Installation**

1. Download `shark_*_windows_amd64.zip` from the [latest release](https://github.com/jwwelbor/shark-task-manager/releases/latest)

2. Download `checksums.txt` and verify (PowerShell):
   ```powershell
   # Calculate hash
   $actual = (Get-FileHash shark_*_windows_amd64.zip -Algorithm SHA256).Hash.ToLower()

   # Extract expected hash
   $expected = (Get-Content checksums.txt | Select-String "windows_amd64").ToString().Split()[0]

   # Verify
   if ($actual -eq $expected) {
       Write-Host "✅ Checksum verified successfully"
   } else {
       Write-Host "❌ Checksum verification FAILED"
       exit 1
   }
   ```

3. Extract the ZIP file

4. Add the directory to your PATH or move `shark.exe` to a directory already in PATH

5. Verify installation:
   ```powershell
   shark --version
   ```

### Security: Verifying Downloads

All releases include SHA256 checksums in `checksums.txt`. Always verify downloads before installation:

**Linux/macOS**:
```bash
sha256sum -c checksums.txt --ignore-missing
```

**Windows PowerShell**:
```powershell
$actual = (Get-FileHash shark.zip -Algorithm SHA256).Hash.ToLower()
$expected = (Get-Content checksums.txt | Select-String "windows").ToString().Split()[0]
if ($actual -eq $expected) { Write-Host "✅ Verified" }
```

For more details, see [SECURITY.md](SECURITY.md).

### Upgrading

**Homebrew**:
```bash
brew upgrade shark
```

**Scoop**:
```powershell
scoop update shark
```

**Manual Installation**: Download and install the latest version following the manual installation steps above.

---

## Shark CLI - AI Agent Task Management

The `shark` CLI is designed for AI agents to manage epics, features, and tasks programmatically. It provides atomic operations, dependency management, and progress tracking.

### Quick Start for AI Agents

```bash
# 1. Initialize infrastructure (first time only)
shark init --non-interactive

# 2. Query available work
shark epic list --json
shark task next --agent=backend --json

# 3. Start working on a task
shark task start T-E04-F06-001

# 4. Mark task ready for review
shark task complete T-E04-F06-001

# 5. Approve and complete
shark task approve T-E04-F06-001
```

### Core Workflows for AI Agents

#### 1. Project Initialization

Set up Shark CLI infrastructure (run once per project):

```bash
# Non-interactive mode for automation
shark init --non-interactive
```

This creates:
- SQLite database (`shark-tasks.db`) with schema
- Folder structure (`docs/plan/`)
- Configuration file (`.sharkconfig.json`)
- Templates (`shark-templates/`) - epic.md, feature.md, task.md

#### 2. Discovering Available Work

**List all epics with progress:**
```bash
shark epic list --json
shark epic list --status=active --json
```

**Get epic details with all features:**
```bash
shark epic get E04 --json
```

**List features in an epic:**
```bash
shark feature list --epic=E04 --json
shark feature list --epic=E04 --status=active --json
```

**Get feature details with all tasks:**
```bash
shark feature get E04-F06 --json
```

**Find the next available task:**
```bash
# Get next task for any agent
shark task next --json

# Get next task for specific agent type
shark task next --agent=backend --json
shark task next --agent=frontend --json

# Get next task in specific epic
shark task next --epic=E04 --json
```

The `task next` command:
- Returns tasks in `todo` status
- Checks all dependencies are completed
- Sorts by priority (1 = highest)
- Returns task key, title, file path, and dependencies

#### 3. Querying Tasks

**List all tasks:**
```bash
shark task list --json
```

**Filter by status:**
```bash
shark task list --status=todo --json
shark task list --status=in_progress --json
shark task list --status=ready_for_review --json
shark task list --status=completed --json
shark task list --status=blocked --json
```

**Filter by epic or agent:**
```bash
shark task list --epic=E04 --json
shark task list --agent=backend --json
shark task list --epic=E04 --agent=backend --status=todo --json
```

**Get task details:**
```bash
shark task get T-E04-F06-001 --json
```

Returns task metadata, dependencies, and dependency status.

#### 4. Creating Tasks

```bash
shark task create \
  --epic=E04 \
  --feature=F06 \
  --title="Implement task validation" \
  --agent=backend \
  --priority=3 \
  --description="Add validation logic for task creation" \
  --depends-on="T-E04-F06-001,T-E04-F06-002"
```

Parameters:
- `--epic` (required): Epic key (e.g., `E04`)
- `--feature` (required): Feature key (e.g., `F06` or `E04-F06`)
- `--title` (required): Task title
- `--agent` (required): Agent type (`frontend`, `backend`, `api`, `testing`, `devops`, `general`)
- `--priority`: Priority 1-10 (default: 5, where 1 = highest)
- `--description`: Detailed task description
- `--depends-on`: Comma-separated list of dependency task keys

The CLI automatically:
- Generates unique task key (e.g., `T-E04-F06-003`)
- Creates markdown file at `docs/plan/epic/feature/task-key.md`
- Sets initial status to `todo`
- Validates epic and feature exist

#### 5. Task Lifecycle Management

**Standard workflow:**

```bash
# 1. Start task (todo → in_progress)
shark task start T-E04-F06-001 --json

# 2. Mark ready for review (in_progress → ready_for_review)
shark task complete T-E04-F06-001 --json

# 3. Approve and mark completed (ready_for_review → completed)
shark task approve T-E04-F06-001 --json
```

**State Transitions:**

| Command | From Status | To Status | Notes |
|---------|-------------|-----------|-------|
| `start` | `todo` | `in_progress` | Begin work on task |
| `complete` | `in_progress` | `ready_for_review` | Implementation done, needs review |
| `approve` | `ready_for_review` | `completed` | Review passed, task complete |
| `reopen` | `ready_for_review` | `in_progress` | Review failed, rework needed |
| `block` | `todo`, `in_progress` | `blocked` | Cannot proceed, needs resolution |
| `unblock` | `blocked` | `todo` | Blocker resolved, ready to start |

**Handling blocked tasks:**

```bash
# Block a task with reason
shark task block T-E04-F06-001 --reason="Waiting for API design approval" --json

# List all blocked tasks
shark task list --blocked --json

# Unblock when resolved
shark task unblock T-E04-F06-001 --json
```

**Handling review feedback:**

```bash
# Reopen for rework
shark task reopen T-E04-F06-001 --notes="Need to add error handling" --json

# Fix issues and mark ready again
shark task complete T-E04-F06-001 --json
```

#### 6. Synchronizing with File System

After Git operations or manual file edits:

```bash
# Preview sync changes (dry-run)
shark sync --dry-run --json

# Sync task files with database
shark sync --json

# Sync with conflict resolution strategy
shark sync --strategy=file-wins --json
shark sync --strategy=database-wins --json
shark sync --strategy=newer-wins --json

# Create missing epics/features from files
shark sync --create-missing --json

# Delete orphaned database records (files deleted)
shark sync --cleanup --json

# Sync specific folder only
shark sync --folder=docs/plan/E04-task-mgmt-cli-core --json
```

**Sync patterns:**
```bash
# Sync task files only (default)
shark sync --pattern=task --json

# Sync PRP (Product Requirement Prompt) files only
shark sync --pattern=prp --json

# Sync both task and PRP files
shark sync --pattern=task --pattern=prp --json
```

**Important:** Status is managed exclusively in the database and is NOT synced from files. This ensures atomic status transitions and audit trails.

#### 7. Progress Tracking

**Epic progress:**
```bash
shark epic list --json
shark epic get E04 --json
```

Returns calculated progress percentage based on completed tasks across all features.

**Feature progress:**
```bash
shark feature list --epic=E04 --json
shark feature get E04-F06 --json
```

Returns:
- Progress percentage
- Task count
- Status breakdown (todo/in_progress/completed/blocked)

### AI Agent Best Practices

1. **Always use `--json` flag** for machine-readable output
2. **Check dependencies** before starting tasks via `shark task next --json`
3. **Use atomic operations** - each command is a single transaction
4. **Handle blocked tasks** - use `block` command with reasons
5. **Sync after Git operations** - run `shark sync` after pulls/checkouts
6. **Track work with agent identifier** - use `--agent` flag for audit trail
7. **Use priority effectively** - 1=highest, 10=lowest for task ordering
8. **Check exit codes** - Non-zero indicates errors (1=not found, 2=db error, 3=invalid state)

### Example: Complete AI Agent Workflow

```bash
# 1. Initialize project (first time)
shark init --non-interactive

# 2. Discover available work
NEXT_TASK=$(shark task next --agent=backend --json | jq -r '.key')

# 3. Start the task
shark task start "$NEXT_TASK" --agent="ai-agent-001"

# 4. Do the implementation work...
# ... code implementation happens here ...

# 5. Mark ready for review
shark task complete "$NEXT_TASK" --agent="ai-agent-001" --notes="Implementation complete, all tests passing"

# 6. After review approval
shark task approve "$NEXT_TASK" --agent="reviewer-001" --notes="LGTM, approved"

# 7. Sync changes to filesystem
shark sync
```

### JSON Output Format

All commands support `--json` for structured output:

```json
{
  "key": "T-E04-F06-001",
  "title": "Implement key generation",
  "status": "todo",
  "priority": 3,
  "agent_type": "backend",
  "depends_on": ["T-E04-F05-001"],
  "dependency_status": {
    "T-E04-F05-001": "completed"
  },
  "file_path": "docs/plan/E04-task-mgmt-cli-core/E04-F06-task-creation/T-E04-F06-001.md"
}
```

### Documentation

#### User Guides
- [Initialization Guide](docs/user-guide/initialization.md) - Set up Shark CLI
- [Synchronization Guide](docs/user-guide/synchronization.md) - Sync tasks with Git workflow
- [Troubleshooting](docs/troubleshooting.md) - Common issues and solutions

#### Reference
- [Complete Documentation Index](docs/DOCUMENTATION_INDEX.md) - Find all documentation
- [CLI Documentation](docs/CLI_REFERENCE.md) - Complete command reference
- [Custom Folder Paths Migration Guide](docs/MIGRATION_CUSTOM_PATHS.md) - Organize with custom folder base paths
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
