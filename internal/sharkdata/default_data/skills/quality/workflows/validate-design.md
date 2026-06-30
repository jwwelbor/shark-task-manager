---
inputs:
  - feature_prd_path: absolute path to the feature PRD markdown
  - design_doc_paths: list of absolute paths to expected design documents (README, architecture, database, API spec, frontend, security/performance, implementation phases, test criteria, research report)
  - epic_prd_path: absolute path to the parent epic PRD markdown (optional)
  - interaction_map_path: absolute path to parent `<epic-id>-interaction-map.md` if present
  - tasks_dir_path: absolute path to the feature's tasks/ directory (used to confirm tasks have not yet been generated)
  - validation_report_path: absolute path where the validation report markdown should be written
outputs:
  - validation_report: structured markdown written to validation_report_path
  - gaps: list of {doc_type, missing_section, severity} where severity is BLOCKER | WARNING
  - prd_completeness_issues: list of {design_element, location, missing_ac_recommendation, severity}
  - verdict: PASS | PASS_WITH_WARNINGS | FAIL
---

# Workflow: Validate Feature Design Documentation (craft)

## Purpose

Verify that all required design documents for a feature are present and complete before proceeding to task generation. This ensures downstream task-writing has all the context it needs.

## What This Workflow Checks

### 1. Required Files Exist

For each path in `design_doc_paths`, verify the file exists. Standard expected document set:

- `README.md` — Navigation hub and overview
- Feature PRD file — name varies (`prd.md`, `feature.md`, etc.); resolved from caller
- `00-research-report.md` — Project research findings
- `02-architecture.md` — System design and integration
- `03-database-design.md` — Schema and data model
- `04-api-specification.md` — API endpoints and contracts (single source of truth for DTOs and contracts)
- `05-frontend-design.md` — UI components and state
- `06-security-performance.md` — Security and optimization
- `07-implementation-phases.md` — Timeline and phases
- `08-test-criteria.md` — Test specifications

### 2. File Completeness Check

For each design document that exists, verify required sections are present. Apply the criteria documented in `../context/design-validation-criteria.md`.

### 3. Cross-Reference Validation

Check that documents properly reference each other:
- README.md links to all design documents
- Design docs use relative paths
- No broken internal links

### 4. Anti-Pattern Detection

Flag issues that violate the architecture design guidelines:
- **No code implementation** — Check for SQL, Python, TypeScript, etc.
- **No placeholders** — Check for "TODO", "TBD", "[to be completed]"
- **Mermaid diagrams present** — Architecture and database sections have diagrams
- **No tasks created** — `tasks_dir_path` should be empty (or only contain a placeholder README) at design-validation time
- **Descriptions, not code** — All specs should be prose, not implementation

## Execution Steps

### Step 1: Validate File Existence

Read each path in `design_doc_paths` in parallel. Record which files exist and which are missing. Missing files are recorded as gaps with severity `BLOCKER`.

### Step 2: Content Analysis

For each file that exists, analyze content for:
- Required sections present (per `../context/design-validation-criteria.md`)
- Appropriate length (with ±20% tolerance)
- No code implementation
- No placeholders/TODOs
- Mermaid diagrams where required

Each finding is added to `gaps` with severity `BLOCKER` or `WARNING`.

### Step 2.5: PRD Acceptance Criteria Completeness Check

**Validate that all technical design elements have corresponding acceptance criteria.** This prevents the gap where design decisions are documented but never validated during development.

#### 2.5.1: Extract Design Elements from Feature PRD

Read `feature_prd_path` and scan for sections that specify technical implementation:

**Design Sections to Check:**
- "Technical Design" or "Architecture"
- "Config Updates" or "Configuration"
- "Database Schema" or "Data Model"
- "API Changes" or "Endpoints"
- "Migration Strategy"
- "File Structure" or "Directory Layout"
- "Template System" or "Template Engine"
- "Caching Strategy"
- "Security Implementation"

For each design element found, note:
- **What**: Description of the design decision (e.g., "add template_directory config field")
- **Where**: Section name and line number
- **Type**: Config/Schema/API/File/etc.

#### 2.5.2: Extract Acceptance Criteria

Scan PRD sections:
- "Success Criteria"
- "Acceptance Criteria"
- "MVP Requirements"
- Individual phase requirements (Phase 1, Phase 2, etc.)
- Any section with "AC-" numbered criteria

Build a list of all acceptance criteria with their references.

#### 2.5.3: Map Design Elements to ACs

Create a coverage matrix:

