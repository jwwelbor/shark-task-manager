# E07-F16: Workflow Improvements - Design Documentation Index

## Overview

This index provides a complete view of the CX design deliverables for E07-F16 (Workflow Improvements). The design validates all five user requirements and provides comprehensive recommendations for implementation.

**Status**: Complete - Ready for Development Sprint
**Date**: 2026-01-08

---

## Quick Start

### For Decision Makers
Read: `/docs/E07-F16-CX-RECOMMENDATIONS.md` (5 min)
- Direct answers to your 5 questions
- Clear recommendations with rationale
- Implementation priorities
- Risk summary

### For Developers
Read: `/docs/WORKFLOW_UX_PATTERNS.md` (15 min)
- 10 UX patterns with code examples
- Implementation guidelines
- Testing patterns
- Configuration requirements

### For Complete Context
Read: `/docs/WORKFLOW_UX_DESIGN_REVIEW.md` (30 min)
- Deep dive on each requirement
- Edge case handling
- Success metrics
- Appendix with full workflow config

---

## Documentation Structure

### 1. E07-F16-CX-RECOMMENDATIONS.md
**Quick Reference - Executive Summary**

**Contains**:
- Direct answers to 5 design questions
- Command syntax patterns
- Implementation order (MVP vs. Phase 2)
- Critical issues to fix
- JSON output examples
- Testing checklist
- Backward compatibility notes

**Best For**: Project leads, decision making, sprint planning

**Key Sections**:
- Q1: Status-based querying pattern → ✓ APPROVED
- Q2: Workflow progression command → ✓ APPROVED (with interactive mode)
- Q3: Phase-based querying → ✓ APPROVED FOR PHASE 2
- Q4: Multiple next statuses → ✓ Use interactive selection
- Q5: Show transitions in task get → ✓ APPROVED

---

### 2. WORKFLOW_UX_PATTERNS.md
**Implementation Guide - Design Patterns**

**Contains**:
- 10 reusable UX patterns with code examples
- Consistency checklist
- Configuration structure
- Testing patterns
- Shell completion support

**Patterns Covered**:
1. Positional Arguments for Discovery
2. Case-Insensitive Input
3. Helpful Error Messages
4. Interactive Selection for Ambiguity
5. Preview Mode for Exploration
6. Backward Compatibility with Flags
7. JSON Output for Integration
8. Status Metadata in Display
9. Workflow Config Validation
10. Shell Completion Support

**Best For**: Backend developers, code implementation

**Code Examples**:
- Go implementation for each pattern
- Error handling strategies
- Test structure templates
- Configuration validation

---

### 3. WORKFLOW_UX_DESIGN_REVIEW.md
**Complete Design Document - Full Context**

**Contains**:
- Current state analysis (existing workflow system)
- Detailed validation of each requirement
- Critical issues identified
- Command syntax patterns and consistency
- Journey flow coherence analysis
- Accessibility & inclusivity review
- Edge cases and error handling
- Alternative pattern comparisons
- Implementation roadmap (Phase 1, 2, 3)
- Risk mitigation strategies
- Success metrics

**Sections**:
1. Current State Analysis
2. User Experience Requirements Validation
   - Requirement 1: Status-Based Querying
   - Requirement 2: Workflow Progression Command
   - Requirement 3: Phase-Based Querying
   - Requirement 4: Multiple Possible Transitions
   - Requirement 5: Show Available Transitions
3. Command Syntax Patterns
4. Workflow Discovery - Best Practices
5. Task Creation Status Fix
6. Journey Flow Coherence
7. Accessibility & Inclusivity
8. Edge Cases & Error Handling
9. Comparison to Alternative Patterns
10. Recommended Implementation Order
11. Risk Mitigation
12. Success Metrics

**Best For**: Deep understanding, validation, stakeholder alignment

**Appendix**:
- Current workflow configuration (full reference)

---

### 4. WORKFLOW_GUIDE.md
**User Guide - How to Use Workflow Features**

Note: This is existing documentation in the codebase.

---

## Key Recommendations Summary

### MVP (E07-F16)

**Status-Based Filtering**
```bash
shark task list status ready-for-development     # Positional (primary)
shark task list --status=ready-for-development   # Flag (backward compatible)
```
✓ Enables direct task querying by workflow status
✓ Case-insensitive
✓ Helpful errors show available statuses

**Workflow Progression**
```bash
shark task next-status E07-F20-001               # Interactive (shows options)
shark task next-status E07-F20-001 --status=xyz  # Explicit (non-interactive)
shark task next-status E07-F20-001 --preview     # Preview mode
```
✓ Interactive mode prevents mistakes
✓ Shows available transitions
✓ Educational for users
✓ Non-interactive option for scripts

