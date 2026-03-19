---
epic_key: E21
title: Entity Polymorphism and Duplication Reduction
description: Introduce a shared Entity interface and generic services to eliminate ~1,255+ lines of cross-entity duplication, enabling new entity types to be added in days instead of weeks.
---

# Entity Polymorphism and Duplication Reduction

**Epic Key**: E21

---

## Goal

### Problem

The Shark codebase has five entity types (Epic, Feature, Task, Bug, ChangeCard) that share 10 common fields and 60-70% identical service logic, yet each is implemented as a fully independent struct with no shared interface. Every cross-cutting feature (notes, context, documents, status transitions, templates, display) must be independently implemented for each entity type. Adding a new entity type requires touching 15+ files across models, repositories, services, CLI commands, and accessors. This creates a maintenance burden where bug fixes must be applied 5 times and new features require repetitive copy-paste implementation.

### Solution

Introduce a `models.Entity` interface that all entity types implement, providing polymorphic access to shared fields (ID, Key, Title, Status, etc.). Build a shared `EntityService` for cross-cutting operations (status transitions, document linking, orchestrator actions) and a registry pattern that replaces per-entity setter methods. Entity-specific services compose the shared service and add only their unique logic (dependency graphs, progress rollups, severity, approvals).

### Impact

- Reduce cross-entity duplication by ~1,255+ lines (conservative, services only)
- Adding a 6th entity type goes from ~2 weeks / 15+ files to ~2 days / 3-4 files
- Bug fixes to cross-cutting logic (status transitions, notes, context) apply once instead of 5 times
- Consistent behavior guaranteed across all entity types

---

## Business Value

**Rating**: High

This is a foundational architectural improvement that directly reduces development cost and defect rate. Every future feature that spans entity types (and most do) will take 1/5th the implementation effort. The current duplication has already caused behavioral inconsistencies between entities and makes the codebase increasingly difficult to maintain as entity count grows.

---

## Current State Analysis

### Scale of the Problem

| Layer | Files | Total Lines | Entity Types |
|-------|-------|-------------|--------------|
| Models | 5 | ~401 | Epic, Feature, Task, Bug, ChangeCard |
| Repositories | 5 | ~4,227 | Epic, Feature, Task, Bug, ChangeCard |
| Services | 5 (core) + 3 (cross-cutting) | ~5,806 | All |
| CLI Commands | ~76 | ~24,041 | All |
| CLI Accessors | 2 | ~647 | All |
| **Total** | **~96** | **~35,122** | |

### Shared Fields Across All Entities

Every entity has these fields (or near-equivalents):

| Field | Epic | Feature | Task | Bug | ChangeCard |
|-------|------|---------|------|-----|------------|
| `ID` (int64) | Y | Y | Y | Y | Y |
| `Key` (string) | Y | Y | Y | Y | Y |
| `Title` (string) | Y | Y | Y | Y | Y |
| `Slug` (*string or string) | Y | Y | Y | Y | Y |
| `Description` (*string) | Y | Y | Y | Y | Y |
| `Status` (typed string) | Y | Y | Y | Y | Y |
| `FilePath` (*string or string) | Y | Y | Y | Y | Y |
| `ContextData` (*string) | Y | Y | Y | Y | Y |
| `CreatedAt` (time.Time) | Y | Y | Y | Y | Y |
| `UpdatedAt` (time.Time) | Y | Y | Y | Y | Y |

10 shared fields. Only the Status type wrapper differs (EpicStatus, FeatureStatus, etc.), but they are all `string` underneath.

### Entity-Specific Fields

- **Epic only:** Priority, BusinessValue
- **Feature only:** EpicID (parent), ProgressPct, StatusOverride, ExecutionOrder
- **Task only:** FeatureID (parent), AgentType, Priority, DependsOn, AssignedAgent, BlockedReason, ExecutionOrder, completion metadata, RejectionCount
- **Bug only:** Severity, LinkedEntityType, LinkedEntityKey
- **ChangeCard only:** Priority, RequestedBy, AssignedTo, EpicID, FeatureID, RelatedTaskID, Justification, ImpactAnalysis, RollbackPlan

### Duplicated Service Patterns

