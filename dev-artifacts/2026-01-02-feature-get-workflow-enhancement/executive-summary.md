# Executive Summary: Feature Get Workflow Enhancement

**Date:** 2026-01-02
**Review Type:** Code Review + Architecture Design
**Scope:** Apply workflow config logic to feature get screen (and related commands)

---

## 🎯 Objective

Enhance the feature get screen to display task statuses dynamically based on workflow configuration, instead of using hardcoded status values.

---

## ✅ Code Review: T-E07-F16-002

### Quality Score: **8/10**

**Strengths:**
- ✅ Clean separation of concerns
- ✅ Proper error handling with graceful fallback
- ✅ Well-documented code
- ✅ Leverages existing config infrastructure (with caching)

**Weaknesses:**
- ⚠️ Pattern creates code duplication risk
- ⚠️ Hardcoded config path (magic string)
- ⚠️ Limited use of workflow metadata

**Verdict:** Good implementation, but needs abstraction to prevent technical debt.

---

## 🏗️ Proposed Architecture

### Current Problems

1. **Hardcoded Statuses** - `GetStatusBreakdown()` returns only 6 hardcoded statuses
2. **Random Ordering** - Go map iteration is non-deterministic
3. **No Metadata** - Missing colors, descriptions, phases from workflow config
4. **Code Duplication** - Each command reconstructs workflow config path

### Solution: WorkflowService

Create a **shared service** to centralize workflow config access:

```
CLI Commands (feature, epic, task)
         ↓
   WorkflowService (NEW)
         ↓
   config.LoadWorkflowConfig() (existing)
```

**Benefits:**
- Single source of truth for workflow config
- Consistent status ordering across all commands
- Rich status metadata (colors, descriptions, phases)
- Easy to test and maintain

---

## 📦 Deliverables

### Phase 1: Core Infrastructure
- `internal/workflow/service.go` - WorkflowService implementation
- `internal/workflow/formatter.go` - Status formatting utilities
- Unit tests with >80% coverage

### Phase 2: Repository Enhancement
- Update `TaskRepository.GetStatusBreakdown()`:
  - Return ordered slice instead of map
  - Include ALL workflow statuses (with zero counts)
  - Add metadata (phase, description)

### Phase 3: Feature Get Enhancement
- Inject WorkflowService
- Display statuses in workflow order
- Add color-coded output
- Show status descriptions
- Group tasks by phase (optional)

### Phase 4: Refactor Existing Code
- Update task creation to use WorkflowService
- Remove `getInitialTaskStatus()` method
- Eliminate config path duplication

### Phase 5: Extend to Other Commands
- Apply same pattern to epic get
- Apply same pattern to task list
- Ensure consistent UX

---

## 📊 API Contracts

### WorkflowService

```go
type Service struct {
    projectRoot string
    workflow    *config.WorkflowConfig
}

func NewService(projectRoot string) *Service
func (s *Service) GetInitialStatus() models.TaskStatus
func (s *Service) GetAllStatuses() []string  // Ordered by phase
func (s *Service) GetStatusMetadata(status string) config.StatusMetadata
func (s *Service) FormatStatusForDisplay(status string, colorEnabled bool) FormattedStatus
```

### Enhanced TaskRepository

```go
type StatusCount struct {
    Status      models.TaskStatus
    Count       int
    Phase       string
    Description string
}

func (r *TaskRepository) GetStatusBreakdown(ctx context.Context, featureID int64) ([]StatusCount, error)
```

---

## 🎨 User Experience Improvements

### Before (Current)

```
Task Status Breakdown
ready_for_review       3
in_progress           2
todo                  5
```

**Issues:**
- Random order
- No context (what does "ready_for_review" mean?)
- No visual hierarchy

### After (Enhanced)

```
Task Status Breakdown
Status                      Count  Phase        Description
draft                       5      planning     Task created but not yet refined
ready_for_refinement        0      planning     Awaiting specification and analysis
ready_for_development       2      development  Spec complete, ready for implementation
ready_for_qa                3      qa           Ready for quality assurance testing
completed                   0      done         Task finished and approved
```

**Improvements:**
- ✅ Workflow-defined ordering
- ✅ Descriptive labels
- ✅ Phase grouping
- ✅ All statuses shown (including zero counts)
- ✅ Color-coded (terminal only)

---

## ⏱️ Estimated Effort

| Phase | Deliverable | Hours |
|-------|-------------|-------|
| 1 | WorkflowService + tests | 4-6 |
| 2 | TaskRepository enhancement | 2-3 |
| 3 | Feature get enhancement | 3-4 |
| 4 | Task creation refactor | 1-2 |
| 5 | Other commands | 2-3 |
| 6 | Testing & docs | 4-5 |
| **Total** | | **16-23** |

---

## ⚠️ Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Breaking changes | Low | High | Maintain backward compatibility |
| Performance regression | Low | Medium | Benchmark, use caching |
| Config parsing errors | Medium | Medium | Graceful fallback to default |
| Terminal color issues | Medium | Low | Respect --no-color flag |

**Primary Mitigation Strategy:** Comprehensive integration tests + gradual rollout

---

## ✅ Success Criteria

### Functional
- [ ] Feature get displays workflow-ordered statuses
- [ ] All workflow statuses shown (including zero counts)
- [ ] Status colors applied (unless --no-color)
- [ ] Status descriptions shown
- [ ] Backward compatible with projects lacking workflow config

### Non-Functional
- [ ] No performance regression (< 1ms overhead)
- [ ] Code duplication removed
- [ ] Test coverage > 80%
- [ ] Documentation updated

---

## 🚀 Recommended Next Steps

1. **Review** this design with team
2. **Create shark tasks** for each phase
3. **Implement Phase 1** (WorkflowService)
4. **Write integration tests**
5. **Roll out** to feature get command
6. **Extend** to other commands

---

## 📚 Related Documents

- **Full Design:** `architectural-review-and-design.md`
- **Code Examples:** `implementation-examples.md`
- **Reference Implementation:** `/internal/taskcreation/creator.go:446-467`
- **Workflow Config Spec:** `docs/specs/configurable-status-workflow.md`

---

## 💡 Key Insights

1. **DRY Principle:** Current implementation creates technical debt through duplication
2. **Workflow as First-Class Citizen:** Status workflow should drive all status display logic
3. **User Experience:** Metadata (colors, descriptions, phases) dramatically improve UX
4. **Backward Compatibility:** Essential for gradual adoption in existing projects
5. **Separation of Concerns:** Config parsing (existing) vs. workflow business logic (new service)

---

## 🎓 Lessons from T-E07-F16-002

**What Worked Well:**
- Graceful fallback when config missing
- Caching for performance
- Clear documentation

**What to Improve:**
- Extract to shared service (avoid duplication)
- Use more workflow metadata (colors, phases)
- Centralize config path construction

**Overall Assessment:** Solid implementation that provides a good foundation for enhancement, but needs architectural refactoring to scale to other commands.

---

**Document Version:** 1.0
**Status:** Ready for Review
**Author:** Architect Agent (Claude)
