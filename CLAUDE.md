# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Shark Task Manager** is a Go-based CLI tool and HTTP API for managing project tasks, features, and epics with AI-driven development workflows. It uses SQLite for persistence and follows clean architecture principles.

### Key Technologies
- **Go**: 1.23.4+ (statically typed, compiled)
- **SQLite**: Local database with WAL mode for concurrency
- **Cobra**: CLI framework for structured command hierarchy
- **Viper**: Configuration management

### Quick Commands

See @.claude/rules/quickref.md for complete command reference.

```bash
# Build
make build              # Build all binaries
make shark             # Build only shark CLI

# Test
make test              # Run all tests
make test-coverage     # Run tests with HTML coverage report

# Common commands
./bin/shark task list           # List tasks
./bin/shark task next           # Get next available task
./bin/shark feature create E07 "Feature Title"
./bin/shark task create E07 F01 "Task Title"
```

---

## ⚠️ Critical Warnings

See @.claude/rules/database-critical.md for full details.

**NEVER delete shark-tasks.db** - it's the single source of truth for all project data. Deleting it causes data loss and sync errors.

**DO NOT**:
- Run `make clean` during development (deletes database)
- Use `rm shark*` glob patterns
- Delete database to fix sync errors
- Modify task files during sync operations

---

## Navigation Guide

This project uses modular documentation. Rules are loaded automatically based on which files you're working with:

### Always-Loaded (Base Context)
- **Quick Reference**: @.claude/rules/quickref.md - Build, test, common commands
- **Database Critical**: @.claude/rules/database-critical.md - Critical DB warnings & recovery
- **Development Workflows**: @.claude/rules/development-workflows.md - Task creation, lifecycle, patterns

### Path-Specific (Auto-Loaded Based on Files)

**Working on Go code** (`internal/**/*.go`, `cmd/**/*.go`):
- Architecture: @.claude/rules/architecture.md
- Go Patterns: @.claude/rules/go/patterns.md
- Error Handling: @.claude/rules/go/error-handling.md

**Working on Database/Repository** (`internal/db/**/*`, `internal/repository/**/*`):
- Database Schema: @.claude/rules/database/schema.md
- Architecture: @.claude/rules/architecture.md

**Working on Cloud/Config** (`internal/db/**/*`, `internal/config/**/*`):
- Cloud/Turso: @.claude/rules/database/cloud-turso.md

**Working on CLI** (`internal/cli/**/*`):
- CLI Patterns: @.claude/rules/cli/patterns.md

**Working on CLI Commands** (`internal/cli/commands/**/*`):
- CLI Commands: @.claude/rules/cli/commands.md

**Writing Tests** (`**/*_test.go`):
- Testing Architecture: @.claude/rules/testing/architecture.md
- Repository Tests: @.claude/rules/testing/repository-tests.md (if in `internal/repository/**/*_test.go`)
- CLI Tests: @.claude/rules/testing/cli-tests.md (if in `internal/cli/**/*_test.go`)

---

## Key Concepts

### Dual Key Format (Slug Architecture)

Shark supports both numeric and human-readable slugged keys:

**Epics**: `E04` or `E04-epic-name`
**Features**: `E04-F02` or `E04-F02-feature-name` or `F02` or `F02-feature-name`
**Tasks**: `T-E04-F02-001` or `T-E04-F02-001-task-name` or `E04-F02-001` (short format)

All keys are **case insensitive**: `E07`, `e07`, `E07-user-management` all work.

Slugs are auto-generated from titles and both formats work in all commands.

### Task Lifecycle

```
todo → in_progress → ready_for_review → completed
                  ↘ blocked ↗
```

Commands:
- `shark task start <task>` - todo → in_progress
- `shark task complete <task>` - in_progress → ready_for_review
- `shark task approve <task>` - ready_for_review → completed
- `shark task reopen <task>` - ready_for_review → in_progress
- `shark task block <task> --reason="..."` - any → blocked
- `shark task unblock <task>` - blocked → previous state

### Project Root Auto-Detection

Shark automatically finds the project root by walking up directories looking for:
1. `.sharkconfig.json` (primary)
2. `shark-tasks.db` (secondary)
3. `.git/` (fallback)

You can run shark commands from any subdirectory.

---

## Documentation References

- **Architecture Details**: @.claude/rules/architecture.md
- **Complete CLI Reference**: @docs/CLI_REFERENCE.md
- **Turso Cloud Setup**: @docs/TURSO_QUICKSTART.md
- **Turso Migration Guide**: @docs/TURSO_MIGRATION.md
- **Original Full Documentation**: @CLAUDE.md.backup (if needed)

---

## Development Principles

See @.claude/rules/development-workflows.md for complete workflows.

**Task Creation**:
1. Create feature: `shark feature create E07 "Feature Title"`
2. Create tasks: `shark task create E07 F01 "Task Title"`
3. Update task file with implementation details
4. Link related docs in task frontmatter

**Testing**:
- Only repository tests use real database
- All other tests use mocks
- See @.claude/rules/testing/architecture.md for details

**Go Patterns**:
- See @.claude/rules/go/patterns.md (auto-loaded for .go files)
- Error handling, transactions, validation patterns