**Workflow Discovery**
```bash
shark task get E07-F20-001                       # Shows available transitions
shark workflow transitions in_development        # Show transitions for status
```
✓ Integrated into task details
✓ Reduces context switching
✓ Teaches workflow structure

**Task Creation Fix**
- Remove hardcoded `TaskStatusTodo` references
- Use workflow config `special_statuses._start_[0]`
- Ensures tasks created in correct status

### Phase 2 (E07-F17)

**Phase-Based Filtering**
```bash
shark task list phase development
shark task list phase review
```
- Secondary priority (after gathering status filtering usage)
- Leverages same infrastructure

### Phase 3 (E07-F18)

**Advanced Features**
- Shell completion for status/phase names
- Agent-aware filtering
- Workflow templates

---

## Implementation Checklist

### Functional Requirements
- [ ] `shark task list status <name>` filters by workflow status
- [ ] `shark task list --status=<name>` works as backward-compatible alias
- [ ] Case-insensitive status name handling (Draft, DRAFT, draft all work)
- [ ] Invalid status shows helpful error with available options
- [ ] `shark task next-status <task-key>` shows available transitions interactively
- [ ] Multiple transitions: user chooses via numbered selection
- [ ] Single transition: advance immediately or ask confirmation
- [ ] Terminal status (completed, cancelled): appropriate error message
- [ ] Tasks created in correct workflow entry status (respect _start_ config)
- [ ] `--status=<name>` override works for non-interactive use
- [ ] `--preview` flag shows transitions without changing status

### UX Requirements
- [ ] Help text shows examples for both positional and flag syntax
- [ ] `--help` output mentions filtering options clearly
- [ ] Commands complete successfully with `--json` output
- [ ] Status names discoverable via shell completion or help
- [ ] Error messages are actionable and show available options
- [ ] Interactive mode is user-friendly (numbered, clear prompts)
- [ ] Available transitions shown in `task get` output
- [ ] Metadata (phase, agents, description) displayed for statuses

### Documentation
- [ ] README updated with workflow examples
- [ ] CLI_REFERENCE.md shows new commands with examples
- [ ] WORKFLOW_GUIDE.md documents status transitions
- [ ] Help text examples include common filtering patterns
- [ ] Edge cases documented (terminal statuses, mismatched config)

### Testing
- [ ] Valid status names filter correctly
- [ ] Invalid status names show helpful errors
- [ ] Case-insensitive status input works
- [ ] `task next-status` shows all available transitions
- [ ] Multiple transitions: interactive selection works
- [ ] Single transition: correct behavior
- [ ] Terminal status: appropriate error
- [ ] `--status` override works in next-status
- [ ] `--preview` shows transitions without changing
- [ ] `task get` displays available transitions
- [ ] JSON output is valid and complete
- [ ] Backward compatibility with existing commands

---

## Design Principles Applied

### 1. User-Centered
- Tasks organized by workflow status (what users think about)
- Interactive mode reduces cognitive load
- Educational (shows workflow structure)

### 2. Consistency
- Positional arguments visible in help (like `task list E04 F01`)
- Error messages follow same pattern
- Status metadata always included when displayed

### 3. Safety
- Interactive selection prevents wrong transitions
- Preview mode lets users explore without committing
- Explicit confirmation for ambiguous options

### 4. Flexibility
- Positional arguments for discovery and human use
- Flags for scripts and automation
- Preview mode for exploration

### 5. Discoverability
- Help text shows what's possible
- Error messages guide users to solutions
- Shell completion reduces memorization

---

## Configuration Reference

Current workflow config in `.sharkconfig.json`:
- 15 statuses with clear phase assignment
- Multiple workflow paths (planning → development → review → qa → approval → done)
- Error recovery paths (blocked, on_hold, back to earlier stages)
- Agent type targeting for workflow states

**View Current Configuration**:
```bash
shark workflow list
shark workflow list --json

# Validate configuration
shark workflow validate
```

**See Full Config**: `/docs/WORKFLOW_UX_DESIGN_REVIEW.md` Appendix A

---

## Critical Issues to Address

### Issue 1: Task Creation Status Mismatch
**Problem**: Config expects tasks in "draft" or "ready_for_development", but some code still references "todo"

**Solution**: Remove hardcoded status constants from workflow-aware code
- Use `workflow.SpecialStatuses[StartStatusKey][0]` for task creation
- Update validation to use workflow config
- Keep constants only for backward compatibility

**Priority**: CRITICAL - Blocks proper workflow function

### Issue 2: Hardcoded Status Transitions
**Problem**: Commands like `task start`, `task complete` don't respect workflow config

**Solution**: Make all status transitions workflow-aware
- Load workflow config in command handlers
- Validate transitions against `status_flow`
- Show available options when multiple transitions exist

