# Remediation Plan

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 6 — Technical Debt Assessment

## Prioritized Remediation Items

### Priority 1: Complete Service Layer Migration (TD-01)

| Property | Value |
|----------|-------|
| **Effort** | Large (multi-sprint) |
| **Risk Reduced** | Duplication, testability, maintainability |
| **Status** | In progress (E15 epic) |
| **Dependencies** | None |

**Action**: Continue systematic migration of CLI commands to call services instead of repositories directly. Follow the 10-step migration process in `docs/guides/service-layer-migration.md`.

**Sequence**:
1. High-priority commands (simple get/list) → already done for many
2. Medium-priority commands (start, complete, status transitions)
3. Low-priority commands (complex create, dependency management)

### Priority 2: Complete Entity Polymorphism (TD-03, TD-09)

| Property | Value |
|----------|-------|
| **Effort** | Large (multi-sprint) |
| **Risk Reduced** | Code duplication, maintenance burden |
| **Status** | In progress (E21 epic) |
| **Dependencies** | TD-01 partially (cleaner with services) |

**Action**: Complete the Entity interface and registry system to unify cross-cutting operations across all entity types.

**Impact**: Eliminates parallel file sets for context, notes, resume, and formatting across 5+ entity types.

### Priority 3: Split db.go Schema File (TD-02)

| Property | Value |
|----------|-------|
| **Effort** | Small |
| **Risk Reduced** | Complexity, cognitive load |
| **Status** | Not started |
| **Dependencies** | None |

**Action**: Split `internal/db/db.go` (2,335 LOC) into:
- `schema.go` — Table CREATE statements
- `sqlite_config.go` — PRAGMA configuration
- `version.go` — Schema versioning logic
- `db.go` — InitDB and connection management

### Priority 4: Pin Turso Client to Tagged Release (TD-04)

| Property | Value |
|----------|-------|
| **Effort** | Small |
| **Risk Reduced** | Dependency stability |
| **Status** | Blocked (waiting for Turso tagged release) |
| **Dependencies** | External (Turso team) |

**Action**: Monitor `tursodatabase/libsql-client-go` for tagged releases. Update `go.mod` when available.

### Priority 5: Consider Pure-Go SQLite Alternative (TD-05)

| Property | Value |
|----------|-------|
| **Effort** | Medium |
| **Risk Reduced** | Build complexity, cross-platform parity |
| **Status** | Not started |
| **Dependencies** | Requires performance benchmarking |

**Action**: Evaluate `modernc.org/sqlite` as a CGO-free alternative. This would:
- Eliminate cross-compiler requirements
- Enable FTS5 on all platforms
- Simplify CI build matrix
- Remove CGO-related build issues

**Tradeoff**: May have different performance characteristics. Benchmark before deciding.

### Priority 6: Add HTTP API Authentication (TD-06)

| Property | Value |
|----------|-------|
| **Effort** | Small |
| **Risk Reduced** | Security (if API is exposed) |
| **Status** | Not started |
| **Dependencies** | None |

**Action**: Add API key or Bearer token authentication if the HTTP server is used beyond localhost.

### Priority 7: Clean Up Deprecated Fields (TD-07)

| Property | Value |
|----------|-------|
| **Effort** | Small |
| **Risk Reduced** | Schema clarity |
| **Status** | Not started |
| **Dependencies** | None |

**Action**: Add migration to remove deprecated `depends_on` TEXT field from tasks table (replaced by `task_relationships` table).

### Priority 8: Set Up Package Distribution (TD-10)

| Property | Value |
|----------|-------|
| **Effort** | Small |
| **Risk Reduced** | Distribution reach |
| **Status** | Not started |
| **Dependencies** | Set up homebrew-shark and scoop-shark repositories |

**Action**: Create Homebrew tap and Scoop bucket repositories, uncomment GoReleaser configuration.

## Recommended Sequence

```
Phase 1 (Current)
├── [In Progress] E15: Service layer migration
└── [In Progress] E21: Entity polymorphism

Phase 2 (Next)
├── Split db.go schema file
├── Pin Turso client (when available)
└── Clean up deprecated fields

Phase 3 (When Needed)
├── Evaluate pure-Go SQLite
├── Add HTTP API auth
└── Set up Homebrew/Scoop distribution
```

---

See also: [Summary](summary.md) | [Maintenance Burden](maintenance-burden.md) | [Migration Readiness](../migration/component-order.md)
