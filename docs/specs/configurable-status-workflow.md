# Configurable Status Workflow System

**Version**: 1.0
**Date**: 2025-12-29
**Status**: Draft

## Overview

Transform Shark's task status system from a hardcoded linear progression to a flexible, configuration-driven workflow engine that supports multi-agent collaboration and complex development processes.

## Problem Statement

Currently, Shark uses a fixed status progression:
```
todo → in_progress → ready_for_review → completed
         ↓
      blocked
```

This rigid model has limitations:
1. **No refinement phase** for AI agents to analyze and specify tasks
2. **No QA/approval gates** for multi-agent workflows
3. **No backward transitions** (e.g., QA finds bugs → back to development)
4. **Hardcoded validation** in repository layer prevents customization
5. **Single workflow** doesn't accommodate different task types (feature, hotfix, spike)

## Goals

### Primary Goals
1. **Configuration-driven workflow**: Define status transitions in `.sharkconfig.json`
2. **Flexible transitions**: Support forward, backward, and lateral moves between statuses
3. **Multi-agent targeting**: Enable agents to query tasks by workflow phase
4. **Validation with escape hatch**: Enforce valid transitions with `--force` override
5. **Backward compatibility**: Migrate existing tasks gracefully

### Non-Goals
- Workflow automation (triggers/webhooks)
- Multiple parallel workflows (future enhancement)
- Time-based status transitions
- External system integrations

## Design

### 1. Configuration Schema

Add `status_flow` section to `.sharkconfig.json`:

```json
{
  "project_name": "shark-default
  "status_flow": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"],

    "draft": ["ready_for_refinement", "cancelled", "on_hold"],
    "ready_for_refinement": ["in_refinement", "cancelled", "on_hold"],
    "in_refinement": ["ready_for_development", "draft", "blocked", "on_hold"],

    "ready_for_development": ["in_development", "ready_for_refinement", "cancelled", "on_hold"],
    "in_development": ["ready_for_review", "ready_for_refinement", "blocked", "on_hold"],

    "ready_for_code_review": ["in_code_review", "in_development", "on_hold"],
    "in_code_review": ["ready_for_qa", "in_development", "ready_for_refinement", "on_hold"],

    "ready_for_qa": ["in_qa", "on_hold"],
    "in_qa": ["ready_for_approval", "in_development", "ready_for_refinement", "blocked", "on_hold"],

    "ready_for_approval": ["in_approval", "on_hold"],
    "in_approval": ["completed", "ready_for_qa", "ready_for_development", "ready_for_refinement", "on_hold"],

    "blocked": ["ready_for_development", "ready_for_refinement", "cancelled"],
    "on_hold": ["ready_for_refinement", "ready_for_development", "cancelled"],

    "completed": [],
    "cancelled": []
  },

  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Task created but not yet refined",
      "phase": "planning"
    },
    "ready_for_refinement": {
      "color": "cyan",
      "description": "Awaiting specification and analysis",
      "phase": "planning",
      "agent_types": ["business-analyst", "architect", "ux-designer"]
    },
    "in_refinement": {
      "color": "blue",
      "description": "Being analyzed and specified",
      "phase": "planning"
    },
    "ready_for_development": {
      "color": "yellow",
      "description": "Spec complete, ready for implementation",
      "phase": "development",
      "agent_types": ["developer", "ai-coder"]
    },
    "in_development": {
      "color": "yellow",
      "description": "Code implementation in progress",
      "phase": "development"
    },
    "ready_for_code_review": {
      "color": "magenta",
      "description": "Code complete, awaiting code review",
      "phase": "review",
      "agent_types": ["tech-lead", "architect"]
    },
    "in_code_review": {
      "color": "magenta",
      "description": "Under code review",
      "phase": "review"
    },
    "ready_for_qa": {
      "color": "green",
      "description": "Ready for quality assurance testing",
      "phase": "qa",
      "agent_types": ["qa", "test-engineer"]
    },
    "in_qa": {
      "color": "green",
      "description": "Being tested",
      "phase": "qa"
    },
    "ready_for_approval": {
      "color": "purple",
      "description": "Awaiting final approval",
      "phase": "approval",
      "agent_types": ["product-manager", "client"]
    },
    "in_approval": {
      "color": "purple",
      "description": "Under final review",
      "phase": "approval"
    },
    "blocked": {
      "color": "red",
      "description": "Temporarily blocked by external dependency",
      "phase": "any"
    },
    "on_hold": {
      "color": "orange",
      "description": "Intentionally paused",
      "phase": "any"
    },
    "completed": {
      "color": "white",
      "description": "Task finished and approved",
      "phase": "done"
    },
    "cancelled": {
      "color": "gray",
      "description": "Task abandoned or deprecated",
      "phase": "done"
    }
  }
}
```

