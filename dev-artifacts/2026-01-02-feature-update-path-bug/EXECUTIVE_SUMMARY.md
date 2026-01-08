# Executive Summary: Path/Filename Architecture Issue

## TL;DR

What started as a simple bug fix ("feature update --path doesn't work") revealed a **fundamental architectural flaw** in how shark handles file paths. The `--path` flag feature was implemented incompletely and has never worked as designed.

**Recommendation:** Convert this from a bug fix to a proper architecture refactoring project. Estimated effort: ~11.5 days across 10 phases.

---

## The Problem

### Original Bug Report (Idea I-2026-01-02-05)
"feature update --path isn't working. no errors but it doesn't change"

### What We Discovered

The problem is much deeper than a single broken command:

1. **Database has columns that are never used**
   - `custom_folder_path` exists for epics and features
   - Values are stored but ignored when calculating file paths
   - It's a "phantom feature" - appears to work but has no effect

2. **Path and filename are conflated**
   - `file_path` stores full paths (mixing directory + filename)
   - `--filename` flag expects full path (should only be filename)
   - No separation of concerns

3. **No inheritance**
   - Features should inherit path from epic
   - Tasks should inherit path from feature
   - Currently each entity calculates path independently

4. **Tasks don't support custom paths at all**
   - Missing `custom_folder_path` column in database
   - No `--path` flag in commands
   - No way to organize task files

5. **Path logic is duplicated**
   - Each command (epic, feature, task) calculates paths independently
   - No single source of truth
   - Difficult to maintain and extend

### Impact

- **Functionality:** Advertised feature completely non-functional
- **Data Integrity:** Database contains misleading values
- **User Experience:** Users who tried this feature have incorrect data
- **Code Quality:** Duplicated logic, incomplete implementation

---

## The Vision: How It Should Work

### Proper Architecture

**Hierarchical Path Composition:**
```
Epic:    {docs_root}/{epic_custom_path OR default}/{epic_filename}
Feature: {docs_root}/{epic_path}/{feature_custom_path OR default}/{feature_filename}
Task:    {docs_root}/{epic_path}/{feature_path}/{task_custom_path OR default}/{task_filename}
```

**Separation of Concerns:**
- `custom_path` - Just the path segment for this entity
- `filename` - Just the filename (not full path)
- Computed `full_path` - Calculated from inheritance + custom values

**Example Usage:**
```bash
# Create epic with custom organization
shark epic create "Q1 2025 Roadmap" --path="roadmap/2025-q1"
# Creates: docs/roadmap/2025-q1/epic.md

# Feature inherits epic's path automatically
shark feature create --epic=E01 "User Growth"
# Creates: docs/roadmap/2025-q1/E01-F01-user-growth/feature.md

# Feature can override with its own custom path
shark feature create --epic=E01 "Retention" --path="metrics"
# Creates: docs/roadmap/2025-q1/metrics/feature.md

# Tasks inherit from feature (which inherited from epic)
shark task create --epic=E01 --feature=F01 "Analytics Dashboard" --agent=frontend
# Creates: docs/roadmap/2025-q1/E01-F01-user-growth/tasks/T-E01-F01-001.md
```

---

## The Solution: Phased Refactoring

### Phase Breakdown

| Phase | Focus | Effort |
|-------|-------|--------|
| 0 | Design approval | S (0.5 day) |
| 1 | Database schema migration | M (1 day) |
| 2 | Path resolution logic | L (1.5 days) |
| 3 | Repository updates | M (1 day) |
| 4 | Epic command fixes | M (1 day) |
| 5 | Feature command fixes | M (1 day) |
| 6 | Task command updates | M (1 day) |
| 7 | Cascading updates | L (1.5 days) |
| 8 | Documentation | M (1 day) |
| 9 | Testing & validation | L (1.5 days) |
| 10 | Cleanup & deprecation | S (0.5 day) |
| **Total** | | **~11.5 days** |

### Key Deliverables

1. **Centralized path resolution** - Single source of truth for path calculation
2. **Proper inheritance** - Children automatically inherit parent paths
3. **Working --path flag** - Actually affects where files are created
4. **Separate --filename flag** - Just the filename, not full path
5. **Task path support** - Tasks can be organized with custom paths
6. **File operations** - Move/rename files when paths change
7. **Comprehensive tests** - Full coverage of all scenarios
8. **Migration guide** - Help existing users upgrade

