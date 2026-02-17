# Quality Gate Failure Analysis - E07-F30 Template Engine

**Date**: 2026-02-15
**Feature**: E07-F30 (Template Engine for Orchestrator Instructions)
**Issue Type**: Quality gate failure - design requirement not implemented
**Severity**: Medium (functionality gap, not critical)
**Status**: Resolved (quality workflows updated)

---

## Executive Summary

During User Acceptance Testing (UAT) for E07-F30, we discovered that `template_directory` configuration was **hardcoded to "templates"** instead of being configurable via `.sharkconfig.json`, despite the feature PRD explicitly specifying it should be configurable.

This document analyzes:
1. **What was missed**: The specific requirement that slipped through
2. **Why it was missed**: Root cause analysis of the quality gate failure
3. **How we're fixing it**: Workflow improvements to prevent recurrence

---

## The Issue

### What Was Specified

**Feature PRD (docs/plan/E07-enhancements/E07-F30-template-engine/feature.md)**

The PRD included a config schema in the **Technical Design** section (line 430):

```go
type Config struct {
    TemplateDirectory string `json:"template_directory,omitempty"`
    MarkdownTemplateDirectory string `json:"markdown_template_directory,omitempty"`
    // ... other fields
}
```

**Design Intent**: Teams should be able to organize templates in custom locations via `.sharkconfig.json`.

### What Was Implemented

**Actual Implementation (internal/templates/orchestrator_renderer.go:83)**

```go
func GetOrchestratorEngine() *OrchestratorRenderer {
    engineOnce.Do(func() {
        templateDir := "templates"  // ❌ HARDCODED
        if testTemplateDir != "" {
            templateDir = testTemplateDir
        }
        engineInstance, engineError = NewOrchestratorRenderer(templateDir)
        // ...
    })
    return engineInstance
}
```

The directory path was hardcoded with no config integration.

### Impact

- Teams cannot customize template locations
- Affects **both** template systems:
  - Orchestrator templates (`templates/` directory)
  - Markdown entity templates (`shark-templates/` directory)
- Not a critical bug, but reduces flexibility

---

## Root Cause Analysis

### Primary Cause: Design Element Without Acceptance Criterion

The feature PRD documented the config schema in the **Technical Design** section, but **did not include a corresponding acceptance criterion** to validate this functionality.

**What was in the PRD:**
- ✅ Technical Design section: Config struct with `template_directory` field
- ❌ Acceptance Criteria section: No AC for config field validation

**Result**: Task decomposition followed acceptance criteria, not design elements. No task was created to implement the config fields.

### Contributing Factor 1: Test-Planning Phase Bypassed

E07-F30 **skipped the `in_test_planning` phase entirely**:
- No `test_plans/` directory exists for E07-F30
- No QA-authored test plan created before development
- Developers implemented directly from task specs without independent test validation

**Why this matters**: Test-planning is designed to catch spec drift by comparing task specs against the parent feature PRD. This would have flagged the missing config implementation.

### Contributing Factor 2: Workflow Gap in validate-design.md

The `validate-design.md` workflow checks:
- ✅ All required design documents exist
- ✅ Required sections present
- ✅ No code implementation in design docs
- ✅ No placeholders (TODO, TBD)
- ✅ Mermaid diagrams present

**But it does NOT check**:
- ❌ Whether all design elements have acceptance criteria
- ❌ Mapping of design → AC → task

**Result**: The workflow validated the PRD structure but not the AC completeness.

### Contributing Factor 3: QA Testing Against Incomplete ACs

QA testing (`qa-testing.md` workflow) validates implementation against acceptance criteria.

**Problem**: If a design element has no AC, QA has nothing to test against. QA can only validate what's specified in ACs.

**E07-F30 Example**:
- QA validated all explicit ACs (template engine singleton, conditionals, partials, etc.)
- All AC-based tests passed ✅
- But no AC existed for config fields → no test was written → gap went unnoticed

---

## Timeline of the Failure

1. **Feature PRD Created** (E07-F30)
   - Technical Design section includes config schema
   - Acceptance Criteria section does NOT include config validation
   - **Gap introduced here** ✋

2. **Design Validation** (`validate-design.md`)
   - Checks document structure ✅
   - Does NOT check AC coverage ❌
   - **Gap not caught**

3. **Task Decomposition**
   - Tasks created from acceptance criteria
   - No AC for config → no task created
   - **Gap persists**

4. **Test-Planning Phase**
   - **E07-F30 skipped this phase** ❌
   - If it had run, would have flagged missing task for PRD requirement
   - **Gap not caught**

5. **Development**
   - Developers implement tasks as specified
   - Task specs don't mention config (correctly, since no AC existed)
   - **Correct behavior given incomplete spec**

6. **Code Review**
   - Validates code matches task spec ✅
   - Task spec doesn't mention config (because no AC)
   - **Gap not caught**

7. **QA Testing**
   - Tests against acceptance criteria ✅
   - All ACs pass ✅
   - No AC for config → no test for config
   - **Gap not caught**

8. **UAT Scenario 1** ← **Gap DETECTED here** ✅
   - User validates implementation against original PRD
   - Notices template_directory is hardcoded
   - **First point where gap is visible**

