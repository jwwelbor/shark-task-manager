# UAT Test Guide - Centralized Workflow Service Authority

**Feature:** E07-F26 - Centralized Workflow Service Authority
**Epic:** E07 - Enhancements
**Generated:** 2026-02-08
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Improve and enhance the Shark Task Manager CLI with quality-of-life features, architectural improvements, and workflow centralization.

**This Feature's Role:** Establishes `workflow.Service` as the single authoritative source for all workflow logic -- status validation, transitions, help text, defaults, and display. Eliminates all hardcoded status strings and duplicated workflow logic across the codebase.

**Related Features:**
- E07-F25: CLI verb-first shortcuts and reorganized help (completed - shared command registration patterns)
- E07-F24: Workflow profiles (completed - provides the profile configs this feature reads)

**Integration Points:**
- `workflow.Service` is used by CLI commands, status package, models validation, and taskfile parser
- `.sharkconfig.json` is the configuration source, shared with init/profile system
- `cli.GetWorkflowService()` global accessor follows same pattern as `cli.GetDB()`

---

## Design Intent

**From Feature PRD:**
> Establish `internal/workflow.Service` as the **sole authority** for all workflow-related queries. Every part of the system that needs to know about statuses, transitions, validation, or display calls `workflow.Service` methods.

> **No other package should contain hardcoded status strings or workflow logic.**

**Key Design Decisions:**
- Model layer (`models/validation.go`, `taskfile/parser.go`) performs only basic structural validation (non-empty checks); workflow-aware validation happens at CLI layer
- Global `cli.GetWorkflowService()` uses `sync.Once` lazy initialization pattern (same as `cli.GetDB()`)
- Deprecated `TaskStatus*` constants removed entirely, replaced by `TaskStatus()` type literals
- `IsDefaultStatus()` and `DefaultStatuses()` standalone helpers removed; `DefaultWorkflow()` remains as the ONE default
- Phase-based metadata checks replace hardcoded status string comparisons in `action_items.go`

---

## Feature Acceptance Validation

| # | Acceptance Criteria | Status |
|---|---------------------|--------|
| AC-1 | `shark task list --help` shows valid statuses from active workflow config | [ ] |
| AC-2 | `shark task list --status=in_development` works on advanced profile | [ ] |
| AC-3 | `shark task list --status=invalid_xyz` returns clear error listing valid options | [ ] |
| AC-4 | No `.sharkconfig.json` = default basic workflow (todo, in_progress, ready_for_review, completed, blocked) | [ ] |
| AC-5 | Zero hardcoded status strings in command logic (excluding test files and config defaults) | [ ] |
| AC-6 | `workflow.Service` provides methods for validation, transitions, status lists, help text, terminal/initial checks, normalization | [ ] |
| AC-7 | Build compiles cleanly (`go build ./...`) | [ ] |
| AC-8 | All tests pass (`go test ./internal/...`) except pre-existing `TestLoadActualSharkConfig` | [ ] |

---

## Test Scenarios

### Scenario 1: Help Text Reflects Active Workflow Config
**Tasks covered:** T-E07-F26-005 (config-driven help text)

**Steps:**
1. Run `./bin/shark task list --help` and check `--status` flag description
2. Verify it lists statuses from the advanced workflow profile (draft, in_development, ready_for_code_review, etc.)
3. Run `./bin/shark unblock --help` and check the Long description
4. Verify it mentions the config-driven default status (e.g., "draft" for advanced, "todo" for basic)

**Success Criteria:**
- [ ] `--status` flag description contains `draft` (advanced profile start status)
- [ ] `--status` flag description contains `in_development` (advanced profile status)
- [ ] `--status` flag description does NOT say "todo, in_progress, completed, blocked" (hardcoded basic list)
- [ ] Unblock help text mentions the configured default status

### Scenario 2: Status Validation is Config-Driven
**Tasks covered:** T-E07-F26-002, T-E07-F26-004

**Steps:**
1. Run `./bin/shark task list --status=in_development` (valid in advanced profile)
2. Run `./bin/shark task list --status=invalid_xyz_status`
3. Check error message lists valid statuses

**Success Criteria:**
- [ ] `--status=in_development` does not produce a validation error
- [ ] `--status=invalid_xyz_status` produces an error containing "invalid status"
- [ ] Error message lists actual valid statuses from the workflow config

### Scenario 3: Workflow Service Methods Work Correctly
**Tasks covered:** T-E07-F26-002, T-E07-F26-003

