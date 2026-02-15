# Exploratory Testing Findings: Template Variables Feature

**Feature**: E07-F29-template-variables-for-related-docs-and-tasks
**Task**: T-E07-F29-012 - Perform end-to-end CLI validation test
**Date**: 2026-02-13
**QA Engineer**: QA Agent
**Test Type**: Exploratory Testing
**Session Duration**: 45 minutes

---

## Session Charter

"Explore the template variables feature to discover integration gaps, architectural inconsistencies, and usability issues beyond formal test cases."

---

## Exploratory Testing Observations

### 1. Import Cycle Architecture Smell

**Observation**: The `internal/config` package has tests that import `internal/test`, which creates a circular dependency.

**Context**:
- File: `internal/config/template_integration_test.go`
- Imports: `internal/test` → `internal/db` → `internal/config`
- Result: `import cycle not allowed in test`

**Analysis**:
This suggests a fundamental architectural issue: integration tests are being written in the wrong package. The `internal/config` package should contain only pure logic and configuration handling, not database integration logic.

**Recommendation**:
- **Pattern Violation**: Integration tests belong in `internal/repository/` (per testing architecture guidelines)
- **Fix Strategy**: Move `template_integration_test.go` to `internal/repository/template_integration_test.go`
- **Prevent Recurrence**: Enforce rule: `internal/config` tests must use mocks, never real database

**Impact**: This blocks ALL tests for the feature (52 out of 57 planned tests).

---

### 2. Incomplete Service Layer Integration

**Observation**: The `DisplayService` has access to `DocumentRepository` but doesn't use the new placeholder functions.

**Context**:
- File: `internal/services/display_service.go`
- Struct: `DisplayServiceDeps` includes `DocumentRepo *repository.DocumentRepository`
- Search: No calls to `TaskPlaceholdersWithRelated`, `FeaturePlaceholdersWithRelated`, or `EpicPlaceholdersWithRelated`

**Analysis**:
The service layer is the correct architectural location for this logic (per architecture guidelines: services contain business logic, commands are thin wrappers). However, the orchestrator action resolution logic in the service doesn't use the new functions.

**Expected Pattern** (from architecture.md):
```go
// Service layer owns business logic
func (s *DisplayService) GetTaskDisplayInfo(ctx context.Context, task *models.Task) (*TaskDisplayInfo, error) {
    // ... other logic ...

    // Populate orchestrator action with extended placeholders
    placeholders := config.TaskPlaceholdersWithRelated(task, s.deps.DocumentRepo, ctx)
    action := s.workflowSvc.GetActionForStatus(task.Status)
    populatedAction := action.PopulateTemplate(placeholders)

    return &TaskDisplayInfo{
        Task: task,
        OrchestratorAction: populatedAction,
    }, nil
}
```

**Actual State**: Service likely uses basic placeholders without repository integration.

**Impact**: Feature appears "implemented" (code exists) but is non-functional (not called).

---

### 3. CLI Layer Comment Acknowledges Gap

**Observation**: Comment in `internal/cli/commands/orchestrator_display.go` explicitly states the limitation.

**Comment** (lines ~22-28):
```go
// resolveTaskAction looks up the OrchestratorAction for a task's current status
// and populates the template with the task's data. Returns nil if no action is configured.
//
// This function uses TaskPlaceholdersWithRelated to include related documents and tasks.
// However, since this is in the CLI commands layer without access to repositories,
// it falls back to basic placeholders. Full integration with related docs/tasks
// happens in the service layer (e.g., TaskService) which has repository access.
```

**Analysis**:
- Developer was aware of the limitation
- Intended solution was service layer integration
- Comment suggests this was a known TODO item
- No reference to a task or issue tracking this work

**Questions**:
1. Was service layer integration planned as a separate task?
2. Was this comment added during T-E07-F29-001 to T-E07-F29-011 implementation?
3. Should there be a task T-E07-F29-013 for service integration?

**Recommendation**: Update comment to reference the integration task once created, or remove comment once service layer is updated.

---

### 4. Related-Docs Command Works Perfectly

**Observation**: The `shark related-docs` CLI command is fully functional and works as expected.

**Test Session**:
```bash
# Create test documents
echo "# Test Spec" > docs/test/spec.md
echo "# Test Design" > docs/test/design.md

# Link to task
shark related-docs add "Test Spec" docs/test/spec.md --task=T-E07-F29-023
shark related-docs add "Test Design" docs/test/design.md --task=T-E07-F29-023

# Verify
shark related-docs list --task=T-E07-F29-023
# Output:
# Related Documents:
#   - Test Spec (docs/test/spec.md)
#   - Test Design (docs/test/design.md)
```