---

## Why UAT Caught It (When Others Didn't)

UAT is **different from QA testing** in a critical way:

| QA Testing | UAT |
|------------|-----|
| Validates against **acceptance criteria** | Validates against **original PRD intent** |
| "Does it meet the AC?" | "Does it solve the problem we set out to solve?" |
| Task-level validation | Feature-level + epic-level validation |
| Tests **what was specified** | Tests **what was needed** |

**UAT's strength**: It goes back to the source (the PRD design section) rather than relying on derived artifacts (ACs, task specs). This allows it to catch gaps in the AC coverage.

---

## The Fix: Two-Level Validation

We implemented two new validation steps to prevent this type of gap:

### Fix 1: validate-design.md - Step 2.5 (Feature-Level)

**New Step**: PRD Acceptance Criteria Completeness Check

**When**: After design documents are created, **before task generation**

**What it does**:
1. Scans PRD for technical design elements:
   - "Config Updates" or "Configuration"
   - "Database Schema" or "Data Model"
   - "API Changes" or "Endpoints"
   - "File Structure" or "Directory Layout"
   - "Migration Strategy"
   - Any other technical implementation details

2. For each design element found:
   - Note **what** it specifies (e.g., "add template_directory config field")
   - Note **where** it's located (section name + line number)
   - Note **type** (Config/Schema/API/File/etc.)

3. Checks if each design element has a corresponding acceptance criterion

4. Creates coverage matrix:

   | Design Element | Location | Has AC? | AC Reference | Notes |
   |----------------|----------|---------|--------------|-------|
   | template_directory config | Config Updates, line 430 | ❌ | None | BLOCKER: No AC |
   | templates/ directory | File Structure, line 28 | ✅ | AC-3.3 | OK |
   | OrchestratorRenderer | Technical Design, line 29 | ✅ | AC-1.4, AC-1.5 | OK |

5. **If missing ACs found**:
   - Set validation status to **FAIL**
   - Block progression to task generation
   - Document missing ACs in validation report
   - Require PRD update before proceeding

**Updated Success Criteria**:
```markdown
9. All design elements in PRD have corresponding acceptance criteria (Step 2.5)
```

**Updated Common Issues**:
```markdown
- **Design Elements Without ACs**: Add explicit acceptance criteria for all
  technical design elements (config fields, schemas, APIs, file structures)
  to PRD before task generation
```

### Fix 2: test-planning.md - Step 4.5 (Task-Level)

**New Step**: Feature-Level Coverage Check (First Task Only)

**When**: During test-planning phase, when reviewing the **first task** in a feature

**Why first task only**: This validates feature decomposition and PRD completeness once per feature, not per task. It's a feature-level check, not a task-level check.

**What it does**:
1. Identifies the first task in execution order:
   ```bash
   FIRST_TASK=$(shark task list --feature=$FEATURE_ID --json | jq -r 'sort_by(.execution_order)[0].task_id')

   if [ "$TASK_ID" = "$FIRST_TASK" ]; then
     # Run feature-level coverage check
   fi
   ```

2. Extracts all design elements from PRD (same as Step 2.5)

3. For each design element, validates:
   - ✅ Has an acceptance criterion in the PRD
   - ✅ Has a task to implement it
   - ✅ Task references the AC

4. Creates coverage matrix:

   | Design Element | Has AC? | AC Reference | Has Task? | Task Reference |
   |----------------|---------|--------------|-----------|----------------|
   | template_directory config | ❌ | None | ❌ | None |
   | templates/ directory | ✅ | AC-3.3 | ✅ | T-E07-F30-003 |

5. **If design elements lack ACs**:
   ```bash
   shark task note add $TASK_ID --type blocker \
     "Feature PRD incomplete: design elements without ACs. See test plan for details."

   shark task update $TASK_ID --status ready_for_refinement_tech \
     --note "PRD missing ACs for: template_directory config. Add ACs before task generation."
   ```

6. **If tasks don't cover all design elements**:
   ```bash
   shark task update $TASK_ID --status ready_for_decomposition \
     --note "Missing tasks for PRD requirements. See test plan for gaps."
   ```

---

## How This Prevents Future Failures

### Scenario: If E07-F30 Were Created Today

**Step 1: Feature PRD Created**
- PRD includes config schema in Technical Design section
- PRD does NOT include AC for config validation
- (Same as before)

**Step 2: Design Validation (validate-design.md Step 2.5)**
- Workflow scans PRD for design elements
- Finds: "template_directory config field" in Config Updates section
- Checks for corresponding AC
- **FINDS NONE** ❌
- **VALIDATION FAILS**
- **BLOCKS TASK GENERATION**

**Output**:
```markdown
## PRD Completeness Issues ❌

### Design Elements Without Acceptance Criteria

1. **template_directory config field** (Config Updates, line 430)
   - Design: `TemplateDirectory string json:"template_directory,omitempty"`
   - Missing AC: How to verify this config field works?
   - **Recommended AC**: "AC-X.X: Template engine reads template_directory
     from config (defaults to 'templates' if missing)"

### Impact
Without ACs for these design elements:
- No validation during development that they work
- QA won't know what to test
- UAT will have no criteria to verify against

### Recommendation
**FAIL validation. Feature PRD must be updated with missing ACs before task generation.**
```

