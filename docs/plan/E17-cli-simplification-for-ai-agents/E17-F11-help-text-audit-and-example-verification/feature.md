---
feature_key: E17-F11-help-text-audit-and-example-verification
epic_key: E17
title: Help Text Audit and Example Verification
description: Audit all Go source --help strings for accuracy and verify every example works against the actual CLI binary.
---

# Help Text Audit and Example Verification

**Feature Key**: E17-F11-help-text-audit-and-example-verification

---

## Goal

### Problem
E17 restructured the CLI from ~45 commands to 121 command paths with new categories (Quick Commands, Core Commands, Entity Management, etc.). The --help text in Go source files may contain inaccurate descriptions, outdated examples, or missing examples for new commands. Since --help is the primary in-situ documentation, inaccuracies directly mislead users.

### Solution
Systematically audit every command's Short/Long/Example text in Go source against actual behavior. Fix inaccuracies, add missing examples, and standardize format across all commands. Verify every example shown in --help actually works.

### Impact
- 100% of --help examples verified working
- Consistent --help format across all 121 command paths
- Establishes ground truth for F12 (unified docs) and F13 (agent context)

---

## Requirements

### Functional Requirements

1. **REQ-F-001**: Audit all command --help strings
   - Compare every command's Short/Long/Example text against actual behavior
   - Fix inaccuracies, add missing examples

2. **REQ-F-002**: Verify all --help examples work
   - Run every example shown in --help output against test database
   - Document failures, fix help text or code

3. **REQ-F-003**: Standardize --help format
   - Consistent structure: Description, Positional Arguments (where applicable), Examples, Usage line

4. **REQ-F-004**: Fix inconsistent flag descriptions
   - Flags like --agent, --force, --reason should have identical descriptions everywhere

5. **REQ-F-005**: Add missing --help examples
   - Commands with no examples get at least 2 (one basic, one with common flags)

6. **REQ-F-006**: Final verification pass
   - Run complete `./bin/shark <cmd> --help` for all paths, diff against docs

---

## Acceptance Criteria

- All 121 command paths have working --help output
- Every example in --help runs without error
- Consistent format across all commands
- No stale references to pre-E17 command structure

---

*Last Updated*: 2026-02-25
