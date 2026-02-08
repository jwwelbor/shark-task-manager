---
feature_key: E07-F26-centralized-workflow-service-authority
epic_key: E07
title: Centralized Workflow Service Authority
description: Establish workflow.Service as the single authoritative source for all workflow logic - status validation, transitions, help text, defaults, and display. Eliminate all hardcoded status strings and duplicated workflow logic across the codebase.
---

# Centralized Workflow Service Authority

**Feature Key**: E07-F26-centralized-workflow-service-authority

---

## Goal

### Problem

Workflow logic is scattered across 10+ files with hardcoded status strings, duplicated validation, and help text that doesn't reflect the configured workflow profile. Specifically:

1. **`models/validation.go`** has two hardcoded status maps (old 6-status + new 14-status) that never read from `.sharkconfig.json`
2. **`taskfile/parser.go`** has its own hardcoded 6-status validation map
3. **`cli/commands/task.go`** help text says `"Filter by status (todo, in_progress, completed, blocked)"` regardless of active profile
4. **`cli/commands/shared_flags.go`** hardcodes `"Status: draft, active, completed, archived"` in 4 places
5. **`cli/commands/aliases.go`** hardcodes status names in help text
6. **`config/workflow_default.go`** has `DefaultStatuses()` and `IsDefaultStatus()` as standalone functions instead of routing through the service
7. **`models/task.go`** has deprecated `TaskStatus*` constants still used as fallbacks in `workflow/service.go`
8. **`cli/commands/task_deps.go`** directly compares `task.Status == "completed"` instead of calling service methods
9. **`status/action_items.go`** hardcodes `"ready_for_approval"` and `"in_development"` in case statements

The result: switching to the advanced workflow profile doesn't update help text, validation rejects custom statuses, and developers must update multiple files when adding a status.

### Solution

Establish `internal/workflow.Service` as the **sole authority** for all workflow-related queries. Every part of the system that needs to know about statuses, transitions, validation, or display calls `workflow.Service` methods. The service reads from `.sharkconfig.json` on initialization, falling back to the default workflow (defined in one place within the service/config layer).

**No other package should contain hardcoded status strings or workflow logic.**

### Impact

- Status validation, help text, and transitions automatically reflect the active workflow profile
- Adding/removing/renaming a status requires changing only `.sharkconfig.json` (or the single default definition)
- Eliminates ~150 lines of duplicated status maps across the codebase
- Prevents status drift bugs where different code paths accept different status sets

---

## Architecture

### Design Principle: Single Source of Truth

```
.sharkconfig.json (or default if missing)
         │
         ▼
   workflow.Service  ← THE authority
   ┌─────────────────────────────────────┐
   │ IsValidStatus(status) bool          │
   │ IsValidTransition(from, to) bool    │
   │ GetAllStatuses() []string           │
   │ GetInitialStatus() string           │
   │ GetTerminalStatuses() []string      │
   │ IsTerminalStatus(status) bool       │
   │ GetValidTransitions(from) []string  │
   │ ValidateStatus(status) error        │
   │ StatusHelpText() string             │
   │ StatusFlagChoices() string          │
   │ GetStatusMetadata(status) ...       │
   │ GetStatusesByPhase(phase) []string  │
   │ GetStatusesByAgent(agent) []string  │
   │ FormatStatus(status) string         │
   │ NormalizeStatus(status) string      │
   │ GetPhases() []string                │
   └─────────────────────────────────────┘
         │
    Used by ALL consumers:
    ├── cli/commands/*        (help text, validation, filtering)
    ├── models/validation.go  (delegates to service)
    ├── taskfile/parser.go    (delegates to service)
    ├── repository/*          (status queries)
    ├── status/*              (progress, action items)
    ├── sync/*                (status handling)
    └── config/*              (default workflow only)
```

### Service Lifecycle

1. **Initialization**: `workflow.NewService(projectRoot)` loads config from `.sharkconfig.json`
2. **Fallback**: If no config exists, uses `config.DefaultWorkflow()` (the ONE place defaults live)
3. **Caching**: Config is loaded once and cached for the process lifetime
4. **Access**: Service is created once per command execution via `cli.GetWorkflowService()`

