---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: general-purpose
dependencies: [T024-update-architecture-documentation.md]
estimated_time: 1 hour
---

# Task: Update Code Examples in README

## Goal

Update the main README.md and any other user-facing documentation to reflect the new architecture improvements, particularly focusing on configuration options and development setup.

## Success Criteria

- [ ] README configuration section updated
- [ ] Environment variables documented
- [ ] Config file usage explained
- [ ] Development setup instructions current
- [ ] Code examples use new patterns (if shown)
- [ ] No outdated information remains
- [ ] Quick start guide updated
- [ ] Contributing guide updated (if exists)

## Implementation Guidance

### Overview

Update the main README and user-facing documentation to reflect the configuration system and any other user-visible changes. This helps users understand how to configure and deploy the application.

### Key Requirements

- Add configuration section to README
- Document environment variables
- Show config file examples
- Update development setup for new patterns
- Remove or update any outdated examples

Reference: [PRD - Documentation](../01-feature-prd.md#documentation)

### Files to Create/Modify

**User Documentation**:
- `README.md` - Add/update configuration section
- `CONTRIBUTING.md` (if exists) - Update development setup
- `docs/DEPLOYMENT.md` (if exists) - Add configuration guide
- Example config files already created in T021

### Documentation Updates

**1. Configuration Section** (add to README):
```markdown
## Configuration

Shark Task Manager supports flexible configuration through environment variables, config files, or defaults.

### Configuration Options

| Option | Environment Variable | Default | Description |
|--------|---------------------|---------|-------------|
| Database Path | `SHARK_DATABASE_PATH` | `shark-tasks.db` | SQLite database file path |
| Max Open Connections | `SHARK_DATABASE_MAX_OPEN_CONNS` | `25` | Database connection pool size |
| Max Idle Connections | `SHARK_DATABASE_MAX_IDLE_CONNS` | `5` | Idle connections in pool |
| Database Timeout | `SHARK_DATABASE_TIMEOUT` | `30s` | Query timeout duration |
| Server Port | `SHARK_SERVER_PORT` | `8080` | HTTP server port |
| Server Read Timeout | `SHARK_SERVER_READ_TIMEOUT` | `5s` | HTTP read timeout |
| Server Write Timeout | `SHARK_SERVER_WRITE_TIMEOUT` | `10s` | HTTP write timeout |
| CLI Default Format | `SHARK_CLI_DEFAULT_FORMAT` | `table` | Output format (json, table, text) |
| CLI Color Output | `SHARK_CLI_COLOR_OUTPUT` | `true` | Enable colored output |

### Using Environment Variables

```bash
# Set custom database path
export SHARK_DATABASE_PATH=/data/tasks.db

# Start server on different port
SHARK_SERVER_PORT=9090 shark-server

# Disable colored output
SHARK_CLI_COLOR_OUTPUT=false shark task list
```

### Using Config File

Create a `.shark.yaml` file in your project directory or home directory:

```yaml
database:
  path: ./shark-tasks.db
  max_open_conns: 25
  timeout: 30s

server:
  port: "8080"

cli:
  default_format: table
  color_output: true
```

### Configuration Precedence

1. Environment variables (highest)
2. Config file (`.shark.yaml`)
3. Default values (lowest)
```

**2. Development Setup** (update existing section):
```markdown
## Development Setup

### Prerequisites

- Go 1.21 or later
- SQLite3

### Building

```bash
# Build CLI tool
make build

# Or manually
go build -o shark ./cmd/pm

# Build server
go build -o shark-server ./cmd/server
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

### Project Structure

```
shark-task-manager/
├── cmd/                    # Application entry points
│   ├── pm/                # CLI tool
│   └── server/            # HTTP server
├── internal/
│   ├── domain/            # Domain layer (interfaces, errors)
│   ├── repository/        # Data access
│   │   ├── sqlite/       # SQLite implementation
│   │   └── mock/         # Mock implementations for testing
│   ├── models/           # Data models
│   ├── cli/              # CLI commands
│   └── config/           # Configuration management
└── docs/                 # Documentation
```
```

**3. Docker/Deployment Section** (if applicable):
```markdown
## Deployment

### Docker

```bash
# Build Docker image
docker build -t shark-task-manager .

# Run with environment variables
docker run -e SHARK_DATABASE_PATH=/data/tasks.db \
           -e SHARK_SERVER_PORT=8080 \
           -v /host/data:/data \
           shark-task-manager
```

### Using Config File in Docker

Mount config file as volume:

```bash
docker run -v ./config/.shark.yaml:/app/.shark.yaml \
           shark-task-manager
```
```

### Integration Points

- **Configuration System**: T019-T023 changes
- **User Documentation**: README is primary user-facing doc
- **Example Configs**: Created in T021
- **Development Setup**: Updated for new architecture

## Validation Gates

**Content Review**:
- Configuration section is complete and accurate
- All environment variables documented
- Config file examples are correct
- Development instructions are current

**Accuracy Verification**:
- Test all commands shown in README
- Verify environment variables work as documented
- Test config file example
- Verify development setup instructions

**Completeness Check**:
- No outdated information remains
- All new features documented
- Links to detailed docs provided
- Examples are helpful and clear

**User Experience**:
- README is easy to follow
- Configuration is clearly explained
- Quick start guide works
- Examples are practical

## Context & Resources

- **Current README**: `README.md`
- **Config Examples**: `.shark.yaml.example`, `.shark.json.example`
- **PRD**: [Documentation Section](../01-feature-prd.md#documentation)
- **Previous Task**: [T024 - Architecture Docs](./T024-update-architecture-documentation.md)

## Notes for Agent

- Focus on user-facing documentation (not internal docs)
- Keep README concise - link to detailed docs for more info
- Configuration is the main user-visible change from this feature
- Test all examples before documenting them
- Use tables for configuration options (easier to scan)
- Show practical examples (real use cases)
- Update Quick Start if configuration affects it
- This is the final task - completes all documentation
- After this, feature is fully implemented and documented
- Users should be able to configure and deploy the system from README
