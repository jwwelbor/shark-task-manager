# Transition Validation Logic

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

This document defines the validation logic that ensures all status transitions conform to workflow configuration rules. Validation is performed at multiple layers for safety and clear error messaging.

**Design Principle**: Simple, fast validation with clear error messages that guide users to correct commands.

---

## Validation Layers

### Layer 1: Command-Level Validation (CLI)

**Location**: Command handlers (task_claim.go, task_finish.go, task_reject.go)

**Validates**:
- Required arguments present (task key, --reason for reject)
- Task key format valid
- Task exists in database

**Example**:
```go
func runTaskClaim(cmd *cobra.Command, args []string) error {
    if len(args) != 1 {
        return fmt.Errorf("task key required")
    }

    taskKey := NormalizeTaskKey(args[0])
    if !isValidTaskKeyFormat(taskKey) {
        return fmt.Errorf("invalid task key format: %s", taskKey)
    }

    // ... continue
}
```

### Layer 2: Workflow Validation (Workflow Service)

**Location**: `internal/workflow/service.go`

**Validates**:
- Status exists in workflow
- Transition allowed by status_flow config
- Command-specific patterns (claim requires ready_for_*, etc.)

**Example**:
```go
func (s *Service) ValidateTransition(from, to, cmdType string) error {
    // Status existence
    if !s.IsValidStatus(from) {
        return fmt.Errorf("unknown status: %s", from)
    }
    if !s.IsValidStatus(to) {
        return fmt.Errorf("unknown status: %s", to)
    }

    // Transition allowed
    if !s.IsValidTransition(from, to) {
        valid := s.GetValidTransitions(from)
        return fmt.Errorf(
            "invalid transition %s → %s. Valid: %v",
            from, to, valid,
        )
    }

    // Command-specific rules
    return s.validateCommandPattern(from, to, cmdType)
}
```

### Layer 3: Database Constraints

**Location**: SQLite schema

**Validates**:
- Foreign key constraints
- NOT NULL constraints
- Transaction atomicity

**Schema**:
```sql
CREATE TABLE tasks (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL CHECK(status IN ('draft', 'ready_for_development', ...)),
    ...
);
```

---

## Validation Rules by Command

### Claim Command Validation

**Prerequisites**:
```go
// 1. Task status must be ready_for_*
if !strings.HasPrefix(task.Status, "ready_for_") {
    return ValidationError{
        Message: fmt.Sprintf("Cannot claim task in '%s' status", task.Status),
        Hint: "Task must be in 'ready_for_*' status to claim",
        CurrentStatus: task.Status,
        ValidCommands: []string{"finish", "reject", "block"},
    }
}

// 2. Target status must be in_*
targetStatus := getClaimStatus(task.Status) // "ready_for_X" → "in_X"
if !strings.HasPrefix(targetStatus, "in_") {
    return ConfigError{
        Message: "Invalid workflow: claim target must be 'in_*' status",
        CurrentStatus: task.Status,
        TargetStatus: targetStatus,
    }
}

// 3. Transition must be allowed
if !workflow.IsValidTransition(task.Status, targetStatus) {
    return TransitionError{
        From: task.Status,
        To: targetStatus,
        ValidTransitions: workflow.GetValidTransitions(task.Status),
    }
}

// 4. Agent type check (warning only)
if !workflow.CanAgentClaimStatus(targetStatus, agentType) {
    log.Warn("Agent type mismatch: %s claiming %s (expected: %v)",
        agentType, targetStatus, workflow.GetAgentTypesForStatus(targetStatus))
}
```

**Error Messages**:

```
Error: Cannot claim task in 'in_development' status
Task is already claimed by backend (started 2026-01-11 09:00)

Suggested actions:
  • shark task finish T-E07-F20-001 (if work complete)
  • shark task reject T-E07-F20-001 --reason="..." (if not ready)
  • shark task block T-E07-F20-001 --reason="..." (if blocked)

Exit Code: 3
```

### Finish Command Validation

**Prerequisites**:
```go
// 1. Task status must be in_*
if !strings.HasPrefix(task.Status, "in_") {
    return ValidationError{
        Message: fmt.Sprintf("Cannot finish task in '%s' status", task.Status),
        Hint: "Task must be 'in progress' (in_*) to finish",
        CurrentStatus: task.Status,
        ValidCommands: []string{"claim"},
    }
}

// 2. Must have valid next status
nextStatus, err := workflow.GetNextPhaseStatus(task.Status)
if err != nil {
    return ConfigError{
        Message: fmt.Sprintf("Workflow error: no valid next status from %s", task.Status),
        Hint: "Check .sharkconfig.json - every in_* status needs outgoing transition",
        CurrentStatus: task.Status,
    }
}

// 3. Transition must be allowed
if !workflow.IsValidTransition(task.Status, nextStatus) {
    return TransitionError{
        From: task.Status,
        To: nextStatus,
        ValidTransitions: workflow.GetValidTransitions(task.Status),
    }
}
```

