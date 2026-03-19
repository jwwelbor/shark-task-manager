# F04: Document Operations Unification

**Feature Key**: E21-F04
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Execution Order**: 4
**Effort Estimate**: S (2-3 tasks)
**Risk**: Low (document operations are simpler than status transitions)

---

## Description

Move the document linking operations (`LinkDocument`, `UnlinkDocument`, `ListRelatedDocumentsByKey`) from individual entity services into the shared `EntityService`. Each of the 5 entity services currently has its own wrapper (~30 lines each) that checks `writableDocRepo`, looks up the entity by key, and then calls the same shared helper. This is a straightforward extraction.

---

## Scope

### In Scope

1. **EntityService document methods** (`internal/services/entity_service.go`)
   - `LinkDocument(ctx, repo EntityRepository, key, docPath, description string) error`
   - `UnlinkDocument(ctx, repo EntityRepository, key, docPath string) error`
   - `ListRelatedDocuments(ctx, repo EntityRepository, key string) ([]Document, error)`

2. **Entity-specific service delegation**
   - Each entity service delegates document operations to `EntityService`
   - Remove duplicated document operation methods from all 5 entity services

3. **Tests**
   - EntityService document operation tests with mock EntityRepository
   - Behavioral parity verification

### Out of Scope

- CLI command changes for document operations
- New document operation features
- Document storage or path resolution changes

---

## Requirements Traced

| Requirement | Description | Coverage |
|-------------|-------------|----------|
| REQ-F-010 | Shared Document Linking | Full |
| REQ-NF-001 | Zero Behavioral Changes | All document operations produce identical results |
| REQ-NF-007 | Code Reduction Target | ~150 lines removed (5x ~30-line wrappers) |
| REQ-NF-009 | Phase Boundary Quality Gate | `make fmt && make lint && make test` passes |

---

## Dependencies

- **F01: Entity Interface Foundation** -- EntityRepository adapters needed for polymorphic entity lookup in document operations.
- **F03 (recommended but not strictly required)**: If F03 creates EntityService, F04 adds methods to it. If F04 runs before F03, F04 creates EntityService with document methods only and F03 adds TransitionStatus later. Either order works.

---

## Acceptance Criteria

1. `EntityService.LinkDocument`, `UnlinkDocument`, `ListRelatedDocuments` implemented
2. All 5 entity services delegate document operations to EntityService
3. Per-entity document wrapper methods removed from entity services
4. All existing document linking behavior preserved (verified by tests)
5. `make fmt && make lint && make test` passes

---

## Estimated Tasks

1. Add document operation methods to EntityService
2. Refactor 5 entity services to delegate document operations
3. Tests for shared document operations

---

*Last Updated*: 2026-03-19
