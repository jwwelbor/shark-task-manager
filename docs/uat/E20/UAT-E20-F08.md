# UAT Test Guide - Code Review Remediation

**Feature:** E20-F08 - Code Review Remediation
**Epic:** E20 - Shark Templates
**Generated:** 2026-03-18
**Status:** Ready for UAT

---

## Epic Context

**Epic Goal:** Externalize and standardize template and workflow configuration, separating workflow definitions from runtime settings in `.sharkconfig.json`.

**This Feature's Role:** Address code review findings from E20 implementation (features F01-F06). Fixes security issues, crash risks, data loss bugs, test coverage gaps, and code quality issues identified during static analysis.

**Related Features:**
- E20-F04 (completed): Workflow File Loading and Precedence
- E20-F05 (completed): Init Update Workflow File Generation
- E20-F06 (completed): Config Validation and Developer Experience
- E20-F07 (cancelled): Red Team Code Review Cleanup (migrated to F08)

---

## Test Scenarios

### Scenario 1: File Permission Preservation (T-E20-F08-001)
**Severity:** HIGH | **Priority:** 9

**Spec:** writeConfig must preserve existing file permissions and default to 0644 for new files.

**Steps:**
1. Read `writeConfig()` in `internal/init/profile_service.go`
2. Verify `os.Stat` captures permissions (not discarded with `_`)
3. Verify `os.Chmod` applies permissions before rename
4. Run `TestWriteConfig_PreservesPermissions`

**Success Criteria:**
- [ ] Existing file permissions preserved after writeConfig call
- [ ] New files default to 0644
- [ ] Unit test verifies both scenarios

---

### Scenario 2: Safe Type Assertion in Config Show (T-E20-F08-002)
**Severity:** HIGH | **Priority:** 8

**Spec:** `shark config show` must never panic with malformed config. Use comma-ok type assertions.

**Steps:**
1. Read `configShowCmd` in `internal/cli/commands/config.go` lines 91-110
2. Verify comma-ok idiom for database map assertion
3. Verify comma-ok idiom for workflow_sources assertion
4. Verify fallback prints value with `%v` format

**Success Criteria:**
- [ ] No bare type assertions in config show
- [ ] Graceful fallback for unexpected types

---

### Scenario 3: Path Traversal Validation (T-E20-F08-003)
**Severity:** MEDIUM | **Priority:** 7

**Spec:** Configurable workflow_config path must be validated to prevent path traversal attacks.

**Steps:**
1. Read `validateWorkflowFilePath()` in `internal/config/workflow_parser.go`
2. Verify both paths resolved to absolute before comparison
3. Run `TestValidateWorkflowFilePath` with traversal attempts
4. Verify `../../../etc/passwd` rejected, `config/workflow.json` accepted

**Success Criteria:**
- [ ] Path traversal attempts rejected with clear error
- [ ] Valid relative paths accepted
- [ ] Both Unix and Windows separators handled

---

### Scenario 4: Test Coverage for Edge Cases (T-E20-F08-004)
**Severity:** MEDIUM | **Priority:** 6

**Steps:**
1. Check `workflow_file_loading_test.go` for loadWorkflowFile edge cases
2. Check `workflow_validation_dx_test.go` for ValidateWorkflowFiles edge cases
3. Run tests and verify all pass

**Success Criteria:**
- [ ] At least 3 edge case tests for loadWorkflowFile (invalid JSON, array, permissions, empty)
- [ ] At least 3 edge case tests for ValidateWorkflowFiles
- [ ] All tests pass deterministically

---

### Scenario 5: require_rejection_reason Extraction (T-E20-F08-005)
**Severity:** MEDIUM | **Priority:** 6

**Spec:** extractWorkflowData must include require_rejection_reason in task_workflow block.

**Steps:**
1. Read extractWorkflowData in `internal/init/profile_service.go`
2. Verify require_rejection_reason extraction follows existing pattern
3. Run `TestExtractWorkflowData_IncludesRequireRejectionReason`

**Success Criteria:**
- [ ] Key extracted when present in merged config
- [ ] Key omitted when not present (no false positives)
- [ ] Tests verify both cases

---

### Scenario 6: os.Exit Removal and Cache Cleanup (T-E20-F08-006)
**Severity:** MEDIUM | **Priority:** 5

**Spec:** No os.Exit calls in configValidateCmd or configGetStatusAction. Cache pollution prevented.

**Steps:**
1. Grep for os.Exit in config.go - must find zero matches
2. Verify commands return errors instead
3. Check for ClearWorkflowCache in test cleanup

**Success Criteria:**
- [ ] Zero os.Exit calls in config.go
- [ ] Commands return errors via Cobra RunE pattern
- [ ] No cache-related test flakiness

---

### Scenario 7: Code Quality Fixes (T-E20-F08-007)
**Severity:** LOW | **Priority:** 3

**Steps:**
1. Run `make lint` - must produce 0 issues
2. Verify no dead code in internal/config/ and internal/init/
3. Verify exported functions have godoc comments

**Success Criteria:**
- [ ] `make lint` produces 0 issues
- [ ] No behavioral changes (all tests pass)

---

### Scenario 8: Documentation Cleanup (T-E20-F08-008)
**Severity:** LOW | **Priority:** 4

**Spec:** All references to REQ-F-021 (export command) must be marked DESCOPED.

**Steps:**
1. Search E20-F06 docs for REQ-F-021 and "export"
2. Verify all references have DESCOPED marker or strikethrough

**Success Criteria:**
- [ ] No unmarked references to export command
- [ ] Out of Scope section documents the descoping

---

## Quality Gate

- [ ] `make fmt` - no changes
- [ ] `make lint` - 0 issues
- [ ] `make test` - all 30 packages pass

---

## Last UAT Status

| Field | Value |
|-------|-------|
| Last Session | 2026-03-18 |
| Result | ALL PASS (8/8) |
| Results File | results/UAT-E20-F08-20260318-results.md |
| Red-Team | 0 CRITICAL, 1 HIGH, 3 MEDIUM, 4 LOW |

**Previous Sessions:** 2026-03-18 - Full UAT + Codex red-team. All 8 scenarios PASS. 4 follow-up items identified.
