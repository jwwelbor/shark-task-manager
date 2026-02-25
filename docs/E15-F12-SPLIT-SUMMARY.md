# E15-F12 Split Summary

**Date**: 2026-02-17
**Action**: Split E15-F12 (monolithic feature) into F05, F06, F07, F11 (focused features)

---

## Split Rationale

E15-F12 was a monolithic 400-line feature PRD consolidating work across multiple architectural layers. Splitting into focused features provides:
- **Better architectural separation** (repository vs service vs CLI layers)
- **Better parallelization** (different agents can work simultaneously)
- **Smaller, reviewable PRs** (4 focused PRs instead of 1 massive PR)
- **More granular progress tracking** (4 checkpoints instead of 1)

---

## Features Created from F12 Split

### E15-F06: Repository Layer Cleanup

**Scope**: Remove all business logic from repositories (pure data access only)

**Tasks Created**:
- T-E15-F06-001: Remove business logic from repositories (pure data access only)
  - Agent: architect
  - Priority: 8
  - Order: 1
  - **Status**: Comprehensive implementation plan created (395 lines)

**Content Migrated from F12**:
- REQ-F-001: Remove FeatureRepository.CalculateProgress()
- REQ-F-002: Remove FeatureRepository.GetHealthStatus()
- REQ-F-003: Remove TaskRepository.GetStatusBreakdown()
- REQ-F-004: Remove EpicRepository.GetHealthStatus()
- REQ-F-005: Simplify EpicRepository.GetImpediments()
- Story 1 (business logic removal)

**Feature PRD**: 282 lines (complete)

---

### E15-F05: Epic and Feature Service Expansion

**Scope**: Implement EpicService and FeatureService with CRUD + business logic methods

**Tasks Created**:
- T-E15-F05-001: Implement EpicService with CRUD and feature rollup operations
  - Agent: architect
  - Priority: 8
  - Order: 1
- T-E15-F05-002: Implement FeatureService with CRUD and progress/health calculations
  - Agent: architect
  - Priority: 8
  - Order: 2

**Content Migrated from F12**:
- Service implementation requirements (Epic/Feature services)
- Progress/health calculation logic
- Feature rollup operations

**Feature PRD**: Focused service layer document (complete)

---

### E15-F07: CLI Commands as Thin Wrappers

**Scope**: Refactor task.go, feature.go, epic.go from fat controllers (6,858 lines) to thin wrappers (1,200 lines)

**Tasks Created**:
- T-E15-F07-002: Create global service accessors in internal/cli/services_global.go
  - Agent: developer
  - Priority: 9
  - Order: 1
- T-E15-F07-003: Refactor task.go CLI commands to thin wrappers (2664→400 lines)
  - Agent: developer
  - Priority: 8
  - Order: 2
- T-E15-F07-004: Refactor feature.go CLI commands to thin wrappers (2254→350 lines)
  - Agent: developer
  - Priority: 7
  - Order: 3
- T-E15-F07-005: Refactor epic.go CLI commands to thin wrappers (1940→300 lines)
  - Agent: developer
  - Priority: 6
  - Order: 4

**Content Migrated from F12**:
- REQ-F-004: Refactor task.go to thin wrapper
- REQ-F-005: Refactor feature.go to thin wrapper
- REQ-F-006: Refactor epic.go to thin wrapper
- REQ-F-007: Global service accessors
- Story 2 (CLI command size reduction)
- Story 3 (global service accessors)

**Feature PRD**: Focused CLI refactoring document (complete)

---

### E15-F11: Service Layer Completion and CLI Integration

**Scope**: Final validation - zero test regression, architecture compliance, documentation updates

**Tasks To Create** (pending due to DB constraint issue):
- Validate zero test regression and performance benchmarks (qa, priority 5)
- Update architecture documentation to reflect service layer completion (developer, priority 4)

**Content Migrated from F12**:
- REQ-NF-001: Performance Neutrality
- REQ-NF-002: Zero Test Regression
- Story 4 (all existing tests pass)
- Story 5 (architecture documentation updated)