### 2. Status Flow Semantics

**Special Keys**:
- `_start_`: Valid initial statuses for `shark task create`
- `_complete_`: Terminal statuses (workflow ends)

**Transition Rules**:
- Key = current status
- Value = array of allowed next statuses
- Empty array = terminal status (no transitions allowed)
- Missing key = invalid status

**Validation Logic**:
```
IF --force flag:
    Allow any transition
ELSE:
    IF current_status not in status_flow:
        ERROR: "Unknown status: {current_status}"
    IF next_status not in status_flow[current_status]:
        ERROR: "Invalid transition from {current_status} to {next_status}"
        HINT: "Valid next statuses: {status_flow[current_status]}"
    ELSE:
        Allow transition
```

### 3. Database Schema Changes

**No schema changes required** - status field already exists as TEXT:
```sql
CREATE TABLE tasks (
    ...
    status TEXT NOT NULL DEFAULT 'todo',
    ...
);
```

However, add validation trigger:
```sql
-- Optional: Add CHECK constraint for valid statuses
-- (dynamically generated from config)
CREATE TRIGGER validate_task_status_update
BEFORE UPDATE ON tasks
FOR EACH ROW
BEGIN
    -- Validation happens in application layer
    -- This trigger just records the change
    SELECT CASE
        WHEN NEW.status != OLD.status THEN
            -- Insert into task_history happens automatically via existing trigger
            1
    END;
END;
```

### 4. Configuration Management

**New package**: `internal/config/workflow.go`

```go
package config

type StatusFlow map[string][]string

type StatusMetadata struct {
    Color       string   `json:"color"`
    Description string   `json:"description"`
    Phase       string   `json:"phase"`
    AgentTypes  []string `json:"agent_types,omitempty"`
}

type WorkflowConfig struct {
    StatusFlow     StatusFlow                `json:"status_flow"`
    StatusMetadata map[string]StatusMetadata `json:"status_metadata,omitempty"`
}

// LoadWorkflowConfig reads workflow configuration from .sharkconfig.json
func LoadWorkflowConfig(configPath string) (*WorkflowConfig, error)

// GetValidNextStatuses returns allowed transitions from current status
func (w *WorkflowConfig) GetValidNextStatuses(currentStatus string) ([]string, error)

// CanTransition checks if transition is valid
func (w *WorkflowConfig) CanTransition(from, to string) error

// GetStartStatuses returns valid initial statuses
func (w *WorkflowConfig) GetStartStatuses() []string

// GetTerminalStatuses returns workflow end statuses
func (w *WorkflowConfig) GetTerminalStatuses() []string

// IsTerminal checks if status ends the workflow
func (w *WorkflowConfig) IsTerminal(status string) bool
```

### 5. Repository Layer Changes

**Update**: `internal/repository/task_repository.go`