**Step 3: PRD Updated**
- Author adds AC-6.3: "Template engine reads template_directory from config"
- Author adds AC-6.4: "Markdown templates read markdown_template_directory from config"
- Re-runs validation → PASS

**Step 4: Task Generation**
- Tasks created from complete set of ACs
- T-E07-F30-009 created: "Add template_directory configuration"

**Step 5: Test-Planning (Step 4.5 - First Task)**
- QA reviews first task
- Extracts all design elements from PRD
- Verifies each has AC ✅
- Verifies each has implementing task ✅
- **Validation PASSES**

**Result**: Gap is caught at design validation, **before any development work begins**.

---

## Lessons Learned

### For PRD Authors

**DO**:
- ✅ Create an acceptance criterion for **every** technical design element
- ✅ If you document a config field, create an AC to validate it works
- ✅ If you document a database schema, create an AC to validate migrations/queries
- ✅ If you document an API endpoint, create an AC to validate request/response

**DON'T**:
- ❌ Include design elements in Technical Design section without corresponding ACs
- ❌ Assume "it's obvious" - if it's not in an AC, it won't be tested
- ❌ Treat ACs as optional documentation - they drive task creation and QA testing

### For QA

**Understand the Limitation**:
- QA can only test what has acceptance criteria
- If PRD has incomplete ACs, QA will have incomplete test coverage
- This is **by design** - QA validates the spec, not the intent

**New Responsibility**:
- When reviewing the **first task** in a feature (test-planning Step 4.5)
- Validate that all PRD design elements have ACs and implementing tasks
- Flag any missing coverage **before development begins**

### For Workflow Design

**Validation Must Happen in Layers**:
1. **Design validation**: Check PRD AC completeness (before task generation)
2. **Test-planning**: Check task coverage of ACs (before development)
3. **QA testing**: Validate implementation against ACs (after development)
4. **UAT**: Validate against original intent (final check)

Each layer catches a different type of gap.

---

## Implementation Status

### Completed ✅

- [x] Root cause analysis documented (this file)
- [x] Updated `~/.claude/skills/quality/workflows/validate-design.md`
  - Added Step 2.5: PRD Acceptance Criteria Completeness Check
  - Updated Success Criteria (added criterion #9)
  - Updated Common Issues section
- [x] Updated `~/.claude/skills/quality/workflows/test-planning.md`
  - Added Step 4.5: Feature-Level Coverage Check (First Task Only)
- [x] Created task T-E07-F30-009 to implement missing config fields

### Pending 🔄

- [ ] Apply new validation steps to existing features (backfill)
- [ ] Update specification-writing skill templates to emphasize AC coverage
- [ ] Add validation step examples to PRD template
- [ ] Document AC coverage requirement in CLAUDE.md

---

## Related Documents

- **Feature PRD**: [E07-F30 Template Engine](../E07-enhancements/E07-F30-template-engine/feature.md)
- **Task Created**: [T-E07-F30-009](../E07-enhancements/E07-F30-template-engine/tasks/T-E07-F30-009.md)
- **UAT Document**: [UAT-E07-F30.md](../../uat/E07/UAT-E07-F30.md)
- **Quality Workflows**:
  - [validate-design.md](../../../.claude/skills/quality/workflows/validate-design.md)
  - [test-planning.md](../../../.claude/skills/quality/workflows/test-planning.md)

---

## Appendix: Validation Step Examples

### Example 1: Config Field Validation

**PRD Design Element**:
```go
type Config struct {
    RetryAttempts int `json:"retry_attempts"`
}
```

**Required Acceptance Criterion**:
```markdown
### AC-X.X: Retry Configuration

**Given** a .sharkconfig.json file with `retry_attempts: 5`
**When** the system encounters a retriable error
**Then** it should retry up to 5 times before failing
```

### Example 2: Database Schema Validation

**PRD Design Element**:
```sql
CREATE TABLE api_keys (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

**Required Acceptance Criteria**:
```markdown
### AC-Y.1: API Key Storage

**Given** a new API key is generated
**When** stored in the database
**Then** the key is saved with a unique constraint and timestamp

### AC-Y.2: API Key Retrieval

**Given** an API key exists in the database
**When** queried by key value
**Then** the system returns the key record with creation timestamp
```

### Example 3: File Structure Validation

**PRD Design Element**:
```
templates/
├── task/
│   ├── ready_for_development.tmpl
│   └── ready_for_qa.tmpl
└── partials/
    └── _tdd_process.tmpl
```

**Required Acceptance Criterion**:
```markdown
### AC-Z.1: Template Directory Structure

**Given** the system initializes
**When** templates are loaded
**Then** all required template files exist in the correct directory structure
**And** partials use _prefix naming convention
```

---

**Last Updated**: 2026-02-15
**Author**: Claude (UAT Session)
**Reviewed By**: N/A
**Status**: Complete
