# Task Index: CLI Infrastructure & Framework (E04-F02)

## Overview

This index tracks all implementation tasks for the CLI Infrastructure & Framework feature (E04-F02).

**Feature Location**: `/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/`
**Total Tasks**: 6
**Estimated Total Time**: 48 hours (6 business days)

## Task List

| Task Key | Title | Status | Assigned Agent | Dependencies | Est. Time | Current Location |
|----------|-------|--------|----------------|--------------|-----------|------------------|
| T-E04-F02-001 | CLI Core Framework & Command Structure | todo | general-purpose | T-E04-F01-006 | 8 hours | docs/tasks/todo/ |
| T-E04-F02-002 | Output Formatting System - JSON & Rich Tables | todo | general-purpose | T-E04-F02-001 | 10 hours | docs/tasks/todo/ |
| T-E04-F02-003 | Configuration Management & User Defaults | todo | general-purpose | T-E04-F02-001 | 6 hours | docs/tasks/todo/ |
| T-E04-F02-004 | Error Handling & Exit Code System | todo | general-purpose | T-E04-F02-002 | 8 hours | docs/tasks/todo/ |
| T-E04-F02-005 | Database Context Integration & Session Management | todo | general-purpose | T-E04-F02-001, T-E04-F02-004, T-E04-F01-002 | 6 hours | docs/tasks/todo/ |
| T-E04-F02-006 | Integration Testing, Documentation & Package Finalization | todo | general-purpose | T-E04-F02-001, T-E04-F02-002, T-E04-F02-003, T-E04-F02-004, T-E04-F02-005 | 10 hours | docs/tasks/todo/ |

## Task Workflow

Tasks move through these folders based on their status:

```
docs/tasks/todo/           → New tasks created here
    ↓
docs/tasks/active/         → Moved when work begins
    ↓
docs/tasks/ready-for-review/ → Moved when implementation complete
    ↓
docs/tasks/completed/      → Moved after successful review
    ↓
docs/tasks/archived/       → Moved when no longer needed
```

## Status Definitions

- **todo**: Not started, waiting for dependencies or assignment
- **active**: Currently being worked on by an agent
- **blocked**: Waiting on external dependency or decision
- **ready-for-review**: Implementation complete, awaiting code review
- **completed**: Reviewed, validated, and merged
- **archived**: No longer active or superseded

## Execution Order

### Phase 1: CLI Core Framework (8 hours)

**Task**: T-E04-F02-001
**Dependencies**: T-E04-F01-006 (Database layer complete)

Create the foundational Click-based CLI framework:
- Hierarchical command structure (`pm <resource> <action>`)
- Global flags (--json, --no-color, --verbose, --config)
- Entry point configuration and package installation
- Argument parsing with type validation
- Help text generation system
- Command registration infrastructure

**Success Gates**:
- `shark` command installed and accessible
- Command hierarchy works (`pm --help`, `pm task --help`)
- Global flags inherited by all subcommands
- CLI startup time <300ms

### Phase 2: Output Formatting System (10 hours)

**Task**: T-E04-F02-002
**Dependencies**: T-E04-F02-001

Implement dual output modes for humans and agents:
- JSON serialization for all model types
- Rich table formatter with responsive columns
- Output context manager (auto-select based on --json)
- Color utilities with terminal detection
- Text truncation and terminal width handling
- Empty result handling for both modes

**Success Gates**:
- JSON output valid and parseable by jq
- Tables fit 80-column terminals
- Colorization respects --no-color flag
- Performance targets met (<200ms for 100 rows)

### Phase 3: Configuration Management (6 hours)

**Task**: T-E04-F02-003
**Dependencies**: T-E04-F02-001

Implement .pmconfig.json system:
- Configuration file format and schema
- Loader with JSON parsing and validation
- Merge logic (CLI flags override config)
- Config commands (validate, get, set, init)
- Default value management

**Success Gates**:
- .pmconfig.json loads correctly
- CLI flags override config values
- Missing config doesn't cause errors
- `pm config validate` reports clear errors

### Phase 4: Error Handling & Exit Codes (8 hours)

**Task**: T-E04-F02-004
**Dependencies**: T-E04-F02-002

Implement comprehensive error handling:
- Exit code constants and hierarchy
- Exception translation (database errors → exit codes)
- Global exception handler decorator
- User-friendly error messages
- Logging configuration (stderr only)
- Verbose mode stack traces

**Success Gates**:
- All exceptions caught and translated
- Exit codes correct (0=success, 1=user error, 2=system error, 3=validation)
- Error messages actionable and suggest next steps
- Stack traces only in --verbose mode

### Phase 5: Database Context Integration (6 hours)

**Task**: T-E04-F02-005
**Dependencies**: T-E04-F02-001, T-E04-F02-004, T-E04-F01-002

Integrate database layer with CLI:
- Session factory integration with Click context
- Session lifecycle management (commit/rollback/close)
- Database initialization command (`pm init`)
- Session access via `ctx.obj['db']`
- Connection error handling

**Success Gates**:
- Session created once per command
- Automatic commit on success, rollback on error
- Session always closed properly
- Commands can access database via context

### Phase 6: Integration Testing & Documentation (10 hours)

**Task**: T-E04-F02-006
**Dependencies**: All previous tasks (T-E04-F02-001 through T-E04-F02-005)

