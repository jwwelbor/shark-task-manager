# F03: Status Transition Unification

**Feature Key**: E21-F03
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Execution Order**: 3
**Effort Estimate**: L (5-8 tasks)
**Risk**: Medium (modifying status transition logic used by all entity types)

---

## Description

Extract the shared status transition logic (currently duplicated ~80 lines across 5 entity services) into a single `EntityService.TransitionStatus` method. Refactor all 5 entity-specific services to delegate shared transition logic to `EntityService` while retaining entity-specific pre/post hooks (e.g., task dependency checks, feature progress cascade, epic feature rollup).

This is the highest-value single feature in the epic because status transitions are the most complex duplicated pattern (~400 lines of duplicated code) and the most frequent source of behavioral inconsistencies between entity types.

---

## Scope

### In Scope

1. **EntityService creation** (`internal/services/entity_service.go`)
   - `TransitionStatus` method handling all 10 steps of the transition flow
   - `TransitionFeatures` config struct for opt-in behavior (backward detection, rejection notes, child counting, orchestrator action resolution)
   - `DefaultTransitionFeatures()` and `SimpleTransitionFeatures()` presets
   - `resolveActionForEntity` shared implementation replacing 5 identical `resolveAction` methods
   - Shared `EntityPlaceholders` function (base set for orchestrator action templates)

2. **Entity-specific service refactoring**
   - **EpicService**: Delegate `TransitionStatus` to `EntityService`, add post-hook for feature counting
   - **FeatureService**: Delegate `TransitionStatus`, add post-hook for task counting and epic cascade
   - **TaskService**: Delegate `TransitionStatus`, add pre-hook for dependency validation
   - **BugService**: Delegate `resolveAction`; TransitionStatus delegation optional (simpler workflow)
   - **ChangeCardService**: Delegate `resolveAction`; TransitionStatus delegation optional

3. **Service constructor updates**
   - Each entity service receives `*EntityService` and `EntityRepository` adapter as new dependencies
   - Remove duplicated `workflowSvc` field from entity services (EntityService owns it)
   - Remove duplicated `noteRepo` field from entity services (EntityService owns it)

4. **Tests**
   - `EntityService.TransitionStatus` parameterized test suite across entity types
   - Entity-specific pre/post hook tests
   - Behavioral parity tests (same inputs produce same outputs as before)

### Out of Scope

- NoteService/ContextService/ResumeService changes (that is F02)
- Document operations (that is F04)
- Template placeholders beyond the base set (that is F05)
- CLI command changes

---

## Requirements Traced

| Requirement | Description | Coverage |
|-------------|-------------|----------|
| REQ-F-008 | Shared TransitionStatus Implementation | Full (all 10 steps) |
| REQ-F-009 | Entity-Specific Transition Extensions | Full (pre/post hooks) |
| REQ-NF-001 | Zero Behavioral Changes | All transitions produce identical results |
| REQ-NF-006 | Shared Service Test Coverage | 80%+ for EntityService |
| REQ-NF-007 | Code Reduction Target | ~525 lines removed (5x ~80-line TransitionStatus + 5x ~25-line resolveAction) |
| REQ-NF-009 | Phase Boundary Quality Gate | `make fmt && make lint && make test` passes |
| REQ-NF-014 | Cross-Cutting Service Test Consolidation | Parameterized transition tests |

---

## Dependencies

- **F01: Entity Interface Foundation** -- EntityRepository adapters required for `EntityService.TransitionStatus` to operate polymorphically.
- **Sequencing note**: Should be sequenced after any E15 (service layer migration) work that touches the same service files to avoid merge conflicts. F02 and F03 can run in parallel if developers work on different files.

---

## Technical Design Reference

See [architecture-design.md](../architecture-design.md) sections:
- Section 4: EntityService Composition Pattern
- Section 5: Migration Strategy (Before/After examples for EpicService and BugService)

---

## Acceptance Criteria

1. `EntityService.TransitionStatus` implements all 10 steps of the shared transition flow
2. `TransitionFeatures` config allows opt-in/opt-out of backward detection, rejection notes, child counting, orchestrator action resolution
3. EpicService delegates to `EntityService.TransitionStatus` and adds feature-counting post-hook
4. FeatureService delegates to `EntityService.TransitionStatus` and adds task-counting and cascade post-hooks
5. TaskService delegates to `EntityService.TransitionStatus` and adds dependency-validation pre-hook
6. BugService and ChangeCardService delegate at minimum `resolveAction` to `EntityService`
7. All 5 entity-specific `resolveAction` methods are removed (replaced by `EntityService.resolveActionForEntity`)
8. Entity services no longer hold direct `workflowSvc` references (EntityService owns it)
9. All status transitions for all entity types produce identical results as before the refactoring
10. `make fmt && make lint && make test` passes

---

## Estimated Tasks

1. Create EntityService with TransitionStatus and TransitionFeatures
2. Create shared resolveActionForEntity replacing 5 duplicated methods
3. Refactor EpicService to compose with EntityService
4. Refactor FeatureService to compose with EntityService
5. Refactor TaskService to compose with EntityService
6. Refactor BugService and ChangeCardService (resolveAction delegation at minimum)
7. Update service constructor wiring in CLI accessors
8. Parameterized TransitionStatus test suite

---

*Last Updated*: 2026-03-19