### New Methods Needed on workflow.Service

```go
// ValidateStatus returns an error if the status is not in the workflow.
// This replaces models.ValidateTaskStatus and all hardcoded validation maps.
func (s *Service) ValidateStatus(status string) error

// StatusHelpText returns a formatted string of valid statuses for --help flags.
// Example: "todo, in_progress, ready_for_review, completed, blocked"
func (s *Service) StatusHelpText() string

// StatusFlagDescription returns the full flag description including valid values.
// Example: "Filter by status (todo, in_progress, ready_for_review, completed, blocked)"
func (s *Service) StatusFlagDescription() string

// IsCompletedStatus returns true if this status represents completion.
// Replaces direct comparisons like `status == "completed"`.
func (s *Service) IsCompletedStatus(status string) bool

// GetDefaultStatus returns the default status for newly created tasks.
// Reads from special_statuses._start_[0], never hardcoded.
func (s *Service) GetDefaultStatus() string
```

### Global Access Pattern

```go
// In internal/cli/ - lazy-initialized workflow service
func GetWorkflowService() *workflow.Service {
    // Similar pattern to GetDB() - lazy init, cached
}

// Usage in any command:
func runTaskListCommand(cmd *cobra.Command, args []string) error {
    ws := cli.GetWorkflowService()
    // Use ws.StatusHelpText() for help, ws.ValidateStatus() for validation, etc.
}
```

---

## Scope of Changes

### Files to Modify

**Phase 1: Enhance workflow.Service (add missing methods)**
- `internal/workflow/service.go` - Add `ValidateStatus()`, `StatusHelpText()`, `StatusFlagDescription()`, `IsCompletedStatus()`, `GetDefaultStatus()`
- `internal/workflow/service_test.go` - Tests for all new methods

**Phase 2: Wire global access**
- `internal/cli/workflow_global.go` (new) - Global lazy-init accessor, similar to `db_global.go`

**Phase 3: Remove hardcoded validation**
- `internal/models/validation.go` - `ValidateTaskStatus` and `ValidateTaskStatusWithWorkflow` delegate to workflow.Service (or accept a validator interface)
- `internal/taskfile/parser.go` - Remove hardcoded `validStatuses` map, use service
- `internal/cli/commands/validators.go` - Remove hardcoded status map

**Phase 4: Fix help text**
- `internal/cli/commands/task.go` - `--status` flag description from service
- `internal/cli/commands/shared_flags.go` - All status help text from service
- `internal/cli/commands/aliases.go` - Dynamic help text

**Phase 5: Replace direct status comparisons**
- `internal/cli/commands/task_deps.go` - Replace `== "completed"` with `IsTerminalStatus()`
- `internal/cli/commands/task.go` - Replace `"todo"` in unblock message with `GetDefaultStatus()`
- `internal/status/action_items.go` - Replace hardcoded status strings with phase/metadata queries

**Phase 6: Clean up deprecated code**
- `internal/models/task.go` - Remove or deprecate `TaskStatus*` constants
- `internal/config/workflow_default.go` - Remove `IsDefaultStatus()` and `DefaultStatuses()` standalone functions (logic stays in `DefaultWorkflow()`)
- `internal/workflow/service.go` - Remove fallbacks to `models.TaskStatus*` constants

### Files NOT Changed (out of scope)
- `internal/init/profiles.go` - Profile definitions are config data, not logic. They define what gets written to `.sharkconfig.json` and are legitimate hardcoded config payloads.
- Test files using status strings in test fixtures - these are test data, not logic
- `internal/config/workflow_schema.go` - Existing schema is fine
- `internal/config/workflow_default.go` `DefaultWorkflow()` function itself - this IS the single default definition

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer using the advanced workflow profile, I want `shark task list --status=in_development` to work and `--help` to show all 19 valid statuses, so I can discover and use the statuses my project actually supports.

