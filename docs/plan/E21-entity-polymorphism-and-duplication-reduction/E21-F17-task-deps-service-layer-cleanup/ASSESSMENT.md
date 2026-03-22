# Complexity Triage Assessment - E21-F17

**Date:** 2026-03-22
**Feature:** Task Deps Service Layer Cleanup
**Status:** Ready for Specification

## Scope Validation

✅ **Correctly scoped as FEATURE** (not misclassified as task)

- **Multi-capability:** 3 distinct tasks addressing different concerns
  - Move helpers to service layer (50+ LOC)
  - Add batch GetTasksByIDs method (20-30 LOC)
  - Unify duplicated helpers (30-40 LOC)
- **Design decisions:** Where to house relationship resolution logic (EntityRelationshipService vs new service)
- **File impact:** 5+ files (task_deps.go, entity_relationship_service.go, task_repository.go, tests)
- **Cross-cutting concerns:** Architectural refactoring affecting CLI, service, and repository layers

## Complexity Triage Score: 11/27 → STANDARD

### Technical Dimensions (7/6 points)
| Dimension | Score | Rationale |
|-----------|-------|-----------|
| File Impact | 2/3 | 5 files affected, localized to task dependencies subsystem |
| Pattern Novelty | 1/3 | Follows established E15 service layer pattern, not novel |
| Data Model | 1/3 | No schema changes, reuses existing models (Task, EntityRelationship) |
| API Surface | 2/3 | 4 new public methods (3 service + 1 repository) |
| Cross-Feature Deps | 1/3 | Self-contained, minimal ecosystem impact, used only by task_deps.go |
| UI Complexity | 0/3 | No new UI, pure refactoring of existing CLI output |
| **Subtotal** | **7/18** | |

### Execution Dimensions (4/3 points)
| Dimension | Score | Rationale |
|-----------|-------|-----------|
| Task Estimation | 1/3 | ~40-60 LOC per task, straightforward implementation |
| Regression Risk | 2/3 | Moderate - touches active query path, N+1 elimination requires validation |
| Execution Effort | 1/3 | Clear steps, established patterns to follow from E21 work |
| **Subtotal** | **4/9** | |

**Total: 11/27 → STANDARD** (7-15 = STANDARD tier)

## Complexity Tier Justification

**STANDARD (not SIMPLE)** because:
- 3 distinct, sequenced tasks (not 1-2 simple changes)
- Requires design decisions about service architecture
- Moderate regression risk from N+1 query elimination
- Affects multiple architectural layers
- Needs comprehensive testing strategy

**Not COMPLEX** because:
- Follows established architecture patterns (no novel design)
- No schema changes or data model redesign
- Localized scope (task dependencies only)
- Straightforward implementation steps
- Low-to-moderate file impact

## Next Phase: Specification

This feature is ready for **ready_for_specification** status:
- ✅ Scope validated
- ✅ Complexity assessed
- ⏭ Pending: Detailed specifications, acceptance criteria, test plan
