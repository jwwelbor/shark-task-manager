# Maintenance Burden

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 6 — Technical Debt Assessment

## High-Cost Maintenance Areas

### TD-02: Fat Controller Pattern (Medium)

- **Location**: `internal/cli/commands/` (various legacy command files)
- **Issue**: Some older commands contain business logic, direct repository calls, and workflow validation instead of delegating to services
- **Impact**: Duplicated logic, harder to test (requires DB), inconsistent behavior between CLI and HTTP API
- **Cost**: Each new feature requires changes in both command and service layers
- **Status**: Active migration underway (Epic E15)
- **Effort**: Medium — incremental refactoring per command

### TD-06: Large Config Package (Medium)

- **Location**: `internal/config/` (27 files, 14,408 LOC)
- **Issue**: Configuration package is disproportionately large relative to its responsibility
- **Impact**: Complex to navigate, potential for circular dependencies
- **Cost**: Changes to config cascade through many dependents
- **Effort**: Medium — would benefit from splitting into sub-packages

### TD-11: Large Repository Package (Medium)

- **Location**: `internal/repository/` (77 files, 30,133 LOC)
- **Issue**: Single package contains all entity repositories, test files, and adapters
- **Impact**: Long compile times for package changes, difficult to navigate
- **Cost**: Any repository change triggers full package recompilation
- **Effort**: Small — could split into sub-packages per entity type

### TD-07: Duplicate History Tables (Low)

- **Location**: `internal/db/db.go` (schema)
- **Issue**: Both `task_history` (legacy) and `entity_history` (new polymorphic) tables track status changes
- **Impact**: Two places to query history, potential for drift
- **Cost**: Must maintain both codepaths
- **Effort**: Small — consolidate queries to use entity_history only

### TD-08: Sequential Test Execution (Low)

- **Location**: `Makefile` (`-p=1` flag)
- **Issue**: Tests run sequentially to avoid database contention
- **Impact**: Slower test suite (~2-3x slower than parallel execution)
- **Cost**: Developer feedback loop is slower
- **Effort**: Medium — would require per-test database isolation

## Template System Complexity

- **Location**: `shark-templates/` (133 files, 80+ entity templates)
- **Issue**: Large number of status-specific templates with shared patterns
- **Impact**: Adding a new status requires creating templates for each entity type
- **Mitigation**: Partial templates (`_partials/`) reduce duplication
- **Risk**: Template drift between similar statuses

## Documentation Volume

- **Metric**: ~1,955 markdown files in docs/
- **Issue**: Large documentation footprint may become stale
- **Mitigation**: Database is source of truth; docs are output artifacts
- **Risk**: Low — stale docs don't affect functionality

See also: [Summary](summary.md) | [Remediation Plan](remediation-plan.md)
=======
> Generated: 2026-03-20
> Phase: 6 — Technical Debt Assessment

## High-Maintenance Areas

### 1. CLI Command Duplication (TD-01, TD-03, TD-09)

**Burden**: Every new entity type requires creating 10-15 parallel command files.

| Pattern | Epic | Feature | Task | Bug | Change |
|---------|------|---------|------|-----|--------|
| CRUD commands | Yes | Yes | Yes | Yes | Yes |
| Context commands | Yes | Yes | Yes | Yes | Yes |
| Note commands | Yes | Yes | Yes | Yes | Yes |
| Resume commands | Yes | Yes | Yes | — | — |
| Helper/formatting | Yes | Yes | Yes | Yes | Yes |
| Status transitions | Yes | Yes | Yes | Yes | Yes |

**Impact**: Adding a new entity type (e.g., "Sprint") would require ~15 new files with ~2,000 LOC of mostly-duplicated code.

**Being Addressed By**: E21 (Entity Polymorphism) — introduces unified entity handling via `EntityService` and `EntityRegistry`.

### 2. Schema and Migration Management (TD-02, TD-08)

**Burden**: `db.go` at 2,335 LOC combines schema, config, migrations, and versioning. `migrate.go` at 1,560 LOC grows with each new migration.

**Impact**:
- Every schema change requires modifying large files
- Migration ordering is manual (risk of conflicts)
- No migration rollback capability
- `skip_migrations` flag adds complexity (must remember to toggle for Turso)

**Recommendation**: Split into separate files by concern; consider a migration framework.

### 3. Fat Controller Legacy Code (TD-01)

**Burden**: ~40% of CLI command files directly call repositories, bypassing the service layer.

**Impact**:
- Business logic duplicated between commands and services
- Commands harder to test (require real DB or complex mocking)
- Changes to business rules may need updates in multiple places
- New developers may follow the wrong pattern

**Being Addressed By**: E15 (Service Layer Migration) — systematic extraction of business logic from commands to services. Migration guide exists at `docs/guides/service-layer-migration.md`.

### 4. Workflow Configuration Complexity

**Burden**: `.sharkconfig.json` contains extensive workflow definitions for 5 entity types with up to 19 statuses each.

**Impact**:
- Config file grows complex (hard to edit manually)
- Multi-level workflow resolution (task → feature → epic) adds cognitive load
- Status metadata (colors, phases, agents) must be kept in sync with flow definitions
- Profile switching (`basic` ↔ `advanced`) must preserve custom changes

**Mitigation**: Profile system automates common configurations; config validation catches errors.

### 5. Cross-Platform Build Complexity (TD-05)

**Burden**: CGO requirement for SQLite FTS5 complicates cross-compilation.

**Impact**:
- macOS and Windows releases lack FTS5 (full-text search)
- Linux ARM64 requires cross-compiler (`aarch64-linux-gnu-gcc`)
- CI build matrix has platform-specific overrides
- Contributors on non-Linux may have build issues

**Mitigation**: GoReleaser handles complexity; most users on Linux get full features.

## Low-Maintenance Areas

| Area | Why Low Maintenance |
|------|-------------------|
| **Models** | Pure data types, minimal logic, no dependencies |
| **Workflow engine** | Small (3 files), clean, config-driven |
| **Test infrastructure** | Well-established patterns with shared helpers |
| **Dependencies** | All current, minimal transitive tree |
| **CI/CD** | Simple GitHub Actions with standard patterns |

## Maintenance Cost Trends

| Trend | Direction | Driver |
|-------|-----------|--------|
| CLI duplication | Decreasing | E21 entity polymorphism |
| Service layer coverage | Increasing | E15 migration ongoing |
| Migration file size | Increasing | Each schema change adds migrations |
| Entity type count | Increasing | Bug and ChangeCard added recently |
| Test maintenance | Stable | Good test-to-code ratio maintained |

---

See also: [Summary](summary.md) | [Remediation Plan](remediation-plan.md) | [Complexity Analysis](../analysis/complexity-analysis.md)
>>>>>>> Stashed changes