**Acceptance Criteria**:
- [ ] `shark task list --help` shows valid statuses from active workflow config
- [ ] `shark task list --status=in_development` works on advanced profile
- [ ] `shark task list --status=invalid_status` returns a clear error listing valid options

**Story 2**: As a project admin who adds a custom status to `.sharkconfig.json`, I want all shark commands to immediately recognize the new status without code changes.

**Acceptance Criteria**:
- [ ] Adding a status to `status_flow` and `status_metadata` in config makes it valid everywhere
- [ ] No Go code changes needed to support a new status
- [ ] Validation, help text, and transitions all reflect the new status

**Story 3**: As a developer, I want a single `workflow.Service` that I can call for any workflow question, so I never need to hardcode status strings in command logic.

**Acceptance Criteria**:
- [ ] `workflow.Service` provides methods for: validation, transition checking, status lists, help text, terminal/initial status checks, normalization
- [ ] No hardcoded status strings exist outside of `config/workflow_default.go` `DefaultWorkflow()` and `init/profiles.go`
- [ ] All CLI commands use `workflow.Service` for status-related logic

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Status help text reflects config**
- **Given** the project uses the advanced workflow profile (19 statuses)
- **When** I run `shark task list --help`
- **Then** the `--status` flag description lists all 19 valid statuses

**Scenario 2: Custom status validation**
- **Given** I add `"custom_review"` to `status_flow` and `status_metadata` in `.sharkconfig.json`
- **When** I run `shark task list --status=custom_review`
- **Then** it filters tasks by that status without error

**Scenario 3: Invalid status error**
- **Given** the project uses the basic workflow profile
- **When** I run `shark task list --status=in_development`
- **Then** I get an error: `invalid status "in_development". Valid statuses: todo, in_progress, ready_for_review, completed, blocked`

**Scenario 4: No config file fallback**
- **Given** `.sharkconfig.json` does not exist or has no workflow section
- **When** I run any shark command
- **Then** the default basic workflow is used (todo, in_progress, ready_for_review, completed, blocked)

**Scenario 5: Zero hardcoded status strings in command logic**
- **Given** the codebase after this feature
- **When** I grep for hardcoded status strings in `internal/cli/commands/`
- **Then** no status-specific logic exists outside of `workflow.Service` calls (excluding test files)

---

## Out of Scope

1. **Epic/Feature status workflow** - Epics and features have their own status model (draft, active, completed, archived). This feature focuses on task workflow only. Epic/feature status centralization can follow the same pattern later.
2. **Runtime config reloading** - Config is loaded once per command. Hot-reloading mid-command is not needed.
3. **Database-stored workflow** - Workflow config stays in `.sharkconfig.json`, not in the database.
4. **Profile definitions in `init/profiles.go`** - These are legitimate config payloads that get written to `.sharkconfig.json`. They are data, not logic.

---

## Dependencies

- **Existing `workflow.Service`** at `internal/workflow/service.go` - enhancing, not replacing
- **Existing `config.WorkflowConfig`** at `internal/config/workflow_schema.go` - unchanged
- **Existing `config.DefaultWorkflow()`** at `internal/config/workflow_default.go` - becomes the ONE default

---

## Test Plan

### Unit Tests
- `workflow.Service.ValidateStatus()` - valid statuses, invalid statuses, case insensitivity
- `workflow.Service.StatusHelpText()` - basic profile output, advanced profile output
- `workflow.Service.IsCompletedStatus()` - terminal vs non-terminal
- `workflow.Service.GetDefaultStatus()` - with config, without config
- All new methods tested with both basic and advanced workflow configs

### Integration Tests
- CLI `--help` output contains dynamic status list
- `shark task list --status=<valid>` works for all configured statuses
- `shark task list --status=<invalid>` returns error with valid status list
- Commands work correctly with no `.sharkconfig.json` (default fallback)

### Regression Tests
- All existing tests continue to pass
- Status transitions work as before
- `shark task start/complete/approve/block/unblock` unchanged behavior

---

*Last Updated*: 2026-02-08
