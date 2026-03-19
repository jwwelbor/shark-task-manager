# F05: Template Placeholder Unification

**Feature Key**: E21-F05
**Epic**: E21 - Entity Polymorphism and Duplication Reduction
**Execution Order**: 5
**Effort Estimate**: S (2-3 tasks)
**Risk**: Low (template placeholders are isolated and well-tested)

---

## Description

Create a shared `EntityPlaceholders` base function that generates template placeholders from any `Entity` interface value. Entity-specific placeholder functions extend the base set with their unique fields (e.g., task adds `agent_type`, bug adds `severity`). This replaces 5 independent placeholder functions that each manually extract the same shared fields.

---

## Scope

### In Scope

1. **EntityPlaceholders base function** (`internal/services/entity_service.go` or `internal/templates/`)
   - Generates `map[string]string` from Entity interface: entity_type, entity_key, entity_title, entity_slug, status
   - Used by orchestrator action template population

2. **Entity-specific placeholder extensions**
   - Each entity's placeholder function calls `EntityPlaceholders` first, then adds entity-specific fields
   - Task: agent_type, depends_on, execution_order
   - Bug: severity, linked_entity_type, linked_entity_key
   - ChangeCard: priority, requested_by, assigned_to
   - Epic/Feature: minimal extensions (mostly shared fields)

3. **Tests**
   - Base `EntityPlaceholders` test with mock Entity
   - Entity-specific extension tests
   - Verify existing template rendering produces identical output

### Out of Scope

- Template system redesign
- New template types or fields
- CLI template command changes

---

## Requirements Traced

| Requirement | Description | Coverage |
|-------------|-------------|----------|
| REQ-F-011 | Base Entity Placeholders | Full |
| REQ-NF-001 | Zero Behavioral Changes | Identical template rendering output |
| REQ-NF-007 | Code Reduction Target | ~75 lines removed (shared field extraction across 5 functions) |
| REQ-NF-009 | Phase Boundary Quality Gate | `make fmt && make lint && make test` passes |

---

## Dependencies

- **F01: Entity Interface Foundation** -- Entity interface required for `EntityPlaceholders` to extract shared fields polymorphically.
- **F03 (if EntityPlaceholders lives in EntityService)**: If F03 creates EntityService, F05 can add `EntityPlaceholders` to it. If F05 runs independently, it creates the function in a location accessible to both EntityService and entity-specific services.

---

## Acceptance Criteria

1. `EntityPlaceholders(entity models.Entity) map[string]string` function exists
2. Base function generates placeholders for: entity_type, entity_key, entity_title, entity_slug, status
3. Entity-specific placeholder functions call `EntityPlaceholders` and add unique fields
4. Existing template rendering produces identical output
5. `make fmt && make lint && make test` passes

---

## Estimated Tasks

1. Create EntityPlaceholders base function
2. Refactor entity-specific placeholder functions to extend base
3. Tests for base and extended placeholder generation

---

*Last Updated*: 2026-03-19
