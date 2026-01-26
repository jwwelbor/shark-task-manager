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

Shark supports two workflow profiles with different status sets and agent assignments.

**Basic Profile (Default):**
```
todo → in_progress → ready_for_review → completed
                  ↘ blocked ↗
```

**Advanced Profile (TDD Workflow):**
Comprehensive multi-stage workflow covering planning, development, review, QA, and approval phases. See [Workflow Profiles Guide](docs/guides/workflow-profiles.md) for details.

Commands:
- `shark task start <task>` - Start task (transitions based on profile)
- `shark task complete <task>` - Mark ready for next review phase
- `shark task approve <task>` - Final approval/completion
- `shark task reopen <task>` - Move back to in-progress
- `shark task block <task> --reason="..."` - Block on dependency
- `shark task unblock <task>` - Remove block

### Workflow Profiles & Agent Routing

Shark supports two workflow profiles that define task status flows and agent responsibilities.

**Basic Profile (5 statuses):**
- Simple linear workflow: todo → in_progress → ready_for_review → completed
- Suitable for solo developers or small teams
- No status flow enforcement
- Single agent responsibility

**Advanced Profile (19 statuses):**
- Comprehensive TDD workflow with multiple phases
- Designed for team development with defined roles
- Status flow enforcement
- Agent routing by status:
  - **ba** (Business Analyst): Refinement phase (ready_for_refinement, in_refinement)
  - **developer**: Development phase (ready_for_development, in_development)
  - **tech_lead**: Code review phase (ready_for_code_review, in_code_review)
  - **qa**: QA phase (ready_for_qa, in_qa)
  - **product_owner**: Approval phase (ready_for_approval, in_approval)

**Switching Profiles:**
```bash
# Apply basic workflow
shark init update --workflow=basic

# Apply advanced workflow
shark init update --workflow=advanced

# Preview changes before applying
shark init update --workflow=advanced --dry-run
```

See [Workflow Profiles Guide](docs/guides/workflow-profiles.md) for comprehensive documentation.

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
- **Workflow Profiles Guide**: @docs/guides/workflow-profiles.md
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
