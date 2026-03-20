---
feature_key: E21-F09-entity-service-delegation-completion
epic_key: E21
title: Entity Service Delegation Completion
description: Make all entity services delegate shared logic to EntityService and eliminate cross-cutting service duplication
---

# Entity Service Delegation Completion

**Feature Key**: E21-F09-entity-service-delegation-completion

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

E21-F01 created `EntityService` with shared `TransitionStatus` logic and `EntityRegistry` for polymorphic repository lookup. E21-F02/F03 were marked completed but **the entity services are not fully using this infrastructure**. The result:

**Service Layer Duplication (Current State):**

| Duplicated Pattern | Instances | Lines Each | Total Waste |
|--------------------|-----------|-----------|-------------|
| TransitionStatus implementation | 5 services | ~80 lines | ~400 lines |
| GetNextStatus implementation | 3 services | ~25 lines | ~75 lines |
| makeResolveActionFn | 5 services | ~40 lines | ~200 lines |
| LinkDocument/UnlinkDocument | 5 services | ~35 lines | ~175 lines |
| Dependency setter methods | 5 services | ~20 lines | ~100 lines |

**Cross-Cutting Service Duplication:**

| Service | Problem | Lines Wasted |
|---------|---------|-------------|
| NoteService | 5 per-entity repo fields, 2 five-branch switch statements | ~50 lines |
| ContextService | 5 per-entity repo fields, 2 five-branch switch statements | ~100 lines |
| ResumeService | 5 per-entity repo fields, switch statements for entity lookup | ~80 lines |

**Total: ~1,180 lines of duplicated code across the service layer.**

### Solution

1. Make all 5 entity services compose `EntityService` and delegate `TransitionStatus`, `GetNextStatus`, and `makeResolveActionFn` to it
2. Refactor NoteService, ContextService, and ResumeService to use `EntityRegistry` instead of 5 per-entity repository fields
3. Replace all 5-branch switch statements with single `registry.GetRepository(entityType)` calls
4. Consolidate document operations into EntityService or a shared helper

### Impact

- Eliminate **~1,180 lines** of duplicated service code
- Bug fixes to transition logic, note resolution, or context handling apply in **one place** instead of 3-5
- Adding a 6th entity type requires **zero changes** to NoteService, ContextService, ResumeService
- Adding a 6th entity type requires only a new service file + EntityRegistry registration (no changes to EntityService)

---

## User Personas

### Persona 1: Go Developer (Maintainer)

**Goals**:
1. Fix a transition bug once, not in 5 services
2. Add a new entity type without modifying existing services
3. Understand the service architecture from EntityService, not 5 copies

**Pain Points**:
- Found a backward-detection bug in EpicService.TransitionStatus — had to check if the same bug exists in FeatureService, TaskService, BugService, ChangeCardService
- Adding ChangeCard support required adding `changeCardRepo` fields and switch branches to NoteService, ContextService, and ResumeService
- The deleted `document_helpers.go` means document linking logic is now inlined and duplicated in each entity service

---

## Current State Analysis

### Entity Services: TransitionStatus Duplication

Each entity service has its own TransitionStatus that follows this identical pattern:

```go
func (s *XxxService) TransitionStatus(ctx, key, targetStatus, opts) (*TransitionResult, error) {
    // 1. Get entity from repo
    // 2. Validate transition via workflow service
    // 3. Detect backward transition
    // 4. Create rejection note (if backward)
    // 5. Update status
    // 6. Resolve orchestrator action
    // 7. Return result
}
```

Steps 2-7 are **identical** across all 5 services. Only step 1 (repository type) and optional post-hooks (Epic counts features, Feature counts tasks) differ.

**EntityService already has this logic** — but nobody calls it.

### Cross-Cutting Services: Switch Statement Duplication

```go
// NoteService.resolveEntityID — CURRENT (duplicated pattern)
switch entityType {
case models.EntityTypeEpic:
    epic, err := s.epicRepo.GetByKey(ctx, key)
    return epic.ID, nil
case models.EntityTypeFeature:
    feature, err := s.featureRepo.GetByKey(ctx, key)
    return feature.ID, nil
case models.EntityTypeTask:
    task, err := s.taskRepo.GetByKey(ctx, key)
    return task.ID, nil
case models.EntityTypeBug:
    bug, err := s.bugRepo.GetByKey(ctx, key)
    return bug.ID, nil
case models.EntityTypeChangeCard:
    cc, err := s.changeCardRepo.GetByKey(ctx, key)
    return cc.ID, nil
}
```

**Should be:**
```go
func (s *NoteService) resolveEntityID(ctx, entityType, key) (int64, error) {
    repo, err := s.registry.GetRepository(entityType)
    if err != nil { return 0, err }
    entity, err := repo.GetByKey(ctx, key)
    if err != nil { return 0, err }
    return entity.GetID(), nil
}
```

