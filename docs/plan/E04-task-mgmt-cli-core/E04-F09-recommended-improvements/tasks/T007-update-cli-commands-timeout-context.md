---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T002-update-task-repository-context.md, T003-update-epic-repository-context.md, T004-update-feature-repository-context.md, T005-update-taskhistory-repository-context.md]
estimated_time: 2 hours
---

# Task: Update CLI Commands to Use Context with Timeout

## Goal

Update all CLI commands to create and use timeout contexts when calling repository methods, enabling automatic cancellation of long-running operations and improving CLI responsiveness.

## Success Criteria

- [ ] All CLI commands create timeout context (e.g., 30 seconds)
- [ ] Repository calls in commands pass timeout context
- [ ] Long-running operations timeout gracefully with helpful error messages
- [ ] All CLI commands tested and working
- [ ] No breaking changes to command outputs
- [ ] CLI tests pass

## Implementation Guidance

### Overview

Now that all repositories accept context (T002-T005), update the CLI commands to pass timeout contexts instead of `context.Background()`. This prevents CLI commands from hanging indefinitely on slow database operations.

### Key Requirements

- Create timeout context at the start of each command's `RunE` function
- Use reasonable timeout (30 seconds for most operations, 5 minutes for bulk operations)
- Pass timeout context to all repository method calls
- Handle `context.DeadlineExceeded` error with user-friendly message
- Call `defer cancel()` to clean up context resources

Reference: [PRD - Context Support Example](../01-feature-prd.md#fr-1-context-support)

### Files to Create/Modify

**CLI Commands**:
- `internal/cli/commands/task/*.go` - Update all task commands
- `internal/cli/commands/epic/*.go` - Update all epic commands
- `internal/cli/commands/feature/*.go` - Update all feature commands
- Any other command files that call repositories

### Integration Points

- **All Repositories**: Commands call TaskRepository, EpicRepository, FeatureRepository, TaskHistoryRepository
- **CLI Framework**: Works with Cobra command structure
- **User Experience**: Commands timeout gracefully, don't hang

Reference: [PRD - Affected Components](../01-feature-prd.md#affected-components)

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors
- All `defer cancel()` calls present

**Unit Tests**:
- CLI command tests pass (update to use context-aware mocks)
- Command output formats unchanged

**Integration Tests**:
- Test all commands with real database:
  - `shark task list`
  - `shark task create`
  - `shark task start <key>`
  - `shark epic list`
  - `shark feature list`
- Verify all commands complete successfully
- Verify timeout behavior (if possible to test)

**Manual Testing**:
- Run common CLI workflows:
  - Create task, start task, complete task
  - List tasks with filters
  - Create epic and feature
- Verify no regressions in output format
- Verify performance is acceptable

## Context & Resources

- **PRD**: [Context Support Requirements](../01-feature-prd.md#fr-1-context-support)
- **PRD**: [CLI Context Example](../01-feature-prd.md#fr-1-context-support)
- **Task Dependencies**: T002, T003, T004, T005 (all repository updates must be complete)
- **Architecture**: [SYSTEM_DESIGN.md](../../../../architecture/SYSTEM_DESIGN.md)
- **Go Best Practices**: [Context with Timeout](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Pattern for CLI commands:
  ```go
  func runCommand(cmd *cobra.Command, args []string) error {
      ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
      defer cancel()

      // Use ctx for all repository calls
      task, err := taskRepo.GetByKey(ctx, taskKey)
      if err == context.DeadlineExceeded {
          return fmt.Errorf("operation timed out after 30 seconds")
      }
      // ... rest of command logic
  }
  ```
- Choose appropriate timeouts:
  - Simple queries (get by ID): 5 seconds
  - List operations: 30 seconds
  - Bulk operations: 5 minutes
- Always call `defer cancel()` immediately after creating context
- Handle `context.DeadlineExceeded` error with user-friendly message
- This task touches many files but changes are mechanical and consistent
