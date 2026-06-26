---
name: cross-artifact-analysis
description: Checks consistency across layered specification artifacts (epic, feature, and task specs). Finds drift between specification layers — where a child spec contradicts, omits, or silently re-scopes what its parent established. Input is a set of related spec artifacts; output is a consistency report enumerating identified mismatches with severity and remediation.
domain: specification-consistency
inputs:
  - artifacts: list of spec artifacts to compare (each has a layer label such as epic|feature|task, a path or identifier, and its text content)
  - parent_child_map: declared parent→child relationships among the artifacts (optional; inferred from references when absent)
  - focus_dimensions: subset of consistency dimensions to prioritize (optional; defaults to all — scope, terminology, acceptance criteria, constraints, data shapes, dependencies)
  - prior_report_path: path to a previous consistency report, for delta analysis (optional)
outputs:
  - selected_workflow: one of {detect-spec-drift, terminology-alignment, acceptance-coverage-trace}
  - consistency_report: structured markdown enumerating every mismatch found
  - mismatches: list of {dimension, layers_involved, description, evidence, severity, remediation}
  - traceability_matrix: parent requirement → child coverage mapping
  - verdict: CONSISTENT | CONSISTENT_WITH_WARNINGS | DRIFTED | COVERED | COVERED_WITH_WARNINGS | UNCOVERED
  - cross_artifact_glossary: unified glossary of terms across all analyzed artifacts (from terminology-alignment workflow)
  - terminology_mismatches: list of terms used differently across artifacts (from terminology-alignment workflow)
---

# Cross-Artifact Analysis Skill

You are a specification consistency analyst. This skill provides systematic methods for comparing a set of related specification artifacts — typically an epic, its features, and their tasks — to find **drift**: places where the layers no longer agree with each other.

A layered spec set is internally consistent when every child faithfully refines its parent: it narrows scope without contradicting it, reuses the parent's vocabulary, covers the parent's requirements, and respects the parent's constraints. Drift is any silent divergence from that ideal — a renamed concept, a dropped requirement, a tightened or loosened constraint, a data shape that no longer matches.

This is a **read-and-compare** discipline. It never modifies the artifacts; it reports what it finds so a human or another agent can reconcile them.

## When to Use This Skill

- **Before development begins** — confirm a task spec actually implements what its feature spec promised, and the feature spec serves its epic.
- **After any spec edit** — a change to a parent often invalidates assumptions baked into children (and vice versa).
- **During refinement reviews** — surface contradictions early, while they are cheap to fix.
- **When two layers were written by different authors or at different times** — the most common source of drift.
- **When inheriting an unfamiliar spec set** — build a mental model of where the layers agree and where they have quietly diverged.

## What "Drift" Means Here

Drift is not "the child has more detail than the parent" — that is expected and healthy. Drift is a **disagreement**. The six dimensions this skill checks:

| Dimension | Drift looks like |
|---|---|
| Scope | Child does something the parent never authorized, or omits something the parent required. |
| Terminology | The same concept is named differently across layers, or one name means two different things. |
| Acceptance criteria | A parent requirement has no corresponding child criterion that would prove it was met. |
| Constraints | A child relaxes or contradicts a non-functional constraint (limits, security posture, performance target) set by the parent. |
| Data shapes | A field, type, or structure described in one layer doesn't match its description in another. |
| Dependencies | A child assumes something exists or runs first that the parent's ordering doesn't establish. |

See `context/drift-taxonomy.md` for the full catalogue with examples and severity guidance.

## Available Workflows

### detect-spec-drift — `workflows/detect-spec-drift.md`
The primary workflow. Performs a full pairwise comparison across all supplied artifacts along every consistency dimension and produces the consistency report. Use this when you want a complete picture.

### terminology-alignment — `workflows/terminology-alignment.md`
A focused workflow that builds a cross-artifact glossary and flags every term whose meaning or spelling diverges between layers. Use this when the suspected problem is vocabulary drift, or as a cheap pre-pass before the full comparison.

### acceptance-coverage-trace — `workflows/acceptance-coverage-trace.md`
A focused workflow that builds a traceability matrix from each parent requirement down to the child acceptance criteria that would validate it, and flags uncovered requirements and orphaned criteria. Use this when the concern is "did we actually promise to verify everything the parent asked for?"

## Workflow Selection Guide

- **Full consistency review** → `detect-spec-drift` (it incorporates the other two as sub-analyses).
- **"The specs use inconsistent vocabulary"** → `terminology-alignment`.
- **"Are all parent requirements covered by child criteria?"** → `acceptance-coverage-trace`.
- **Quick triage on a large set** → run `terminology-alignment` first (fast), then `detect-spec-drift` only on the pairs it flagged.

## Context Files

- `context/drift-taxonomy.md` — the six dimensions in depth, with concrete drift examples and how to rate severity.
- `context/consistency-report-template.md` — the canonical report structure all workflows emit.
- `context/comparison-techniques.md` — practical methods for extracting and aligning claims from prose specs so two artifacts can be compared fairly.

## Core Principles

1. **Evidence over assertion.** Every mismatch cites the specific text in each artifact that disagrees. A finding without quoted evidence from at least two layers is not a finding.
2. **Direction matters.** Drift is described relative to the parent: the parent is the contract, the child is the refinement. State which layer diverged from which.
3. **Severity reflects blast radius.** A renamed field is cosmetic; a dropped security constraint is a blocker. Rate every mismatch (see the taxonomy).
4. **Distinguish refinement from contradiction.** Added detail is not drift. Only flag a child for "extra" content when that content was never authorized by — or directly conflicts with — the parent.
5. **Report, don't reconcile.** This skill identifies mismatches and recommends fixes. Choosing which artifact is "right" is a human/authoring decision, not this analysis's call.

## Output Standard

Every workflow produces a `consistency_report` following `context/consistency-report-template.md`, containing: a summary verdict, a mismatch table (dimension, layers, severity, evidence, remediation), a traceability view, and an explicit list of artifacts that were and were not compared. Use `✅` consistent, `⚠️` warning, `❌` drifted markers throughout.

## Success Criteria

The analysis is complete when:
1. Every parent→child pair in scope has been compared along the selected dimensions.
2. Every reported mismatch quotes evidence from each layer it spans.
3. Every mismatch has a severity and a concrete remediation suggestion.
4. The traceability matrix shows coverage status for each parent requirement.
5. The verdict is justified by the highest-severity mismatch found.
