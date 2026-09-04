---
name: quality
description: Authoritative source for all validation, code review, and quality assurance workflows. Provides consistent quality gates across all development phases.
version: 1.0.0
created: 2025-12-09
inputs:
  - entity_type: the type of entity being validated (task|feature|epic|design|code)
  - entity_key: the entity key being validated
  - spec_path: absolute path to the specification document
  - implementation_paths: list of paths to implemented code or documentation
  - test_paths: list of paths to test files (optional)
  - design_refs: list of paths to design documents or wireframes (optional)
  - acceptance_criteria: list of acceptance criteria text to validate against
outputs:
  - selected_workflow: one of {validate-design, validate-tasks, review-code, test-planning, qa-testing, generate-standards, defect-class-sweep}
  - outcome: pass | fail | blocked
  - validation_report: structured validation results (PASS|FAIL)
  - issues_found: list of {severity, description, location, remediation}
  - coverage_analysis: validation coverage and gaps
  - recommendations: actionable recommendations for improvement
---

# Quality Skill

This skill provides quality assurance capabilities with standardized validation and review workflows for maintaining high-quality deliverables throughout the development lifecycle.

## Outcome contract

Every quality response includes one semantic `outcome`: `pass`, `fail`, or
`blocked`. `pass` and `fail` describe a completed validation decision; `blocked`
means required evidence, environment access, or authority is unavailable. The
host, not this craft skill, maps the outcome to workflow state.

## Workflow Selection

Based on what needs validation, invoke the appropriate workflow:

### Design Document Validation
**When**: Validating feature design documentation before task generation
**Invoke**: `workflows/validate-design.md`
**Inputs**: `feature_prd_path`, `design_doc_paths` (expected design docs), `epic_prd_path` (optional), `interaction_map_path` (optional), `tasks_dir_path` (confirms tasks not yet generated)
**Output**: `validation_report`, `gaps` (`{doc_type, missing_section, severity}`), `prd_completeness_issues`, `verdict` (`PASS | PASS_WITH_WARNINGS | FAIL`)
**Use case**: Ensure all required design docs exist and are complete

### Task Readiness Validation
**When**: Validating tasks are complete and ready for implementation
**Invoke**: `workflows/validate-tasks.md`
**Inputs**: `task_specs` (`{task_id, spec_path, ac_list, depends_on, assigned_agent, estimated_time}`), `feature_prd_path`, `design_doc_paths`, `interaction_map_path` (optional), `tasks_index_path`
**Output**: `validation_report`, `blockers`/`warnings` (`{task_id, issue_type, description}`), `dependency_graph`, `integration_coverage`, `verdict` (`READY | READY_WITH_WARNINGS | NOT_READY`)
**Use case**: Verify tasks are properly structured and sequenced

### Code Review
**When**: Reviewing code implementation against requirements and standards
**Invoke**: `workflows/review-code.md`
**Inputs**: `task_spec_path`, `feature_prd_path`, `acceptance_criteria`, `changed_files`, `diff_summary`, `language`, `toolchain`, `coding_standards_path` (optional), `prior_rejection_count`, `codex_command`
**Output**: `verdict` (`PASS | PASS-with-triage | FAIL`), `code_review_report`, `blockers`/`non_blockers_to_triage`/`nits`, `counter_factual_per_ac`, `production_caller_chains`, `codex_assessment_verbatim`
**Use case**: Assess code quality, PRD alignment, and engineering standards

### Test Planning (Shift-Left QA)
**When**: Reviewing task specifications BEFORE development begins
**Invoke**: `workflows/test-planning.md`
**Inputs**: `task_spec_path`, `feature_prd_path`, `ac_list`, `impl_signals` (optional), `is_first_task_in_feature` (gates the Step 4.5 coverage check), `sibling_task_specs` (when first task), `codex_command`
**Output**: `test_plan_doc`, `tc_list` (`{tc_id, ac_id, scenario, technique, iso_characteristics, ...}`), `cross_feature_contract_tests`, `drift_findings`, `codex_verdict` (`PASS | CONCERNS | FAIL`), `verdict` (`APPROVED | NEEDS_REFINEMENT`)
**Use case**: Compare task spec against feature PRD for drift, write test plan that becomes the developer's source of truth. This is the quality gate between planning and development.

### QA Testing (Post-Development)
**When**: Validating implementation after code review
**Invoke**: `workflows/qa-testing.md`
**Inputs**: `task_spec_path`, `feature_prd_path`, `test_plan_path` (optional), `impl_paths`, `test_paths`, `acceptance_criteria`, `has_frontend` (gates `dev_server_command`)
**Output**: `verdict` (`PASS | FAIL`), `qa_report`, `exploratory_findings`, `bugs` (`{severity, summary, reproduction, expected, actual, fix_location}`), `wiring_coverage`
**Use case**: Validate implementation against pre-written test plan and acceptance criteria

### Defect-Class Sweep
**When**: A code-review, QA, or approval kickback; a UAT/red-team re-review round after a prior rejection; or a development/rework pass on a task carrying a rejection section or a kickback reason naming a defect class
**Invoke**: `workflows/defect-class-sweep.md`
**Inputs**: `finding` (the point-instance finding that triggered the sweep), `touched_module_paths` (default search scope), `repair_record` (prior fix/disposition history for this class, if any), `decision_sources` (prior designs/decisions to search — feature/epic notes, tech-debt records, prior review-finding notes, spec/standards docs), `calling_gate` (`code_review | approval | uat_redteam | qa`, drives which rubric re-runs during full-class re-verification) — this workflow's contract is richer than the other entries above; see the file's own frontmatter for the authoritative shape.
**Output**: an I-03 `DefectClassSweep` record — `class_key`, `class_statement`, `search_scope`, `prior_designs`, `searched_count`/`matching_count`/`fixed_count`/`dispositioned_count`/`open_count`, `instances[]` (`fingerprint`, `site_pointer`, `disposition`, `evidence`), `guard` (`kind`, `implementation_pointer`, `counterfactual_pointer`, `status`), and `status` (`open`|`complete`) — nested in the calling gate's `GateResult.remediation_sweeps` array. A severity conflict does not close inside this record: it routes to the outer `GateResult.Finding.disposition = severity_conflict` (with `disposition_pointer`) instead.
**Use case**: Generalize one finding into a defect class, enumerate every sibling instance across the declared search scope in one pass, and require a verified guard before the class closes

## Quality Resources

All workflows reference these quality criteria files:

- `context/design-validation-criteria.md` - Design document requirements
- `context/task-validation-criteria.md` - Task completeness checks
- `context/review-rubric.md` - Code review standards
- `context/quality-gates.md` - General quality standards

## Usage Pattern

Commands and agents reference this skill using:

```markdown
## Your Process
1. Analyze what needs validation
2. Invoke quality skill: quality/workflows/validate-{type}.md
3. Apply criteria from quality/context/{criteria-name}.md
4. Generate validation report with actionable feedback
```

## Quality Standards

All validation activities must:
- Be objective and measurable
- Provide clear pass/fail criteria
- Include actionable feedback for failures
- Reference specific standards and requirements
- Generate consistent, structured reports

---

*For detailed usage instructions, see [README.md](./README.md)*