**Error Messages**:

```
Error: Cannot finish task in 'ready_for_development' status
Task has not been claimed yet

Suggested action:
  • shark task claim T-E07-F20-001 --agent=<type> (to start work)

Exit Code: 3
```

```
Error: Workflow configuration error
Status 'in_qa' has no outgoing transitions defined

This is a configuration error. Every 'in_*' status must have at least
one valid next status in .sharkconfig.json

Fix: Add transitions to status_flow["in_qa"], for example:
  "in_qa": ["ready_for_approval", "in_development", "blocked"]

Exit Code: 2
```

### Reject Command Validation

**Prerequisites**:
```go
// 1. Reason required
if reason == "" {
    return ArgumentError{
        Message: "--reason is required when rejecting a task",
        Hint: "Explain why the task cannot proceed",
        Example: `shark task reject T-E07-F20-001 --reason="Missing acceptance criteria"`,
    }
}

// 2. If --to specified, validate it exists
if toStatus != "" {
    if !workflow.IsValidStatus(toStatus) {
        return ValidationError{
            Message: fmt.Sprintf("Unknown status: %s", toStatus),
            Hint: "Use 'shark workflow list' to see valid statuses",
        }
    }

    // Validate transition allowed
    if !workflow.IsValidTransition(task.Status, toStatus) {
        return TransitionError{
            From: task.Status,
            To: toStatus,
            ValidTransitions: workflow.GetValidTransitions(task.Status),
        }
    }

    // Validate backward direction
    if !isBackwardTransition(workflow, task.Status, toStatus) {
        return ValidationError{
            Message: "Reject must move task to earlier workflow phase",
            From: task.Status,
            FromPhase: workflow.GetPhaseFromStatus(task.Status),
            To: toStatus,
            ToPhase: workflow.GetPhaseFromStatus(toStatus),
            Hint: "Use 'shark task finish' for forward transitions",
        }
    }
}

// 3. If --to not specified, auto-determine
if toStatus == "" {
    toStatus, err = workflow.GetPreviousPhaseStatus(task.Status)
    if err != nil {
        return ConfigError{
            Message: "No backward transition available",
            CurrentStatus: task.Status,
            Hint: "Specify target status with --to=<status>",
            ValidTransitions: workflow.GetValidTransitions(task.Status),
        }
    }
}
```

**Backward Direction Check**:
```go
func isBackwardTransition(w *workflow.Service, from, to string) bool {
    fromPhase := w.GetPhaseFromStatus(from)
    toPhase := w.GetPhaseFromStatus(to)

    phaseOrder := map[string]int{
        "planning": 0,
        "development": 1,
        "review": 2,
        "qa": 3,
        "approval": 4,
        "done": 5,
    }

    fromOrder := phaseOrder[fromPhase]
    toOrder := phaseOrder[toPhase]

    // Backward means earlier phase OR same phase (rework within phase)
    return toOrder <= fromOrder
}
```

**Error Messages**:

```
Error: --reason is required when rejecting a task
Provide an explanation of why the task cannot proceed

Example:
  shark task reject T-E07-F20-001 --reason="Acceptance criteria incomplete"

Exit Code: 1
```

```
Error: Cannot reject from 'in_development' to 'ready_for_qa'
Rejection must move task to earlier workflow phase

Current: in_development (development phase)
Target: ready_for_qa (qa phase)

Valid backward transitions from in_development:
  • in_refinement (planning phase)
  • ready_for_refinement (planning phase)

Exit Code: 3
```

---

## Error Types and Exit Codes

### ArgumentError (Exit Code 1)

**Triggers**:
- Missing required argument
- Invalid argument format
- Invalid flag combination

**Handling**: Show usage help + example

### ConfigError (Exit Code 2)

**Triggers**:
- Workflow configuration invalid
- Status not defined in workflow
- Missing required workflow sections

**Handling**: Point to configuration file, suggest fix

### ValidationError (Exit Code 3)

**Triggers**:
- Invalid status transition
- Task in wrong state for command
- Workflow rules violated

**Handling**: Explain rule, suggest alternative commands

### DatabaseError (Exit Code 2)

**Triggers**:
- Transaction failure
- Constraint violation
- Connection error

**Handling**: Log technical details, rollback transaction

---

## Validation Helper Functions

### Task Key Validation

```go
// NormalizeTaskKey converts various formats to canonical form
// Supports: T-E##-F##-###, E##-F##-###, T-E##-F##-###-slug
func NormalizeTaskKey(input string) (string, error) {
    input = strings.ToUpper(strings.TrimSpace(input))

    // Pattern 1: T-E##-F##-### (canonical)
    if matched, _ := regexp.MatchString(`^T-E\d{2}-F\d{2}-\d{3}$`, input); matched {
        return input, nil
    }

    // Pattern 2: E##-F##-### (add T- prefix)
    if matched, _ := regexp.MatchString(`^E\d{2}-F\d{2}-\d{3}$`, input); matched {
        return "T-" + input, nil
    }

    // Pattern 3: Slugged key (validate and return)
    if matched, _ := regexp.MatchString(`^T-E\d{2}-F\d{2}-\d{3}-.+$`, input); matched {
        return input, nil
    }

    return "", fmt.Errorf("invalid task key format: %s", input)
}
```

