# Quick Reference Card: Feature Get Workflow Enhancement

**Date:** 2026-01-02 | **Status:** ✅ Design Complete

---

## 📋 At a Glance

| Item | Value |
|------|-------|
| **Reference Task** | T-E07-F16-002 (Task Creation Workflow Config) |
| **Code Review Score** | 8/10 |
| **Effort Estimate** | 16-23 hours |
| **Total Tasks** | 13 |
| **Phases** | 5 |
| **Risk Level** | Low-Medium |
| **Test Coverage Target** | >80% |

---

## 🎯 Problem & Solution

### Problem
```
❌ Hardcoded task statuses in feature get screen
❌ Random status ordering (Go map iteration)
❌ Missing workflow metadata (colors, descriptions, phases)
❌ Code duplication across commands
```

### Solution
```
✅ Create WorkflowService for centralized workflow config access
✅ Update TaskRepository to return ordered status counts
✅ Enhance feature get to display workflow-aware status breakdown
✅ Refactor task creation to use shared service
```

---

## 🏗️ Architecture (One-Liner)

```
Commands → WorkflowService → config.LoadWorkflowConfig (cached) → .sharkconfig.json
```

---

## 📦 New Components

### 1. WorkflowService (`internal/workflow/service.go`)
```go
type Service struct {
    projectRoot string
    workflow    *config.WorkflowConfig
}

// Key Methods:
- GetInitialStatus() TaskStatus
- GetAllStatuses() []string (ordered)
- GetStatusMetadata(status) StatusMetadata
- FormatStatusForDisplay(status, color) FormattedStatus
```

### 2. Enhanced TaskRepository
```go
// Before: map[TaskStatus]int
// After:  []StatusCount

type StatusCount struct {
    Status      TaskStatus
    Count       int
    Phase       string
    Description string
}
```

---

## 📊 Implementation Phases

| Phase | What | Hours | Priority |
|-------|------|-------|----------|
| 1 | WorkflowService + tests | 6-9 | ⭐⭐⭐ |
| 2 | TaskRepository enhancement | 4-5 | ⭐⭐⭐ |
| 3 | Feature get enhancement | 4-5 | ⭐⭐⭐ |
| 4 | Task creation refactor | 1-2 | ⭐⭐ |
| 5 | Testing & docs | 5-6 | ⭐⭐⭐ |

---

## 🎨 User Experience

### Before
```
Task Status Breakdown
ready_for_review       3
in_progress           2
todo                  5
```

### After
```
Task Status Breakdown
Status                      Count  Phase        Description
draft                       5      planning     Task created but not refined
ready_for_refinement        0      planning     Awaiting specification
ready_for_development       2      development  Ready for implementation
ready_for_qa                3      qa           Ready for testing
completed                   0      done         Finished and approved
```

---

## ✅ Success Criteria (Checklist)

### Functional
- [ ] Statuses display in workflow order
- [ ] All workflow statuses shown (including zeros)
- [ ] Status colors applied (unless --no-color)
- [ ] Status descriptions shown
- [ ] Backward compatible

### Non-Functional
- [ ] No performance regression (<1ms)
- [ ] Code duplication removed
- [ ] Test coverage >80%
- [ ] Docs updated

---

## 🚀 Getting Started

### For Review
```bash
cd dev-artifacts/2026-01-02-feature-get-workflow-enhancement
cat README.md              # Start here
cat executive-summary.md   # High-level overview
```

### For Implementation
```bash
cat task-breakdown.md           # Task details
cat implementation-examples.md  # Code examples
cat architecture-diagram.md     # Visual reference
```

### Create Tasks in Shark
```bash
# See task-breakdown.md for full commands
shark task create --epic=E07 --feature=F16 --priority=8 \
  "Create WorkflowService infrastructure"
```

---

## 📚 Documents

