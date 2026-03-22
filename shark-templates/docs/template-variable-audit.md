# Template Variable Audit

> Audit date: 2026-03-16
> Scope: All 88 shared templates in ~/projects/shark-templates

## Variables Available but Not Used Where They Should Be

### 1. `{{.complexity_tier}}` — available directly, but fetched via CLI

Multiple feature templates run `shark feature get {{.id}} --json | jq -r '.metadata.complexity_tier'` when `{{.complexity_tier}}` is already a template variable. This wastes an agent CLI call on every invocation.

**Affected templates:**
- `feature/in_triage.tmpl`
- `feature/ready_for_triage.tmpl`
- `feature/in_refinement_ba.tmpl`
- `feature/ready_for_refinement_ba.tmpl`
- `feature/in_refinement_tech.tmpl`
- `feature/ready_for_refinement_tech.tmpl`
- `feature/in_test_planning.tmpl`
- `feature/ready_for_test_planning.tmpl`
- `feature/in_task_generation.tmpl`
- `feature/ready_for_task_generation.tmpl`
- `feature/ready_for_ba_check.tmpl`
- `feature/ready_for_tech_check.tmpl`

### 2. `{{.depends_on}}` — never used in task templates

When a developer picks up `ready_for_development`, they have no idea what this task depends on. They should check dependency outputs before starting. Same for code review and QA — they should know the dependency chain to understand integration context.

**Affected templates:**
- `task/ready_for_development.tmpl`
- `task/in_development.tmpl`
- `task/ready_for_code_review.tmpl`
- `task/in_code_review.tmpl`
- `task/ready_for_qa.tmpl`
- `task/in_qa.tmpl`

### 3. `{{.blocked_reason}}` — not shown in blocked templates

All three `blocked.tmpl` files (task, feature, epic) say "wait for blocker resolution" but don't render `{{.blocked_reason}}`. The user has to run a separate command to find out why.

**Affected templates:**
- `task/blocked.tmpl`
- `feature/blocked.tmpl`
- `epic/blocked.tmpl`

### 4. `{{.files_changed}}` — not used in code review or QA

`ready_for_code_review` and `ready_for_qa` tell agents to review "implementation code changes" but don't point them at `{{.files_changed}}`. If populated from the dev phase, this would focus the review immediately.

**Affected templates:**
- `task/ready_for_code_review.tmpl`
- `task/in_code_review.tmpl`
- `task/ready_for_qa.tmpl`
- `task/in_qa.tmpl`

### 5. `{{.completion_notes}}` — not used anywhere

When a task loops back from code review to development, the developer gets zero context about what the reviewer found. `{{.completion_notes}}` from the previous phase could carry that forward.

**Affected templates:**
- `task/ready_for_development.tmpl` (when returning from code review)
- `task/in_development.tmpl` (when returning from code review)

### 6. `{{.linked_entity_key}}` / `{{.linked_entity_type}}` — not used in bug templates

When a bug is linked to a feature or task, the analysis and fix templates should tell the agent to read that entity's docs. Currently bug templates are disconnected from the feature context.

**Affected templates:**
- `bug/ready_for_analysis.tmpl`
- `bug/in_analysis.tmpl`
- `bug/ready_for_fix.tmpl`
- `bug/in_fix.tmpl`
- `bug/ready_for_verification.tmpl`
- `bug/in_verification.tmpl`

### 7. `{{.business_value}}` — never used in epic templates

Research, feasibility reviews, and decomposition should all be informed by business value. It's available but ignored.

**Affected templates:**
- `epic/ready_for_research.tmpl`
- `epic/in_research.tmpl`
- `epic/ready_for_feasibility_review_ba.tmpl`
- `epic/in_feasibility_review_ba.tmpl`
- `epic/ready_for_feasibility_review_tech.tmpl`
- `epic/in_feasibility_review_tech.tmpl`
- `epic/ready_for_decomposition.tmpl`
- `epic/in_decomposition.tmpl`

### 8. `{{.description}}` — almost never used

