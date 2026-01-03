# E07-F16 Implementation Plan: Workflow Config Integration

**Feature:** Workflow Config Integration for Status Display
**Epic:** E07 - Enhancements
**Total Tasks:** 15 (13 new + 2 existing)
**Estimated Effort:** 16-23 hours

---

## Quick Summary

This feature centralizes workflow configuration access and integrates it into all status display commands (feature get, epic get, task list). It eliminates code duplication and enables dynamic status display based on workflow config.

**Key Benefits:**
- ✅ Single source of truth for workflow config (WorkflowService)
- ✅ Dynamic status display with colors and metadata
- ✅ Workflow-aware ordering in all commands
- ✅ Eliminates hardcoded status values

---

## Recommended Execution Order

### Phase 1: Core Infrastructure (6-9 hours)
**Build the foundation - WorkflowService**

1. **T-E07-F16-003** ⭐ - Implement WorkflowService core
   - Priority: 8, Agent: backend
   - Creates `internal/workflow/service.go` and `types.go`
   - No dependencies

2. **T-E07-F16-004** - Implement status formatter
   - Priority: 7, Agent: backend
   - Depends on: T-E07-F16-003

3. **T-E07-F16-005** - Write comprehensive unit tests for WorkflowService
   - Priority: 8, Agent: backend
   - Depends on: T-E07-F16-003, T-E07-F16-004

### Phase 2: Repository Enhancement (4-5 hours)
**Update TaskRepository to return ordered, metadata-rich status breakdowns**

4. **T-E07-F16-006** - Update GetStatusBreakdown signature
   - Priority: 8, Agent: backend
   - Depends on: T-E07-F16-003

5. **T-E07-F16-007** - Implement status ordering logic
   - Priority: 7, Agent: backend
   - Depends on: T-E07-F16-006

6. **T-E07-F16-008** - Update repository tests for StatusCount changes
   - Priority: 7, Agent: backend
   - Depends on: T-E07-F16-006, T-E07-F16-007

### Phase 3: Feature Get Command (4-5 hours)
**Integrate WorkflowService into feature get display**

7. **T-E07-F16-009** - Inject WorkflowService into feature get command
   - Priority: 8, Agent: backend
   - Depends on: T-E07-F16-003

8. **T-E07-F16-010** ⭐ - Update renderFeatureDetails for workflow-aware display
   - Priority: 8, Agent: backend
   - Depends on: T-E07-F16-006, T-E07-F16-009

9. **T-E07-F16-011** - Add phase grouping to feature display (optional)
   - Priority: 5, Agent: backend
   - Depends on: T-E07-F16-010

### Phase 4: Refactor Task Creation (1-2 hours)
**Remove code duplication from task creation**

10. **T-E07-F16-012** - Replace getInitialTaskStatus with WorkflowService
    - Priority: 7, Agent: backend
    - Depends on: T-E07-F16-003
    - Completes refactor from T-E07-F16-002

### Phase 5: Testing & Documentation (5-6 hours)
**Comprehensive testing and user documentation**

11. **T-E07-F16-013** ⭐ - Write integration tests for workflow-aware feature get
    - Priority: 8, Agent: backend
    - Depends on: All implementation tasks

12. **T-E07-F16-014** - Update CLI_REFERENCE.md for workflow features
    - Priority: 6, Agent: general
    - Depends on: All implementation tasks

13. **T-E07-F16-015** - Add workflow config examples
    - Priority: 5, Agent: general
    - Creates example configs in `docs/examples/workflows/`

### Existing Tasks

14. **T-E07-F16-001** - Display 'all tasks completed' when listing completed feature
    - Priority: 5, Agent: general
    - Independent enhancement to task list messaging

15. **T-E07-F16-002** ✅ - Use workflow config entry status for new task creation
    - Priority: 7, Agent: general
    - Status: ready_for_code_review
    - Already implemented! Needs code review

---

## Dependency Graph

```
T-E07-F16-003 (WorkflowService core) ⭐
    ├─→ T-E07-F16-004 (Formatter)
    │       └─→ T-E07-F16-005 (Unit tests)
    ├─→ T-E07-F16-006 (GetStatusBreakdown) ⭐
    │       ├─→ T-E07-F16-007 (Ordering logic)
    │       │       └─→ T-E07-F16-008 (Repository tests)
    │       └─→ T-E07-F16-010 (renderFeatureDetails) ⭐
    │               └─→ T-E07-F16-011 (Phase grouping)
    ├─→ T-E07-F16-009 (Inject WorkflowService)
    │       └─→ T-E07-F16-010 (renderFeatureDetails) ⭐
    └─→ T-E07-F16-012 (Refactor task creation)

All implementation tasks
    ├─→ T-E07-F16-013 (Integration tests) ⭐
    ├─→ T-E07-F16-014 (Update docs)
    └─→ T-E07-F16-015 (Workflow examples)
```

⭐ = Critical path tasks

---

## Critical Path

The fastest path to completion focuses on these tasks:

1. T-E07-F16-003 - WorkflowService core (3-4h)
2. T-E07-F16-006 - GetStatusBreakdown (2-3h)
3. T-E07-F16-009 - Inject WorkflowService (1h)
4. T-E07-F16-010 - renderFeatureDetails (2-3h)
5. T-E07-F16-013 - Integration tests (2-3h)

**Critical Path Total: 10-14 hours**

---

## Task Status Summary

- **Draft:** 13 tasks (ready to start)
- **Ready for Code Review:** 1 task (T-E07-F16-002)
- **Todo:** 1 task (T-E07-F16-001)

---

## Next Steps

1. **Code Review:** Review T-E07-F16-002 (already implemented)
2. **Start Phase 1:** Begin with T-E07-F16-003 (WorkflowService core)
3. **Parallel Work:** After Phase 1, can parallelize:
   - Phase 2 (Repository) and Phase 4 (Task creation refactor)
   - Both depend only on T-E07-F16-003

---

## Documentation Resources

**Design Documents:**
- `/dev-artifacts/2026-01-02-feature-get-workflow-enhancement/architectural-review-and-design.md` - Full design spec
- `/dev-artifacts/2026-01-02-feature-get-workflow-enhancement/implementation-examples.md` - Code examples
- `/dev-artifacts/2026-01-02-feature-get-workflow-enhancement/QUICKREF.md` - Quick reference

**Implementation Guide:**
- See task files in `docs/plan/E07-enhancements/E07-F16-improved-empty-task-list-messaging/tasks/`
- Each task has detailed acceptance criteria and test plans

---

## Success Metrics

✅ **WorkflowService created** and tested
✅ **Feature get command** displays workflow-aware status breakdown
✅ **Epic get command** uses workflow config (future task)
✅ **Task list command** uses workflow config (future task)
✅ **All tests pass** (unit + integration)
✅ **Documentation updated** with examples
✅ **Code duplication eliminated** from task creation
