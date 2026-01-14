# System Architecture: Workflow-Aware Task Command System

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

This document defines the overall system architecture for implementing phase-aware task commands (`claim`, `finish`, `reject`) that dynamically adapt to any workflow configuration. The design follows proven CLI patterns and integrates cleanly with shark's existing codebase.

---

## Architectural Principles

All architecture decisions follow the **Appropriate, Proven, Simple** philosophy:

- **Appropriate**: Right for a CLI tool with SQLite database and AI orchestrator integration
- **Proven**: Uses established patterns from existing shark codebase (workflow service, repository pattern)
- **Simple**: No unnecessary abstractions, clear data flow, maintainable

---

## System Components

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ task claim   │  │ task finish  │  │ task reject  │      │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘      │
│         │                  │                  │              │
└─────────┼──────────────────┼──────────────────┼──────────────┘
          │                  │                  │
          └──────────────────┼──────────────────┘
                             │
┌────────────────────────────┴─────────────────────────────────┐
│                 Command Execution Service                      │
│  • Parses arguments and flags                                  │
│  • Loads workflow configuration                                │
│  • Orchestrates repositories                                   │
│  • Handles transactions                                        │
└────────────────────┬───────────────────────────────────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
┌───────▼──────┐ ┌──▼──────────┐ ┌▼─────────────────┐
│ Workflow     │ │ Task        │ │ WorkSession      │
│ Service      │ │ Repository  │ │ Repository       │
│              │ │             │ │                  │
│ • Config     │ │ • CRUD      │ │ • Session CRUD   │
│   loading    │ │ • Status    │ │ • Start/End      │
│ • Validation │ │   updates   │ │ • Duration calc  │
│ • Transition │ │ • Queries   │ │                  │
│   logic      │ │             │ │                  │
└───────┬──────┘ └──┬──────────┘ └┬─────────────────┘
        │           │             │
        └───────────┴─────────────┘
                    │
┌───────────────────▼──────────────────────────┐
│            SQLite Database                   │
│  • tasks table (status, metadata)            │
│  • task_sessions table (work tracking)       │
│  • task_history table (audit trail)          │
└──────────────────────────────────────────────┘
                    │
┌───────────────────▼──────────────────────────┐
│         Configuration Files                   │
│  • .sharkconfig.json (workflow definition)   │
└──────────────────────────────────────────────┘
```

---

## Data Flow

### Claim Command Flow

```
User: shark task claim T-E07-F20-001 --agent=backend

1. CLI Command Handler
   ↓ Parse args: task_key, agent_type

2. Workflow Service
   ↓ Load .sharkconfig.json
   ↓ Get current task status → "ready_for_development"
   ↓ Determine target status → "in_development"
   ↓ Validate transition: ready_for_development → in_development ✓

3. Task Repository (Transaction)
   ↓ BEGIN TRANSACTION
   ↓ UPDATE tasks SET status='in_development', assigned_agent='backend'
   ↓ INSERT INTO task_history (task_id, from_status, to_status, agent)

4. WorkSession Repository
   ↓ INSERT INTO task_sessions (task_id, agent_id, started_at)
   ↓ COMMIT TRANSACTION

5. Response
   ↓ Success: "Task T-E07-F20-001 claimed. Status: in_development"
   ↓ Next phase: ready_for_code_review (tech-lead, code-reviewer)
```

### Finish Command Flow

```
User: shark task finish T-E07-F20-001 --notes="API complete"

1. CLI Command Handler
   ↓ Parse args: task_key, notes

2. Workflow Service
   ↓ Load .sharkconfig.json
   ↓ Get current task status → "in_development"
   ↓ Get valid transitions → ["ready_for_code_review", "ready_for_refinement", "blocked"]
   ↓ Select standard forward flow → "ready_for_code_review"
   ↓ Validate transition: in_development → ready_for_code_review ✓

3. Task Repository (Transaction)
   ↓ BEGIN TRANSACTION
   ↓ UPDATE tasks SET status='ready_for_code_review', completed_at=NOW()
   ↓ INSERT INTO task_history (task_id, from_status, to_status, notes)

4. WorkSession Repository
   ↓ UPDATE task_sessions SET ended_at=NOW(), outcome='completed', notes='...'
   ↓ WHERE task_id=X AND ended_at IS NULL
   ↓ COMMIT TRANSACTION

5. Response
   ↓ Success: "Task completed. Status: ready_for_code_review"
   ↓ Next phase: code_review (tech-lead, code-reviewer)