**3 lines replace 20+, and works for any entity type including future ones.**

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want entity services to delegate TransitionStatus to EntityService so that transition logic exists in one place.

**Acceptance Criteria**:
- [ ] EpicService.TransitionStatus delegates to EntityService.TransitionStatus
- [ ] FeatureService.TransitionStatus delegates to EntityService.TransitionStatus
- [ ] TaskService.TransitionStatus delegates to EntityService.TransitionStatus
- [ ] BugService.SetStatus delegates to EntityService.TransitionStatus (with simplified features)
- [ ] ChangeCardService.SetStatus delegates to EntityService.TransitionStatus
- [ ] Entity-specific post-hooks (child counting, cascade) remain in entity services
- [ ] All existing transition tests pass

**Story 2**: As a developer, I want NoteService to use EntityRegistry so that adding a new entity type requires zero NoteService changes.

**Acceptance Criteria**:
- [ ] NoteService constructor accepts `*EntityRegistry` instead of 5 separate repos
- [ ] `resolveEntityID` uses `registry.GetRepository(entityType)` instead of switch statement
- [ ] `GetEntityDetails` uses `registry.GetRepository(entityType)` instead of switch statement
- [ ] 5 setter methods (SetEpicRepo, SetFeatureRepo, etc.) removed
- [ ] All note tests pass

**Story 3**: As a developer, I want ContextService to use EntityRegistry so that context operations work for any registered entity type.

**Acceptance Criteria**:
- [ ] ContextService constructor accepts `*EntityRegistry`
- [ ] `getContextJSON` and `setContextJSON` use registry instead of switch statements
- [ ] Per-entity repo fields removed
- [ ] All context tests pass

**Story 4**: As a developer, I want ResumeService to use EntityRegistry for entity lookup.

**Acceptance Criteria**:
- [ ] ResumeService constructor accepts `*EntityRegistry`
- [ ] Entity lookup uses registry instead of per-entity repo fields
- [ ] All resume tests pass

**Story 5**: As a developer, I want document operations (LinkDocument, UnlinkDocument) centralized so that each entity service doesn't duplicate this logic.

**Acceptance Criteria**:
- [ ] Shared document helper function or EntityService method for link/unlink
- [ ] All 5 entity services delegate to shared implementation
- [ ] Or: EntityDocumentService handles all document operations polymorphically
- [ ] Per-entity LinkDocument/UnlinkDocument methods become thin wrappers

---

### Should-Have Stories

**Story 6**: As a developer, I want service accessor wiring simplified so that adding a new entity type doesn't require 30 lines of boilerplate in service_accessors.go.

**Acceptance Criteria**:
- [ ] EntityRegistry initialized once in service_accessors.go
- [ ] Cross-cutting services (NoteService, ContextService, ResumeService) receive registry at construction
- [ ] Per-entity setter calls (SetEpicRepo, SetFeatureRepo, etc.) eliminated

---

## Requirements

### Functional Requirements

**Category: Entity Service Delegation**

1. **REQ-F-001**: EntityService Composition in Entity Services
   - **Description**: Each entity service stores an `*EntityService` field and delegates shared transition logic
   - **Priority**: Must-Have
   - **Pattern**:
     ```go
     type EpicService struct {
         repo        EpicRepository
         entitySvc   *EntityService       // shared logic
         entityRepo  EntityRepository     // adapter wrapping EpicRepository
         workflowSvc *workflow.Service
         // ... entity-specific deps
     }

     func (s *EpicService) TransitionStatus(ctx, key, target, opts) (*TransitionResult, error) {
         features := DefaultTransitionFeatures()
         features.CountChildren = true  // Epic-specific

         result, err := s.entitySvc.TransitionStatus(ctx, s.entityRepo, models.EntityTypeEpic,
             key, target, opts, features, s.makeResolveActionFn(ctx))
         if err != nil { return nil, err }

         // Epic-specific post-hook: count features
         if s.featureCounter != nil {
             result.ChildCount, _ = s.featureCounter.CountByEpic(ctx, result.EntityID)
         }
         return result, nil
     }
     ```

2. **REQ-F-002**: EntityRegistry in Cross-Cutting Services
   - **Description**: NoteService, ContextService, ResumeService accept EntityRegistry and use it for entity lookup
   - **Priority**: Must-Have
   - **Pattern**:
     ```go
     type NoteService struct {
         noteRepo  EntityNoteRepository
         registry  *EntityRegistry       // replaces 5 per-entity repo fields
     }

     func (s *NoteService) resolveEntityID(ctx, entityType, key) (int64, error) {
         repo, err := s.registry.GetRepository(entityType)
         if err != nil { return 0, err }
         entity, err := repo.GetByKey(ctx, key)
         if err != nil { return 0, err }
         return entity.GetID(), nil
     }
     ```

