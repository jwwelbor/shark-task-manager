---
name: assessment-readiness-check
mode: readiness_check
---

# Workflow: Readiness Check

**Purpose**: Validate that a work item meets the entry criteria, deliverables, and exit criteria for a given gate.

**Key principle**: Gates protect downstream phases from incomplete work. Better to catch issues early than discover them mid-implementation.

## Process

### Step 1: Resolve Gate Criteria

Look up the gate by `gate_id` in `../references/readiness-gates.md`. Each gate defines:

- Entry criteria
- Phase deliverables
- Exit criteria

### Step 2: Validate Entry Criteria

Walk each entry criterion. For each, check `phase_artifacts` and `upstream_state`. Mark PASS or FAIL with a note explaining the evidence.

### Step 3: Validate Deliverables

Walk each phase deliverable. For each, check whether the artifact exists, is complete, and meets the structural requirements.

### Step 4: Validate Exit Criteria

Walk each exit criterion. Mark PASS or FAIL.

### Step 5: Compute Verdict

- All entry criteria PASS + all deliverables PASS + all exit criteria PASS -> **PASS**
- Any FAIL -> **FAIL**

Verdict rules:

- "Almost complete" is FAIL.
- "Could fix later" is FAIL.
- "Conditional pass" is not a valid verdict.

### Step 6: Generate Readiness Report

```markdown
# Readiness Check — {gate_id}

**Date**: {date}

## Entry Criteria
- [x|fail] {criterion} — {note}

## Phase Deliverables
- [x|fail] {deliverable} — {note}

## Exit Criteria
- [x|fail] {criterion} — {note}

## Validation Result

**Verdict**: {PASS|FAIL}

{If FAIL}
**Blocking Issues**:
- {issue 1}
- {issue 2}

**Required Actions**:
- {action 1}
- {action 2}
```

### Step 7: Return Structured Output

Return: `verdict`, `entry_criteria_results`, `deliverables_results`, `exit_criteria_results`, `blocking_issues`, `required_actions`, `readiness_report`.