```

### Reject Command Flow

```
User: shark task reject T-E07-F20-001 --reason="Missing AC" --to=in_refinement

1. CLI Command Handler
   ↓ Parse args: task_key, reason, target_status

2. Workflow Service
   ↓ Load .sharkconfig.json
   ↓ Get current task status → "in_development"
   ↓ Validate target_status exists in workflow
   ↓ Validate transition: in_development → in_refinement ✓
   ↓ Check backward flow (earlier phase) ✓

3. Task Repository (Transaction)
   ↓ BEGIN TRANSACTION
   ↓ UPDATE tasks SET status='in_refinement'
   ↓ INSERT INTO task_history (task_id, from_status, to_status, rejection_reason)
   ↓ INSERT INTO task_notes (task_id, type='rejection', content='...')

4. WorkSession Repository
   ↓ UPDATE task_sessions SET ended_at=NOW(), outcome='rejected', notes='...'
   ↓ WHERE task_id=X AND ended_at IS NULL
   ↓ COMMIT TRANSACTION

5. Response
   ↓ Success: "Task rejected. Status: in_refinement. Reason: Missing AC"
   ↓ Next phase: refinement (business-analyst, architect)
```

---

## Component Responsibilities

### CLI Command Layer

**Location**: `internal/cli/commands/task_claim.go`, `task_finish.go`, `task_reject.go`

**Responsibilities**:
- Parse command-line arguments and flags
- Validate required parameters (task key, agent type, reason)
- Get database connection via `cli.GetDB()`
- Orchestrate workflow service and repositories
- Format output (JSON or human-readable)
- Handle errors and display user-friendly messages

**Does NOT**:
- Directly manipulate database
- Implement business logic
- Parse workflow configuration

### Workflow Configuration Reader

**Location**: `internal/workflow/service.go` (existing, enhance)

**Responsibilities**:
- Load `.sharkconfig.json` workflow configuration
- Cache workflow in memory (singleton per process)
- Provide API for transition queries:
  - `GetValidTransitions(currentStatus string) []string`
  - `IsValidTransition(from, to string) bool`
  - `GetStatusMetadata(status string) StatusInfo`
  - `GetAgentTypes(status string) []string`
- Validate workflow structure on load
- Normalize status names (case-insensitive)

**Caching Strategy**:
```go
// Singleton pattern with file modification check
var (
    cachedWorkflow *WorkflowConfig
    cachedModTime  time.Time
    cacheMutex     sync.RWMutex
)

func LoadWorkflow(configPath string) (*WorkflowConfig, error) {
    cacheMutex.RLock()
    currentModTime := getFileModTime(configPath)
    if cachedWorkflow != nil && currentModTime.Equal(cachedModTime) {
        defer cacheMutex.RUnlock()
        return cachedWorkflow, nil
    }
    cacheMutex.RUnlock()

    // Cache miss or outdated - reload
    cacheMutex.Lock()
    defer cacheMutex.Unlock()

    // ... load and parse config ...
    cachedWorkflow = workflow
    cachedModTime = currentModTime
    return workflow, nil
}
```

### Task Repository

**Location**: `internal/repository/task_repository.go` (existing, enhance)

**New Methods**:
```go
// ClaimTask transitions task from ready_for_X to in_X
func (r *TaskRepository) ClaimTask(ctx context.Context, taskID int64, agentID string) error

// FinishTask transitions task to next phase in workflow
func (r *TaskRepository) FinishTask(ctx context.Context, taskID int64, notes *string) error

// RejectTask sends task backward to previous phase
func (r *TaskRepository) RejectTask(ctx context.Context, taskID int64, targetStatus string, reason string) error
```

**Transaction Management**:
- All status updates wrapped in transactions
- Atomic: status update + history record + session update
- Rollback on any failure
- Use `repository.DB.BeginTx()` pattern

### WorkSession Repository

**Location**: `internal/repository/work_session_repository.go` (existing, enhance)

**Responsibilities**:
- Track work session start/end times
- Calculate session duration
- Record session outcomes (completed, rejected, blocked)
- Support queries for analytics

**Schema** (see session-tracking.md for details)

---

## Integration Points

### With AI Orchestrator

**Query Pattern**:
```bash
# Orchestrator polls for ready tasks by agent type
shark task list --status=ready_for_development --agent=backend --json

# Orchestrator claims task
shark task claim T-E07-F20-001 --agent=backend --json

