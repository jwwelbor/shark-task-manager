# UAT Test Guide - Entity Interface Foundation

**Feature:** E21-F01 - Entity Interface Foundation
**Epic:** E21 - Entity Polymorphism and Duplication Reduction
**Generated:** 2026-03-19
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Eliminate code duplication across entity types (Epic, Feature, Task, Bug, ChangeCard) by introducing polymorphic interfaces, unified services, and shared status transitions.

**This Feature's Role:** F01 lays the foundation — defines the `Entity` interface, `EntityRepository` interface, repository adapters for all 5 entity types, and the `EntityRegistry`. F02-F05 build on this foundation.

**Related Features:**
- E21-F02: Cross-Cutting Service Unification (draft) — will use EntityRegistry to replace switch statements
- E21-F03: Status Transition Unification (draft) — will use EntityRepository.UpdateStatus()
- E21-F04: Document Operations Unification (draft) — will use EntityRepository.GetByKey()
- E21-F05: Template Placeholder Unification (draft) — will use EntityRegistry.RegisteredTypes()

**Integration Points:**
- F02 depends on EntityRegistry.GetRepository() to access adapters polymorphically
- F03 depends on Entity.SetStatus() and EntityRepository.UpdateStatus()
- F04 depends on EntityRepository.GetByKey() returning models.Entity
- F05 depends on EntityRegistry.RegisteredTypes() for dynamic enumeration

---

## Design Intent

**From Architecture (02-architecture.md):**
> Entity interface is additive — existing direct field access (e.g., epic.Key) continues to work unchanged. The interface is used only by cross-cutting services that need to operate on entities polymorphically.

**Key Design Decisions:**
- ChangeCard Slug/FilePath normalized to `*string` as prerequisite (was `string`)
- Entity interface has 14 methods matching shared fields across all 5 models
- Adapters use get-set-update pattern for Feature/Task UpdateStatus (no dedicated typed method)
- EntityRegistry is NOT wired into services_global.go in F01 (deferred to F02)
- EntityType constants reused from existing entity_note.go (no new type)

---

## Test Scenarios

### Scenario 1: Entity Interface Completeness and Satisfaction
**Tasks covered:** T-E21-F01-001 (ChangeCard normalization), T-E21-F01-002 (Entity interface)

**Verification Points:**
- [ ] Entity interface defined in `internal/models/entity.go` with 14 methods
- [ ] Compile-time satisfaction checks: `var _ Entity = (*Epic)(nil)` etc. for all 5 types
- [ ] All 5 models implement all 14 accessor methods
- [ ] Nil `*string` fields (Slug, FilePath, Description, ContextData) return `""` not panic
- [ ] SetStatus/SetContextData mutate correctly for all 5 types
- [ ] Zero-value structs don't panic on any accessor
- [ ] ChangeCard.Slug and ChangeCard.FilePath are `*string` (normalized)

### Scenario 2: EntityRepository Interface and Adapter Delegation
**Tasks covered:** T-E21-F01-003 (EntityRepository + adapters)

**Verification Points:**
- [ ] EntityRepository interface defined in `internal/services/entity_repository.go` with 6 methods
- [ ] All 5 adapters satisfy EntityRepository (compile-time checks)
- [ ] Epic adapter: delegates GetByKey/GetByID/UpdateStatus/Update/GetContextData/UpdateContextData
- [ ] Feature adapter: UpdateStatus uses get-set-update pattern (no direct UpdateStatus on repo)
- [ ] Task adapter: UpdateStatus uses get-set-update pattern
- [ ] Bug adapter: GetContextData/UpdateContextData use get-set-update
- [ ] ChangeCard adapter: delegates directly (repo has UpdateContextData)
- [ ] All adapters reject wrong concrete type on Update() with clear error message
- [ ] Error propagation works correctly for all adapters

### Scenario 3: EntityRegistry Operations
**Tasks covered:** T-E21-F01-004 (EntityRegistry)

**Verification Points:**
- [ ] Register + GetRepository returns correct adapter
- [ ] GetRepository for unregistered type returns error (not panic)
- [ ] MustGetRepository panics for unregistered type
- [ ] Duplicate registration panics
- [ ] RegisteredTypes returns all registered types, sorted alphabetically
- [ ] Empty registry returns empty slice
- [ ] Thread safety: concurrent reads don't race (tested with -race flag)

### Scenario 4: Backward Compatibility
**Tasks covered:** All tasks

**Verification Points:**
- [ ] `make fmt` — zero formatting changes
- [ ] `make lint` — zero lint warnings
- [ ] `make test` — all 30 packages pass, zero failures
- [ ] Existing direct field access (e.g., `epic.Key`, `task.FeatureID`) still works
- [ ] ChangeCard tests pass with `*string` for Slug/FilePath

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-03-19 |
| Result | APPROVED |
| Results File | docs/uat/E21/results/UAT-E21-F01-20260319-180000-results.md |

**Previous Sessions:**
- 2026-03-19: APPROVED (Codex: ACCEPT WITH CONDITIONS → conditions resolved → user approved)
