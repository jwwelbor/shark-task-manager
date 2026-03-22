# Maintenance Burden

> Part of the Shark Task Manager Brownfield Analysis
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