| Design Element | Location | Has AC? | AC Reference | Notes |
|----------------|----------|---------|--------------|-------|
| template_directory config | Config Updates, line 430 | ❌ | None | BLOCKER: No AC |
| templates/ directory | File Structure, line 28 | ✅ | AC-3.3 | OK |
| OrchestratorRenderer | Technical Design, line 29 | ✅ | AC-1.4, AC-1.5 | OK |

**Validation Rules:**
- ❌ **BLOCKER**: Design element with NO acceptance criterion
- ⚠️ **WARNING**: Design element with ambiguous AC ("works correctly")
- ⚠️ **WARNING**: AC that doesn't map to any design element (orphaned AC)

#### 2.5.4: Flag Missing ACs

For each design element lacking an AC, populate `prd_completeness_issues` with `{ design_element, location, missing_ac_recommendation, severity: BLOCKER }`. Document each gap in the validation report:

```markdown
## PRD Completeness Issues ❌

### Design Elements Without Acceptance Criteria

The following technical design elements are specified but have no acceptance criteria to validate them:

1. **template_directory config field** (Config Updates, line 430)
   - Design: `TemplateDirectory string json:"template_directory,omitempty"`
   - Missing AC: How to verify this config field works?
   - **Recommended AC**: "AC-X.X: Template engine reads template_directory from config (defaults to 'templates' if missing)"

### Impact

Without ACs for these design elements:
- No validation during development that they work
- QA won't know what to test
- UAT will have no criteria to verify against
- Implementation gaps will only be caught in production

### Recommendation

**FAIL validation. Feature PRD must be updated with missing ACs before task generation.**
```

#### 2.5.5: Update Validation Status

If any `prd_completeness_issues` have `severity = BLOCKER`, set `verdict = FAIL`.

### Step 2.6: Interaction Map Closure Check

If `interaction_map_path` exists, validate the cross-feature interaction section
before task generation:

1. Read the interaction map and extract every I-## row.
2. Read the feature PRD's `Cross-feature interactions` section.
3. Verify every I-## this feature produces or consumes is declared in the PRD.
4. Verify producer and consumer references use the SAME shape source from the map.
5. Verify each I-## has one shared contract test pointer.

Produce an interaction-map closure table:

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | Status |
|------|----------|-------------|--------------|-----------------------|--------|

Orphan wires, missing shape sources, or mismatched contract test pointers are
BLOCKER findings and set `verdict = FAIL`.

### Step 3: Generate Validation Report

Write a validation report to `validation_report_path` with:

- Summary (files found, issues, warnings)
- Detailed results for each file
- Anti-pattern detection results
- Cross-reference check results
- PRD completeness issues
- Recommendations
- Ready/Not Ready determination

### Step 4: Decide Verdict

- **PASS** when: all required files exist, all required sections present, no code, no placeholders, mermaid present where required, no tasks created yet, all cross-references valid, lengths in range, all design elements have ACs.
- **PASS_WITH_WARNINGS** when: only `WARNING`-severity findings remain.
- **FAIL** when: any `BLOCKER` finding (missing required file, missing required section, code implementation present, missing ACs for design elements, tasks already created).
- Interaction-map closure failures are BLOCKER findings.

## Success Criteria

Validation passes when:
1. All required files exist
2. All required sections present in each file
3. No code implementation found (only descriptions)
4. No placeholder text (TODO, TBD, etc.)
5. Mermaid diagrams present where required
6. No tasks created yet (tasks_dir_path empty or only contains placeholder README)
7. All cross-references and links are valid
8. File lengths within acceptable ranges
9. All design elements in PRD have corresponding acceptance criteria (Step 2.5)
10. If an interaction map exists, all relevant I-## rows close with matching
    producer/consumer declarations, shape sources, and contract test pointers

## Common Issues

- **Missing Mermaid Diagrams**: Add diagrams to architecture and database docs
- **SQL Code Found**: Replace with prose descriptions
- **TODO/TBD Placeholders**: Complete sections or remove placeholders
- **Tasks Already Created**: Tasks should not exist before design validation passes — generate them only after this validation succeeds
- **File Too Short/Long**: Adjust detail level (±20% of target)
- **Design Elements Without ACs**: Add explicit acceptance criteria for all technical design elements (config fields, schemas, APIs, file structures) to PRD before task generation

## Output Format

The validation report should clearly indicate:
- ✅ Items that pass
- ⚠️ Items with warnings
- ❌ Items that fail
- Specific line numbers for issues
- Actionable recommendations for fixing

See `../context/design-validation-criteria.md` for complete validation criteria.
