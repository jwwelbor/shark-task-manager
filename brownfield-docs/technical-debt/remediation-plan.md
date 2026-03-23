# Remediation Plan

> Part of the Shark Task Manager Brownfield Analysis
<<<<<<< Updated upstream
> Generated: 2026-03-22
> Phase: 6 — Technical Debt Assessment

## Prioritized Action Items

| Priority | ID | Action | Why | Effort | Dependencies |
|----------|-----|--------|-----|--------|-------------|
| 1 | TD-02 | Complete service layer migration (E15) | Eliminates duplicated logic, enables HTTP API, improves testability | Medium | None |
| 2 | TD-04 | Implement HTTP API endpoints | Enables web/API access to task management | Large | TD-02 (services must exist) |
| 3 | TD-01 | Monitor and pin Turso client to stable release | Reduces risk of breaking changes in cloud backend | Small | External: Turso stable release |
| 4 | TD-11 | Split repository package into sub-packages | Faster compilation, better navigation | Small | None |
| 5 | TD-06 | Split config package into sub-packages | Reduce complexity, clearer boundaries | Small | None |
| 6 | TD-07 | Consolidate history tables | Eliminate dual-write, simplify queries | Small | None |
| 7 | TD-05 | Add structured logging | Better observability for HTTP server mode | Small | TD-04 (more valuable with server) |
| 8 | TD-08 | Per-test database isolation | Enable parallel test execution | Medium | None |
| 9 | TD-12 | Add graceful shutdown to HTTP server | Proper connection draining | Small | TD-04 |
| 10 | TD-09 | Connection pooling for Turso | Better performance under concurrent access | Small | TD-04 |

## Recommended Sequence

```mermaid
gantt
    title Remediation Timeline
    dateFormat  YYYY-MM
    section Phase 1 (Active)
    Service layer migration (TD-02)       :active, p1, 2026-03, 2026-05
    section Phase 2 (Near-term)
    HTTP API implementation (TD-04)       :p2, after p1, 2026-05, 2026-07
    Pin Turso client (TD-01)              :p3, 2026-04, 2026-04
    Split repository package (TD-11)      :p4, 2026-04, 2026-05
    Split config package (TD-06)          :p5, 2026-05, 2026-06
    section Phase 3 (Low priority)
    Consolidate history tables (TD-07)    :p6, 2026-06, 2026-06
    Structured logging (TD-05)            :p7, after p2, 2026-07, 2026-08
    Parallel test execution (TD-08)       :p8, 2026-07, 2026-08
    Graceful shutdown (TD-12)             :p9, after p2, 2026-08, 2026-08
```

## Quick Wins

These items require minimal effort and can be done opportunistically:

1. **Pin Turso client** (TD-01): Update `go.mod` when stable release is available
2. **Consolidate history queries** (TD-07): Switch all queries to use `entity_history` table
3. **Graceful shutdown** (TD-12): Add `context.WithCancel` + signal handler in `cmd/server/main.go`

See also: [Summary](summary.md) | [Maintenance Burden](maintenance-burden.md)
=======
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
>>>>>>> Stashed changes
