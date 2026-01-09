# E07-F16 CX Design Review - README

## Executive Summary

The CX Designer has completed a comprehensive design review of E07-F16 workflow improvements, validating all 5 user requirements and providing detailed implementation guidance.

**Status**: Complete and Ready for Development
**Date**: 2026-01-08

---

## Your 5 Questions - Quick Answers

### Q1: Is `shark task list status <status-name>` the right UX pattern?

**Answer: YES** - Use positional syntax as primary, `--status` flag as backward-compatible alias.

```bash
shark task list status ready-for-development      # Primary (discoverable)
shark task list --status=ready-for-development    # Alias (backward compatible)
```

**Why**: Positional arguments are visible in help text, match existing patterns, more discoverable.

---

### Q2: Is `shark task next-status <task-key>` the right command?

**Answer: YES** - With interactive mode strongly recommended when multiple transitions exist.

```bash
shark task next-status E07-F20-001               # Interactive (primary)
shark task next-status E07-F20-001 --status=xyz  # Explicit (non-interactive)
shark task next-status E07-F20-001 --preview     # Preview mode
```

**Why**: Interactive mode prevents mistakes, educational, shows all options, safe with confirmation.

**When Multiple Options Exist**:
```
Available transitions:
  1) ready_for_code_review (phase: review)
  2) ready_for_refinement (phase: planning)
  3) blocked
  4) on_hold

Enter selection [1-4]:
```

---

### Q3: Should we support querying by phase?

**Answer: YES** - But defer to Phase 2 (E07-F17).

**MVP (E07-F16)**: Status filtering (primary need)
**Phase 2 (E07-F17)**: Phase filtering (secondary, after gathering usage data)

---

### Q4: How to handle multiple possible next statuses?

**Answer: Interactive Selection** (superior to smart defaults).

| Criteria | Interactive | Smart Default |
|----------|-------------|---------------|
| Safety | Excellent | Fair |
| Clarity | Shows all | Hidden |
| Education | Excellent | None |
| Automation | Use --status | Might guess wrong |

Interactive mode wins on safety and learning. For scripts, use `--status=<name>`.

---

### Q5: Should we show available next statuses in `task get`?

**Answer: YES** - Strongly recommended.

Shows available transitions in task details output, reducing context switching and enabling faster decision-making.

---

## What You Get

### 5 Comprehensive Design Documents

#### 1. E07-F16-CX-RECOMMENDATIONS.md (5 min read)
Quick answers to your 5 questions with implementation priorities and critical issues to fix first.

**Read if**: You need quick decisions and want to start sprint planning.

#### 2. WORKFLOW_UX_PATTERNS.md (15 min read)
10 reusable UX patterns with Go code examples, testing templates, and consistency checklist.

**Read if**: You're a developer implementing the feature.

#### 3. WORKFLOW_UX_DESIGN_REVIEW.md (30 min read)
Complete design analysis including edge cases, workflow coherence validation, accessibility review, and risk mitigation.

**Read if**: You want deep understanding and need to validate all design decisions.

#### 4. E07-F16-DESIGN-INDEX.md (5 min read)
Navigation guide, implementation checklist, design principles, and timeline.

**Read if**: You're managing the project and need to organize the work.

#### 5. E07-F16-VISUAL-SUMMARY.txt (10 min read)
Visual quick reference with command examples, error handling, and implementation roadmap.

**Read if**: You want a visual overview or quick reference.

---

## Implementation Roadmap

### E07-F16 MVP (2 weeks)
1. Status-based filtering: `shark task list status <name>`
2. Workflow progression: `shark task next-status <task-key>` (interactive)
3. Task creation fix: Remove hardcoded status, use workflow config
4. Workflow discovery: Show transitions in task get output

### E07-F17 Phase 2 (1 week)
5. Phase-based filtering: `shark task list phase <name>`
6. Shell completion and enhancements

### E07-F18 Phase 3 (1 week)
7. Workflow templates, custom workflows, migrations

---

## Critical Issues to Fix First

### 1. Task Creation Status Mismatch (CRITICAL)
- **Problem**: Config expects "draft" or "ready_for_development", code still references "todo"
- **Impact**: Tasks created in wrong status, workflow validation fails
- **Fix**: Remove hardcoded constants, use `workflow.SpecialStatuses[StartStatusKey][0]`

### 2. Hardcoded Status Transitions (HIGH)
- **Problem**: Commands like `task start/complete` ignore workflow config
- **Impact**: Custom workflows don't work properly
- **Fix**: Make all transitions workflow-aware, validate against status_flow

### 3. No Workflow Discovery (HIGH)
- **Problem**: Users can't see available transitions
- **Impact**: Users guess wrong, hard to learn workflow
- **Fix**: Show transitions in task get, add workflow transitions command

---

## Key Design Principles

1. **Safety** - Interactive mode prevents unintended transitions
2. **Discoverability** - Positional arguments visible in help
3. **Learnability** - Workflow visible and learnable by using
4. **Flexibility** - Interactive for humans, flags for scripts
5. **Consistency** - All workflow commands follow same patterns

---

## Testing Checklist

- Valid status names filter correctly
- Invalid status shows helpful error with available options
- Case-insensitive status input (Draft, DRAFT, draft)
- Task next-status shows all available transitions
- Interactive selection with numbered choices
- Terminal statuses show appropriate errors
- Preview mode shows without changing
- JSON output is valid and complete
- Backward compatibility maintained

---

## Files to Review

**Start Here** (5 min):
```
/docs/E07-F16-CX-RECOMMENDATIONS.md
```

**Then Read** (15 min):
```
/docs/WORKFLOW_UX_PATTERNS.md       # If implementing
/docs/E07-F16-DESIGN-INDEX.md       # If managing
```

**For Deep Dive** (30 min):
```
/docs/WORKFLOW_UX_DESIGN_REVIEW.md
```

**Quick Reference** (10 min):
```
/docs/E07-F16-VISUAL-SUMMARY.txt
```

---

## Next Steps

1. **Review** (Today): Read E07-F16-CX-RECOMMENDATIONS.md
2. **Discuss** (This Week): Team aligns on approach
3. **Plan** (Next Week): Break into sprint tasks
4. **Implement** (Next Sprint): Follow WORKFLOW_UX_PATTERNS.md
5. **Test** (During Sprint): Use testing checklist
6. **Release** (End of Sprint): E07-F16 complete

---

## Key Metrics

Success will be measured by:
- Adoption of status filtering in daily workflows
- Low error rates on invalid input
- User confidence in status transitions
- Fewer workflow mistakes
- Task status updates completing faster

---

## Summary

**All 5 requirements are VALID and RECOMMENDED for implementation.**

Your requirements are:
- Aligned with user needs
- Consistent with Shark patterns
- Implementable with provided code examples
- Testable with provided patterns
- Documented with comprehensive guides

**Ready for development sprint.**

---

## Questions?

- **Quick answers?** → E07-F16-CX-RECOMMENDATIONS.md
- **Implementation details?** → WORKFLOW_UX_PATTERNS.md
- **Complete context?** → WORKFLOW_UX_DESIGN_REVIEW.md
- **Visual overview?** → E07-F16-VISUAL-SUMMARY.txt
- **Project management?** → E07-F16-DESIGN-INDEX.md

All files in `/docs/`

---

**CX Designer**: CX Designer Agent
**Date**: 2026-01-08
**Status**: Complete - Ready for Development
