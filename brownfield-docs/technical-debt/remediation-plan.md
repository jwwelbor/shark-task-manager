# Remediation Plan

> Part of the Shark Task Manager Brownfield Analysis
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