Validate and document the complete framework:
- Integration tests for all components
- End-to-end test scenarios
- Performance benchmarks
- Framework architecture documentation
- Command registration guide
- Output formatting API docs
- Configuration guide
- Error handling guide
- Database integration guide
- Integration checklist for downstream features

**Success Gates**:
- All integration tests pass
- Performance targets met
- Test coverage >85%
- All documentation complete with examples
- Package exports cleanly
- Zero mypy/ruff errors

## Design Documentation

All tasks reference this design document in the feature directory:

- [PRD](../docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/prd.md) - Complete product requirements, user stories, and acceptance criteria

## Dependencies & Critical Path

```
T-E04-F01-006 (Database Package Export - E04-F01) [EXTERNAL]
    ↓
T-E04-F02-001 (CLI Core Framework) [8h]
    ↓
    ├─→ T-E04-F02-002 (Output Formatting) [10h]
    │       ↓
    │   T-E04-F02-004 (Error Handling) [8h]
    │
    └─→ T-E04-F02-003 (Configuration) [6h]
            │
            └─→ (merges with T-E04-F02-004)
                    ↓
T-E04-F01-002 (Session Management - E04-F01) [EXTERNAL]
    ↓
T-E04-F02-005 (Database Context) [6h]
    ↓
T-E04-F02-006 (Integration Testing & Docs) [10h]
    ↓
READY FOR E04-F03 (Task Lifecycle Operations)
READY FOR E04-F04 (Epic & Feature Queries)
```

**Critical Path**:
1. T-E04-F02-001 (8h)
2. T-E04-F02-002 (10h) ← Longest task in Phase 2
3. T-E04-F02-004 (8h)
4. T-E04-F02-005 (6h)
5. T-E04-F02-006 (10h)

**Total Critical Path Time**: 42 hours (longest sequential path)

**Parallelization Opportunities**:
- T-E04-F02-002 (Output) and T-E04-F02-003 (Config) can run in parallel after T-E04-F02-001
- Saves ~6 hours if two agents work simultaneously

## Downstream Feature Dependencies

Once all tasks are completed, these features can begin:

### E04-F03: Task Lifecycle Operations
**What they need**:
- Click command registration pattern from T-E04-F02-001
- Output formatters (JSON and table) from T-E04-F02-002
- Error handling and exit codes from T-E04-F02-004
- Database session access via context from T-E04-F02-005
- Integration guide from T-E04-F02-006

**Integration pattern**: Register task commands in `pm/cli/groups/task.py`, use output formatters for display, access repositories via `ctx.obj['db']`.

### E04-F04: Epic & Feature Queries
**What they need**:
- Same as E04-F03 (command registration, output, errors, database context)
- Configuration defaults for epic selection from T-E04-F02-003

**Integration pattern**: Register epic/feature commands in respective group files, use table formatter for list views, JSON for structured output.

### E04-F05: Folder Management
**What they need**:
- Error handling for file operations from T-E04-F02-004
- Database transactions for atomic file + DB updates from T-E04-F02-005

### E04-F06: Task Creation
**What they need**:
- Command registration pattern
- Configuration defaults for epic and agent
- Validation error handling

### E04-F07: Initialization & Sync
**What they need**:
- Database initialization command pattern from T-E04-F02-005
- Bulk output formatting from T-E04-F02-002

## Feature Metrics

**Code Volume Estimate**:
- `pm/cli/main.py`: ~200 lines
- `pm/cli/groups/*.py`: ~600 lines (placeholders and config commands)
- `pm/cli/output/*.py`: ~680 lines (JSON, table, colors, utils)
- `pm/cli/config/*.py`: ~380 lines (loader, validator, merger, schema)
- `pm/cli/errors/*.py`: ~360 lines (exit codes, handlers, formatters, logger)
- `pm/cli/database/*.py`: ~230 lines (context, lifecycle)
- Tests: ~1450 lines
- Documentation: ~1000 lines
- **Total**: ~4900 lines of production code + tests + docs

**Test Coverage Targets**:
- `pm/cli/main.py`: 85%
- `pm/cli/output/*.py`: 90%
- `pm/cli/config/*.py`: 90%
- `pm/cli/errors/*.py`: 95%
- `pm/cli/database/*.py`: 95%
- **Overall**: >85% code coverage

**Performance Targets** (from PRD):
- CLI startup time: <300ms (from invocation to first code execution)
- Help text rendering: <50ms
- Table formatting (100 rows): <200ms
- JSON serialization (1000 tasks): <500ms

## Notes

- **CLI-Only Feature**: No frontend or API components. Purely backend CLI framework.
- **Foundation for All Commands**: E04-F03 through E04-F07 all depend on this infrastructure being complete.
- **Click & Rich**: Heavy reliance on Click framework for command routing and Rich library for terminal formatting. Both are mature, well-documented libraries.
- **Type Safety**: Full Python 3.10+ type hints required. Mypy strict mode must pass.
- **Agent Compatibility**: JSON output and exit codes critical for AI agent usage. Prioritize machine-parseable output over fancy terminal features.
- **Performance Critical**: CLI must feel instant (<300ms startup). Lazy loading and minimal imports at top level essential.
- **Documentation Required**: Comprehensive developer guides mandatory for handoff to developers implementing E04-F03, E04-F04, etc.

---

**Task Index Created**: 2025-12-14
**Feature Status**: Ready for implementation (PRD complete, E04-F01 dependency available)
**Next Action**: Begin Phase 1 (T-E04-F02-001) after E04-F01-006 completes, or assign to implementation agent