#### CRUD Operations (duplicated 5x)

```go
// EpicService.GetEpic
func (s *EpicService) GetEpic(ctx context.Context, key string) (*models.Epic, error) {
    epic, err := s.repo.GetByKey(ctx, key)
    if err != nil { return nil, fmt.Errorf("failed to get epic %s: %w", key, err) }
    if epic == nil { return nil, fmt.Errorf("epic not found: %s", key) }
    return epic, nil
}

// BugService.GetBug -- structurally identical
func (s *BugService) GetBug(ctx context.Context, key string) (*models.Bug, error) {
    bug, err := s.repo.GetByKey(ctx, key)
    if err != nil { return nil, fmt.Errorf("failed to get bug %s: %w", key, err) }
    return bug, nil
}
```

#### Status Transition Operations (duplicated 5x, ~80 lines each)

The TransitionStatus method is nearly identical across all services:
1. Get entity by key
2. Extract current status string
3. Validate transition (unless forced)
4. Normalize status
5. Check backward transition
6. Require reason for backward
7. Update entity status
8. Create rejection note if backward
9. Count children for warning
10. Resolve orchestrator action
11. Return TransitionResult

#### Document Operations (duplicated 5x)

LinkDocument, UnlinkDocument, ListRelatedDocumentsByKey -- each entity service has its own wrapper that checks writableDocRepo, looks up entity by key, then calls the same shared helper.

#### Orchestrator Action Resolution (duplicated 5x)

Each entity service has its own resolveAction that does: get workflow → check metadata → get action → build placeholders → return.

### Duplicated Cross-Cutting Services

**NoteService** -- 5-branch switch on EntityType in both `resolveEntityID` and `GetEntityDetails`. Each branch does the same thing: `repo.GetByKey(ctx, key)` → extract `.ID`.

**ContextService** -- Two 5-branch switch statements (`getContextJSON` and `setContextJSON`). Each branch: `repo.GetByKey` → get/set ContextData.

### Estimated Duplication

| Duplicated Pattern | Instances | Lines Each | Total |
|---|---|---|---|
| CRUD service methods | 5x | ~40 | ~200 |
| Status transition methods | 5x | ~80 | ~400 |
| Document linking methods | 5x | ~30 | ~150 |
| Action resolution methods | 5x | ~25 | ~125 |
| Repository interface defs | ~15 | ~5 | ~75 |
| CLI accessor functions | 5x | ~25 | ~125 |
| NoteService switch branches | 2x5 | ~8 | ~80 |
| ContextService switch branches | 2x5 | ~10 | ~100 |
| **Total** | | | **~1,255** |

Conservative estimate -- does not include CLI command duplication (~24,000 lines across 76 files).

---

## Organizational Tier Model

The only fundamental difference between entities is their tier in the organizational hierarchy:

| Entity | Tier | Has Parent | Has Children | Key Pattern |
|--------|------|-----------|--------------|-------------|
| Epic | Top | No | Features | `E##` |
| Feature | Mid | Epic | Tasks | `E##-F##` |
| Task | Leaf | Feature | None | `E##-F##-###` |
| Bug | Standalone | No (linked optionally) | None | `B###` |
| ChangeCard | Standalone | No (linked optionally) | None | `CC-###` |

### Entity-Specific Business Logic (What NOT to Unify)

- **Task:** Dependency graph, agent assignment, completion metadata, work sessions
- **Feature/Epic:** Progress rollup, status cascade to children
- **Bug:** Severity levels, triage workflow
- **ChangeCard:** Approval workflow, impact analysis, rollback plan

---

## Proposed Architecture

### Core Entity Interface

```go
// internal/models/entity.go
type Entity interface {
    GetID() int64
    GetKey() string
    GetTitle() string
    GetSlug() string
    GetEntityType() EntityType
    GetStatus() string
    SetStatus(status string)
    GetDescription() string
    GetFilePath() string
    GetContextData() *string
    SetContextData(data *string)
    GetCreatedAt() time.Time
    GetUpdatedAt() time.Time
    Validate() error
}
```

Each existing model implements this via simple accessor methods. Non-invasive -- existing code using `epic.Key` continues to work.

