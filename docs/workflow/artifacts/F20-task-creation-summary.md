# F20 Task Creation Summary

**Created**: 2026-01-03
**Feature**: E07-F20 - CLI Command Options Standardization
**Total Tasks**: 19

---

## Tasks Created

### Phase 1: Case Insensitivity (5 tasks, Priority 7-8)

| Task Key | Title | Status | Priority |
|----------|-------|--------|----------|
| T-E07-F20-001 | Add key normalization function for case insensitivity | Draft | 8 ✅ UPDATED |
| T-E07-F20-002 | Update validation functions for case insensitivity | Draft | 8 |
| T-E07-F20-003 | Update parsing functions for case insensitivity | Draft | 8 |
| T-E07-F20-004 | Add unit tests for case insensitive keys | Draft | 8 |
| T-E07-F20-005 | Add integration tests for case insensitive keys | Draft | 7 |

**Estimated Effort**: 8 hours
**Implementation Guide Reference**: Lines 15-454

### Phase 1.5: Short Task Key Format (3 tasks, Priority 7)

| Task Key | Title | Status | Priority |
|----------|-------|--------|----------|
| T-E07-F20-006 | Add short task key pattern and normalization function | Draft | 7 ✅ UPDATED |
| T-E07-F20-007 | Update task commands to use short key normalization | Draft | 7 |
| T-E07-F20-008 | Add tests for short task key format | Draft | 7 |

**Estimated Effort**: 3 hours
**Implementation Guide Reference**: Lines 458-688

### Phase 2: Positional Arguments (4 tasks, Priority 6)

| Task Key | Title | Status | Priority |
|----------|-------|--------|----------|
| T-E07-F20-009 | Update feature create command with positional arguments | Draft | 6 |
| T-E07-F20-010 | Update task create command with positional arguments | Draft | 6 |
| T-E07-F20-011 | Add unit tests for positional argument parsing | Draft | 6 |
| T-E07-F20-012 | Add integration tests for positional argument syntax | Draft | 6 |

**Estimated Effort**: 10 hours
**Implementation Guide Reference**: Lines 690-938

### Phase 3: Enhanced Errors (3 tasks, Priority 5)

| Task Key | Title | Status | Priority |
|----------|-------|--------|----------|
| T-E07-F20-013 | Create error template system for user-friendly messages | Draft | 5 |
| T-E07-F20-014 | Update error messages throughout CLI commands | Draft | 5 |
| T-E07-F20-015 | Test enhanced error messages | Draft | 5 |

**Estimated Effort**: 6 hours
**Implementation Guide Reference**: Lines 940-1003

### Phase 4: Documentation (4 tasks, Priority 4)

| Task Key | Title | Status | Priority |
|----------|-------|--------|----------|
| T-E07-F20-016 | Update CLI_REFERENCE.md with new syntax patterns | Draft | 4 |
| T-E07-F20-017 | Update CLAUDE.md with command examples | Draft | 4 |
| T-E07-F20-018 | Update README.md with CLI improvements | Draft | 4 |
| T-E07-F20-019 | Create migration guide for new syntax | Draft | 4 |

**Estimated Effort**: 8 hours
**Implementation Guide Reference**: Lines 1005-1142

---

## Total Project Metrics

- **Total Tasks**: 19
- **Total Estimated Effort**: 35 hours
- **Timeline**: 4 weeks (no delay from original scope)
- **Phases**: 5 (including 1.5)

---

## Task File Locations

All task files are in:
```
docs/plan/E07-enhancements/E07-F20-cli-command-options-standardization/tasks/
```

Files:
- T-E07-F20-001.md ✅ Updated with implementation details
- T-E07-F20-002.md → T-E07-F20-019.md (need implementation details)
- T-E07-F20-006.md ✅ Updated with implementation details

---

## Next Steps to Complete Task Specifications

### For Each Remaining Task File:

1. **Read the Implementation Guide** section for that task
   - Location: `docs/workflow/artifacts/F20-implementation-guide.md`
   - Find the corresponding section (e.g., "1.2 Update Validation Functions" for T-E07-F20-002)

2. **Extract Key Information**:
   - Goal/objective
   - File locations to modify
   - Code snippets to add/change
   - Acceptance criteria
   - Test requirements
   - Dependencies on other tasks

3. **Update the Task File** with:
   ```markdown
   ## Goal
   [Clear 1-2 sentence description]

   ## Implementation Details
   ### File Location
   [Path to file(s) to modify]

   ### Changes Required
   [Code snippets, function signatures, or detailed steps]

   ### Estimated Time
   [Hours from implementation guide]

   ## Acceptance Criteria
   - [ ] [Specific checkboxes]

   ## Testing Requirements
   [Test cases and coverage needed]

   ## Related Documentation
   - Implementation Guide: [path and line numbers]
   - Design documents: [relevant docs]

   ## Dependencies
   [List of tasks that must complete first]

   ## Notes
   [Important context, trade-offs, or warnings]
   ```

### Task Update Priority

