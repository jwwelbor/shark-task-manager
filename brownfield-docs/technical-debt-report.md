# Technical Debt Report — Executive Summary

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-20
> Phase: 10 — Finalization

## Overall Assessment: GOOD HEALTH

The Shark Task Manager codebase is in **good health** with no critical technical debt. The architecture is sound, dependencies are current, and test coverage is strong (1.87:1 test-to-production ratio). Two significant modernization efforts are already underway.

## Top Recommendation

**Complete the ongoing service layer migration (E15) and entity polymorphism (E21) epics.** These two efforts address the project's largest structural debt items and are already well-planned with migration guides and progress tracking.

## Debt Summary

| Severity | Count | Description |
|----------|-------|-------------|
| **Critical** | 0 | No critical issues |
| **High** | 1 | Fat controller pattern (business logic in CLI commands) |
| **Medium** | 5 | Code duplication, pre-release dependency, build complexity |
| **Low** | 4 | Minor schema/security/distribution items |

## Top 5 Issues

### 1. Fat Controller Anti-Pattern (HIGH)
**~40% of CLI commands call repositories directly**, bypassing the service layer. This duplicates business logic and makes commands hard to test.
- **Business Impact**: Higher maintenance cost, slower feature development
- **Being Addressed By**: E15 epic (service layer migration) — in progress
- **Effort**: Large (multi-sprint), but migration guide exists

### 2. Entity-Type Code Duplication (MEDIUM)
Parallel file sets for each entity type (epic, feature, task, bug, change) in CLI commands — context, notes, resume, formatting helpers.
- **Business Impact**: Adding new entity types requires ~15 new files
- **Being Addressed By**: E21 epic (entity polymorphism) — in progress

### 3. Schema God File (MEDIUM)
`internal/db/db.go` at 2,335 LOC combines schema, config, migrations, and versioning.
- **Business Impact**: Harder to navigate and modify schema changes
- **Recommendation**: Split into 4 focused files
- **Effort**: Small

### 4. Pre-Release Turso Client (MEDIUM)
Cloud database client uses commit-hash version, not a tagged release.
- **Business Impact**: API stability risk, harder to track security updates
- **Recommendation**: Pin to tagged release when available
- **Effort**: Small (blocked on external dependency)

### 5. CGO Cross-Compilation (MEDIUM)
SQLite driver requires C compiler, complicating cross-platform builds.
- **Business Impact**: macOS/Windows lack FTS5; build matrix is complex
- **Recommendation**: Evaluate pure-Go SQLite alternative
- **Effort**: Medium (requires benchmarking)

## Immediate Actions

1. Continue E15 and E21 epics — they address items #1 and #2
2. Split `db.go` into focused files — quick win for #3
3. Monitor Turso client for tagged releases — resolve #4 when possible

## Strategic Timeline

| Timeframe | Actions |
|-----------|---------|
| **Current** | Complete E15 (service migration) + E21 (entity polymorphism) |
| **Next sprint** | Split db.go, pin Turso client, clean deprecated fields |
| **When needed** | Evaluate pure-Go SQLite, add HTTP auth, set up Homebrew/Scoop |

## Strengths

- Excellent test coverage (1.87:1 ratio)
- All dependencies current with no known CVEs
- Clean layered architecture (services → repositories → database)
- Config-driven workflow system (flexible, no hardcoded statuses)
- Active modernization already underway (E15, E21)
- Comprehensive documentation (CLAUDE.md, CLI reference, migration guides)

---

See also: [Full Technical Debt Assessment](technical-debt/summary.md) | [Remediation Plan](technical-debt/remediation-plan.md)
