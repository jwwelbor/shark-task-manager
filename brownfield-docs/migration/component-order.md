# Component Migration Order

> Part of the Shark Task Manager Brownfield Analysis
> Generated: 2026-03-22
> Phase: 8 — Migration Readiness

## Migration Context

The primary migration effort is the ongoing service layer refactoring (Epic E15): moving business logic from CLI commands into services. This document provides the dependency-ordered sequence for this and potential future migrations.

## Dependency-Ordered Migration Sequence

### Phase 1: Foundation (No Dependencies)

| Order | Component | Reason | Effort |
|-------|-----------|--------|--------|
| 1.1 | `models/` | Leaf node — no internal deps, stable interface | Done |
| 1.2 | `workflow/` | Depends only on config — stable interface | Done |
| 1.3 | `config/` | Foundation for workflow and status | Done |

### Phase 2: Data Access (Depends on Phase 1)

| Order | Component | Reason | Effort |
|-------|-----------|--------|--------|
| 2.1 | `db/` | Schema and migrations — depends on config | Done |
| 2.2 | `repository/` | Data access — depends on db, models | Done |

### Phase 3: Business Logic (Depends on Phase 2)

| Order | Component | Reason | Effort |
|-------|-----------|--------|--------|
| 3.1 | `services/entity_service.go` | Polymorphic base — used by all entity services | Done |
| 3.2 | `services/task_service.go` | Most commands; highest value | In Progress |
| 3.3 | `services/feature_service.go` | Feature lifecycle | Partial |
| 3.4 | `services/epic_service.go` | Epic lifecycle | Partial |
| 3.5 | `services/bug_service.go` | Bug tracking | Done |
| 3.6 | `services/change_card_service.go` | Change cards | Done |
| 3.7 | `services/note_service.go` | Notes management | Done |

### Phase 4: CLI Migration (Depends on Phase 3)

| Order | Component | Reason | Effort |
|-------|-----------|--------|--------|
| 4.1 | Simple get commands | Low complexity, high value | Small |
| 4.2 | List commands with filtering | Medium complexity | Medium |
| 4.3 | Status transition commands | High value | Medium |
| 4.4 | Create commands | Complex (file + DB) | Large |
| 4.5 | Analytics/progress commands | Report generation | Medium |

### Phase 5: HTTP API (Depends on Phase 3-4)

| Order | Component | Reason | Effort |
|-------|-----------|--------|--------|
| 5.1 | Handler infrastructure | Base patterns, middleware | Medium |
| 5.2 | Task API endpoints | Most used entity | Medium |
| 5.3 | Feature/Epic API endpoints | Hierarchy management | Medium |
| 5.4 | Bug/ChangeCard API endpoints | Standalone entities | Small |

## Components That Can Be Migrated Independently

- **Repository splitting** (TD-11): Can be done anytime, no service changes needed
- **Config splitting** (TD-06): Can be done anytime, internal refactoring only
- **History consolidation** (TD-07): Can switch queries independently
- **Structured logging** (TD-05): Additive change, no breaking changes

See also: [Test Specifications](test-specifications.md) | [Validation Criteria](validation-criteria.md)
