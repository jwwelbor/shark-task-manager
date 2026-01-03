---
timestamp: 2026-01-02T13:42:37-06:00
session_type: continuation
context: path-filename-architecture-refactoring
shark_task: T-E07-F18-001
shark_idea: I-2026-01-02-05
status: decision_made_ready_for_implementation
---

# Continue: Path/Filename Architecture Refactoring

## Context

Investigation of bug report "feature update --path doesn't work" (Idea I-2026-01-02-05) revealed a **fundamental architectural flaw** in how shark handles file paths. The `--path` flag was incompletely implemented and has never worked as designed.

## Current Status

**Shark State:**
- Task: `T-E07-F18-001` - "Fix feature update --path not updating custom_folder_path"
- Status: `ready_for_code_review` (but incomplete fix - only display layer)
- Feature: `E07-F18` - CLI Bug Fixes
- Epic: `E07` - Enhancements

**Analysis Complete:**
All documentation in `dev-artifacts/2026-01-02-feature-update-path-bug/`:
1. **EXECUTIVE_SUMMARY.md** - Decision summary for stakeholders (~11.5 day refactor)
2. **PATH_FILENAME_ARCHITECTURE.md** - Complete architecture design
3. **CURRENT_IMPLEMENTATION_ANALYSIS.md** - What's broken and why
4. **REFACTORING_PLAN.md** - 10-phase implementation plan