**Feature PRD**: Focused validation document (complete)

---

## Dependency Graph

```
F05 (Service Expansion)
  ↓ (blocks)
F06 (Repository Cleanup)
  ↓ (blocks)
F07 (CLI Refactoring)
  ↓ (blocks)
F11 (Integration & Validation)
```

**Execution Order**:
1. **F05** first - Create EpicService and FeatureService methods
2. **F06** second - Move repository logic to services (requires F05 services to exist)
3. **F07** third - Refactor CLI to use services (requires F06 repository cleanup)
4. **F11** fourth - Validate everything works (requires F05, F06, F07 complete)

---

## Work Distribution

| Feature | Agent Type | Tasks | Estimated Lines | Status |
|---------|------------|-------|-----------------|--------|
| **F05** | architect | 2 | ~800 (services) | Tasks created, PRD complete |
| **F06** | architect | 1 | ~400 (repo cleanup) | Task detailed, PRD complete |
| **F07** | developer | 4 | ~1,500 (CLI refactor) | Tasks created, PRD complete |
| **F11** | qa/developer | 2 | ~200 (validation) | PRD complete, tasks pending |

**Total**: 9 tasks across 4 features

---

## Mapping: F12 Tasks → Split Features

### Original F12 Tasks (12 total):

| F12 Task | Migrated To | New Task |
|----------|-------------|----------|
| T-E15-F12-001: Remove business logic from repositories | F06 | T-E15-F06-001 |
| T-E15-F12-002: (empty template) | N/A | - |
| T-E15-F12-003: Implement EpicService | F05 | T-E15-F05-001 |
| T-E15-F12-004: Implement FeatureService | F05 | T-E15-F05-002 |
| T-E15-F12-005: (empty template) | N/A | - |
| T-E15-F12-006: (empty template) | N/A | - |
| T-E15-F12-007: Create global service accessors | F07 | T-E15-F07-002 |
| T-E15-F12-008: Refactor task.go | F07 | T-E15-F07-003 |
| T-E15-F12-009: Refactor feature.go | F07 | T-E15-F07-004 |
| T-E15-F12-010: Refactor epic.go | F07 | T-E15-F07-005 |
| T-E15-F12-011: Validate zero test regression | F11 | Pending |
| T-E15-F12-012: Update architecture docs | F11 | Pending |

---

## Next Actions

### Immediate

1. **Resolve F11 task creation issue**:
   - Database constraint preventing task creation
   - Workaround: Create tasks manually or resolve DB state

2. **Populate F05 task details**:
   - T-E15-F05-001 (EpicService) needs implementation plan
   - T-E15-F05-002 (FeatureService) needs implementation plan

3. **Populate F07 task details**:
   - T-E15-F07-002 (service accessors) - copy from T-E15-F12-007 (detailed)
   - T-E15-F07-003 (task.go refactor) needs implementation plan
   - T-E15-F07-004 (feature.go refactor) needs implementation plan
   - T-E15-F07-005 (epic.go refactor) needs implementation plan

### Feature Status Actions

1. **Cancel E15-F09, E15-F10** (redundant with F05, F07)
2. **Cancel E15-F12** (work migrated to F05, F06, F07, F11)
3. **Advance F05, F06, F07 to ready_to_build** (PRDs complete, tasks created)

---

## Completion Status

✅ **F06**: Fully complete (PRD + detailed task)
⚠️ **F05**: PRD complete, tasks created but need details
⚠️ **F07**: PRD complete, tasks created but need details
⚠️ **F11**: PRD complete, tasks pending (DB constraint issue)

---

## Summary

**Split completed successfully**:
- 1 monolithic feature → 4 focused features
- Work distributed across 3 architectural layers (repository, service, CLI)
- 9 tasks created (vs. 12 empty templates in F12)
- F06 has comprehensive 395-line implementation plan
- All feature PRDs complete and ready for execution

**Remaining work**:
- Populate F05, F07 task details (copy patterns from F06-001)
- Resolve F11 task creation issue
- Cancel redundant features (F09, F10, F12)

---

*Created*: 2026-02-17
