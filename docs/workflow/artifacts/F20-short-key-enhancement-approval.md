# F20 Enhancement Approval: Short Task Key Format

**Feature**: E10-F20 - Standardize CLI Command Options
**Enhancement**: Alternative 3 - Short Task Key Format
**Date**: 2026-01-03
**Status**: ✅ APPROVED - Ready for Implementation

---

## Client Decision

**Client has approved Alternative 3**: Allow dropping the `T-` prefix from task keys.

### What This Means

Users can now use either format:
```bash
# Full format (original)
shark task start T-E01-F02-001

# Short format (new) ✨
shark task start e01-f02-001
```

Both formats work identically. The CLI automatically adds the `T-` prefix internally.

---

## Integration with F20

This enhancement has been **fully integrated** into the F20 design documents:

### ✅ Updated Documents

1. **F20-cli-ux-specification.md**
   - Added task key prefix rules
   - Updated examples throughout
   - Added Phase 1.5 implementation section
   - Included test cases for short format

2. **F20-implementation-guide.md**
   - Added Phase 1.5: Short Task Key Format (3 hours)
   - Detailed implementation steps
   - Code examples for `NormalizeTaskKey()`
   - Test cases and validation logic

3. **F20-quick-reference.md**
   - Updated all task command examples
   - Added short format to cheat sheets
   - Updated key format reference section
   - Modified workflow examples

4. **F20-design-summary.md**
   - Added short key format to core changes
   - Updated implementation timeline
   - Added Phase 1.5 with effort estimate
   - Total timeline unchanged: 4 weeks

---

## Timeline Impact

### Original F20 Timeline
- **Week 1**: Case Insensitivity (8 hours)
- **Week 2**: Positional Arguments (10 hours)
- **Week 3**: Enhanced Errors (6 hours)
- **Week 4**: Documentation (8 hours)

### Updated F20 Timeline (with Short Keys)
- **Week 1**: Case Insensitivity (8 hours)
- **Week 2**:
  - Short Task Key Format (3 hours) ✨ NEW
  - Positional Arguments (10 hours)
  - **Total: 13 hours (runs in parallel)**
- **Week 3**: Enhanced Errors (6 hours)
- **Week 4**: Documentation (8 hours)

### Impact Analysis

✅ **NO SCHEDULE DELAY**
- Short key format fits naturally in Week 2
- Uses same infrastructure as case normalization
- Runs in parallel with positional arguments
- Total timeline remains 4 weeks

✅ **MINIMAL COST INCREASE**
- Original F20 effort: ~32 hours
- With short keys: ~35 hours (+3 hours, +9%)
- Easily absorbed in existing schedule

✅ **HIGH VALUE-TO-COST RATIO**
- Cost: 3 hours
- Benefit: 2.5% keystroke reduction + cleaner syntax
- Consistency with epic/feature keys (no prefix needed)

---

## Technical Summary

### Implementation

**New Functions**:
```go
// Pattern for short task key (without T- prefix)
shortTaskKeyPattern = regexp.MustCompile(`^E\d{2}-F\d{2}-\d{3}$`)

// Normalize task key - add T- prefix if missing
func NormalizeTaskKey(input string) (string, error) {
    normalized := strings.ToUpper(input)
    if strings.HasPrefix(normalized, "T-") {
        return normalized, nil
    }
    if shortTaskKeyPattern.MatchString(normalized) {
        return "T-" + normalized, nil
    }
    // Handle slugged keys...
    return normalized, nil
}
```

**Updated Commands**:
- `shark task get <key>`
- `shark task start <key>`
- `shark task complete <key>`
- `shark task approve <key>`
- `shark task reopen <key>`
- `shark task block <key>`
- `shark task unblock <key>`

**No Changes Needed**:
- Database schema (unchanged)
- Repository layer (unchanged)
- Task creation logic (unchanged)
- JSON output format (unchanged)

### Examples

```bash
# All these work identically:
shark task start T-E01-F02-001
shark task start t-e01-f02-001
shark task start E01-F02-001
shark task start e01-f02-001        # ✨ Short format

# With slugs:
shark task get T-E01-F02-001-implement-auth
shark task get e01-f02-001-implement-auth  # ✨ Short format

# Output always shows canonical format:
Task T-E01-F02-001 started successfully
```

---

## Benefits

