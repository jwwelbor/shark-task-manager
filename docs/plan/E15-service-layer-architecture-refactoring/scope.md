# Scope Boundaries

**Epic**: [Service Layer Architecture Refactoring](./epic.md)

---

## In Scope

### Service Layer Creation

✅ **Create dedicated service layer in `internal/services/`**
- `TaskService`: All task-related business logic
- `FeatureService`: All feature-related business logic
- `EpicService`: All epic-related business logic
- `QueryService`: Cross-entity querying and filtering logic

✅ **Service interface design**
- Clearly defined method signatures with godoc documentation
- Constructor dependency injection (repositories passed in)
- Context-aware methods (`ctx context.Context` as first parameter)
- Error wrapping for observability

✅ **Integration with existing services**
- Leverage `workflow.Service` for status validation
- Leverage `status.CalculationService` for progress calculations
- Leverage `taskcreation.Creator` for task generation

---

### CLI Command Refactoring

✅ **Slim down all 47 CLI command files to <500 lines each**
- Extract business logic to service layer
- Keep only: argument parsing, service calls, output formatting
- Use dependency injection via `cli.GetTaskService()` pattern

✅ **Maintain CLI interface compatibility**
- Same command syntax (flags, arguments)
- Same output format (JSON structure, table columns)
- Same error messages
- Same exit codes (0, 1, 2, 3)

✅ **Update CLI tests to use service layer**
- Test structure improvements (unit tests for services, integration tests for commands)
- Mock repositories in service tests
- Verify CLI commands call correct service methods

---

### Repository Layer Cleanup

✅ **Remove business logic from repositories**
- Remove progress calculation (`FeatureRepository.CalculateProgress()`)
- Remove status derivation (`TaskRepository.GetStatusBreakdown()`)
- Remove health calculations (`EpicRepository.GetHealthStatus()`)

✅ **Make repositories pure data access**
- CRUD operations only (Create, Read, Update, Delete)
- Query operations returning raw data models
- No validation, no orchestration, no calculations

✅ **Maintain repository interfaces**
- Existing repository methods continue to work during migration
- Gradual deprecation of business-logic methods
- Repository tests remain integration tests with real database

---

### HTTP API Integration

✅ **Wire HTTP API to service layer**
- API endpoints call same service methods as CLI
- Achieve 100% feature parity (all CLI operations available via API)
- Same validation, same errors, same calculations

✅ **Add missing API endpoints**
- `/api/tasks/next?agent={type}` for agent-specific task querying
- `/api/features/{key}/progress` for progress calculations
- `/api/epics/{key}/rollup` for feature/task aggregations
- All other CLI commands without API equivalents

✅ **API documentation**
- Document all endpoints with CLI command equivalents
- Provide request/response examples
- List feature parity status

---

### Documentation

✅ **Update architecture documentation**
- `.claude/rules/architecture.md`: Reflect service layer architecture
- `CLAUDE.md`: Add service layer development guidance
- README.md: Add architecture diagram

✅ **Create migration guide**
- Guide for contributors: "Where does code belong now?"
- Service layer usage examples
- Testing patterns for service layer

✅ **API documentation**
- OpenAPI/Swagger spec for HTTP API
- CLI-to-API mapping table

---

## Out of Scope

### New Features or Capabilities

❌ **No new CLI commands**
- This epic refactors existing code, does not add new functionality
- New commands can be added in future epics leveraging the service layer
- Exception: Helper commands for debugging/testing the service layer itself

❌ **No new API capabilities beyond CLI parity**
- API gets what CLI already has, no more
- New API-specific features (GraphQL, webhooks, etc.) are future work

❌ **No UI or web interface**
- HTTP API is for programmatic access, not human-facing web UI
- Web dashboard is a separate epic

---

### Database or Schema Changes

❌ **No database schema modifications**
- Refactoring is code-only, database structure unchanged
- Exception: Minor schema cleanups (dropping unused columns) if low-risk

❌ **No migration scripts**
- Data migrations not required for architecture refactoring
- No changes to `.sharkconfig.json` schema

❌ **No performance optimizations**
- Performance must not degrade (NFR3), but active optimization is out of scope
- Exception: Performance improvements that naturally arise from cleaner architecture

---

### Testing Infrastructure

❌ **No new test frameworks**
- Use existing test utilities in `internal/test/`
- No introduction of new testing libraries (testify, gomock, etc.)
- Maintain current testing patterns (table-driven tests, subtests)

❌ **No test refactoring beyond service layer**
- CLI tests updated to call services, but not rewritten from scratch
- Repository tests remain as-is (integration tests)
- New service tests added, but existing test suite structure unchanged