3. **REQ-F-003**: Document Operations Consolidation
   - **Description**: Shared implementation for LinkDocument/UnlinkDocument used by all entity services
   - **Priority**: Must-Have
   - **Acceptance Criteria**:
     - [ ] Single function or method handles document linking for any entity type
     - [ ] Per-entity services call shared implementation
     - [ ] Regression from deleted `document_helpers.go` reversed

4. **REQ-F-004**: Service Accessor Simplification
   - **Description**: Update `service_accessors.go` to initialize EntityRegistry once and pass to cross-cutting services
   - **Priority**: Should-Have
   - **Acceptance Criteria**:
     - [ ] EntityRegistry created once with all 5 entity repositories registered
     - [ ] NoteService, ContextService, ResumeService constructed with registry
     - [ ] Per-entity setter calls eliminated from accessor functions

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: Behavioral Equivalence
   - **Description**: All status transitions, note operations, context operations must produce identical results
   - **Measurement**: Existing test suite passes without modification
   - **Target**: Zero behavioral changes

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Transition Delegation**
- **Given** EpicService composes EntityService
- **When** `svc.TransitionStatus(ctx, "E21", "active", opts)` is called
- **Then** EntityService.TransitionStatus handles validation, backward detection, rejection notes, and status update
- **And** EpicService only runs its post-hook (feature counting)

**Scenario 2: Registry-Based Note Resolution**
- **Given** NoteService uses EntityRegistry
- **When** a note is added for entity type "bug" with key "B001"
- **Then** NoteService resolves the entity ID via `registry.GetRepository("bug").GetByKey(ctx, "B001")`
- **And** no switch statement is evaluated

**Scenario 3: New Entity Type (Future-Proof)**
- **Given** a hypothetical 6th entity type "milestone" is registered in EntityRegistry
- **When** a note is added for entity type "milestone" with key "M001"
- **Then** NoteService works without any code changes (registry handles lookup)

---

## Out of Scope

### Explicitly Excluded

1. **Merging Entity Services Into One**
   - **Why**: Entity-specific logic (Task dependencies, Feature progress, Epic rollups) justifies separate services
   - **Future**: Only if entity-specific logic converges significantly

2. **CLI Command Consolidation**
   - **Why**: Addressed separately in E21-F10
   - **Dependency**: This feature makes F10 easier by providing unified service methods

---

## Estimated Impact

### Lines Removed (Duplication Eliminated)

| Component | Current Lines | After Delegation | Savings |
|-----------|-------------|-----------------|---------|
| 5x TransitionStatus | ~400 | ~100 (thin wrappers + hooks) | ~300 |
| 5x makeResolveActionFn | ~200 | ~50 (entity-specific only) | ~150 |
| 3x GetNextStatus | ~75 | 0 (delegated to EntityService) | ~75 |
| 5x LinkDocument/UnlinkDocument | ~175 | ~25 (thin wrappers) | ~150 |
| NoteService switch statements | ~50 | ~6 (registry calls) | ~44 |
| ContextService switch statements | ~100 | ~12 (registry calls) | ~88 |
| ResumeService switch statements | ~80 | ~10 (registry calls) | ~70 |
| 5x dependency setters per service | ~100 | 0 (constructor injection) | ~100 |
| Service accessor wiring | ~175 | ~75 (registry-based) | ~100 |
| **TOTAL** | **~1,355** | **~278** | **~1,077** |

---

## Dependencies & Integrations

### Dependencies

- **E21-F01** (completed): Entity interface, EntityRepository adapters, EntityRegistry
- **EntityService** (exists): TransitionStatus, GetNextStatus already implemented

### Integration Points

- **service_accessors.go**: Must be updated to create EntityRegistry and pass to services
- **cmd/server/services.go**: HTTP server wiring must also use EntityRegistry pattern
- **All entity service tests**: Must be updated to verify delegation (mock EntityService)

---

## Success Metrics

### Primary Metrics

1. **Duplication Reduction**
   - **Target**: ~1,077 lines of duplicated code eliminated
   - **Measurement**: Diff line count before/after

2. **Switch Statement Elimination**
   - **Target**: All entity-type switch statements in NoteService, ContextService, ResumeService replaced with registry lookups
   - **Measurement**: `grep -r "case models.EntityType" internal/services/` returns 0 matches in cross-cutting services

3. **New Entity Type Cost**
   - **Target**: Adding a new entity type requires 0 changes to NoteService, ContextService, ResumeService
   - **Measurement**: Code review of hypothetical 6th entity type addition

---

*Last Updated*: 2026-03-20