```go
// Add workflow config to repository
type TaskRepository struct {
    db       *db.DB
    workflow *config.WorkflowConfig
}

// NewTaskRepository now loads workflow config
func NewTaskRepository(database *db.DB) *TaskRepository {
    cfg, err := config.LoadWorkflowConfig(".sharkconfig.json")
    if err != nil {
        // Fall back to default workflow
        cfg = config.DefaultWorkflowConfig()
    }

    return &TaskRepository{
        db:       database,
        workflow: cfg,
    }
}

// UpdateStatus validates transition before updating
func (r *TaskRepository) UpdateStatus(ctx context.Context, taskKey, newStatus string, force bool) error {
    // Get current task
    task, err := r.GetByKey(ctx, taskKey)
    if err != nil {
        return err
    }

    // Validate transition (unless --force)
    if !force {
        if err := r.workflow.CanTransition(task.Status, newStatus); err != nil {
            return fmt.Errorf("invalid status transition: %w\nUse --force to override", err)
        }
    }

    // Update in transaction
    // ... existing update logic ...
}
```

### 6. CLI Command Changes

#### New Commands

**`shark workflow list`** - Show all statuses and valid transitions
```bash
$ shark workflow list
STATUS FLOW:
  _start_ → draft, ready_for_development

  draft → ready_for_refinement, cancelled, on_hold
  ready_for_refinement → in_refinement, cancelled, on_hold
  ...

  _complete_ → completed, cancelled
```

**`shark workflow validate`** - Check workflow configuration
```bash
$ shark workflow validate
✓ Status flow configuration is valid
✓ 14 statuses defined
✓ 2 start statuses
✓ 2 terminal statuses
✓ No orphaned statuses
✓ All statuses are reachable from start
```

**`shark workflow graph`** - Generate Mermaid diagram (optional)
```bash
$ shark workflow graph > workflow.mmd
# Generates Mermaid state diagram from config
```

#### Updated Commands

**`shark task create`** - Support `--status` flag with validation
```bash
$ shark task create --epic=E11 --feature=F01 "New task" --status=draft

# Validates that 'draft' is in _start_ array
```

**`shark task set-status`** - Generic status transition command
```bash
$ shark task set-status <task-key> <new-status> [--force] [--notes="..."]

# Examples:
$ shark task set-status T-E11-F01-001 in_development
$ shark task set-status T-E11-F01-001 completed --force  # Skip validation
```

**Existing convenience commands remain**:
```bash
$ shark task start <key>     # → in_development (validates from current status)
$ shark task complete <key>  # → ready_for_review (validates)
$ shark task approve <key>   # → completed (validates)
$ shark task block <key>     # → blocked (validates)
$ shark task unblock <key>   # → previous status (validates)
```

#### Agent-Targeted Queries

```bash
# Find tasks ready for specific agent types
$ shark task list --status=ready_for_refinement --agent=business-analyst
$ shark task list --status=ready_for_development --agent=developer
$ shark task list --status=ready_for_qa --agent=qa

# Find all tasks in a phase
$ shark task list --phase=development  # in_development, ready_for_development
$ shark task list --phase=qa           # in_qa, ready_for_qa
```

### 7. Migration Strategy

**Existing task statuses** (`todo`, `in_progress`, `ready_for_review`, `completed`):

**Option A: Automatic mapping** (on first run with new config)
```
todo             → ready_for_development
in_progress      → in_development
ready_for_review → ready_for_review
completed        → completed
blocked          → blocked
```

**Option B: Explicit migration command**
```bash
$ shark migrate workflow
Found 150 tasks with legacy statuses:
  - 45 tasks: todo → ready_for_development
  - 23 tasks: in_progress → in_development
  - 15 tasks: ready_for_review → ready_for_review
  - 67 tasks: completed → completed

Proceed? [y/N]: y
✓ Migrated 150 tasks
```

**Option C: No migration** (allow old statuses to coexist)
- Add old statuses to workflow config
- Gradually transition to new flow

**Recommendation**: Option B (explicit migration with confirmation)

## Implementation Plan

### Phase 1: Core Infrastructure (High Priority)
1. **Config schema**: Add `status_flow` to `.sharkconfig.json` schema
2. **Workflow package**: Create `internal/config/workflow.go` with validation logic
3. **Repository integration**: Add workflow validation to `TaskRepository.UpdateStatus()`
4. **Unit tests**: Test workflow validation, transitions, edge cases