**Key Finding:**
Database has `custom_folder_path` column that is stored but **never used** in file path calculation. The problem extends to epics, features, and tasks (tasks don't even have the column).

## The Problem in Detail

### Current Behavior (Broken)
```bash
shark feature create --epic=E01 "My Feature" --path="docs/roadmap/2025"
# Stores custom_folder_path="docs/roadmap/2025" in database ✓
# But creates file at: docs/plan/E01-epic/E01-F01-my-feature/feature.md ❌
# The --path value has NO EFFECT on actual file location!
```

### Desired Behavior (User's Vision)
```
Epic:    {docs_root}/{epic_custom_path OR default}/{epic_filename}
Feature: {docs_root}/{epic_path}/{feature_custom_path OR default}/{feature_filename}
Task:    {docs_root}/{epic_path}/{feature_path}/{task_custom_path OR default}/{task_filename}
```

With proper inheritance: features inherit epic's path, tasks inherit feature's path.

## Decision Made ✅

**Chosen Approach: Simplified Full Path Storage**

After PM/UX/Architect review, the team chose a **simpler alternative** to the original hierarchical refactoring:

**Use single `file_path` column storing full file path. Drop `custom_folder_path` entirely.**

**Rationale:**
- Product: Files don't move in typical workflow (95% use defaults, 5% custom at creation)
- UX: Simple mental model (1 concept vs 6+ concepts in hierarchical approach)
- Architecture: Performance optimized for common case (no joins on hot path)

**Key Insight:** The hierarchical path system was solving an imaginary reorganization problem. Users need fast status tracking and easy file imports, not automatic cascading file moves.

See full decision: `dev-artifacts/2026-01-02-feature-update-path-bug/ARCHITECTURE_DECISION.md`

## How to Continue

### Implementation Plan (1.5 days)

The task has been updated with full implementation details: `docs/plan/E07-enhancements/E07-F18-cli-bug-fixes/tasks/T-E07-F18-001.md`

**6 Implementation Steps:**
1. Fix feature update command (use `file_path` not `custom_folder_path`)
2. Update flag names to `--file` with `--filepath`/`--path` aliases
3. Apply same fix to epic create/update
4. Apply same fix to task create/update
5. Schema migration (drop `custom_folder_path` columns)
6. Update models (remove CustomFolderPath field)

### Execution Strategy

**Option A: Single developer agent with TDD** (recommended for simplicity)
```bash
# Start the task
shark task start T-E07-F18-001

# Spawn developer agent for full implementation
# Agent will:
# - Write tests first (mocked repos for CLI tests)
# - Implement fixes step-by-step
# - Run integration tests with real DB
# - Update documentation
```

**Option B: Parallel agents** (if optimizing for speed)
```bash
# Break into 3 parallel workstreams:
# 1. Developer Agent: Feature + Epic commands
# 2. Developer Agent: Task command + Models
# 3. Developer Agent: Schema migration + Tests

# Main thread coordinates and merges
```

**Estimated: 12 hours total (~1.5 days)**

## Key Files to Reference

**Analysis Documents:**
- `dev-artifacts/2026-01-02-feature-update-path-bug/EXECUTIVE_SUMMARY.md`
- `dev-artifacts/2026-01-02-feature-update-path-bug/PATH_FILENAME_ARCHITECTURE.md`
- `dev-artifacts/2026-01-02-feature-update-path-bug/CURRENT_IMPLEMENTATION_ANALYSIS.md`
- `dev-artifacts/2026-01-02-feature-update-path-bug/REFACTORING_PLAN.md`

**Code to Modify (if doing full refactor):**
- `internal/db/db.go` - Schema migrations
- `internal/utils/path_resolver.go` - NEW: Centralized path resolution
- `internal/cli/commands/epic.go` - Fix create/update
- `internal/cli/commands/feature.go` - Fix create/update (lines 828-1125, 1585-1599)
- `internal/cli/commands/task.go` - Add path support
- `internal/repository/*_repository.go` - Support new columns

**Database Schema:**
```sql
-- Current (partially broken)
epics: file_path, custom_folder_path
features: file_path, custom_folder_path
tasks: file_path (NO custom_folder_path)

-- Proposed (from architecture doc)
All tables: custom_path, filename
(compute full path from inheritance)
```

## Execution Path Optimization

### Recommended Agent Strategy

**Main Thread (You):**
- Read analysis documents
- Make decision on approach
- Create feature/tasks if doing full refactor
- Coordinate agents
- Review outputs
- Merge results

**Parallel Agents (to avoid context bloat):**
1. **Developer Agent (Schema)** - Phase 1: Database migrations
2. **Developer Agent (Path Logic)** - Phase 2: Path resolver implementation
3. **Developer Agent (Epic)** - Phase 4: Epic command updates
4. **Developer Agent (Feature)** - Phase 5: Feature command updates (can run parallel with #3)
5. **Developer Agent (Task)** - Phase 6: Task command updates (can run parallel with #3 & #4)
6. **Developer Agent (Cascade)** - Phase 7: Cascading update logic
7. **Tech-Lead Agent** - Code review after each phase
8. **QA Agent** - Phase 9: Testing and validation

**Benefits:**
- Each agent has focused context (single phase)
- Phases 4-6 can run in parallel (independent)
- Main thread stays lightweight (coordination only)
- Code review happens incrementally (not all at end)

### Minimal Context Approach

If doing minimal fix:
1. Single developer agent with TDD
2. Focus only on feature.go:1585-1599
3. Write test first, implement fix, verify

## Quick Start Commands

```bash
# Check current state
shark task get T-E07-F18-001 --json
shark idea get I-2026-01-02-05 --json

# Read analysis
cat dev-artifacts/2026-01-02-feature-update-path-bug/EXECUTIVE_SUMMARY.md

# Make decision and proceed based on choice above
```

## Questions Resolved ✅

1. **Scope decision:** ✅ Use simplified full path storage (not hierarchical refactoring)
2. **File moving:** ✅ Auto-move files when `--file` flag updates path (with rollback on DB error)
3. **Cascading:** ✅ Not needed - files don't move in typical workflow
4. **Backward compat:** ✅ Drop `custom_folder_path` after code deploy (safe - column unused)
5. **Default paths:** ✅ Keep existing defaults unchanged (backward compatible)
6. **Flag naming:** ✅ Use `--file` primary, `--filepath` and `--path` as aliases

## Success Criteria

**Simplified Full Path Approach:**
- ✅ `feature update --file` works correctly (moves file + updates database)
- ✅ `epic update --file` and `task create --file` work
- ✅ All three flag aliases work (`--file`, `--filepath`, `--path`)
- ✅ Default paths work when flag omitted (95% case - backward compatible)
- ✅ `custom_folder_path` column removed from schema
- ✅ File moves are atomic (rollback on DB error)
- ✅ All tests pass (unit, integration, e2e)
- ✅ Documentation updated
- ✅ No regressions

## Estimated Effort

- **Simplified Fix:** ~12 hours / 1.5 days (see task T-E07-F18-001 for breakdown)

---

**Next Action:** Start implementation of T-E07-F18-001 using developer agent with TDD approach.
