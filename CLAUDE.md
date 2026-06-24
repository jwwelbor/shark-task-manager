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

# Core Commands (auto-detect entity type from key)
./bin/shark get E07                                    # Get epic details
./bin/shark get E07-F01                                # Get feature details
./bin/shark get E07-F01-001                            # Get task details
./bin/shark get E07-F01-001 --field status             # Extract single field
./bin/shark list                                       # List epics
./bin/shark list E07                                   # List features in epic
./bin/shark list E07 F01                               # List tasks in feature

# Status & Analytics
./bin/shark status                                     # Project dashboard
./bin/shark status E07-F01                             # Feature status
./bin/shark status history E07-F01-001                 # Status change history
./bin/shark status advance E07-F01-001                 # Advance to next status
./bin/shark progress E07                               # Epic progress

# Entity Management
./bin/shark task create E07 F01 "Task Title"           # Create task
./bin/shark feature create E07 "Feature Title"         # Create feature
./bin/shark task next-status E07-F01-001               # Advance task workflow
```

---

## ⚠️ Mandatory Quality Gate

**After ANY Go code changes, ALWAYS run before declaring work complete:**
```bash
make fmt && make lint && make test
```
Fix all failures. No exceptions. See @.claude/rules/development-workflows.md for details.

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
- Input Sanitization: @.claude/rules/go/input-sanitization.md

**Working on Database/Repository** (`internal/db/**/*`, `internal/repository/**/*`):
- Database Schema: @.claude/rules/database/schema.md
- Architecture: @.claude/rules/architecture.md

**Working on Cloud/Config** (`internal/db/**/*`, `internal/config/**/*`):
- Cloud/Turso: @.claude/rules/database/cloud-turso.md

**Working on CLI** (`internal/cli/**/*`):
- CLI Patterns: @.claude/rules/cli/patterns.md

**Working on CLI Commands** (`internal/cli/commands/**/*`):
- CLI Commands: @.claude/rules/cli/commands.md

**Working on Service Layer** (`internal/services/**/*`):
- Service Design: @.claude/rules/services/service-design.md
- CLI Integration: @.claude/rules/services/cli-integration.md
- HTTP Integration: @.claude/rules/services/http-integration.md
- Service Testing: @.claude/rules/services/testing.md
- Migration Guide: @docs/guides/service-layer-migration.md

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

Shark loads its workflow definitions (statuses, transitions, agent routing)
from per-entity YAML files referenced by `workflow_config` in
`.sharkconfig.json`. `workflow_config` may point at:

- a **directory** of per-entity YAML files (the default, `shark-data/workflow/`
  with `task.yaml`, `feature.yaml`, `epic.yaml`, …), or
- a **master index file** that maps each entity to its workflow file (see
  [Route-Based Workflow Guide](docs/guides/route-based-workflow.md) §3).

`shark admin init` materializes the `shark-data/` tree (workflows, prompts,
skills, agents) and leaves your `shark-data/overrides/` subtree untouched.

> A bare Shark 1.x JSON workflow file (e.g. `.sharkworkflow.json`) is **no
> longer a valid `workflow_config` target** — the loader rejects it with a
> migration hint. Run `shark init` to materialize the `shark-data/` tree.

Task lifecycle at a glance (default route-based task workflow):
```
draft → development → completed
   ↘ blocked / on_hold (parking)   ↘ cancelled (terminal)
```

Commands:
- `shark task next-status <task>` / `shark status advance <task>` — Advance to next workflow status
- `shark status advance <task> --outcome <pass|fail|blocked>` — Release a semantic outcome (route-based)
- `shark status set <task> <status>` / `shark task set-status <task> <status>` — Set status directly
- `shark task approve <task>` — Final approval/completion
- `shark task reopen <task>` — Move back to in-progress

### Switching / customizing workflows

Edit the per-entity YAML under `shark-data/workflow/` directly, or point
`workflow_config` at your own directory or master index file. Files under
`shark-data/overrides/` layer on top of the bundled defaults and are never
overwritten by `shark admin init`.

> The basic/advanced "profile" subsystem (`shark init update --workflow=...`,
> `shark init merge ...`) was removed. See
> [Workflow Configuration Guide](docs/guides/workflow-profiles.md) for the
> migration table.

### Route-Based Workflows (Shark 2.x — E35)

A consolidated route-based workflow schema is supported alongside the legacy
two-map shape. It merges `status_flow` + `status_metadata` into one per-step
block (`steps:`), replaces the transition graph with a per-step `outcomes:` map
(skills release a semantic `pass`/`fail`/`blocked` outcome and the engine
routes), collapses `ready_for_X`/`in_X` into a phase + a claim/session lease,
and supports a master index file mapping each entity to its workflow.

Both shapes coexist: the loader derives the legacy maps from `steps:`, so every
existing reader keeps working and the default shipped workflows remain on the
legacy shape until explicitly switched. New CLI surface: `shark status advance
--outcome <name>`, `shark claim/release/heartbeat/claims`, and `shark admin
migrate statuses` (gated). See
[Route-Based Workflow Guide](docs/guides/route-based-workflow.md).

### Project Root Auto-Detection

Shark automatically finds the project root by walking up directories looking for:
1. `.sharkconfig.json` (primary)
2. `shark-tasks.db` (secondary)
3. `.git/` (fallback)

You can run shark commands from any subdirectory.

---

## Documentation References

- **Architecture Details**: @.claude/rules/architecture.md
- **CLI Reference (Unified)**: @docs/cli-reference/README.md
- **Workflow Profiles Guide**: @docs/guides/workflow-profiles.md
- **Route-Based Workflow (Shark 2.x)**: @docs/guides/route-based-workflow.md
- **Turso Cloud Setup**: @docs/TURSO_QUICKSTART.md
- **Turso Migration Guide**: @docs/TURSO_MIGRATION.md

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
- Viewer HTML is covered by Go asset/API/service tests; do not add Playwright,
  npm, or `node_modules` for viewer testing
- See @.claude/rules/testing/architecture.md for details

**Go Patterns**:
- See @.claude/rules/go/patterns.md (auto-loaded for .go files)
- Error handling, transactions, validation patterns