# Orchestrator finishes task
shark task finish T-E07-F20-001 --notes="Implementation complete" --json
```

**JSON Response Format**:
```json
{
  "task_key": "T-E07-F20-001",
  "previous_status": "in_development",
  "new_status": "ready_for_code_review",
  "next_phase": {
    "phase": "code_review",
    "agent_types": ["tech-lead", "code-reviewer"]
  },
  "session": {
    "started_at": "2026-01-11T10:00:00Z",
    "ended_at": "2026-01-11T12:30:00Z",
    "duration_minutes": 150
  }
}
```

### With Existing Commands

**Backward Compatibility**:
- Keep existing commands (`start`, `complete`, `approve`) functional
- Add deprecation warnings in Phase 2
- Commands use same underlying repositories

**Migration Path**:
```
Phase 1 (Release 1): Add claim/finish/reject, keep old commands
Phase 2 (Release 2): Add deprecation warnings to old commands
Phase 3 (Release 3): Remove old commands (after 2 releases)
```

### With Workflow Config

**Configuration Loading**:
1. Check `--config` flag for custom path
2. Fall back to `.sharkconfig.json` in project root
3. Fall back to default hardcoded workflow if missing

**Validation on Load**:
- Every status in `status_flow` keys has outgoing transitions (unless terminal)
- All transition targets exist as status_flow keys or in terminal list
- All statuses referenced in `status_metadata`
- No orphaned statuses

**Error Handling**:
```bash
# Invalid transition
$ shark task claim T-E07-F20-001
Error: Cannot claim task in status 'in_development'
Task is already claimed. Use 'shark task finish' to complete or 'shark task reject' to send back.
Current status: in_development
Valid commands: finish, reject, block