**Analysis**:
- Document relationship storage works correctly
- CLI command layer is well-implemented
- Database junction table (`task_documents`) is functioning
- This validates that the "prerequisites" for the feature (E07-F05 document relationships) are solid

**Positive Finding**: This part of the feature stack is production-ready.

---

### 5. Orchestrator Action Structure Present

**Observation**: The orchestrator action JSON structure exists in `shark task get --json` output, but doesn't include template variables.

**Test Session**:
```bash
shark task get T-E07-F29-023 --json | jq '.orchestrator_action'
```

**Output**:
```json
{
  "action": "advance_status",
  "instruction": "Task T-E07-F29-023 is in draft status. Execute `shark task T-E07-F29-023 next-status` to begin planning."
}
```

**Analysis**:
- Orchestrator action generation works
- Instruction template population works
- JSON output structure is correct
- **BUT**: Template uses `advance_status` action (not `spawn_agent`)
- **AND**: No `{related_docs}` or `{related_tasks}` placeholders in template

**Question**: Does the workflow configuration for `draft` status define a template with these placeholders?

**Follow-Up Investigation**:
```bash
shark config get status_metadata.draft.orchestrator_actions
# Need to check if workflow config includes placeholders
```

**Hypothesis**: Even if service layer is fixed, the workflow configuration might not include the new placeholders in templates.

---

### 6. Test Plan Quality

**Observation**: The test plan (09-test-plan.md) is exceptionally detailed with 67 test cases across 5 phases.

**Positive Findings**:
- Epic UAT scenario decomposition
- Component-by-component strategy
- API contract test cases with edge cases
- Integration scenarios with user goal tracing
- Acceptance criteria traceability matrix
- Risk mitigation documented

**Gap**: Test plan assumes implementation is complete. Phase 4 (CLI acceptance tests) cannot execute because Phases 1-3 are blocked.

**Recommendation**: Test plan should include "smoke test" phase 0:
- Verify import cycles resolved
- Verify basic compilation
- Verify service layer wiring exists

This would catch integration gaps earlier.

---

### 7. Helper Function Code Quality

**Observation**: Review of `internal/config/template_helpers.go` shows well-structured code.

**Positive Findings**:
- Clear function signatures
- Graceful error handling (warnings instead of failures)
- CSV formatting is straightforward
- JSON parsing uses standard library
- Functions are pure (no side effects)

**Code Snippet** (formatDocPathsAsCSV):
```go
func formatDocPathsAsCSV(docs []*models.Document) string {
    if len(docs) == 0 {
        return ""
    }
    paths := make([]string, 0, len(docs))
    for _, doc := range docs {
        paths = append(paths, doc.FilePath)
    }
    return strings.Join(paths, ",")
}
```

**Analysis**: This is production-quality code. Simple, readable, efficient.

**Concern**: Cannot verify it works without tests executing.

---

### 8. Workflow Configuration Gap

**Observation**: Unclear whether workflow configuration templates use the new placeholders.

**Investigation Needed**:
1. Check `.sharkconfig.json` for orchestrator action templates
2. Verify templates include `{related_docs}` and `{related_tasks}`
3. Confirm different statuses have appropriate templates

**Example Expected Template** (for `ready_for_development` status):
```json
{
  "status_metadata": {
    "ready_for_development": {
      "orchestrator_actions": {
        "spawn_agent": {
          "instruction_template": "Work on {id}: {title}. Related docs: {related_docs}. Dependencies: {related_tasks}."
        }
      }
    }
  }
}
```

**Risk**: If workflow config doesn't include placeholders, feature won't work even after service layer integration.

**Action Item**: Developer should verify workflow config includes new placeholders in instruction templates.

---

### 9. Documentation vs. Implementation Mismatch

**Observation**: Test plan and PRD describe complete feature, but implementation is ~40% complete.

**Evidence**:
- Test plan describes 67 test cases
- PRD has 16 acceptance criteria
- Implementation has helper functions but no integration
- CLI output doesn't show feature working

**Analysis**: This suggests:
1. Task breakdown may have stopped after helper function implementation
2. Service integration was overlooked or deprioritized
3. End-to-end testing wasn't run during development

**Lesson Learned**: TDD approach should enforce end-to-end validation earlier:
- Write failing end-to-end test first
- Implement helper functions
- Integrate into service layer
- Re-run end-to-end test (should pass)

