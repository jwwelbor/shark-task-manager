# Test Specifications

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 8 — Migration Readiness

## Current Test Coverage

| Layer | Test Type | Database | Count (files) | Status |
|-------|-----------|----------|---------------|--------|
| Repository | Integration | Real SQLite | ~40 | Comprehensive |
| Services | Unit | Mocked repos | ~30 | Growing |
| CLI Commands | Unit | Mocked services | ~15 | Partial (legacy uses real DB) |
| Models | Unit | None | ~10 | Comprehensive |
| Config | Unit | None | ~10 | Comprehensive |
| Workflow | Unit | None | ~5 | Comprehensive |

## Test Requirements Per Migration Phase

### Service Layer Migration Tests

| Service | Required Tests | Priority |
|---------|---------------|----------|
| TaskService | CRUD, status transitions, dependency validation, blocking, filtering | High |
| FeatureService | CRUD, completion logic, progress, cascade | High |
| EpicService | CRUD, rollups, impediments, cascade | High |
| EntityService | Polymorphic transitions, history recording | High |
| NoteService | Add, list, filter by type | Medium |
| ContextService | Get/set/clear context fields | Medium |

### CLI Migration Tests

| Command Category | Required Tests | Approach |
|-----------------|---------------|----------|
| Get commands | Arg parsing, JSON output, field extraction | Mock service |
| List commands | Filter parsing, table output, JSON output | Mock service |
| Status commands | Transition flow, error mapping, exit codes | Mock service |
| Create commands | Input parsing, success output, error handling | Mock service |

### HTTP API Tests

| Category | Required Tests | Approach |
|----------|---------------|----------|
| Handlers | Request parsing, response format, status codes | Mock service, httptest |
| Error mapping | Service errors → HTTP status codes | Table-driven |
| Middleware | Auth, logging, request ID | Integration |

## Test Data Requirements

| Data Type | Source | Format |
|-----------|--------|--------|
| Entity fixtures | `test-fixtures/` | JSON, YAML |
| Mock repositories | `services/mocks_test.go` | Go function fields |
| Test database | `internal/test/testdb.go` | In-memory SQLite |
| Workflow configs | `.sharkworkflow-short.json` | JSON |

## Recommended Test Frameworks

| Framework | Version | Use For |
|-----------|---------|---------|
| testify/assert | v1.11.1 | Assertions (already in use) |
| testify/require | v1.11.1 | Fatal assertions |
| net/http/httptest | stdlib | HTTP handler tests |
| testing/fstest | stdlib | Filesystem mock (if needed) |

See also: [Component Order](component-order.md) | [Validation Criteria](validation-criteria.md)
