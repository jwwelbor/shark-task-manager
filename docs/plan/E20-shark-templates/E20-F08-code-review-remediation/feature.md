---
feature_key: E20-F08-code-review-remediation
epic_key: E20
title: Code Review Remediation
description: Address code review findings from E20 implementation - bug fixes, security hardening, test coverage gaps, and code quality improvements.
---

# Code Review Remediation

**Feature Key**: E20-F08-code-review-remediation

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

During code review of E20 implementation (features F01-F06), several issues were identified across severity levels: two high-severity bugs (file permission silently reset, panic on malformed config), three medium-severity issues (path traversal risk, missing test coverage, data loss during config migration), and low-severity code quality items. These were originally tracked as E20-F07 but that feature was cancelled as misclassified; the work items were migrated here.

### Solution

Address all code review findings as independent, atomic fixes. Each task can be implemented and tested independently. The fixes span `internal/config/`, `internal/init/`, and `internal/cli/commands/config.go`.

### Impact

- Eliminate 2 potential crashes (panic from type assertion, data loss from permission reset)
- Close security gap (path traversal in workflow file path)
- Improve test coverage for edge cases in config validation and file loading
- Fix data loss during config migration (require_rejection_reason dropped)
- Improve code quality and documentation accuracy

---

## Task Summary

| Task | Severity | Description |
|------|----------|-------------|
| T-E20-F08-001 | HIGH | Fix silent file permission reset in writeConfig |
| T-E20-F08-002 | HIGH | Fix unsafe type assertion panic in config show |
| T-E20-F08-003 | MEDIUM | Add path traversal validation for workflow file path |
| T-E20-F08-004 | MEDIUM | Add test coverage for ValidateWorkflowFiles and loadWorkflowFile |
| T-E20-F08-005 | MEDIUM | Include require_rejection_reason in extractWorkflowData |
| T-E20-F08-006 | MEDIUM | Fix os.Exit in configValidateCmd and test cache pollution |
| T-E20-F08-007 | LOW | Dead code, param order, comments, naming cleanup |
| T-E20-F08-008 | LOW | Update E20-F06 docs to remove descoped export command |

All tasks are independent and can be implemented in any order. Priority ordering reflects severity (high first).

---

## Acceptance Criteria

- [ ] All HIGH severity issues (001, 002) are fixed with tests
- [ ] All MEDIUM severity issues (003-006) are addressed
- [ ] LOW severity items (007, 008) are cleaned up
- [ ] `make fmt && make lint && make test` passes after all changes
- [ ] No new warnings introduced

---

*Last Updated*: 2026-03-18