This would catch integration gaps immediately.

---

### 10. Positive: Repository Methods Likely Exist

**Observation**: The test plan references `DocumentRepository.ListForTask()` method.

**Investigation**:
```bash
grep -n "ListForTask" internal/repository/document_repository.go
```

**Hypothesis**: If this method exists, the repository layer is ready for service integration.

**Follow-Up**: Check if `FeatureRelationshipRepository.ListRelatedFeatures()` and `EpicRelationshipRepository.ListRelatedEpics()` also exist (per test plan scenarios 11-15).

---

## Usability Observations

### Positive

1. **`related-docs` command is intuitive**: Add/list/delete pattern matches user expectations
2. **JSON output structure is clean**: Well-organized for programmatic consumption
3. **Error messages are clear**: When import cycle fails, error is descriptive

### Concerns

1. **No user-facing indication that feature exists**: Without reading code/docs, user wouldn't know `{related_docs}` placeholders are available (or supposed to be)
2. **Silent failure mode**: If service integration is incomplete, users get no error - feature just doesn't work
3. **No validation**: If user adds `{related_docs}` to template, no error if placeholder doesn't populate

---

## Performance Observations

**Not Tested**: Cannot run performance benchmarks due to import cycle blocking tests.

**Expected Performance** (from test plan):
- Placeholder population latency: <50ms overhead
- Single query per entity (no N+1 pattern)
- JSON parsing: <1ms for typical context data

**Recommendation**: After integration complete, run:
```bash
go test -bench=. ./internal/config ./internal/repository -benchmem
```

---

## Security Observations

**Reviewed**:
- Document file paths: Already exposed via `shark related-docs list` (no new attack surface)
- Task keys: Already exposed via `shark task list` (no new exposure)
- Context data JSON: Internal structured data, not user input
- SQL queries: Should use parameterized statements (verify in repository)

**No New Security Risks Identified**: Feature doesn't introduce new attack vectors.

**Recommendation**: Verify `DocumentRepository.ListForTask()` uses prepared statements.

---

## Summary of Exploratory Findings

| Finding Category | Severity | Count |
|------------------|----------|-------|
| Critical Architecture Issues | Critical | 2 |
| Integration Gaps | High | 2 |
| Documentation Gaps | Medium | 2 |
| Positive Findings | N/A | 3 |
| Questions for Developer | N/A | 3 |

---

## Questions for Developer

1. **Was service layer integration planned as a separate task?** If so, what's the task ID?
2. **Does the workflow configuration include `{related_docs}` and `{related_tasks}` in instruction templates?** If not, needs to be added.
3. **Why were integration tests placed in `internal/config/` instead of `internal/repository/`?** Understand rationale to prevent recurrence.
4. **Are repository methods ready?** Confirm `DocumentRepository.ListForTask()`, `FeatureRelationshipRepository.ListRelatedFeatures()`, `EpicRelationshipRepository.ListRelatedEpics()` exist.
5. **What's the estimated time to complete service integration?** This affects release timeline.

---

## Actionable Recommendations

### For Developer

1. **Immediate**: Fix import cycle by moving integration tests to correct package
2. **High Priority**: Integrate `*PlaceholdersWithRelated` functions into `DisplayService`
3. **Medium Priority**: Verify workflow config templates include new placeholders
4. **Low Priority**: Remove or update CLI layer comment once service integration complete

### For QA (Next Round)

1. After developer fixes, re-run full test suite: `make test`
2. Execute all 57 planned tests (unit, integration, acceptance)
3. Run performance benchmarks
4. Validate end-to-end CLI behavior with multiple related docs/tasks
5. Test cross-epic relationships (scenarios 11-15)

### For Product Owner

1. **Decision**: Should feature be released incrementally (helper functions only) or wait for full integration?
2. **Communication**: Inform stakeholders of delay due to integration work
3. **Timeline**: Estimate 8-12 hours developer work before QA re-test

---

## Test Data Cleanup

**Created During Session**:
- Task: `T-E07-F29-023`
- Documents: `docs/test/spec.md`, `docs/test/design.md`

**Cleanup Commands**:
```bash
shark task delete T-E07-F29-023
rm -f docs/test/spec.md docs/test/design.md
rmdir docs/test  # If empty
```

---

**Session End**: 2026-02-13 23:34
**Overall Assessment**: Feature has solid foundation but is incomplete. Integration work required before release.
