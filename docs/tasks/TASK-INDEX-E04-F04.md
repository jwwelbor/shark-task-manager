# Task Index: Epic & Feature Queries (E04-F04)

## Overview

This index tracks all implementation tasks for the Epic & Feature Queries feature (E04-F04).

**Feature Location**: `/home/jwwelbor/.claude/docs/plan/E04-task-mgmt-cli-core/E04-F04-epic-feature-queries/`
**Total Tasks**: 6
**Estimated Total Time**: 32 hours (4 business days)

## Task List

| Task Key | Title | Status | Assigned Agent | Dependencies | Est. Time | Current Location |
|----------|-------|--------|----------------|--------------|-----------|------------------|
| T-E04-F04-001 | Progress Calculation Service | todo | general-purpose | E04-F01, E04-F02 | 6 hours | docs/tasks/todo/ |
| T-E04-F04-002 | Epic Query Commands | todo | general-purpose | T-E04-F04-001 | 8 hours | docs/tasks/todo/ |
| T-E04-F04-003 | Feature Query Commands | todo | general-purpose | T-E04-F04-001 | 8 hours | docs/tasks/todo/ |
| T-E04-F04-004 | JSON Output & Filtering | todo | general-purpose | T-E04-F04-002, T-E04-F04-003 | 4 hours | docs/tasks/todo/ |
| T-E04-F04-005 | Integration Tests & Performance Validation | todo | general-purpose | T-E04-F04-004 | 4 hours | docs/tasks/todo/ |
| T-E04-F04-006 | Documentation & Usage Examples | todo | general-purpose | T-E04-F04-005 | 2 hours | docs/tasks/todo/ |

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

### Phase 1: Progress Calculation Service (6 hours)

**Task**: T-E04-F04-001
**Dependencies**: E04-F01 (Database), E04-F02 (CLI Framework)

Create the core progress calculation engine:
- Feature progress calculation (completed tasks / total tasks × 100)
- Epic progress calculation (weighted average of feature progress)
- Efficient SQL queries using JOINs/aggregations to avoid N+1 queries
- Handle edge cases (zero tasks, zero features, division by zero)
- Service layer functions callable by CLI commands

**Success Gates**:
- Feature progress calculation accurate for all scenarios
- Epic weighted average calculation correct
- No N+1 query problems (use EXPLAIN QUERY PLAN to verify)
- Division by zero handled gracefully (returns 0.0)
- Unit tests cover all edge cases

### Phase 2: Epic Query Commands (8 hours)

**Task**: T-E04-F04-002
**Dependencies**: T-E04-F04-001

Implement `pm epic list` and `pm epic get` commands:
- `pm epic list` - List all epics with status, progress, priority
- `pm epic get <epic-key>` - Show epic details with feature breakdown
- Rich table formatting (80-column width, progress right-aligned)
- Error handling for non-existent epics
- Integration with progress calculation service

**Success Gates**:
- Both commands work with human-readable table output
- Progress percentages display correctly (one decimal place)
- Error messages are clear and helpful
- Exit codes correct (0=success, 1=user error, 2=system error)
- Commands integrate cleanly with E04-F02 CLI framework

### Phase 3: Feature Query Commands (8 hours)

**Task**: T-E04-F04-003
**Dependencies**: T-E04-F04-001

Implement `pm feature list` and `pm feature get` commands:
- `pm feature list` - List all features with epic, status, progress
- `pm feature list --epic=<key>` - Filter features by epic
- `pm feature list --status=<status>` - Filter features by status
- `pm feature get <feature-key>` - Show feature details with task breakdown
- Task status breakdown (completed: X, in_progress: Y, etc.)
- Rich table formatting

**Success Gates**:
- All commands work with filtering
- Task breakdown shows accurate counts by status
- Empty results show helpful messages
- All filters validated (invalid status returns error with valid options)
- Progress calculations match expected values

### Phase 4: JSON Output & Filtering (4 hours)

**Task**: T-E04-F04-004
**Dependencies**: T-E04-F04-002, T-E04-F04-003

Add JSON output and advanced filtering:
- `--json` flag for all epic/feature commands
- JSON structure matches PRD specification
- Sorting support (--sort-by=progress, --sort-by=status)
- Machine-readable output for agent consumption
- Nested objects in JSON (epic with features array, feature with tasks array)

**Success Gates**:
- JSON output is valid and parseable
- jq queries work correctly (e.g., `pm epic get E01 --json | jq '.features[0].progress_pct'`)
- Sorting produces correct order
- JSON and table output show same data
- Agent can consume JSON without errors

### Phase 5: Integration Tests & Performance Validation (4 hours)

**Task**: T-E04-F04-005
**Dependencies**: T-E04-F04-004

