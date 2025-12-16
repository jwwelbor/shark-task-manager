---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T016-update-repositories-return-domain-errors.md]
estimated_time: 3 hours
---

# Task: Update CLI Commands to Handle Domain Errors

## Goal

Update all CLI commands to check for specific domain error types and provide helpful, context-specific error messages to users instead of generic error output.

## Success Criteria

- [ ] All commands check for domain errors using `errors.Is()`
- [ ] "Not found" errors provide helpful suggestions
- [ ] Validation errors explain what's wrong and how to fix it
- [ ] Foreign key errors explain relationships
- [ ] Duplicate key errors suggest alternatives
- [ ] Generic errors still provide useful debug information
- [ ] All CLI commands tested and working
- [ ] Error messages are user-friendly

## Implementation Guidance

### Overview

Now that repositories return typed domain errors (T016), update CLI commands to check for specific error types and provide helpful, actionable error messages to users. This dramatically improves user experience.

### Key Requirements

- Use `errors.Is(err, domain.ErrTaskNotFound)` to check error types
- Provide context-specific help messages for each error type
- Include suggestions for next steps (e.g., "Use 'pm task list' to see available tasks")
- Keep error messages concise but informative
- Maintain consistent error message format across commands

Reference: [PRD - CLI Error Handling Example](../01-feature-prd.md#fr-3-domain-specific-errors)

### Files to Create/Modify

**CLI Command Files**:
- `internal/cli/commands/task/*.go` - Update error handling in all task commands
- `internal/cli/commands/epic/*.go` - Update error handling in all epic commands
- `internal/cli/commands/feature/*.go` - Update error handling in all feature commands
- Any other command files that call repositories

### Error Handling Pattern

**Before (generic error)**:
```go
task, err := taskRepo.GetByKey(ctx, taskKey)
if err != nil {
    return fmt.Errorf("failed to get task: %w", err)  // generic, unhelpful
}
```

**After (specific, helpful)**:
```go
task, err := taskRepo.GetByKey(ctx, taskKey)
if errors.Is(err, domain.ErrTaskNotFound) {
    fmt.Fprintf(os.Stderr, "Error: Task '%s' not found.\n", taskKey)
    fmt.Fprintf(os.Stderr, "Use 'pm task list' to see available tasks.\n")
    return nil
}
if err != nil {
    return fmt.Errorf("failed to get task: %w", err)
}
```

**Validation error example**:
```go
err := taskRepo.Create(ctx, task)
if errors.Is(err, domain.ErrInvalidTaskKey) {
    fmt.Fprintf(os.Stderr, "Error: Invalid task key format '%s'.\n", task.Key)
    fmt.Fprintf(os.Stderr, "Task keys must match pattern: E##-F##-T### (e.g., E01-F02-T003)\n")
    return nil
}
if errors.Is(err, domain.ErrDuplicateKey) {
    fmt.Fprintf(os.Stderr, "Error: Task with key '%s' already exists.\n", task.Key)
    fmt.Fprintf(os.Stderr, "Use 'pm task show %s' to view the existing task.\n", task.Key)
    return nil
}
```

Reference: [PRD - Error Usage Example](../01-feature-prd.md#fr-3-domain-specific-errors)

### Error Message Guidelines

**Not Found Errors**:
- State what wasn't found
- Suggest command to list available items
- Example: "Task 'E01-F01-T001' not found. Use 'pm task list' to see available tasks."

**Validation Errors**:
- Explain what's invalid
- Show correct format or constraints
- Example: "Invalid priority '15'. Priority must be between 1 and 10."

**Business Logic Errors**:
- Explain why operation failed
- Suggest valid alternative
- Example: "Cannot complete task E01-F01-T002. Dependency E01-F01-T001 must be completed first."

**Constraint Errors**:
- Explain the constraint violation
- Suggest how to resolve
- Example: "Cannot delete epic E01. It contains 3 features. Delete or move the features first."

### Integration Points

- **Domain Errors**: Check errors from `internal/domain/errors.go`
- **User Experience**: Error messages guide users to resolution
- **Consistency**: Similar errors have similar message formats
- **Debugging**: Generic errors still provide stack trace information

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Manual Testing**:
- Test each error scenario with CLI:
  - Try to get non-existent task: `pm task show INVALID-KEY`
  - Try to create duplicate task
  - Try invalid status transition
  - Try to delete epic with features
- Verify error messages are helpful
- Verify suggestions are actionable

**Error Message Review**:
- Error messages are user-friendly (not technical)
- Messages explain what went wrong
- Messages suggest next steps
- Format is consistent across commands

**Integration Tests**:
- Run full CLI test suite
- Verify all commands still work
- Verify error cases are handled properly

## Context & Resources

- **PRD**: [Domain-Specific Errors](../01-feature-prd.md#fr-3-domain-specific-errors)
- **PRD**: [CLI Error Example](../01-feature-prd.md#fr-3-domain-specific-errors)
- **Task Dependency**: [T016 - Repository Domain Errors](./T016-update-repositories-return-domain-errors.md)
- **Domain Errors**: `internal/domain/errors.go`
- **CLI Commands**: `internal/cli/commands/`
- **Go Errors**: [errors.Is() documentation](https://pkg.go.dev/errors#Is)

## Notes for Agent

- Use `errors.Is(err, domain.ErrXxx)` pattern for error checking
- Write to stderr for error messages: `fmt.Fprintf(os.Stderr, ...)`
- Include helpful suggestions in error messages
- Consider error message format:
  1. What went wrong (brief)
  2. Why it failed (if not obvious)
  3. What to do next (suggestion)
- Don't overdo it - keep messages concise
- This greatly improves user experience with better error messages
- Test each error path manually to verify messages are helpful
- Common patterns:
  - Not found → suggest list command
  - Duplicate → suggest show command
  - Validation → explain constraints
  - Business logic → explain requirement