### Phase 2: CLI Commands (High Priority)
1. **`shark workflow list`**: Display status flow from config
2. **`shark workflow validate`**: Check config validity
3. **`shark task set-status`**: Generic status transition command
4. **Update existing commands**: Add workflow validation to `start`, `complete`, `approve`
5. **Add `--force` flag**: Bypass validation for all status change commands

### Phase 3: Migration & Metadata (Medium Priority)
1. **`shark migrate workflow`**: Migrate legacy statuses to new flow
2. **Status metadata support**: Load and display color, description, phase
3. **Agent targeting**: `--agent` and `--phase` filters for `task list`
4. **CLI output enhancements**: Color-code statuses, show phase in table view

### Phase 4: Advanced Features (Optional)
1. **Workflow graph visualization**: Generate Mermaid diagrams
2. **Multiple workflow support**: Different flows for feature/bug/spike
3. **Workflow analytics**: Report on task flow through statuses
4. **API endpoints**: Expose workflow config via HTTP API

## Testing Strategy

### Unit Tests
- **Workflow validation**: Test valid/invalid transitions
- **Edge cases**: Empty flow, missing statuses, circular references
- **Migration logic**: Old → new status mapping

### Integration Tests
- **Full workflow**: Create task → move through all statuses → complete
- **Backward transitions**: QA → development, approval → QA
- **Force flag**: Override validation correctly
- **Config reload**: Changes to config are picked up

### Manual Testing
- **Multi-agent simulation**: Different agents query and update tasks
- **Invalid transitions**: Verify helpful error messages
- **Config errors**: Test with malformed workflow config

## Acceptance Criteria

### Must Have
- [x] Workflow defined in `.sharkconfig.json`
- [ ] Status transitions validated against config
- [ ] `--force` flag bypasses validation
- [ ] Backward transitions supported (e.g., QA → development)
- [ ] `shark workflow list` shows all valid transitions
- [ ] Existing commands respect new workflow
- [ ] Migration path for legacy statuses
- [ ] Comprehensive error messages for invalid transitions

### Should Have
- [ ] Status metadata (color, description, phase)
- [ ] Agent targeting (`--status`, `--phase` filters)
- [ ] `shark workflow validate` command
- [ ] Colored status output in CLI

### Nice to Have
- [ ] Workflow visualization (Mermaid diagrams)
- [ ] Multiple workflows per project
- [ ] Workflow analytics/reports

## Open Questions

1. **Default workflow**: Should we ship with the full 14-status workflow, or a simpler default?
2. **Config location**: `.sharkconfig.json` or separate `workflow.json`?
3. **Validation strictness**: Should invalid statuses in DB be errors or warnings?
4. **Transition hooks**: Future support for running scripts on status change?
5. **Audit trail**: Should task_history record workflow validation failures?

## References

- Current task status handling: `internal/repository/task_repository.go`
- Task history: `internal/repository/task_history_repository.go`
- Configuration: `internal/config/config.go`
- CLI commands: `internal/cli/commands/task.go`

## Appendix: Example Workflows

### Simple Workflow (Minimal)
```json
{
  "_start_": ["todo"],
  "_complete_": ["done"],
  "todo": ["in_progress", "done"],
  "in_progress": ["done", "todo"],
  "done": []
}
```

### Kanban Workflow
```json
{
  "_start_": ["backlog"],
  "_complete_": ["done"],
  "backlog": ["ready", "archived"],
  "ready": ["in_progress", "backlog"],
  "in_progress": ["review", "blocked"],
  "review": ["done", "in_progress"],
  "blocked": ["ready"],
  "done": [],
  "archived": []
}
```

### GitFlow-style Workflow
```json
{
  "_start_": ["draft"],
  "_complete_": ["merged", "rejected"],
  "draft": ["ready_for_dev"],
  "ready_for_dev": ["in_dev"],
  "in_dev": ["in_review", "blocked"],
  "in_review": ["approved", "changes_requested"],
  "changes_requested": ["in_dev"],
  "approved": ["merged"],
  "blocked": ["ready_for_dev"],
  "merged": [],
  "rejected": []
}
```