Templates always say "READ: at `{{.file_path}}`" but the description is available inline. Showing it gives the agent immediate context before reading the full file.

**Affected templates:** Nearly all — especially useful for task and bug templates where a one-line description can orient the agent before it reads the spec.

### 9. Change-card variables — mostly unused

`{{.justification}}`, `{{.impact_analysis}}`, and `{{.rollback_plan}}` exist but the change templates don't reference them. The `approved.tmpl` and `in_progress.tmpl` templates just say "READ: Change-card description" without surfacing this structured data.

**Affected templates:**
- `change/approved.tmpl`
- `change/in_progress.tmpl`
- `change/proposed.tmpl`

### 10. `{{.execution_order}}` — never used

Available for both tasks and features but not referenced. Agents driving active features/epics should use this for sequencing rather than relying on the agent to figure it out.

**Affected templates:**
- `feature/active.tmpl`
- `epic/active.tmpl`

### 11. `{{.priority}}` — available for tasks but unused

Task priority should inform how thorough the development, review, and QA processes are.

**Affected templates:**
- `task/ready_for_development.tmpl`
- `task/in_development.tmpl`
- `task/ready_for_code_review.tmpl`
- `task/ready_for_qa.tmpl`

---

## New Variables That Would Make a Real Difference

These don't exist in shark yet but would eliminate the most common "first thing the agent does is run a CLI command" patterns.

### 1. `{{.parent_title}}`

Task templates know `{{.feature_key}}` but not the feature title. Every agent has to `shark feature get` just to understand context.

### 2. `{{.previous_status}}`

Is this task in `ready_for_development` for the first time, or was it rejected from code review? Templates can't branch on this. Currently agents check notes to figure it out.

### 3. `{{.notes_summary}}` or `{{.latest_note}}`

Almost every `in_*` resume template starts with "check notes via `shark notes`". If the latest note was injected, agents could branch immediately.

### 4. `{{.sibling_progress}}`

`active.tmpl` for both epics and features tells the agent to list children and check statuses. A summary like "3/7 completed, 1 blocked" would let the template give smarter instructions.

### 5. `{{.context_data}}` fields as template variables

The PLAN GATE pattern (`shark feature context get {{.id}} --json`) is repeated in 8+ templates. If `remaining_steps`, `completed_steps`, and `implementation_decisions` were template variables, the resume logic could be handled in template conditionals instead of CLI calls.

---

## Priority Ranking

| # | Item | Impact | Effort |
|---|------|--------|--------|
| 1 | `complexity_tier` direct use | High — eliminates CLI call in 12 templates | Template-only fix |
| 2 | `depends_on` in task templates | High — prevents dependency violations | Template-only fix |
| 3 | `blocked_reason` in blocked templates | Medium — UX improvement | Template-only fix |
| 4 | `linked_entity_key` in bug templates | Medium — better bug-to-feature context | Template-only fix |
| 5 | Change-card variables | Medium — structured data available but ignored | Template-only fix |
| 6 | `description` inline | Low-Medium — nice-to-have context | Template-only fix |
| 7 | `files_changed` in review/QA | Medium — depends on dev phase populating it | Template-only fix |
| 8 | `business_value` in epics | Low — informs but doesn't block | Template-only fix |
| 9 | `execution_order` in active templates | Low — agents figure it out via list | Template-only fix |
| 10 | `priority` in task templates | Low — rarely drives behavior differences | Template-only fix |
| 11 | `completion_notes` on loop-back | Medium — prevents rework | Template-only fix |
| 12 | `previous_status` (new var) | High — enables smart branching | Requires Go code change |
| 13 | `parent_title` (new var) | Medium — saves CLI call per task | Requires Go code change |
| 14 | `context_data` fields (new vars) | High — eliminates PLAN GATE CLI calls | Requires Go code change |
| 15 | `latest_note` (new var) | High — eliminates resume CLI calls | Requires Go code change |
| 16 | `sibling_progress` (new var) | Medium — smarter active templates | Requires Go code change |