# Invalid workflow config
$ shark task claim T-E07-F20-001
Error: Workflow configuration invalid
Status 'in_qa' has no outgoing transitions. Every non-terminal status must have valid next states.
Fix .sharkconfig.json or use --force to bypass validation.
```

---

## Error Handling Strategy

### Command Validation Errors (Exit Code 1)

**Examples**:
- Missing required argument: `--reason` not provided for reject
- Invalid task key format
- Task not found in database

**Response**: User-actionable error message with example

### Workflow Validation Errors (Exit Code 3)

**Examples**:
- Invalid status transition
- Task in wrong status for command (can't claim if already in `in_X`)
- Workflow config invalid or missing

**Response**: Error + suggestion + `--force` option

### Database Errors (Exit Code 2)

**Examples**:
- Transaction failure
- Connection error
- Constraint violation

**Response**: Technical error message + rollback confirmation

### Override with --force

**Bypasses**:
- Workflow transition validation
- Status prerequisite checks

**Does NOT bypass**:
- Database constraints
- Required argument validation
- Transaction atomicity

**Warning**: Always display warning when --force used

---

## Performance Considerations

### Workflow Config Caching

**Target**: < 50ms overhead per command

**Implementation**:
- Cache loaded config in memory
- Check file modification time on each command
- Only reload if file changed
- Mutex for thread-safety

### Database Queries

**Target**: < 500ms for 90th percentile (claim/finish/reject)

**Optimization**:
- Single transaction for all updates
- Use prepared statements
- Index on `tasks.key` (existing)
- Index on `task_sessions.task_id` (new)

### Bulk Operations

**Future Enhancement** (Could Have):
```bash
# Finish multiple tasks at once
shark task finish --status=ready_for_approval --all --yes
```

---

## Testing Strategy

### Unit Tests

**Mock Repositories** (CLI commands):
- Test command argument parsing
- Test output formatting
- Test error handling
- Mock workflow service

**Real Database** (Repositories):
- Test transaction atomicity
- Test status transitions
- Test session tracking
- Clean up before each test

### Integration Tests

**Workflow Scenarios**:
```go
func TestFullWorkflowCycle(t *testing.T) {
    // 1. Create task (draft)
    // 2. Claim for refinement (in_refinement)
    // 3. Finish refinement (ready_for_development)
    // 4. Claim for development (in_development)
    // 5. Finish development (ready_for_code_review)
    // 6. Claim for review (in_code_review)
    // 7. Finish review (ready_for_qa)
    // 8. Claim for QA (in_qa)
    // 9. Finish QA (ready_for_approval)
    // 10. Claim for approval (in_approval)
    // 11. Finish approval (completed)
}
```

### Workflow Compatibility Tests

**Test Workflows**:
- Default 3-state: todo → in_progress → completed
- Simple 5-state custom
- Complex 10-state enterprise
- Minimal 2-state: draft → completed
- Branching with skip paths

**Validation**:
- All transitions work correctly
- Agent type filtering works
- Error messages accurate

---

## Security Considerations

### Input Validation

- Task keys: alphanumeric + hyphens only
- Agent types: string, max 100 chars
- Notes/Reason: string, max 5000 chars
- Status names: alphanumeric + underscores only

### SQL Injection Prevention

- Use parameterized queries everywhere
- No string concatenation for SQL
- Validate enum values (TaskStatus type)

### Authorization

**Phase 1 (MVP)**:
- No strict agent type enforcement
- Warning if wrong agent type claims task
- Humans can override with --force

**Future**:
- Strict RBAC mode (optional)
- API keys for orchestrator
- Audit trail of all agent actions

---

## Deployment & Migration

### Phase 1: Non-Breaking Addition

**Changes**:
- Add new commands: `claim`, `finish`, `reject`
- Add `task_sessions` table
- Enhance workflow service
- All existing commands still work

**Migration**:
```bash
# No migration needed - new commands available
shark task claim T-E07-F20-001 --agent=backend
```

### Phase 2: Deprecation Warnings

**Changes**:
- Add deprecation warnings to old commands
- Update documentation
- Provide migration guide

**Example**:
```bash
$ shark task start T-E07-F20-001
Warning: 'task start' is deprecated and will be removed in Release 3.
Use 'task claim' instead: shark task claim T-E07-F20-001 --agent=<type>
```

### Phase 3: Removal

**Changes**:
- Remove deprecated commands
- Clean up dead code
- Update tests

**Breaking Change Notice**:
- 2 releases advance notice
- Migration guide in CHANGELOG
- Error message points to new commands

---

## Monitoring & Observability

### Logging

**What to Log**:
- All status transitions (success and failure)
- Workflow validation failures
- Force overrides used
- Session durations

**Format**:
```
[2026-01-11 12:30:45] INFO: Task T-E07-F20-001 claimed by backend (in_development)
[2026-01-11 14:15:20] INFO: Task T-E07-F20-001 finished by backend (ready_for_code_review) [session: 105m]
[2026-01-11 14:30:10] WARN: Task T-E07-F20-001 force-transitioned (bypassed workflow validation)
[2026-01-11 15:00:00] ERROR: Transition failed: in_qa → completed (invalid per workflow)
```

### Metrics (Future)

**Track**:
- Command usage counts (claim vs. start)
- Average session duration by phase
- Rejection rate by phase
- Force override frequency

---

## Open Questions & Decisions

### Q1: Should `claim` auto-detect agent type?

**Options**:
- A. Require `--agent` flag always
- B. Read from `SHARK_AGENT_TYPE` env var
- C. Read from user config `~/.sharkconfig`

**Decision**: B + C (env var, then user config, with explicit flag override)

**Rationale**: Reduces typing for humans, clear for AI orchestrator

### Q2: What if workflow has no `in_` states?

**Scenario**: Workflow is `draft → completed`

**Solution**: `claim` transitions directly to `completed` (no-op effectively)

**Rationale**: Supports minimal workflows gracefully

### Q3: Should `finish` require confirmation for terminal transitions?

**Scenario**: User runs `finish` and task goes to `completed`

**Options**:
- A. Require `--confirm` flag
- B. Show warning, proceed
- C. No special handling

**Decision**: B (show warning, proceed)

**Rationale**: Trust users, avoid extra typing

---

## Future Enhancements

### Not in MVP

1. **Bulk Operations**: Finish multiple tasks at once
2. **Interactive Mode**: Prompt for next status if multiple options
3. **Workflow Designer UI**: Visual editor for `.sharkconfig.json`
4. **Advanced Analytics**: Time-in-phase reports, bottleneck detection
5. **Workflow Hot-Reload**: Detect config changes without restart
6. **Multi-Tenancy**: Different workflows per epic/feature

---

## References

- [Workflow Configuration Reader Design](./workflow-config-reader.md)
- [Command Specifications](./command-specifications.md)
- [Session Tracking Design](./session-tracking.md)
- [Transition Validation Logic](./transition-validation.md)
- [Migration Strategy](./migration-strategy.md)
- [Epic Requirements](../requirements.md)
- [User Journeys](../user-journeys.md)
