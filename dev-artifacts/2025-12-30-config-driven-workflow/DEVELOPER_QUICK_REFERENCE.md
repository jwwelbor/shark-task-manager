# Config-Driven Workflow - Developer Quick Reference

## For Developers Using Shark Task Manager

### What Changed?

Workflows are now **fully configurable** via `.sharkconfig.json`. You can define any custom status workflow without changing code.

---

## Quick Start: Define Your Workflow

Edit `.sharkconfig.json`:

```json
{
  "status_flow_version": "1.0",
  "status_flow": {
    "todo": ["in_progress", "blocked"],
    "in_progress": ["ready_for_review", "blocked"],
    "ready_for_review": ["completed", "in_progress"],
    "blocked": ["todo", "in_progress"],
    "completed": []
  },
  "special_statuses": {
    "_start_": ["todo"],
    "_complete_": ["completed"]
  }
}
```

**That's it!** No code changes needed.

---

## Using the New Validation API

### ✅ Recommended (Config-Driven)

```go
import "github.com/jwwelbor/shark-task-manager/internal/validation"

// Create validator from workflow config
workflow := config.GetWorkflowOrDefault(".sharkconfig.json")
validator := validation.NewStatusValidator(workflow)

// Check if status is valid
if validator.IsValidStatus("in_development") {
    // Status exists in workflow
}

// Check if transition is allowed
if validator.CanTransition("in_development", "ready_for_code_review") {
    // Transition is valid
}

// Get all valid statuses
statuses := validator.GetAllStatuses()
// ["todo", "in_progress", "ready_for_review", "completed", "blocked"]

// Get start statuses
startStatuses := validator.GetStartStatuses()
// ["todo"]

// Get allowed transitions from a status
transitions := validator.GetAllowedTransitions("in_progress")
// ["ready_for_review", "blocked"]
```

### ⚠️ Deprecated (Hardcoded)

```go
// DEPRECATED - Will be removed in future version
if task.Status == models.TaskStatusTodo {
    // ...
}

// Use this instead:
if task.Status == TaskStatus("todo") {
    // ...
}

// Or better yet:
if validator.IsStartStatus(string(task.Status)) {
    // ...
}
```

---

## Common Use Cases

### 1. Validate a Task Status

```go
import "github.com/jwwelbor/shark-task-manager/internal/models"

// Old way (still works but limited)
err := models.ValidateTaskStatus("in_development")

// New way (accepts any workflow status)
workflow := getWorkflow()
validator := validation.NewStatusValidator(workflow)
err := validator.ValidateStatus("in_development")
```

### 2. Check if Transition is Allowed

```go
// Old way (hardcoded in repository)
canTransition := repo.isValidTransition(fromStatus, toStatus)

// New way (config-driven)
validator := validation.NewStatusValidator(repo.GetWorkflow())
canTransition := validator.CanTransition(string(fromStatus), string(toStatus))
```

### 3. Get All Statuses for UI Dropdown

```go
workflow := config.GetWorkflowOrDefault(".sharkconfig.json")
validator := validation.NewStatusValidator(workflow)

// Get all statuses
allStatuses := validator.GetAllStatuses()

// Render in UI
for _, status := range allStatuses {
    fmt.Printf("<option value='%s'>%s</option>\n", status, status)
}
```

### 4. Determine if Task Can Be Started

```go
validator := validation.NewStatusValidator(workflow)

// Check if current status is a "start" status
if validator.IsStartStatus(string(task.Status)) {
    fmt.Println("This task is ready to be started")
}
```

### 5. Determine if Task is Complete

```go
validator := validation.NewStatusValidator(workflow)

// Check if current status is a "complete" status
if validator.IsCompleteStatus(string(task.Status)) {
    fmt.Println("This task is finished")
}
```

---

## Example: Custom 14-Status Workflow

```json
{
  "status_flow": {
    "draft": ["ready_for_refinement", "cancelled"],
    "ready_for_refinement": ["in_refinement", "cancelled"],
    "in_refinement": ["ready_for_development", "blocked"],
    "ready_for_development": ["in_development", "cancelled"],
    "in_development": ["ready_for_code_review", "blocked"],
    "ready_for_code_review": ["in_code_review"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development", "blocked"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed", "ready_for_qa"],
    "blocked": ["ready_for_development", "cancelled"],
    "completed": [],
    "cancelled": []
  },
  "status_metadata": {
    "draft": {
      "color": "gray",
      "description": "Initial draft, not yet refined",
      "phase": "planning",
      "agent_types": ["business-analyst"]
    },
    "in_development": {
      "color": "blue",
      "description": "Actively being coded",
      "phase": "development",
      "agent_types": ["developer", "backend", "frontend"]
    },
    "in_qa": {
      "color": "green",
      "description": "Being tested",
      "phase": "qa",
      "agent_types": ["qa", "test-engineer"]
    }
  },
  "special_statuses": {
    "_start_": ["draft", "ready_for_development"],
    "_complete_": ["completed", "cancelled"]
  }
}
```