### For Users
- **2.5% fewer keystrokes** for manual typing
- **Cleaner, more readable** syntax
- **Consistency** - epic/feature keys don't have prefixes either
- **Optional** - full format still works

### For AI Agents
- **No change required** - agents already have full keys from database
- **Both formats accepted** - robust parsing
- **No normalization code needed** - CLI handles it

### For Project
- **Low risk** - purely additive change
- **Easy to implement** - leverages existing infrastructure
- **Well documented** - comprehensive specs and tests
- **Fits F20 philosophy** - "more permissive, not more complex"

---

## Decision Rationale

### Why Alternative 3 (Short Keys) Over Alternative 2 (Positional)?

| Factor | Alt 2 (Positional) | Alt 3 (Short Keys) |
|--------|-------------------|-------------------|
| Cost | 7 hours | 3 hours |
| Complexity | High (35 lines, 8 edge cases) | Low (15 lines, 3 edge cases) |
| Real benefit | 2.8% usage | 2.5% usage |
| Ambiguity risk | Medium (single arg) | None (single arg) |
| AI agent value | None (won't use) | None (won't use) |
| Human value | Marginal | Marginal |
| Consistency | Less (different from epic/feature) | More (matches epic/feature) |

**Result**: Alt 3 provides 90% of the benefit for 40% of the cost.

---

## Next Steps

### For Implementation Team

1. **Start with F20 Week 1** (Case Insensitivity)
   - This is the foundation for all other changes
   - Must complete before short keys

2. **Week 2: Implement Short Keys + Positional Arguments**
   - Short keys use case normalization infrastructure
   - Can be developed in parallel with positional args
   - Both merge together in Week 2

3. **Follow F20-implementation-guide.md**
   - All details documented in Phase 1.5
   - Code examples provided
   - Test cases specified

### For Product Manager

1. **Monitor Week 2 Progress**
   - Ensure short keys and positional args stay on track
   - Watch for any integration issues
   - Verify tests pass for both features

2. **User Communication**
   - Announce short key format in release notes
   - Update user documentation
   - Highlight in changelog

3. **Metrics to Track**
   - Usage of short vs full format (if telemetry available)
   - Error rates with new format
   - User feedback on clarity

---

## Success Criteria

### Phase 1.5 Complete When:

✅ `NormalizeTaskKey()` function implemented and tested
✅ All task commands accept short format
✅ All tests pass (unit + integration)
✅ Error messages mention both formats
✅ Documentation updated with examples
✅ Manual testing confirms both formats work

### Validation Commands:

```bash
# These should all work:
shark task start e01-f02-001
shark task complete E01-F02-001
shark task get e01-f02-001-task-name

# These should fail with clear errors:
shark task start e1-f2-1          # Wrong digit count
shark task start E01-F02          # Missing task number
```

---

## Documentation Updated

All F20 documents now include short key format:

- ✅ **Specification** - Technical details and parsing rules
- ✅ **Implementation Guide** - Code snippets and test cases
- ✅ **Quick Reference** - User-facing examples
- ✅ **Design Summary** - Timeline and effort estimates

Additional documents created during analysis:

- **F20-enhancement-one-pager.md** - Decision summary (1 page)
- **F20-enhancement-decision-summary.md** - Executive summary (5 pages)
- **F20-enhancement-alternatives-comparison.md** - Side-by-side comparison (12 pages)
- **F20-enhancement-positional-task-number.md** - Full UX analysis (15 pages)

---

## Approval Checklist

- [x] Client approved Alternative 3
- [x] Design documents updated
- [x] Timeline verified (no schedule impact)
- [x] Implementation guide complete
- [x] Test cases specified
- [x] Error messages defined
- [x] Examples provided
- [x] Risk assessment complete
- [x] Success criteria defined

---

## Final Recommendation

**PROCEED WITH IMPLEMENTATION**

The short task key format enhancement:
- ✅ Approved by client
- ✅ Fully integrated into F20 design
- ✅ No schedule impact (fits in Week 2)
- ✅ Low cost (3 hours, +9% of F20)
- ✅ Low risk (additive, well-documented)
- ✅ High consistency (matches epic/feature patterns)

**Implementation can begin immediately following completion of Phase 1 (Case Insensitivity).**

---

**Product Manager**: Ready to proceed?
**Development Team**: Phase 1.5 specifications are complete and ready for implementation.