**High Priority** (needed for development to start):
1. T-E07-F20-001 ✅ Done
2. T-E07-F20-002 (Validation functions)
3. T-E07-F20-003 (Parsing functions)
4. T-E07-F20-006 ✅ Done

**Medium Priority** (needed for Phase 2):
5. T-E07-F20-009 (Feature create command)
6. T-E07-F20-010 (Task create command)

**Lower Priority** (can be specified as needed):
- All testing tasks (004, 005, 008, 011, 012, 015)
- Documentation tasks (016-019)
- Error handling tasks (013-014)

---

## Template for Bulk Updates

Use this pattern for efficiency:

```bash
# Example script to help update remaining files
for task in 002 003 004 005 007 008 009 010 011 012 013 014 015 016 017 018 019; do
  echo "Updating T-E07-F20-${task}..."
  # 1. Read implementation guide section
  # 2. Extract specifications
  # 3. Update task file
done
```

---

## Design Documentation References

All design documents are in `docs/workflow/artifacts/`:

1. **F20-cli-ux-specification.md** - Technical specification
2. **F20-implementation-guide.md** - Developer implementation details
3. **F20-design-summary.md** - Executive overview
4. **F20-user-journey-comparison.md** - UX impact analysis
5. **F20-quick-reference.md** - Developer cheat sheet
6. **F20-short-key-enhancement-approval.md** - Client approval for short keys

---

## Implementation Guide Section Map

Quick reference for finding task details in the implementation guide:

| Task | Guide Section | Line Range |
|------|---------------|------------|
| T-E07-F20-001 | 1.1 Add Key Normalization | 17-36 |
| T-E07-F20-002 | 1.2 Update Validation Functions | 38-59 |
| T-E07-F20-003 | 1.3 Update Parsing Functions | 64-167 |
| T-E07-F20-004 | 1.4 Add Tests | 170-345 |
| T-E07-F20-005 | 1.5 Integration Tests | 347-454 |
| T-E07-F20-006 | 1.5.1-1.5.2 Short Key Pattern | 466-520 |
| T-E07-F20-007 | 1.5.4 Update Task Commands | 562-589 |
| T-E07-F20-008 | 1.5.5 Add Tests for Short Format | 591-655 |
| T-E07-F20-009 | 2.1 Update Feature Create | 693-767 |
| T-E07-F20-010 | 2.2 Update Task Create | 769-868 |
| T-E07-F20-011 | 2.3 Add Tests (Positional) | 872-938 |
| T-E07-F20-012 | Integration Tests (implied) | Manual testing section |
| T-E07-F20-013 | 3.1 Create Error Template | 945-987 |
| T-E07-F20-014 | 3.2 Update Error Messages | 989-1003 |
| T-E07-F20-015 | Testing (implied) | Error testing section |
| T-E07-F20-016 | 4.1 Update CLI_REFERENCE | 1010-1013 |
| T-E07-F20-017 | 4.2 Update CLAUDE.md | 1015-1018 |
| T-E07-F20-018 | README updates (implied) | 1010-1018 |
| T-E07-F20-019 | Migration guide (implied) | Testing checklist |

---

## Success Criteria for Task Specifications

Each task file should have:

- [ ] Clear, actionable goal statement
- [ ] Specific file paths to modify
- [ ] Code snippets or detailed change instructions
- [ ] 3-5 measurable acceptance criteria
- [ ] Test requirements (what to test, not how)
- [ ] Links to related documentation
- [ ] Dependency list (which tasks must complete first)
- [ ] Estimated hours (from implementation guide)
- [ ] Any important notes or warnings

---

## How to Verify Tasks Are Ready

Before starting development on a task:

1. **Read the task file** - Should be clear what to do
2. **Check dependencies** - Are prerequisite tasks complete?
3. **Review acceptance criteria** - Do you know when it's "done"?
4. **Check implementation guide** - Any additional context needed?
5. **Verify test requirements** - Clear what needs to be tested?

If any of these are unclear, the task specification needs more detail.

---

## For AI Agents Working on F20

When you start working on a task:

1. Read the task file first
2. Read the corresponding section of the implementation guide
3. Check if dependencies are complete
4. Review related documentation
5. Implement following the specifications exactly
6. Run tests (unit + integration)
7. Update task status: `shark task complete <task-key>`

---

## Questions or Issues

If task specifications are unclear:

1. Check the implementation guide for more details
2. Review design documents for context
3. Look at T-E07-F20-001 or T-E07-F20-006 as examples
4. Ask for clarification before implementing

---

## Summary

✅ **Created**: Feature E07-F20 with 19 implementation tasks
✅ **Updated**: Sample tasks (001, 006) with full specifications
✅ **Organized**: Tasks by phase with clear priorities
✅ **Documented**: Complete mapping to implementation guide

**Next Action**: Update remaining task files (002-005, 007-019) with specifications from implementation guide

**Estimated Time to Complete Specifications**: 2-3 hours
**Ready to Start Development**: Phase 1 tasks (001-005) after specifications complete