**Priority**: HIGH - Inconsistent behavior

### Issue 3: No Workflow Discovery
**Problem**: Users can't see available transitions for a task

**Solution**:
- Show transitions in `task get` output
- Add `workflow transitions` command
- Document available transitions in help text

**Priority**: HIGH - Improves usability

---

## Success Metrics

### Adoption
- Percentage of `task list` commands using status filtering
- Usage rate of `task next-status` command
- Help/documentation page hits

### Quality
- Error rate for invalid status names (should be low)
- User satisfaction with workflow progression
- Task status update speed

### Learning
- Users understand workflow by using commands
- Fewer mistakes in status transitions
- Higher confidence in task status changes

---

## Timeline

### E07-F16 MVP (2 weeks)
1. Status-based filtering (with positional + flag support)
2. Workflow progression command (interactive mode)
3. Task creation fix (respect workflow config)
4. Workflow discovery (show transitions)
5. Documentation and testing

### E07-F17 Phase 2 (1 week)
6. Phase-based filtering
7. Enhanced testing and validation
8. User feedback integration

### E07-F18 Phase 3 (1 week)
9. Shell completion
10. Advanced features (templates, migrations)

---

## Related Documentation

### In Codebase
- `.sharkconfig.json` - Workflow configuration
- `internal/config/workflow_schema.go` - Configuration structure
- `internal/config/workflow_default.go` - Default workflow
- `docs/WORKFLOW_GUIDE.md` - User guide

### In This Review
- `/docs/E07-F16-CX-RECOMMENDATIONS.md` - Quick start
- `/docs/WORKFLOW_UX_PATTERNS.md` - Implementation patterns
- `/docs/WORKFLOW_UX_DESIGN_REVIEW.md` - Full design document

---

## How to Use This Documentation

### For Sprint Planning
1. Read: `E07-F16-CX-RECOMMENDATIONS.md` (5 min)
2. Discuss: Do we agree with the recommendations?
3. Plan: Map tasks to implementation order
4. Review: Check implementation checklist

### For Development
1. Understand: Read your assigned pattern in `WORKFLOW_UX_PATTERNS.md`
2. Reference: Use code examples as implementation guide
3. Test: Follow testing patterns for your component
4. Validate: Check against design consistency checklist

### For Code Review
1. Verify: Commands match recommended syntax
2. Check: Error messages follow pattern
3. Validate: JSON output structure
4. Confirm: Tests cover edge cases listed in design

### For Documentation
1. Reference: Use examples from `E07-F16-CX-RECOMMENDATIONS.md`
2. Patterns: Show commands following `WORKFLOW_UX_PATTERNS.md`
3. Edge Cases: Document behaviors from `WORKFLOW_UX_DESIGN_REVIEW.md`

---

## Questions & Feedback

**Design questions?** Review the detailed sections in `WORKFLOW_UX_DESIGN_REVIEW.md`

**Implementation questions?** Check code examples in `WORKFLOW_UX_PATTERNS.md`

**Quick decision needed?** See `E07-F16-CX-RECOMMENDATIONS.md`

---

## Document Metadata

| Document | Purpose | Audience | Length | Location |
|----------|---------|----------|--------|----------|
| E07-F16-CX-RECOMMENDATIONS.md | Quick reference | Leads, developers | 5 min | `/docs/` |
| WORKFLOW_UX_PATTERNS.md | Implementation guide | Developers | 15 min | `/docs/` |
| WORKFLOW_UX_DESIGN_REVIEW.md | Complete design | All stakeholders | 30 min | `/docs/` |

**Total Design Package**: ~50 minutes to read completely, but can use each document independently.

---

## Deliverable Checklist

✓ Design Review Complete
✓ Five Requirements Validated
✓ Recommendations Documented
✓ Patterns with Code Examples Provided
✓ Edge Cases Identified and Solutions Proposed
✓ Implementation Roadmap Created
✓ Testing Strategies Defined
✓ Success Metrics Established
✓ Risk Mitigation Documented
✓ Backward Compatibility Addressed
✓ Configuration Reference Provided
✓ Team Documentation Prepared

---

## Next Steps

1. **Review** (Today): Team reads recommendations and design review
2. **Discuss** (Tomorrow): Align on approach, confirm recommendations
3. **Plan** (This Week): Break requirements into tasks for sprint
4. **Implement** (Next Sprint): Follow patterns from WORKFLOW_UX_PATTERNS.md
5. **Test** (During Sprint): Use testing checklist
6. **Release** (End of Sprint): E07-F16 complete

---

## Sign-Off

**CX Design Review**: Complete ✓
**Recommendations**: Approved ✓
**Ready for Implementation**: Yes ✓

**Designer**: CX Designer Agent
**Date**: 2026-01-08
**Status**: Ready for Development

