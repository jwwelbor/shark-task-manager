---
status: created
feature: /docs/plan/E04-task-mgmt-cli-core/E04-F09-recommended-improvements
created: 2025-12-16
assigned_agent: api-developer
dependencies: [T002-update-task-repository-context.md, T003-update-epic-repository-context.md, T004-update-feature-repository-context.md, T005-update-taskhistory-repository-context.md]
estimated_time: 1 hour
---

# Task: Update HTTP Handlers to Use Request Context

## Goal

Update all HTTP handlers to extract and use the request's context (`r.Context()`) when calling repository methods, enabling proper request cancellation and timeout handling for API operations.

## Success Criteria

- [ ] All HTTP handlers use `r.Context()` instead of `context.Background()`
- [ ] Repository calls in handlers pass request context
- [ ] Request cancellation properly aborts database operations
- [ ] All API endpoints tested and working
- [ ] No breaking changes to API responses
- [ ] HTTP server tests pass

## Implementation Guidance

### Overview

Now that all repositories accept context (T002-T005), update the HTTP handlers to pass the request context instead of `context.Background()`. This enables automatic cancellation when clients disconnect or request timeouts occur.

### Key Requirements

- Extract request context using `ctx := r.Context()` at the start of each handler
- Pass `ctx` to all repository method calls
- Request cancellation propagates to database queries automatically
- No changes to response formats or status codes
- Maintain current error handling behavior

Reference: [PRD - Context Support Example](../01-feature-prd.md#fr-1-context-support)

### Files to Create/Modify

**Backend**:
- `cmd/server/main.go` - Update HTTP handler functions to use request context
- Any handler helper functions that call repositories

### Integration Points

- **All Repositories**: Handlers call TaskRepository, EpicRepository, FeatureRepository, TaskHistoryRepository
- **HTTP Middleware**: Context flows through middleware chain (if any exists)
- **Client Timeout**: If client sets timeout or cancels request, operations abort

Reference: [PRD - Affected Components](../01-feature-prd.md#affected-components)

## Validation Gates

**Linting & Type Checking**:
- Code passes `go vet`
- Code passes `golangci-lint run`
- No compilation errors

**Unit Tests**:
- HTTP handler tests pass (update to use context-aware mocks if needed)
- Handler response formats unchanged

**Integration Tests**:
- Start server and test all API endpoints with curl/Postman
- Verify task CRUD operations work
- Verify epic/feature operations work
- Verify task history operations work

**Manual Testing**:
- Start server: `make server` or `go run cmd/server/main.go`
- Test endpoints:
  - `curl http://localhost:8080/tasks`
  - `curl http://localhost:8080/epics`
  - `curl -X POST http://localhost:8080/tasks -d '...'`
- Verify all responses match expected format

## Context & Resources

- **PRD**: [Context Support Requirements](../01-feature-prd.md#fr-1-context-support)
- **PRD**: [HTTP Handler Example](../01-feature-prd.md#fr-1-context-support)
- **Task Dependencies**: T002, T003, T004, T005 (all repository updates must be complete)
- **Architecture**: [SYSTEM_DESIGN.md](../../../../architecture/SYSTEM_DESIGN.md)
- **Go Best Practices**: [Context in HTTP Handlers](../../../../architecture/GO_BEST_PRACTICES.md)

## Notes for Agent

- Request context is obtained via `r.Context()` from `*http.Request`
- Pattern: `ctx := r.Context()` at start of handler, then pass to all repository calls
- Request context is automatically cancelled when:
  - Client closes connection
  - Request timeout occurs
  - Server is shutting down
- No need to call `cancel()` for request context - it's managed by HTTP server
- This connects the HTTP request lifecycle to database operations
- Quick task - mainly find/replace `context.Background()` with `ctx` in handlers