| File | Purpose | Read Time | Audience |
|------|---------|-----------|----------|
| `README.md` | Workspace overview | 5 min | All |
| `executive-summary.md` | High-level decisions | 10 min | PMs, Architects |
| `architectural-review-and-design.md` | Full design spec | 30 min | Architects |
| `implementation-examples.md` | Code samples | 20 min | Developers |
| `task-breakdown.md` | Detailed tasks | 15 min | PMs, Devs |
| `architecture-diagram.md` | Visual diagrams | 10 min | All |

**Total:** ~1.5 hours to read everything (or 15 min for key docs)

---

## 🎓 Key Insights

### Code Review Findings
1. **Pattern works well** but creates duplication risk
2. **Config path hardcoded** - needs constant
3. **Limited metadata usage** - opportunity for enhancement

### Architectural Decisions
1. **Create WorkflowService** - centralize config access (DRY)
2. **Change return type** - slice instead of map (deterministic order)
3. **Add metadata** - colors, descriptions, phases (better UX)
4. **Cache workflow config** - performance (negligible overhead)

### Risk Mitigation
1. **Backward compatibility** - graceful fallback to default workflow
2. **Integration tests** - prevent regressions
3. **Gradual rollout** - feature get first, then others

---

## 🔗 Quick Links

### Code References
- T-E07-F16-002: `/internal/taskcreation/creator.go:446-467`
- Current feature get: `/internal/cli/commands/feature.go:633-734`
- Workflow schema: `/internal/config/workflow_schema.go`
- Workflow parser: `/internal/config/workflow_parser.go`

### Documentation
- Workflow spec: `docs/specs/configurable-status-workflow.md`
- CLI reference: `docs/CLI_REFERENCE.md`
- Project config: `.sharkconfig.json`

---

## 📞 Questions?

### Technical
- Architecture concerns? → Read `architectural-review-and-design.md`
- Implementation questions? → Check `implementation-examples.md`
- Need code samples? → See `implementation-examples.md`

### Process
- Task dependencies? → See `task-breakdown.md` (includes DAG)
- Effort estimates? → See `executive-summary.md`
- Success criteria? → See `architectural-review-and-design.md` Part 10

---

## 🎯 Next Actions

### For Architect
1. [ ] Review design docs
2. [ ] Validate API contracts
3. [ ] Approve approach
4. [ ] Schedule design review meeting

### For Product Manager
1. [ ] Review executive summary
2. [ ] Approve effort estimate (16-23h)
3. [ ] Prioritize phases
4. [ ] Create epic/feature in shark

### For Developer
1. [ ] Read executive summary
2. [ ] Skim architecture design
3. [ ] Study implementation examples
4. [ ] Start Phase 1 (WorkflowService)

---

## 📈 Progress Tracking

| Milestone | Status | Date |
|-----------|--------|------|
| Code review | ✅ Done | 2026-01-02 |
| Architecture design | ✅ Done | 2026-01-02 |
| API contracts | ✅ Done | 2026-01-02 |
| Implementation plan | ✅ Done | 2026-01-02 |
| Task breakdown | ✅ Done | 2026-01-02 |
| Design review | ⏳ Pending | TBD |
| Approval | ⏳ Pending | TBD |
| Implementation | ⏳ Pending | TBD |
| Testing | ⏳ Pending | TBD |
| Deployment | ⏳ Pending | TBD |

---

## 💡 Pro Tips

### For Reviewers
- Start with `executive-summary.md` (10 min read)
- Skim `architecture-diagram.md` for visuals
- Deep dive only if needed

### For Implementers
- Create all tasks first (use task-breakdown.md)
- Implement in phase order (1 → 2 → 3 → 4 → 5)
- Write tests as you go
- Reference implementation-examples.md during coding

### For Testing
- Unit test each component (>80% coverage)
- Integration test feature get command
- Manual test with multiple workflow configs
- Visual review in terminal

---

**Version:** 1.0
**Last Updated:** 2026-01-02
**Status:** Ready for Review
