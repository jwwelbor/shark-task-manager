---
feature_key: E17-F14-service-srp-refactoring
epic_key: E17
title: Service SRP Refactoring
description: Split oversized services (TaskService, FeatureService, EpicService) into focused sub-services following SRP, with a shared EntityDocumentService.
---

# Service SRP Refactoring

**Feature Key**: E17-F14-service-srp-refactoring

---

## Goal

### Problem

Three core services have grown far beyond maintainable size:
- **TaskService**: 2,279 lines, 66+ methods mixing CRUD, lifecycle, dependencies, documents, history, and analytics
- **FeatureService**: 1,400 lines, 43 methods mixing CRUD, lifecycle, progress, health, documents, and paths
- **EpicService**: 1,386 lines, 32 methods mixing CRUD, lifecycle, completion orchestration, analytics, and documents

All three contain nearly identical document-linking methods (Link, Unlink, List) that duplicate logic across entity types.

### Solution

Extract cohesive responsibility groups into focused sub-services while preserving the existing public API through delegation. Create a shared `EntityDocumentService` that eliminates ~300 lines of duplicated document-linking code.

### Impact

- All service files ≤500 lines
- Zero CLI command or HTTP handler changes (delegation preserves public API)
- Shared document service eliminates code duplication
- Improved testability through smaller, focused units

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Shared EntityDocumentService
   - Create `entity_document_service.go` parameterized by entity type
   - Replace duplicate Link/Unlink/List methods across all three services
   - Absorb existing `document_helpers.go` helpers
   - Priority: Must-Have

2. **REQ-F-002**: TaskService decomposition
   - Extract `TaskDependencyService` (~250 lines): dependency graph, cycle detection, relationships
   - Extract `TaskHistoryService` (~150 lines): history, work sessions, analytics
   - Extract `TaskQueryService` (~200 lines): listing, filtering, pagination, display data
   - Core `task_service.go` retains CRUD + lifecycle (~450 lines)
   - Priority: Must-Have

3. **REQ-F-003**: FeatureService decomposition
   - Extract `FeatureProgressService` (~200 lines): progress calculation, status breakdown, health, action items
   - Core `feature_service.go` retains CRUD + lifecycle + completion (~450 lines)
   - Priority: Must-Have

4. **REQ-F-004**: EpicService decomposition
   - Extract `EpicAnalyticsService` (~250 lines): progress, rollups, impediments, health, display data
   - Core `epic_service.go` retains CRUD + lifecycle + completion (~450 lines)
   - Priority: Must-Have

### Non-Functional Requirements

1. **REQ-NF-001**: No public API changes
   - All existing method signatures on TaskService/FeatureService/EpicService remain unchanged
   - CLI commands and HTTP handlers require zero modifications
   - Delegation is transparent to callers

2. **REQ-NF-002**: Test coverage preservation
   - All existing tests continue to pass without modification
   - New extracted services get their own test files
   - `make fmt && make lint && make test` passes after each extraction

---

## Acceptance Criteria

### Feature-Level Acceptance

- [ ] No service file exceeds 500 lines
- [ ] `entity_document_service.go` replaces all duplicate document methods
- [ ] `document_helpers.go` is removed (absorbed into new service)
- [ ] All existing tests pass unchanged
- [ ] `make fmt && make lint && make test` passes
- [ ] `make build` succeeds
- [ ] CLI smoke test (`shark task list`, `shark feature list`, `shark epic list`, `shark status`) works

---

## Out of Scope

1. **Changing CLI command implementations** - Commands continue calling parent services
2. **Changing constructor signatures** - New sub-services injected via setter methods
3. **Adding new functionality** - Pure structural refactoring only
4. **Test file splitting** - Existing test files remain as-is; new test files added for extracted services

---

## Dependencies

- T-E17-F14-001 must complete first (creates shared EntityDocumentService used by T-E17-F14-002 and T-E17-F14-003)

---

*Last Updated*: 2026-02-28
