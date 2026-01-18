# Exploratory QA Findings: T-E07-F22-014

**Date:** 2026-01-17
**Task:** T-E07-F22-014 - Update CLI reference documentation for rejection reasons
**QA Agent:** qa-agent

---

## Exploratory Testing Scope

This exploratory testing session focused on:
1. Verifying documentation accuracy against actual implementation
2. Cross-referencing status names with workflow configuration
3. Validating all command examples and flag names
4. Testing document link integrity

---

## Critical Finding: Status Name Mismatch

### Discovery

While cross-referencing the backward transition table with the actual workflow configuration in `.sharkconfig.json`, discovered that the documentation uses incorrect status names.

### Analysis

The workflow operates on a two-tier status system:

**Tier 1: Waiting States (`ready_for_*`)**
- Tasks are queued waiting for a phase to begin
- Examples: `ready_for_code_review`, `ready_for_qa`, `ready_for_approval`
- These are *blocking* states - task is waiting for someone to pick it up

**Tier 2: Active States (`in_*`)**
- Tasks are actively being worked on in a phase
- Examples: `in_code_review`, `in_qa`, `in_approval`
- These are *active* states - work is happening now

**Workflow Pattern:**
```
ready_for_code_review → in_code_review → (rejection) → in_development
ready_for_qa → in_qa → (rejection) → in_development
ready_for_approval → in_approval → (rejection) → ready_for_qa
```

### Impact

The documentation's backward transition table incorrectly shows transitions FROM `ready_for_*` states, when rejections actually occur FROM `in_*` states. This is confusing because:

1. You can't reject from a waiting state - there's nothing to reject yet
2. Rejections happen during active review (`in_code_review`), not while waiting (`ready_for_code_review`)
3. Users following the documentation will try to use wrong status names

### Root Cause

Likely the documentation was written before the two-tier status system was implemented, or the author confused waiting states with active states.

---

## Positive Findings

### Well-Structured Documentation

The rejection-reasons.md document is comprehensive and includes:
- Clear overview and feature benefits
- Detailed flag descriptions with examples
- Best practices for reviewers and developers
- Multiple command examples
- Error message documentation
- Cross-references to related docs

### Accurate Flag Names

All command examples use correct flag names:
- `--rejection-reason` (not `--reason`)
- `--reason-doc` (correct optional flag)
- `--force` (documented with appropriate warnings)

### Good JSON Documentation

The JSON structure example accurately reflects the actual `rejection_history` field returned by `shark task get --json`.

### Proper Cross-Referencing

Document includes proper links to:
- task-commands-full.md (for reopen command reference)
- workflow-config.md (for status flow details)
- error-messages.md (for error handling)

All referenced documents exist and are properly linked.

---

## Minor Issues Observed

### 1. Repository Error Message Bug

**Location:** internal/repository/task_repository.go

**Issue:** Error message says "use --reason flag" but should say "use --rejection-reason flag"

**Severity:** Low (documentation is correct, code message is slightly wrong)

**Recommendation:** Create separate low-priority task to fix error message

### 2. Terminology Consistency

**Issue:** Documentation doesn't explain the difference between `ready_for_*` and `in_*` statuses

**Recommendation:** Add brief explanation of two-tier status system to help users understand workflow phases

### 3. Duplicate/Inconsistent Transitions

**Issue:** Table shows both:
- `ready_for_qa → in_development` (should be `in_qa → in_development`)
- `in_qa → in_development` (correct but redundant with above)

**Impact:** Confuses readers about which status actually triggers rejection requirement

---

## Testing Methodology

### Commands Tested
- ✅ `shark task reopen --help` - Verified rejection reason flags exist
- ✅ `shark task approve --help` - Verified rejection reason flags exist
- ✅ `shark task get --json` - Verified rejection_history structure
- ✅ `shark task timeline --help` - Verified command exists (referenced in docs)

### Files Analyzed
- ✅ .sharkconfig.json - Validated actual workflow configuration
- ✅ docs/cli-reference/*.md - Verified all cross-references
- ✅ internal/repository/task_repository.go - Checked error message implementation
- ✅ internal/cli/commands/task.go - Verified flag registration

### Cross-Reference Checks
- ✅ All internal links resolve to existing documents
- ✅ All example file paths follow project conventions
- ✅ JSON structure matches actual API response format
- ✅ Flag names match command --help output

---

## Recommendations for Future Documentation

### 1. Add Workflow Diagram

Include visual diagram showing:
```
Draft → Ready for Refinement → In Refinement → Ready for Development
    → In Development → Ready for Code Review → In Code Review
    → Ready for QA → In QA → Ready for Approval → In Approval → Completed
```

With backward transitions clearly marked.

### 2. Create Status Reference

Add a dedicated page explaining:
- Two-tier status system (`ready_for_*` vs `in_*`)
- Which transitions require rejection reasons
- How to determine if a transition is "backward"

### 3. Add Real-World Example

Include end-to-end scenario:
```
Developer completes code → moves to ready_for_code_review
Reviewer picks up task → moves to in_code_review
Reviewer finds issues → rejects with --rejection-reason → back to in_development
Developer fixes → moves to ready_for_code_review again
Reviewer approves → moves to ready_for_qa
```

---

## Usability Observations

### Strengths
- Documentation is comprehensive and well-organized
- Examples are clear and actionable
- Best practices are highlighted effectively
- Error messages are well-documented

### Improvement Opportunities
- Status name confusion could frustrate new users
- Missing visual workflow diagram
- No troubleshooting section for common mistakes
- Could benefit from FAQ section

---

## Conclusion

The rejection-reasons.md documentation is high-quality and comprehensive, but contains a **critical accuracy issue** with status names in the backward transition table. This must be fixed before approval.

Once status names are corrected, the documentation will be excellent reference material for users implementing rejection workflows.

**Recommendation:** REJECT and send back to developer for status name corrections.
