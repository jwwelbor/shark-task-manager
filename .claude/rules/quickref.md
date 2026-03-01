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

## Core Commands (Entity Auto-Detection)

Core commands auto-detect entity type from key format:

```bash
# Get entity details
shark get E07                              # Get epic
shark get E07-F01                          # Get feature
shark get E07-F01-001                      # Get task
shark get E07-F01-001 --field status       # Extract single field

# List entities
shark list                                 # List all epics
shark list E07                             # List features in epic E07
shark list E07 F01                         # List tasks in feature E07-F01

# Create entities
shark create epic "Epic Title"             # Create epic
shark create feature E07 "Feature Title"   # Create feature in epic
shark create task E07 F01 "Task Title"     # Create task in feature

# Delete entities
shark delete E07-F01-001                   # Delete task
shark delete E07-F01                       # Delete feature

# View entity markdown
shark view E07-F01-001                     # View task file
```

## Status & Analytics

```bash
# Status dashboard
shark status                               # Project dashboard
shark status E07                           # Epic status with feature rollups
shark status E07-F01                       # Feature status with task breakdown

# Status management
shark status set E07-F01-001 in_development   # Set status directly
shark status advance E07-F01-001              # Advance to next status
shark status options E07-F01-001              # Show valid next statuses
shark status history E07-F01-001              # Status change history

# Progress & analytics
shark progress E07                         # Epic progress breakdown
shark analytics                            # Project-wide analytics
shark analytics E07                        # Epic analytics
```

## Entity Management

### Task Commands (18 subcommands)

```bash
# CRUD
shark task create E07 F01 "Task Title" --order=1 --agent=backend
shark task get E07-F01-001 --json
shark task list --status=todo --agent=backend
shark task update E07-F01-001 --title="New Title"
shark task delete E07-F01-001

# Lifecycle
shark task approve E07-F01-001
shark task reopen E07-F01-001
shark task next-status E07-F01-001         # Advance to next workflow status
shark task set-status E07-F01-001 blocked  # Set status directly

# Dependencies
shark task deps E07-F01-001                # Show dependency tree
shark task link E07-F01-001 E07-F01-002 --type=depends_on
shark task unlink E07-F01-001 E07-F01-002

# Context & Notes
shark task context set E07-F01-001 --field current_step --value "Implementing API"
shark task note add E07-F01-001 --content="Progress update" --type=progress
shark task notes E07-F01-001
shark task resume E07-F01-001              # Resume with full context
```

### Feature Commands (13 subcommands)

```bash
shark feature create E07 "Feature Title"
shark feature get E07-F01 --json
shark feature list E07
shark feature complete E07-F01
shark feature next-status E07-F01
shark feature context set E07-F01 --field complexity --value "standard"
shark feature note add E07-F01 --content="Design decision" --type=decision
```

### Epic Commands (14 subcommands)

```bash
shark epic create "Epic Title"
shark epic get E07 --json
shark epic list
shark epic complete E07
shark epic next-status E07
shark epic status E07                      # Epic-level status with rollups
shark epic context set E07 --field phase --value "development"
shark epic note add E07 --content="Kickoff notes" --type=progress
```

### Idea Commands (6 subcommands)

```bash
shark idea create "Idea Title" --description="..."
shark idea list
shark idea get 1
shark idea update 1 --status=approved
shark idea delete 1
shark idea promote 1 --epic=E07            # Promote to task/feature
```

## Discovery Commands

```bash
shark search "authentication"              # Search across entities
shark notes E07-F01-001                    # View entity notes
shark related-docs list --feature=E07-F01  # List related documents
shark related-docs add --feature=E07-F01 --path="docs/design.md"
```

## Configuration & Setup

```bash
# Initialize
shark init --non-interactive
shark init update --workflow=advanced      # Apply advanced workflow

# Configuration
shark config show                          # Show full config
shark config validate                      # Validate config file
shark config get-status-action ready_for_development  # Debug workflow

# Cloud database
shark cloud init --url="libsql://..." --auth-token="..." --non-interactive
shark cloud status
```

## Global Flags

```bash
--json              # Machine-readable JSON output
--field <name>      # Extract single field (implies --json)
--no-color          # Disable colored output
--verbose / -v      # Debug logging
--db <path>         # Override database path
--config <path>     # Override config path
```

## Key Format Notes

- **Case insensitive**: `E07`, `e07`, `E07-user-management` all work
- **Short format**: `E07-F20-001` (recommended) or `T-E07-F20-001` (traditional)
- **Slugged keys**: `E07-F20-001-task-name` also works
- **Feature keys**: `E07-F01`, `F01`, `E07-F01-feature-name`
- **Auto-detection**: Key format determines entity type (E## = epic, E##-F## = feature, E##-F##-### = task)
