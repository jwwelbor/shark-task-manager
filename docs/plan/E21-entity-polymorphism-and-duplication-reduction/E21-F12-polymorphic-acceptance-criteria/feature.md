---
feature_key: E21-F12-polymorphic-acceptance-criteria
epic_key: E21
title: "Remove Unused Acceptance Criteria System"
description: Remove the unused task_criteria table, repository, service, CLI commands, and all related code
---

# Remove Unused Acceptance Criteria System

**Feature Key**: E21-F12-polymorphic-acceptance-criteria

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The acceptance criteria system was built (E10-F04) but **has never been used**. The `task_criteria` table has **0 rows** in production. Despite this, it has a full stack of code that must be maintained:

- **Model**: `internal/models/task_criteria.go`
- **Repository**: `internal/repository/task_criteria_repository.go` + tests
- **Service**: `internal/services/criteria_service.go` + tests
- **CLI Commands**: `internal/cli/commands/task_criteria.go` + `feature_criteria.go` + tests
- **Parser**: `internal/taskfile/criteria_parser.go` + tests
- **Schema**: `task_criteria` table + indexes in `internal/db/db.go`
- **References**: scattered across context_service, task_dto, search_repository, templates, discovery, validation (~41 files reference "criteria")

This is dead code adding maintenance burden. It was referenced across 41 Go files at time of analysis.

### Solution

Remove the entire acceptance criteria system:
1. Drop the `task_criteria` table (0 rows, no data loss)
2. Delete criteria model, repository, service, CLI commands, and parser
3. Remove criteria references from other services (context, search, templates, discovery)
4. Clean up imports and tests

### Impact

- Remove dead code from ~41 files
- Eliminate maintenance burden for an unused feature
- Reduce test suite runtime (criteria tests run but test nothing useful)
- Simplify the entity model (one fewer cross-cutting concern to think about)

---

## Scope

### What to Remove

| Component | File(s) | Action |
|-----------|---------|--------|
| Model | `internal/models/task_criteria.go` | Delete |
| Repository | `internal/repository/task_criteria_repository.go`, `*_test.go` | Delete |
| Service | `internal/services/criteria_service.go`, `*_test.go` | Delete |
| CLI Commands | `internal/cli/commands/task_criteria.go`, `feature_criteria.go`, `*_test.go` | Delete |
| Parser | `internal/taskfile/criteria_parser.go`, `*_test.go` | Delete |
| DB Schema | `task_criteria` table creation in `internal/db/db.go` or `migrate.go` | Remove DDL |
| DB Migration | Add migration to `DROP TABLE IF EXISTS task_criteria` | Add |
| References | ~41 files that import or reference criteria | Clean up |

### What to Keep

Nothing. The feature has 0 usage.

---

## Requirements

1. **REQ-F-001**: Drop `task_criteria` table via migration
   - Add `DROP TABLE IF EXISTS task_criteria` migration
   - Bump `CurrentSchemaVersion`
   - Developer must set `skip_migrations: false` temporarily

2. **REQ-F-002**: Delete all criteria code files (model, repo, service, CLI, parser)

3. **REQ-F-003**: Remove criteria references from other files
   - Search for `criteria`, `TaskCriteria`, `task_criteria` across all Go files
   - Remove imports, struct fields, service dependencies, CLI registrations

4. **REQ-F-004**: All tests pass after removal (`make fmt && make lint && make test`)

---

## Acceptance Criteria

- [ ] `task_criteria` table dropped from schema
- [ ] All criteria-specific files deleted
- [ ] No remaining references to `criteria` in Go code (except this feature doc)
- [ ] `make fmt && make lint && make test` passes
- [ ] `shark task criteria` command no longer exists

---

## Rationale

- **0 rows** in production database (verified 2026-03-20)
- Feature was built in E10-F04 (marked completed) but never adopted into any workflow
- The `/uat` skill mentions criteria but works without them
- If criteria are needed in the future, they should be designed from scratch as polymorphic (`entity_criteria`) rather than extending the unused task-only system
- Removing dead code before E21 refactoring prevents wasted effort consolidating code that should be deleted

---

*Last Updated*: 2026-03-20