Validate end-to-end functionality and performance:
- Integration tests for all commands with realistic data
- Progress calculation accuracy tests (all acceptance criteria scenarios)
- Performance benchmarks against PRD targets:
  - `pm epic list` <100ms for 100 epics
  - `pm epic get` <200ms for epics with 50 features
  - `pm feature get` <200ms for features with 100 tasks
- Error handling tests (non-existent keys, database errors)
- Empty result tests

**Success Gates**:
- All acceptance criteria scenarios pass
- Performance benchmarks meet targets
- Error messages match PRD specifications
- Edge cases handled correctly
- No query performance issues (verify with EXPLAIN)

### Phase 6: Documentation & Usage Examples (2 hours)

**Task**: T-E04-F04-006
**Dependencies**: T-E04-F04-005

Create user documentation and examples:
- Command reference for all epic/feature commands
- Usage examples for common scenarios
- JSON output examples with jq processing
- Error message catalog
- Integration guide for agents

**Success Gates**:
- All commands documented with examples
- Examples run successfully
- Agent integration patterns documented
- Help text accurate and helpful

## Design Documentation

All tasks reference the PRD:

- [PRD](../docs/plan/E04-task-mgmt-cli-core/E04-F04-epic-feature-queries/prd.md) - Complete requirements and acceptance criteria

## Dependencies & Critical Path

```
E04-F01 (Database Schema) [COMPLETE]
E04-F02 (CLI Infrastructure) [COMPLETE]
    ↓
T-E04-F04-001 (Progress Calculation) [6h]
    ↓
T-E04-F04-002 (Epic Commands) [8h]  ← CRITICAL PATH (longest task)
    ↓ AND
T-E04-F04-003 (Feature Commands) [8h]  ← CRITICAL PATH (longest task)
    ↓
T-E04-F04-004 (JSON & Filtering) [4h]
    ↓
T-E04-F04-005 (Integration Tests) [4h]
    ↓
T-E04-F04-006 (Documentation) [2h]
    ↓
READY FOR E05-F01 (Status Dashboard)
```

**Critical Path**: T-E04-F04-002 and T-E04-F04-003 are the longest tasks at 8 hours each. They can be worked in parallel after T-E04-F04-001 completes.

**Parallelization**: Tasks T-E04-F04-002 (Epic Commands) and T-E04-F04-003 (Feature Commands) can be developed in parallel since they both depend only on T-E04-F04-001.

## Downstream Feature Dependencies

Once all tasks are completed, these features can begin:

### E05-F01: Status Dashboard
**What they need**:
- Epic/Feature listing with progress calculations
- JSON output for data processing
- Filtering capabilities
- Progress calculation service

**Integration pattern**: Dashboard aggregates data from epic/feature queries to show comprehensive project status.

### E04-F03: Task Lifecycle Operations
**What they need**:
- Feature progress updates when task status changes
- Epic progress recalculation when feature progress changes

**Integration pattern**: Task status updates trigger progress recalculation.

## Feature Metrics

**Code Volume Estimate**:
- `pm/services/progress.py`: ~300 lines
- `pm/cli/epic.py`: ~400 lines
- `pm/cli/feature.py`: ~400 lines
- `pm/formatters/table.py`: ~200 lines
- `pm/formatters/json.py`: ~100 lines
- Tests: ~800 lines
- **Total**: ~2200 lines of production code + tests

**Test Coverage Targets**:
- `progress.py`: 100% (critical business logic)
- `epic.py`: 90%
- `feature.py`: 90%
- `formatters/`: 85%
- **Overall**: >90% code coverage

**Performance Targets** (from PRD):
- `pm epic list`: <100ms for 100 epics
- `pm epic get`: <200ms for epics with 50 features
- `pm feature list`: <100ms for 100 features
- `pm feature get`: <200ms for features with 100 tasks
- Progress calculations: No N+1 queries

## Notes

- **Read-Only Feature**: This feature provides query/read capabilities only. No create/update/delete operations for epics or features.
- **Progress Calculation Critical**: Accuracy of progress percentages is critical for project visibility and reporting. All edge cases must be handled.
- **Performance Focus**: Queries must be efficient (use JOINs, avoid N+1). Progress calculation must not load all data into memory.
- **Agent-Friendly**: JSON output must be clean and parseable for AI agent consumption. Structure must be consistent across all commands.
- **Error Messages**: Clear, helpful error messages with suggestions (e.g., "Use 'pm epic list' to see available epics").
- **Formatting**: Table output must fit in 80-column terminals. Long titles truncated with "...".

---

**Task Index Created**: 2025-12-14
**Feature Status**: PRD complete, ready for implementation
**Next Action**: Begin Phase 1 (T-E04-F04-001) or assign to implementation agent
