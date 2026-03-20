# Test Specifications

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 8 — Migration Readiness

## Current Test Coverage

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
