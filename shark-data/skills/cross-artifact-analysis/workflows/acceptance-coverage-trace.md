---
inputs:
  - artifacts: list of {layer, identifier, content} including at least one parent and its children
  - parent_child_map: declared parent→child relationships (optional)
  - report_path: where the coverage report should be written (optional; otherwise returned inline)
outputs:
  - traceability_matrix: parent requirement → child criterion(s) with coverage status
  - coverage_gaps: list of {parent_requirement, status, evidence, severity, remediation}
  - orphaned_criteria: child criteria with no parent requirement they trace to
  - verdict: COVERED | COVERED_WITH_WARNINGS | UNCOVERED
---

# Workflow: Acceptance Coverage Trace

## Purpose

Build a traceability matrix linking each requirement stated in a parent spec to the child acceptance criteria that would prove it was satisfied, and flag the two failure modes:

- **Coverage gaps** — a parent requirement with no child criterion that validates it. The requirement was promised but nothing downstream proves it gets met.
- **Orphaned criteria** — a child criterion that traces to no parent requirement. Either the child took on unauthorized scope, or the parent forgot to state a requirement the child correctly anticipated.

This workflow answers: **"Did the children actually promise to verify everything the parent asked for, and did they promise anything the parent never asked for?"**

## What Counts as a Requirement vs a Criterion

- A **requirement** (parent side) is a stated need or capability: "the system must let a reviewer reject a submission", "exports must complete within the configured limit", numbered MVP items, "in scope" bullets.
- An **acceptance criterion** (child side) is a verifiable condition that, if true, demonstrates a requirement is met: "given a submission in review, when the reviewer rejects it with a reason, the submission moves to rejected and the reason is recorded".

Requirements describe *what*; criteria describe *how you'd know it's done*. A healthy spec set has every parent requirement decomposed into one or more child criteria.

## Execution Steps

### Step 1: Collect parent requirements

From each parent artifact, extract every requirement and assign it a stable local reference (R1, R2, …). Capture the evidence quote and location. Include explicit out-of-scope statements — they are negative requirements ("must NOT do X") and children can violate them.

### Step 2: Collect child criteria

From each child artifact, extract every acceptance criterion / success condition and assign a reference (C1, C2, …) tagged with its source artifact. Capture evidence quotes. Anything phrased as "done when", "must", "should", "verify that", or a numbered acceptance item qualifies.

### Step 3: Map criteria to requirements

For each child criterion, determine which parent requirement(s) it helps satisfy. A criterion may cover part of a requirement, all of it, or contradict it. Record the linkage. Use the claim-alignment techniques in `../context/comparison-techniques.md` to match criteria to requirements that are worded differently — the same need is often phrased one way in the parent and another in the child.

### Step 4: Build the traceability matrix

| Parent req | Description | Covered by | Status |
|---|---|---|---|
| R1 | Reviewer can reject with a reason | C3, C4 | ✅ COVERED |
| R2 | Export completes within configured limit | — | ❌ MISSING |
| R3 | Submissions are immutable after approval | C7 (but C7 allows edits) | ❌ CONTRADICTED |
| R4 | Out of scope: bulk import | C9 (adds bulk import) | ❌ CONTRADICTED (violates out-of-scope) |

Statuses: `COVERED`, `PARTIAL` (some aspects unverified), `MISSING` (no criterion), `CONTRADICTED` (a criterion conflicts with the requirement).

### Step 5: Identify orphaned criteria

List every child criterion that maps to no parent requirement. For each, judge whether it is:
- **Unauthorized scope** — the child is verifying something the parent never asked for (WARNING, or BLOCKER if it also violates an out-of-scope statement).
- **A parent gap** — the criterion is reasonable and the *parent* should have stated the requirement (WARNING; remediation targets the parent).

### Step 6: Rate severity

- **BLOCKER** — a `MISSING` status on a requirement that is essential to the parent's purpose, or any `CONTRADICTED` status (including violating an out-of-scope statement).
- **WARNING** — a `PARTIAL` status, or an orphaned criterion.
- **INFO** — a fully covered requirement noted for completeness, or trivial wording mismatches that didn't affect mapping.

### Step 7: Decide the verdict and write the report

- **COVERED** — every parent requirement `COVERED`, no contradictions, no concerning orphans.
- **COVERED_WITH_WARNINGS** — only `PARTIAL` statuses and/or orphaned criteria.
- **UNCOVERED** — any `MISSING` essential requirement or any `CONTRADICTED` status.

Emit the report per `../context/consistency-report-template.md`, scoped to coverage. Include the full matrix, the orphaned-criteria list, evidence for every gap, and a remediation per finding ("add a child criterion for R2", "remove or escalate the unauthorized criterion C9", "add R-new to the parent to legitimize C-orphan").

## Anti-Patterns to Avoid

- **Counting restatements as coverage.** A child that merely repeats the parent's requirement verbatim has not provided a *verifiable* criterion. Coverage requires a condition you could check, not an echo.
- **Treating every orphan as a defect.** Some orphaned criteria reveal a gap in the *parent*, not an overreach in the child. Classify before condemning.
- **Ignoring negative requirements.** Out-of-scope statements are requirements too; a child criterion that does the forbidden thing is a CONTRADICTED finding, not an orphan.

## Success Criteria

1. Every parent requirement extracted and referenced, including out-of-scope statements.
2. Every child criterion extracted and referenced.
3. Mapping built from criteria to requirements.
4. Traceability matrix produced with a status per requirement.
5. Orphaned criteria identified and classified.
6. Each gap carries severity, evidence, and remediation; verdict justified by the worst finding.