### Generic Entity Repository Interface

```go
type EntityRepository interface {
    GetByKey(ctx context.Context, key string) (models.Entity, error)
    GetByID(ctx context.Context, id int64) (models.Entity, error)
    Create(ctx context.Context, entity models.Entity) error
    Update(ctx context.Context, entity models.Entity) error
    Delete(ctx context.Context, id int64) error
    UpdateStatus(ctx context.Context, id int64, status string) error
    GetContextData(ctx context.Context, id int64) (*string, error)
    UpdateContextData(ctx context.Context, id int64, data *string) error
}
```

Existing typed repositories remain -- EntityRepository is an additional interface for cross-cutting services.

### Shared EntityService

```go
type EntityService struct {
    workflowSvc     *workflow.Service
    noteRepo        EntityNoteRepository
    writableDocRepo WritableDocumentRepository
}

func (s *EntityService) TransitionStatus(ctx, repo EntityRepository, key, target string, opts TransitionOptions) (*TransitionResult, error) {
    // SINGLE implementation for all entity types
}
```

Entity-specific services compose the shared service:

```go
type EpicService struct {
    repo        EpicRepository   // typed repo for epic-specific queries
    entitySvc   *EntityService   // shared cross-cutting logic
    featureRepo EpicFeatureCounter
}
```

### Registry Pattern

```go
type EntityRegistry struct {
    repos    map[models.EntityType]EntityRepository
    services map[models.EntityType]EntityTypeService
}
```

Replaces all `SetBugRepo()`/`SetChangeCardRepo()` setters with a single registration mechanism.

---

## Features

### F01: Entity Interface Foundation
Define the Entity interface, implement on all 5 model structs, create EntityRepository interface with adapters.
- **Risk:** Zero (purely additive)
- **Effort:** M (3-5 tasks)

### F02: Cross-Cutting Service Unification
Refactor NoteService, ContextService, ResumeService to use EntityRepository map and registry pattern instead of per-entity repos + switch statements.
- **Risk:** Medium
- **Effort:** L (5-8 tasks)
- **Depends on:** F01

### F03: Status Transition Unification
Extract shared TransitionStatus into EntityService. Refactor all 5 entity services to delegate + add entity-specific logic.
- **Risk:** Medium
- **Effort:** L (5-8 tasks)
- **Depends on:** F01

### F04: Document Operations Unification
Move LinkDocument/UnlinkDocument/ListRelatedDocumentsByKey to EntityService.
- **Risk:** Low
- **Effort:** S (2-3 tasks)
- **Depends on:** F01

### F05: Template Placeholder Unification
Create EntityPlaceholders base function. Entity-specific placeholder functions extend from base.
- **Risk:** Low
- **Effort:** S (2-3 tasks)
- **Depends on:** F01

### F06: Generic CLI Commands
Create generic EntityCommand builder for get, list, create, update, delete, note, context commands. Entity-specific commands remain as extensions.
- **Risk:** Medium (high effort)
- **Effort:** XL (10+ tasks, incremental per entity)
- **Depends on:** F01, F02, F03

---

## Success Criteria

- All 5 entity types implement the Entity interface with full test coverage
- NoteService, ContextService, ResumeService have zero per-entity switch statements
- Status transition logic exists in one place (EntityService) with entity-specific extensions
- Adding a hypothetical 6th entity type requires modifying ≤5 files (vs current 15+)
- All existing tests pass with zero behavioral changes
- `make fmt && make lint && make test` passes at each phase boundary

---

## Trade-offs and Risks

| Risk | Mitigation |
|------|------------|
| Over-abstraction losing type safety | Keep typed repos/services for entity-specific ops. Entity interface only for shared logic. |
| Performance from interface indirection | Negligible in Go (~1-2ns per dispatch) |
| Migration disruption | Incremental phases. Each self-contained. Existing typed code works alongside new generic code. |
| Loss of explicitness | Clear naming: `EntityService` for shared, `EpicService` for epic-specific |

---

## Reference

- **Architecture Review:** [entity-polymorphism-architecture-review.md](./entity-polymorphism-architecture-review.md)

---

*Last Updated*: 2026-03-18