**Result:** All 14 statuses work immediately. No code changes required!

---

## Migration Guide

### Migrating from Hardcoded Constants

**Before:**
```go
import "github.com/jwwelbor/shark-task-manager/internal/models"

if task.Status == models.TaskStatusTodo {
    // Start work
}

if task.Status == models.TaskStatusCompleted {
    // Task done
}
```

**After:**
```go
import (
    "github.com/jwwelbor/shark-task-manager/internal/validation"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

workflow := config.GetWorkflowOrDefault(".sharkconfig.json")
validator := validation.NewStatusValidator(workflow)

if validator.IsStartStatus(string(task.Status)) {
    // Start work
}

if validator.IsCompleteStatus(string(task.Status)) {
    // Task done
}
```

**Benefits:**
- Works with ANY workflow (not just 6 hardcoded statuses)
- Respects `.sharkconfig.json` definitions
- More maintainable and flexible

---

## Error Handling

### Old Error Messages (Hardcoded)
```
invalid task status: must be todo, in_progress, blocked, ready_for_review, completed, or archived
```

### New Error Messages (Helpful)
```
invalid task status "in_development": not found in default or extended workflow.
Ensure status is defined in .sharkconfig.json workflow
```

```
invalid transition from "in_qa" to "completed": allowed transitions from "in_qa" are [ready_for_approval, in_development, blocked]
```

---

## Testing Your Workflow

```go
import (
    "testing"
    "github.com/jwwelbor/shark-task-manager/internal/validation"
    "github.com/jwwelbor/shark-task-manager/internal/config"
)

func TestCustomWorkflow(t *testing.T) {
    workflow := &config.WorkflowConfig{
        StatusFlow: map[string][]string{
            "draft": {"ready_for_refinement"},
            "ready_for_refinement": {"in_refinement"},
            "in_refinement": {"ready_for_development"},
            // ... more statuses
        },
    }

    validator := validation.NewStatusValidator(workflow)

    // Test valid status
    if !validator.IsValidStatus("draft") {
        t.Error("draft should be valid")
    }

    // Test valid transition
    if !validator.CanTransition("draft", "ready_for_refinement") {
        t.Error("transition should be allowed")
    }

    // Test invalid transition
    if validator.CanTransition("draft", "completed") {
        t.Error("transition should not be allowed")
    }
}
```

---

## Best Practices

### ✅ DO:
- Define workflows in `.sharkconfig.json`
- Use `validation.StatusValidator` for all status checks
- Use `config.GetWorkflowOrDefault()` to load workflow
- Add helpful metadata (colors, descriptions, phases) to statuses
- Test your workflow with the validator

### ❌ DON'T:
- Use hardcoded `models.TaskStatus*` constants (they're deprecated)
- Hardcode status strings in new code
- Create custom validation logic (use the validator)
- Modify workflow config structure (follow the schema)

---

## API Reference

### `validation.StatusValidator`

```go
type StatusValidator struct { ... }

// Create new validator
func NewStatusValidator(workflow *WorkflowConfig) *StatusValidator

// Validate status exists
func (v *StatusValidator) ValidateStatus(status string) error

// Validate transition is allowed
func (v *StatusValidator) ValidateTransition(fromStatus, toStatus string) error

// Check if status is valid (bool)
func (v *StatusValidator) IsValidStatus(status string) bool

// Check if transition is allowed (bool)
func (v *StatusValidator) CanTransition(fromStatus, toStatus string) bool

// Get all defined statuses
func (v *StatusValidator) GetAllStatuses() []string

// Get start statuses from special_statuses._start_
func (v *StatusValidator) GetStartStatuses() []string

// Get complete statuses from special_statuses._complete_
func (v *StatusValidator) GetCompleteStatuses() []string

// Check if status is a start status
func (v *StatusValidator) IsStartStatus(status string) bool

// Check if status is a complete status
func (v *StatusValidator) IsCompleteStatus(status string) bool

// Get allowed transitions from a status
func (v *StatusValidator) GetAllowedTransitions(fromStatus string) []string
```

---

## Questions?

See:
- `/dev-artifacts/2025-12-30-config-driven-workflow/IMPLEMENTATION_SUMMARY.md` - Technical details
- `/dev-artifacts/2025-12-30-config-driven-workflow/COMPLETION_REPORT.md` - Full report
- `/internal/validation/workflow_validator_test.go` - Example usage
- `.sharkconfig.json` - Your workflow definition

**Summary:** Define your workflow in JSON, use the validator API, enjoy unlimited flexibility!