**Steps:**
1. Run `go test -v ./internal/workflow/ -run "TestService_ValidateStatus|TestService_StatusHelpText|TestService_IsCompletedStatus|TestService_GetDefaultStatus"`
2. Run `go test -v ./internal/cli/ -run "TestGetWorkflowService"`
3. Verify all pass

**Success Criteria:**
- [ ] All `workflow.Service` new method tests pass
- [ ] Global accessor tests pass (returns non-nil, same instance, thread-safe)

### Scenario 4: Sync Command Removed
**Tasks covered:** T-E07-F26-001

**Steps:**
1. Run `./bin/shark sync` and verify it no longer exists
2. Check `./bin/shark --help` does not list `sync` command
3. Verify build passes without sync code

**Success Criteria:**
- [ ] `shark sync` returns "unknown command" error
- [ ] No `sync` in help output
- [ ] `go build ./...` succeeds

### Scenario 5: Deprecated Constants Removed
**Tasks covered:** T-E07-F26-007

**Steps:**
1. Run `grep -rn "TaskStatusTodo\|TaskStatusCompleted\|TaskStatusBlocked\|TaskStatusInProgress\|TaskStatusReadyForReview\|TaskStatusArchived" internal/ --include="*.go"` (exclude test files)
2. Verify no production code references the old constants
3. Check `internal/models/task.go` no longer defines the constants
4. Check `internal/config/workflow_default.go` no longer has `IsDefaultStatus()` or `DefaultStatuses()`

**Success Criteria:**
- [ ] No `TaskStatus*` constants in `models/task.go`
- [ ] No `IsDefaultStatus()` in `config/workflow_default.go`
- [ ] No `DefaultStatuses()` in `config/workflow_default.go`
- [ ] `DefaultWorkflow()` still exists and returns valid config

### Scenario 6: Status Comparisons Use Service Methods
**Tasks covered:** T-E07-F26-006

**Steps:**
1. Check `internal/cli/commands/task_deps.go` for `IsTerminalStatus()` usage instead of `== "completed"`
2. Check `internal/cli/commands/task.go` `filterTasksByCompletedStatus` uses `IsTerminalStatus()`
3. Check `internal/cli/commands/task.go` `isTaskAvailable` uses `IsTerminalStatus()`
4. Check `internal/status/action_items.go` uses phase-based metadata checks

**Success Criteria:**
- [ ] No `task.Status == "completed"` in task_deps.go
- [ ] `filterTasksByCompletedStatus` uses `ws.IsTerminalStatus()`
- [ ] `isTaskAvailable` uses `ws.IsTerminalStatus()`
- [ ] `action_items.go` uses `cfg.GetStatusMetadata()` for phase-based categorization

### Scenario 7: Key Service (Bonus - T-009)
**Tasks covered:** T-E07-F26-009

**Steps:**
1. Run `go test -v ./internal/keys/` to verify KeyService tests pass
2. Verify `internal/keys/service.go` exists with key parsing/normalization methods

**Success Criteria:**
- [ ] All key service tests pass
- [ ] KeyService provides centralized key parsing functionality

### Scenario 8: Full Build and Test Suite
**Tasks covered:** All tasks

**Steps:**
1. Run `go build ./...`
2. Run `go test ./internal/...`
3. Run `go vet ./...`

**Success Criteria:**
- [ ] Build compiles with zero errors
- [ ] All tests pass (except pre-existing `TestLoadActualSharkConfig`)
- [ ] `go vet` reports no issues

---

## Task Coverage Map

| Task | Title | Scenario Coverage |
|------|-------|-------------------|
| T-E07-F26-001 | Remove sync command | Scenario 4 |
| T-E07-F26-002 | Add workflow.Service methods | Scenarios 2, 3 |
| T-E07-F26-003 | Global workflow accessor | Scenario 3 |
| T-E07-F26-004 | Replace hardcoded validation | Scenario 2 |
| T-E07-F26-005 | Config-driven help text | Scenario 1 |
| T-E07-F26-006 | Replace status comparisons | Scenario 6 |
| T-E07-F26-007 | Remove deprecated constants | Scenario 5 |
| T-E07-F26-008 | Update help text for --order | N/A (docs only) |
| T-E07-F26-009 | Centralize key parsing | Scenario 7 |

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | (none yet) |
| Result | - |
| Results File | - |

**Previous Sessions:** None