---

### Workflow or Status Changes

❌ **No workflow system changes**
- Workflow configuration (`.sharkconfig.json`) unchanged
- Status transitions unchanged
- `workflow.Service` remains as-is (used by service layer, not modified)

❌ **No file operations changes**
- File creation, discovery, sync logic unchanged
- `fileops.EntityFileWriter` used as-is by service layer

---

### External Dependencies

❌ **No new external libraries**
- Use Go stdlib and existing dependencies only
- No ORM frameworks (GORM, ent, etc.)
- No DI frameworks (uber/dig, google/wire, etc.)
- No service mesh or RPC frameworks

❌ **No infrastructure changes**
- SQLite database unchanged (no PostgreSQL, no Turso changes)
- No Docker, no Kubernetes, no deployment changes
- No CI/CD pipeline changes (beyond test execution)

---

## Edge Cases and Clarifications

### Partial Service Implementations

**Question**: What about existing services like `workflow.Service` and `status.CalculationService`?

**Answer**:
- ✅ **In scope**: Integrate them into the service layer architecture
- ✅ **In scope**: Use them as dependencies for Task/Feature/Epic services
- ❌ **Out of scope**: Rewrite or refactor them (they already work well)

### CLI Global State

**Question**: What about global state in `internal/cli/` (GlobalConfig, database singleton)?

**Answer**:
- ✅ **In scope**: Services accept repositories via dependency injection
- ✅ **In scope**: CLI commands wire services using global helpers (`cli.GetTaskService()`)
- ❌ **Out of scope**: Eliminating global state in CLI layer (not required for service layer)

### Incremental Migration

**Question**: Can commands use both old patterns and new service layer during migration?

**Answer**:
- ✅ **In scope**: Yes, intermediate states are valid (gradual migration)
- ✅ **In scope**: Commands can call services for some operations, repositories for others
- ✅ **In scope**: Main branch must always be deployable during refactoring

### Error Handling

**Question**: Should we standardize error types across service layer?

**Answer**:
- ✅ **In scope**: Define common error types (`NotFoundError`, `ValidationError`, etc.)
- ✅ **In scope**: Services return wrapped errors with context
- ❌ **Out of scope**: Global error handling framework or error codes system

---

## Boundary Enforcement

### Code Review Checklist

When reviewing PRs for this epic, enforce these boundaries:

**Must Have**:
- [ ] Business logic moved to service layer
- [ ] CLI command file <500 lines (excluding tests)
- [ ] Service method has godoc comment
- [ ] Service method has unit test with mocks
- [ ] Existing tests pass (zero regressions)

**Must Not Have**:
- [ ] New external dependencies
- [ ] Database schema changes (unless explicitly approved)
- [ ] New CLI commands (unless service-layer helpers)
- [ ] Breaking changes to CLI interface
- [ ] Performance degradation >10%

---

## Future Considerations

These items are explicitly deferred to future epics:

### Epic E16: API-First Development Model
- GraphQL API layer on top of service layer
- Webhook system for event notifications
- API rate limiting and authentication
- OpenAPI/Swagger auto-generation

### Epic E17: Service Layer Observability
- Structured logging in service methods
- Distributed tracing support
- Metrics collection (Prometheus)
- Service health endpoints

### Epic E18: Advanced Service Patterns
- Service layer caching (Redis)
- Event sourcing for audit trail
- CQRS pattern for read-heavy operations
- Background job processing (async tasks)

### Configurable File Naming (Mentioned in Original Epic)
- Epic/feature filename configuration (`epic_filename`, `feature_filename` in `.sharkconfig.json`)
- This is trivially configurable (2-3 references each)
- **Deferred decision**: Wait for user demand before implementing
- **If implemented**: Would be done in F06 (Slim Down CLI Commands) when init/config is touched
- **Estimated effort**: 2-4 hours (low complexity, low priority)

---

## Scope Change Process

If scope changes are proposed during epic execution:

1. **Evaluate impact** on timeline and complexity
2. **Update this document** with rationale for inclusion/exclusion
3. **Notify stakeholders** (maintainers, contributors)
4. **Adjust success metrics** if scope significantly changes

**Examples of acceptable scope additions**:
- Discover additional duplicated pattern worth centralizing
- Find critical bug that must be fixed during refactoring
- Identify missing service method required for API parity

**Examples of unacceptable scope additions**:
- "While we're refactoring, let's add GraphQL"
- "Let's optimize database queries during this epic"
- "We should add a web UI for the service layer"

---

*See also*: [Requirements](./requirements.md), [Success Metrics](./success-metrics.md)
