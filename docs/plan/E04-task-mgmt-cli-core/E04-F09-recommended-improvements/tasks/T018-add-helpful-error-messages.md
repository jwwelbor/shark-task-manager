---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T017-update-cli-commands-handle-domain-errors.md]
estimated_time: 2 hours
---

# Task: Add Helpful Error Messages for Common Failures

## Goal

Enhance CLI error handling by adding helpful error messages for common user mistakes and failure scenarios, improving the overall user experience and reducing support requests.

## Success Criteria

- [ ] Common failure scenarios identified and documented
- [ ] Helpful error messages added for each scenario
- [ ] Error messages include examples where appropriate
- [ ] Suggestions are actionable and correct
- [ ] Error output is consistent and well-formatted
- [ ] All error scenarios tested manually
- [ ] User documentation updated (if applicable)

## Implementation Guidance

### Overview

Build on the domain error handling from T017 by adding enhanced error messages for common user mistakes and edge cases. Focus on making the CLI forgiving and helpful, especially for new users.

### Key Requirements

- Identify common failure scenarios through code review and user workflows
- Add helpful, actionable error messages for each scenario
- Include examples in error messages where helpful
- Suggest correct command syntax or alternatives
- Handle edge cases gracefully

Reference: [PRD - Testing Strategy](../01-feature-prd.md#testing-strategy)

### Common Failure Scenarios to Handle

**Invalid Task Key Format**:
```go
if errors.Is(err, domain.ErrInvalidTaskKey) {
    fmt.Fprintf(os.Stderr, "Error: Invalid task key format '%s'\n", key)
    fmt.Fprintf(os.Stderr, "Expected format: E##-F##-T### (e.g., E01-F02-T003)\n")
    fmt.Fprintf(os.Stderr, "  E## = Epic number (e.g., E01)\n")
    fmt.Fprintf(os.Stderr, "  F## = Feature number (e.g., F02)\n")
    fmt.Fprintf(os.Stderr, "  T### = Task number (e.g., T003)\n")
    return nil
}
```

**Status Transition Errors**:
```go
if errors.Is(err, domain.ErrInvalidStatus) {
    fmt.Fprintf(os.Stderr, "Error: Cannot transition from '%s' to '%s'\n", currentStatus, newStatus)
    fmt.Fprintf(os.Stderr, "Valid transitions from '%s': %s\n", currentStatus, strings.Join(validTransitions, ", "))
    return nil
}
```

**Dependency Not Met**:
```go
if errors.Is(err, domain.ErrDependencyNotMet) {
    fmt.Fprintf(os.Stderr, "Error: Cannot complete task. Dependency not satisfied.\n")
    fmt.Fprintf(os.Stderr, "Task '%s' depends on: %s\n", task.Key, task.Dependencies)
    fmt.Fprintf(os.Stderr, "Use 'pm task show %s' to check dependency status\n", dependencyKey)
    return nil
}
```

**Empty or Invalid Input**:
```go
if errors.Is(err, domain.ErrEmptyTitle) {
    fmt.Fprintf(os.Stderr, "Error: Task title cannot be empty\n")
    fmt.Fprintf(os.Stderr, "Example: pm task create E01-F01-T001 'Implement user login'\n")
    return nil
}
```

**Foreign Key Violations**:
```go
if errors.Is(err, domain.ErrForeignKey) {
    fmt.Fprintf(os.Stderr, "Error: Cannot delete epic '%s'. It contains active features/tasks.\n", epicKey)
    fmt.Fprintf(os.Stderr, "To delete this epic:\n")
    fmt.Fprintf(os.Stderr, "  1. List features: pm feature list --epic=%s\n", epicKey)
    fmt.Fprintf(os.Stderr, "  2. Complete or delete all features\n")
    fmt.Fprintf(os.Stderr, "  3. Try delete again\n")
    return nil
}
```

### Files to Create/Modify

**CLI Command Files**:
- `internal/cli/commands/task/*.go` - Add enhanced error messages
- `internal/cli/commands/epic/*.go` - Add enhanced error messages
- `internal/cli/commands/feature/*.go` - Add enhanced error messages
- `internal/cli/commands/errors.go` (optional) - Shared error formatting helpers

**Optional Helper**:
- Create helper functions for consistent error formatting
- Example: `func formatValidationError(field, value, expected string)`

### Error Message Formatting Guidelines

**Structure**:
1. **Error Statement**: Clear statement of what went wrong
2. **Context**: Why it's a problem (if not obvious)
3. **Solution**: What to do to fix it
4. **Example**: Show correct usage (if helpful)

**Tone**:
- Be helpful, not judgmental
- Use "cannot" instead of "failed to"
- Suggest solutions, don't just state problems
- Include examples for complex cases

**Formatting**:
```
Error: [Brief description of what went wrong]
[Optional: Additional context or explanation]
[Actionable suggestion or next steps]
[Optional: Example showing correct usage]
```

### Integration Points

- **Domain Errors**: Enhanced messages for all domain error types
- **CLI Help**: Error messages align with help text
- **User Workflows**: Messages guide users through common workflows
- **Documentation**: Error messages reference documented patterns

## Validation Gates

**Manual Testing**:
- Test each error scenario:
  - Invalid task key
  - Status transition errors
  - Missing dependencies
  - Empty/invalid input
  - Foreign key violations
  - Duplicate keys
- Verify error messages are helpful
- Verify suggestions are correct
- Verify examples are accurate

**User Experience Review**:
- Error messages are clear and understandable
- Suggestions are actionable
- Examples demonstrate correct usage
- Tone is helpful, not condescending

**Consistency Check**:
- Similar errors have similar format
- Terminology is consistent across commands
- Suggestions follow same pattern

**Integration Testing**:
- Run through common user workflows
- Verify error handling doesn't break normal operations
- Verify no regressions in command behavior

## Context & Resources

- **PRD**: [Domain-Specific Errors](../01-feature-prd.md#fr-3-domain-specific-errors)
- **Task Dependency**: [T017 - CLI Error Handling](./T017-update-cli-commands-handle-domain-errors.md)
- **Domain Errors**: `internal/domain/errors.go`
- **CLI Commands**: `internal/cli/commands/`
- **User Workflows**: Common task management workflows

## Notes for Agent

- Focus on most common user mistakes first
- Use actual examples from the codebase (real task keys, epic keys)
- Test error messages by triggering them manually
- Get feedback on error message clarity (if possible)
- This task polishes the user experience and completes Phase 3 (Domain Errors)
- Consider creating error message helper functions for consistency
- Don't over-explain - keep messages concise but helpful
- Example patterns:
  - Show valid formats when input is invalid
  - List valid options when choice is invalid
  - Show current state when operation not allowed
  - Suggest alternative commands when appropriate
- After this task, CLI should feel much more user-friendly and helpful
