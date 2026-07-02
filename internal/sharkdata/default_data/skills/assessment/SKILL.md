---
name: assessment
description: Workflow decision-making assessments for project management. Use when evaluating feature complexity (SIMPLE/STANDARD/COMPLEX tier assignment), validating scope (feature vs task classification), checking phase readiness (gate validation), or estimating implementation effort. Invoked during feature triage, scope validation, phase transitions, or when complexity/effort estimates are needed for planning.
inputs:
  # The assessment skill exposes four distinct activities (modes). Each has its own input/output
  # contract. The host selects mode and supplies inputs accordingly.
  - mode: one of "complexity_triage" | "scope_validation" | "readiness_check" | "effort_estimation"

  # mode = complexity_triage
  - feature_title: short title of the feature being triaged (string)
  - feature_description: feature description / scope text (string)
  - epic_context: parent epic business context (string or path to epic PRD)
  - codebase_summary: existing-pattern summary, file counts, integration points (string, optional — produced upstream by research)
  - existing_task_count: integer count of already-decomposed tasks if any (optional)

  # mode = scope_validation
  - work_item_description: text describing the candidate work (string)
  - estimated_task_count: integer (optional, if known)
  - estimated_loc: integer lines of code (optional)
  - estimated_duration_days: number (optional)
  - is_user_visible: bool (optional — caller may pre-classify)

  # mode = readiness_check
  - gate_id: gate identifier from the readiness-gates reference (e.g. "G1_triage", "G3_ba_refinement", "G4_tech_refinement", "G5_task_generation", "G6_autonomous_build", "G7_uat", "T1_development", "T2_code_review", "T3_qa", "T4_approval")
  - phase_artifacts: map of artifact_name → path or content for deliverables produced in the current phase
  - upstream_state: structured summary of upstream phase outputs (e.g. complexity_tier, prd_path, arch_path, task_list, test_results)

  # mode = effort_estimation
  - file_impact_score: 0-3 (from triage)
  - task_count: integer
  - pattern_novelty_score: 0-3 (from triage)
  - regression_risk_score: 0-3 (from triage)
  - context_multiplier_reason: optional adjustment rationale (e.g. "team learning new tech")

outputs:
  # mode = complexity_triage
  - complexity_score: integer 0-27
  - tier: SIMPLE | STANDARD | COMPLEX
  - dimension_scores: map of 9 dimensions → score + rationale
  - triage_report: structured markdown report
  - autonomous_build_feasible: bool (with rationale)
  - tier_rationale: 1-2 sentence explanation

  # mode = scope_validation
  - classification: FEATURE | TASK | SPLIT | CONSOLIDATE
  - decision_rationale: structured analysis text
  - recommended_action: text describing next step (create as feature/task, split into N, consolidate)

  # mode = readiness_check
  - verdict: PASS | FAIL
  - entry_criteria_results: list of {criterion, status, note}
  - deliverables_results: list of {deliverable, status, note}
  - exit_criteria_results: list of {criterion, status, note}
  - blocking_issues: list of strings (empty on PASS)
  - required_actions: list of strings (empty on PASS)
  - readiness_report: structured markdown

  # mode = effort_estimation
  - base_effort_days: number
  - adjusted_effort_days: number
  - confidence_level: HIGH | MEDIUM | LOW
  - assumptions: list of strings
  - risks: list of strings
  - estimate_report: structured markdown
---

# Assessment Skill (craft)

Standardized assessment workflows for workflow routing, scope validation, quality gates, and effort estimation. This skill is the **craft layer** — it provides the methodology for each decision activity. The host selects a mode, supplies the inputs above, receives the structured outputs, and is responsible for translating those outputs into workflow state changes.

## Overview

The assessment skill provides four distinct decision-making activities, each invoked independently via `mode`:

1. **Complexity Triage** — Score features across 9 dimensions to assign SIMPLE/STANDARD/COMPLEX tier.
2. **Scope Validation** — Determine if work belongs at feature or task level (or should be split/consolidated).
3. **Readiness Check** — Validate phase transition requirements (entry criteria, deliverables, exit criteria).
4. **Effort Estimation** — Size work items for planning and capacity allocation.

Each activity produces a structured output. The host decides what to do with that output (advance status, store metadata, route through workflow, etc.).

## Mode Router

| Mode | Workflow file | Primary references |
|---|---|---|
| `complexity_triage` | `workflows/complexity-triage.md` | `references/complexity-dimensions.md`, `references/tier-thresholds.md` |
| `scope_validation` | `workflows/scope-validation.md` | `references/scope-criteria.md` |
| `readiness_check` | `workflows/readiness-check.md` | `references/readiness-gates.md` |
| `effort_estimation` | `workflows/effort-estimation.md` | `references/tier-thresholds.md` |

Load the workflow file matching the requested mode. The slug remains `assessment`; only the edit surface is split by responsibility.

## Resources

- `references/complexity-dimensions.md` — full 9-dimension scoring rubric
- `references/scope-criteria.md` — feature vs task classification guidance
- `references/tier-thresholds.md` — default tier definitions and customization guidance
- `references/readiness-gates.md` — gate criteria and deliverables

## Editing Guide

- Change `SKILL.md` when the input or output contract changes.
- Change `workflows/*.md` when the process for a specific mode changes.
- Change `references/*.md` when the scoring rubric or gate criteria changes.