---

## Why This Matters

### Current State Problems

❌ **Feature is broken** - Documented but doesn't work
❌ **Data is misleading** - Database has values that mean nothing
❌ **Code is fragile** - Duplicated logic in multiple places
❌ **Can't be extended** - No foundation for future enhancements
❌ **Tests are inadequate** - Bug wasn't caught because no tests

### Future State Benefits

✅ **Feature works correctly** - --path flag does what it says
✅ **Data is accurate** - Database reflects reality
✅ **Code is maintainable** - Single path resolution function
✅ **Extensible** - Easy to add new path features
✅ **Well tested** - Comprehensive test coverage
✅ **Flexible organization** - Users can organize files their way

---

## Decision Points

### Option 1: Quick Fix (Not Recommended)

**Approach:** Just make --path update the database field and move the file

**Pros:**
- Fast (1-2 days)
- Fixes immediate bug

**Cons:**
- Doesn't fix root cause
- Leaves architectural flaws in place
- Will need full refactor eventually anyway
- Technical debt accumulates

**Recommendation:** ❌ Don't do this. Kicking the can down the road.

### Option 2: Proper Refactoring (Recommended)

**Approach:** Full architectural refactoring as outlined in this document

**Pros:**
- Fixes root cause
- Creates solid foundation
- Enables future features
- Clean, maintainable code
- Comprehensive test coverage

**Cons:**
- Takes longer (~11.5 days)
- More complex
- Requires careful planning

**Recommendation:** ✅ Do this. The right way to solve the problem.

### Option 3: Hybrid Approach

**Approach:** Minimal fix now, refactor later (break into phases)

**Phase A (Quick):** Fix feature update --path to actually work (1-2 days)
**Phase B (Later):** Full refactoring when time permits (9-10 days)

**Pros:**
- Users get immediate fix
- Can schedule larger refactor separately

**Cons:**
- Work is duplicated (fix twice)
- Phase A code will be thrown away
- Still accumulating tech debt

**Recommendation:** ⚠️ Only if time pressure demands it

---

## Recommended Next Steps

1. **Get Stakeholder Approval** (~1 hour)
   - Review this summary
   - Review detailed architecture (PATH_FILENAME_ARCHITECTURE.md)
   - Review refactoring plan (REFACTORING_PLAN.md)
   - Decide on approach (Option 1, 2, or 3)

2. **Answer Open Questions** (~1 hour)
   - File moving behavior on path update?
   - Cascading updates automatic or opt-in?
   - How long to maintain backward compatibility?
   - Approve default path structures?

3. **Create Project Structure** (~1 hour)
   - Create new feature (not just a bug fix task)
   - Break down into tasks per phase
   - Assign to appropriate agents
   - Set up tracking

4. **Begin Implementation** (~11 days)
   - Follow phased approach in REFACTORING_PLAN.md
   - Test thoroughly at each phase
   - Document as you go

5. **Review & Deploy** (~1 day)
   - Final code review
   - User acceptance testing
   - Merge and deploy
   - Monitor for issues

---

## Risk Assessment

### High Risk
- Data migration could lose information
- Breaking changes could disrupt users

**Mitigation:**
- Auto-backup before migration
- Maintain backward compatibility
- Thorough testing on production copy

### Medium Risk
- Performance degradation with path computation
- Complex inheritance logic hard to debug

**Mitigation:**
- Cache computed paths
- Add comprehensive logging
- Performance test with large datasets

### Low Risk
- New bugs introduced during refactor

**Mitigation:**
- Test-driven development
- Code review at each phase
- Comprehensive test suite

---

## Documents for Review

1. **PATH_FILENAME_ARCHITECTURE.md** - Detailed architecture design
2. **CURRENT_IMPLEMENTATION_ANALYSIS.md** - What's broken and why
3. **REFACTORING_PLAN.md** - Phase-by-phase implementation plan
4. **This document** - Executive summary for decision makers

---

## Questions?

Contact the technical team for clarification on any aspect of this proposal.

**Decision needed by:** [Set deadline]
**Implementation start:** [After approval]
**Target completion:** [~2-3 weeks from start]
