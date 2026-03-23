# Test Specifications

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
=======
> Generated: 2026-03-20
>>>>>>> Stashed changes
> Phase: 8 — Migration Readiness

## Current Test Coverage

<<<<<<< Updated upstream
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
=======
| Category | Files | LOC | Approach |
|----------|-------|-----|----------|
| Repository tests | 50 | ~50K | Real database + cleanup |
| Service tests | 22 | ~15K | Mocked repositories |
| CLI command tests | 100 | ~40K | Mocked services/repos |
| Config tests | 14 | ~8K | Unit tests |
| Model tests | 6 | ~3K | Pure logic tests |
| Status tests | 9 | ~5K | Mock configs |
| Other tests | ~87 | ~30K | Various |

**Test-to-production ratio**: 1.87:1 (excellent)

## Test Strategy by Migration Phase

### Phase 2: Service Layer Migration

**Required test types**:
- **Service unit tests** — Mock all repository dependencies, test business logic
- **Pattern**: Function field mocks (no mocking framework)

**Per-service test requirements**:

| Service | Tests Needed | Priority |
|---------|-------------|----------|
| TaskService | Status transitions, dependency validation, create/update | High |
| FeatureService | Progress rollup, status cascading | High |
| EpicService | Feature rollups, impediment tracking | High |
| EntityService | Transition validation, rejection notes | High |
| NoteService | CRUD, entity type routing | Medium |
| ContextService | Context aggregation | Medium |
| ResumeService | Full context assembly | Medium |

### Phase 3: CLI Command Migration

**Required test types**:
- **Command tests** — Mock service layer, verify argument parsing and output formatting
- **Integration tests** — End-to-end via CLI (existing e2e tests)

**Key test scenarios**:
- JSON output format correctness
- Field extraction (`--field` flag)
- Error propagation and exit codes
- Argument parsing (positional + flag syntax)

### Phase 4: HTTP API

**Required test types**:
- **Handler tests** — Mock service layer, use `httptest.NewRecorder()`
- **Integration tests** — Full server with test database

## Test Data Requirements

| Type | Source | Notes |
|------|--------|-------|
| Unit tests | Inline test data | No external fixtures |
| Repository tests | `test.GetTestDB()` + `test.SeedTestData()` | Shared test DB |
| CLI tests | Mock data in test functions | No DB needed |
| E2E tests | `test/e2e/` shell scripts | Full binary execution |

## Recommended Test Frameworks

Already in use — no changes needed:
- **testify** (v1.11.1) — Assertions and require
- **testing** (stdlib) — Test runner, subtests, table-driven tests
- **httptest** (stdlib) — HTTP handler testing

## Test Quality Gates

Existing mandatory gate (enforced by CI):
```bash
make fmt    # Format code
make lint   # Static analysis
make test   # Full test suite (sequential, -p=1)
```

All three must pass before any change is considered complete.

---

See also: [Component Order](component-order.md) | [Validation Criteria](validation-criteria.md) | [Testing Architecture](../.claude/rules/testing/architecture.md)
>>>>>>> Stashed changes
