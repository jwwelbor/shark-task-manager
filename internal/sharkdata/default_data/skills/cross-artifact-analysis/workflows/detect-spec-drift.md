---
inputs:
  - artifacts: list of {layer, identifier, content} for every spec to compare (layer is e.g. epic|feature|task)
  - parent_child_map: declared parent→child relationships (optional; inferred from references if absent)
  - focus_dimensions: subset of {scope, terminology, acceptance_criteria, constraints, data_shapes, dependencies} (optional; defaults to all)
  - report_path: where the consistency report should be written (optional; otherwise returned inline)
outputs:
  - consistency_report: structured markdown per ../context/consistency-report-template.md
  - mismatches: list of {dimension, layers_involved, description, evidence, severity, remediation}
  - traceability_matrix: parent requirement → child coverage status
  - verdict: CONSISTENT | CONSISTENT_WITH_WARNINGS | DRIFTED
---

# Workflow: Detect Spec Drift

## Purpose

Compare a set of layered specification artifacts and enumerate every place where the layers disagree. This is the primary cross-artifact-analysis workflow; it covers all six consistency dimensions and produces the full consistency report.

The output answers one question for a reviewer: **"Where have these specs quietly diverged from each other, how bad is each divergence, and what would it take to reconcile them?"**

## Mental Model

Treat the artifact set as a tree. The parent at each edge is the contract; the child is its refinement. A consistent edge means the child narrows the parent without contradicting it. Drift is any edge where the child and parent disagree. Your job is to walk every edge and inspect it along each dimension.

You are comparing **prose**, not structured data. Two specs can describe the same thing in different words (consistent) or the same words for different things (drifted). The work is mostly in extracting comparable claims from each artifact before judging them — see `../context/comparison-techniques.md`.

## Execution Steps

### Step 1: Establish the comparison graph

1. If `parent_child_map` is supplied, use it directly.
2. Otherwise, infer relationships from the artifacts: a child names or references its parent; a parent enumerates its children; layer labels (epic > feature > task) imply nesting.
3. List every parent→child edge you will examine. Record any artifact that has **no** established relationship — it cannot be compared and must be reported as such (an orphan is itself a finding worth noting).

### Step 2: Extract claims from each artifact

For each artifact, distill its content into a set of comparable claims, grouped by dimension. Use the extraction methods in `../context/comparison-techniques.md`. At minimum capture:

- **Scope claims** — what this artifact says it will and won't do; in/out-of-scope statements.
- **Defined terms** — every named concept, entity, role, field, or state, with the local meaning.
- **Acceptance criteria / success conditions** — anything phrased as "must", "should", "is done when", or numbered criteria.
- **Constraints** — limits, thresholds, security/privacy posture, performance targets, compatibility requirements.
- **Data shapes** — named structures, fields, types, formats, enumerated values.
- **Dependencies / ordering assumptions** — "after X", "requires Y to exist", "builds on Z".

Keep a pointer (quote + location) for every extracted claim so findings can cite evidence.

### Step 3: Compare each parent→child edge, dimension by dimension

For each edge, and each dimension in `focus_dimensions`, ask the dimension's drift question (full catalogue in `../context/drift-taxonomy.md`):

- **Scope** — Does the child do anything the parent never authorized? Does it omit a child-relevant scope item the parent required? Does it contradict an explicit out-of-scope statement?
- **Terminology** — Is any shared concept named differently across the two layers? Does any single term carry two different meanings? (Defer the full vocabulary sweep to the terminology-alignment workflow; here, flag the obvious collisions on this edge.)
- **Acceptance criteria** — Does each applicable parent requirement have a child criterion that would prove it satisfied? Does the child assert criteria the parent contradicts? (Defer exhaustive coverage tracing to the acceptance-coverage-trace workflow; here, flag direct conflicts and glaring gaps.)
- **Constraints** — Does the child relax, tighten, or contradict any parent constraint? A child silently allowing what the parent forbade is high severity.
- **Data shapes** — Do field names, types, formats, or allowed values match across layers? A field that is optional in one layer and required in another is drift.
- **Dependencies** — Does the child assume an ordering or precondition the parent doesn't establish? Does it depend on a sibling the parent never linked?

Record each disagreement as a mismatch: `{dimension, layers_involved, description, evidence (quote from each layer), severity, remediation}`.

### Step 4: Rate severity

Apply the severity scale from `../context/drift-taxonomy.md`:

- **BLOCKER** — a contradiction that, if implemented as written, produces wrong or unsafe behavior, or violates a parent constraint that exists for a reason (security, data integrity, compliance, performance budget). Also: a required parent capability with zero child coverage.
- **WARNING** — a divergence that is unlikely to cause incorrect behavior but creates ambiguity, maintenance risk, or reviewer confusion (e.g., inconsistent terminology, an orphaned child criterion, added scope that is plausibly in-bounds but unauthorized).
- **INFO** — a cosmetic or stylistic difference worth noting but not requiring action (e.g., a synonym used consistently within one layer).

When in doubt between two levels, rate up and explain the uncertainty in the description.

### Step 5: Build the traceability matrix

Roll up the acceptance and scope dimensions into a matrix: each parent requirement on a row, with its coverage status in the child layer(s) — `COVERED`, `PARTIAL`, `MISSING`, or `CONTRADICTED` — and the child criterion (or absence) that justifies the status. This is the at-a-glance view of how faithfully children serve their parents.

### Step 6: Decide the verdict

- **CONSISTENT** — no mismatches above INFO across all examined edges.
- **CONSISTENT_WITH_WARNINGS** — one or more WARNING-level mismatches, no BLOCKERs.
- **DRIFTED** — at least one BLOCKER-level mismatch.

The verdict is governed by the single highest-severity finding.

### Step 7: Write the report

Emit the report following `../context/consistency-report-template.md`. It must include:

- The verdict and a one-paragraph summary.
- The full mismatch table.
- The traceability matrix.
- An explicit "Compared / Not Compared" list naming every artifact and every edge, so the reader knows the coverage of the analysis itself.
- Per-mismatch remediation suggestions phrased as "reconcile by …" (e.g., "rename the child's term to match the parent", "add a child criterion for parent requirement R3", "raise the contradiction to the spec authors — the layers genuinely disagree").

## Anti-Patterns to Avoid

- **Flagging refinement as drift.** A child with more detail than its parent is doing its job. Only flag added content when it conflicts with, or was forbidden by, the parent.
- **Single-sided evidence.** "The task doesn't mention X" is only a finding if the parent *required* X. Quote both sides.
- **Reconciling silently.** Do not pick a winner and rewrite. Report the disagreement and recommend; let an author decide.
- **Comparing unrelated artifacts.** Only edges in the comparison graph are valid comparisons. Two unrelated tasks under different features should not be diffed against each other.
- **Ignoring orphans.** An artifact with no parent or child relationship is a structural finding, not something to skip.

## Success Criteria

1. Every edge in the comparison graph examined along every selected dimension.
2. Every mismatch cites quoted evidence from each layer it spans.
3. Every mismatch carries a severity and a concrete remediation.
4. Traceability matrix produced with a status for each parent requirement.
5. Report names every artifact under "Compared / Not Compared".
6. Verdict justified by the highest-severity mismatch.
