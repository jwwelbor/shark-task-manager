# F02: Cross-Cutting Service Unification

**Feature Key**: E21-F02
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Execution Order**: 2
**Effort Estimate**: L (5-8 tasks)
**Risk**: Medium (modifying service constructors and wiring)

---

## Description

Refactor the three cross-cutting services (NoteService, ContextService, ResumeService) to use the `EntityRegistry` and `EntityRepository` interfaces instead of maintaining per-entity repository fields and 5-branch switch statements. Update CLI accessor wiring in `services_global.go` to use `EntityRegistry` initialization instead of per-entity setter methods.

This is one of the two highest-value features (alongside F03) because it eliminates the most visible duplication pattern: every new entity type currently requires adding a new repository field, setter method, and switch branch to each cross-cutting service.

---

## Scope

### In Scope

1. **NoteService refactoring** (`internal/services/note_service.go`)
   - Replace 5 per-entity repository fields with single `EntityRegistry`
   - Replace `resolveEntityID` 5-branch switch with registry lookup
   - Replace `GetEntityDetails` 5-branch switch with registry lookup
   - Remove all `SetXxxRepo()` setter methods
   - Update constructor to accept `EntityRegistry`

2. **ContextService refactoring** (`internal/services/context_service.go`)
   - Replace 5 per-entity repository fields with single `EntityRegistry`
   - Replace `getContextJSON` 5-branch switch with registry `GetContextData`
   - Replace `setContextJSON` 5-branch switch with registry `UpdateContextData`
   - Remove all `SetXxxRepo()` setter methods
   - Update constructor to accept `EntityRegistry`

3. **ResumeService refactoring** (`internal/services/resume_service.go`)
   - Replace per-entity repository fields with `EntityRegistry`
   - Replace entity lookup switch with registry dispatch
   - Remove setter methods

4. **CLI accessor wiring update** (`internal/cli/services_global.go`, `services_global_entities.go`)
   - Replace per-entity setter calls with `EntityRegistry` initialization
   - Create `GetEntityRegistry()` global accessor with `sync.Once` lazy init
   - Update `GetNoteService()`, `GetContextService()`, `GetResumeService()` to pass registry

5. **Tests**
   - Update NoteService tests to use mock `EntityRegistry`
   - Update ContextService tests to use mock `EntityRegistry`
   - Update ResumeService tests
   - Verify behavioral parity (identical results for all existing operations)

### Out of Scope

- Status transition changes (that is F03)
- Document operation changes (that is F04)
- Entity-specific service changes (EpicService, TaskService, etc.)
- CLI command changes

---

## Requirements Traced

| Requirement | Description | Coverage |
|-------------|-------------|----------|
| REQ-F-005 | NoteService Registry-Based Dispatch | Full |
| REQ-F-006 | ContextService Registry-Based Dispatch | Full |
| REQ-F-007 | ResumeService Registry-Based Dispatch | Full |
| REQ-F-013 | CLI Accessor Consolidation | Partial (accessor wiring only) |
| REQ-NF-001 | Zero Behavioral Changes | All operations produce identical results |
| REQ-NF-006 | Shared Service Test Coverage | 80%+ for refactored services |
| REQ-NF-007 | Code Reduction Target | ~230 lines removed (switch branches + setter methods + repo fields) |
| REQ-NF-009 | Phase Boundary Quality Gate | `make fmt && make lint && make test` passes |

---

## Dependencies

- **F01: Entity Interface Foundation** -- EntityRepository interface, adapters, and EntityRegistry must exist before cross-cutting services can use them.

---

## Technical Design Reference

See [architecture-design.md](../architecture-design.md) sections:
- Section 3: EntityRegistry Design (registry initialization in CLI)
- Section 6: NoteService/ContextService Refactoring Pattern

---

## Acceptance Criteria

1. NoteService has zero per-entity switch statements
2. NoteService constructor accepts `EntityRegistry` (not individual repos)
3. All `SetXxxRepo()` setter methods removed from NoteService
4. ContextService has zero per-entity switch statements
5. ContextService constructor accepts `EntityRegistry`
6. ResumeService uses registry-based entity lookup
7. CLI `services_global.go` creates and wires `EntityRegistry` lazily
8. All existing note, context, and resume operations produce identical results
9. Adding a hypothetical 6th entity type requires only a `Register()` call, no modifications to NoteService/ContextService/ResumeService
10. `make fmt && make lint && make test` passes

---

## Estimated Tasks

1. Refactor NoteService to use EntityRegistry
2. Refactor ContextService to use EntityRegistry
3. Refactor ResumeService to use EntityRegistry
4. Update CLI accessor wiring (services_global.go) to use EntityRegistry
5. Update NoteService tests with mock EntityRegistry
6. Update ContextService tests with mock EntityRegistry
7. Update ResumeService tests
8. Integration verification (end-to-end smoke test of note/context/resume commands)

---

*Last Updated*: 2026-03-19