### Status Pattern Checks

```go
// IsReadyForPhase checks if status matches ready_for_* pattern
func IsReadyForPhase(status string) bool {
    return strings.HasPrefix(status, "ready_for_")
}

// IsInPhase checks if status matches in_* pattern
func IsInPhase(status string) bool {
    return strings.HasPrefix(status, "in_")
}

// ExtractPhase extracts phase name from status
// "ready_for_development" → "development"
// "in_code_review" → "code_review"
func ExtractPhase(status string) string {
    if strings.HasPrefix(status, "ready_for_") {
        return strings.TrimPrefix(status, "ready_for_")
    }
    if strings.HasPrefix(status, "in_") {
        return strings.TrimPrefix(status, "in_")
    }
    return status
}
```

---

## Validation Error Struct

```go
// ValidationError provides structured error information
type ValidationError struct {
    Type            string   // "argument", "config", "transition", "database"
    Message         string   // Human-readable error
    Hint            string   // Suggestion for how to fix
    CurrentStatus   string   // Current task status (if applicable)
    TargetStatus    string   // Attempted target status (if applicable)
    ValidTransitions []string // Valid next statuses (if applicable)
    ValidCommands   []string // Alternative commands (if applicable)
    ExitCode        int      // Process exit code
}

func (e ValidationError) Error() string {
    var b strings.Builder

    fmt.Fprintf(&b, "Error: %s\n", e.Message)

    if e.Hint != "" {
        fmt.Fprintf(&b, "%s\n", e.Hint)
    }

    if e.CurrentStatus != "" {
        fmt.Fprintf(&b, "\nCurrent status: %s\n", e.CurrentStatus)
    }

    if len(e.ValidTransitions) > 0 {
        fmt.Fprintf(&b, "\nValid next statuses: %v\n", e.ValidTransitions)
    }

    if len(e.ValidCommands) > 0 {
        fmt.Fprintf(&b, "\nSuggested commands:\n")
        for _, cmd := range e.ValidCommands {
            fmt.Fprintf(&b, "  • shark task %s\n", cmd)
        }
    }

    return b.String()
}
```

---

## Force Override (--force Flag)

### What --force Bypasses

**Workflow validation only**:
- Status transition validation
- Command pattern checks (claim requires ready_for_*, etc.)
- Backward direction enforcement for reject

**Does NOT bypass**:
- Required argument validation (e.g., --reason for reject)
- Task existence check
- Database constraints
- Transaction atomicity

### Usage

```bash
# Force claim from any status
shark task claim T-E07-F20-001 --force

# Force finish to specific status (bypass workflow)
shark task finish T-E07-F20-001 --force

# Force forward reject (normally not allowed)
shark task reject T-E07-F20-001 --reason="Skip QA" --to=completed --force
```

### Implementation

```go
func validateTransition(task *Task, targetStatus string, force bool) error {
    if force {
        log.Warn("--force flag used, bypassing workflow validation")
        return nil // Skip validation
    }

    // Normal validation
    return workflow.ValidateTransition(task.Status, targetStatus, cmdType)
}
```

### Warning Message

```
Warning: Task T-E07-F20-001 force-claimed from 'completed' status
Workflow validation bypassed (--force used)
Status: completed → in_development

This is an administrative override. Ensure this transition is intentional.
```

---

## Testing Strategy

### Unit Tests

```go
func TestValidateTransition_Claim(t *testing.T) {
    tests := []struct{
        name        string
        currentStatus string
        agentType   string
        expectError bool
        errorType   string
    }{
        {"valid claim", "ready_for_development", "backend", false, ""},
        {"already claimed", "in_development", "backend", true, "ValidationError"},
        {"terminal status", "completed", "backend", true, "ValidationError"},
        {"invalid status", "unknown", "backend", true, "ConfigError"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateClaimTransition(tt.currentStatus, tt.agentType)
            // ... assertions
        })
    }
}
```

### Integration Tests

Test full command execution with various invalid states:
- Task not found
- Invalid workflow config
- Wrong status for command
- Multiple validation failures

---

## Performance Considerations

### Validation Overhead

**Target**: < 10ms per validation

**Optimization**:
- Cache workflow config (50ms first load, < 1ms cached)
- Precompile regex patterns
- Use string prefix checks (fast)
- Defer expensive checks (agent type validation is warning only)

---

## References

- [System Architecture](./system-architecture.md)
- [Workflow Config Reader](./workflow-config-reader.md)
- [Command Specifications](./command-specifications.md)
- [Epic Requirements](../requirements.md) - REQ-F-005, REQ-NF-005
