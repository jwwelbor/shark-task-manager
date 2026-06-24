---
inputs:
  - artifacts: list of {layer, identifier, content} for every spec to compare
  - parent_child_map: declared parent→child relationships (optional)
  - report_path: where the terminology report should be written (optional; otherwise returned inline)
outputs:
  - cross_artifact_glossary: every defined concept with each layer's name and meaning for it
  - terminology_mismatches: list of {concept, layers_involved, kind, evidence, severity, remediation}
  - verdict: CONSISTENT | CONSISTENT_WITH_WARNINGS | DRIFTED
---

# Workflow: Terminology Alignment

## Purpose

Build a single cross-artifact glossary from a layered spec set and flag every place where vocabulary has drifted. Terminology drift is the most common and most insidious form of spec drift: two layers describe the same thing under different names (so a reader thinks they are two things), or use one name for two different things (so a reader thinks they are one thing). Either way, downstream work is built on a misunderstanding.

This workflow is useful on its own, and as a fast pre-pass before `detect-spec-drift` — terminology collisions often point straight at the deeper scope and data-shape disagreements.

## What Counts as a Term

A "term" is any named domain concept: an entity, a role or actor, a state or status value, a field or attribute, an action or operation, a feature name, or a defined phrase. Capture both the surface name (how it's written) and the local meaning (what the artifact says it is or does).

## The Two Failure Modes

| Kind | Symptom | Why it matters |
|---|---|---|
| Synonym split | One concept, multiple names across layers ("user" in the epic, "account holder" in the feature, "member" in the task). | Readers and tools treat the names as distinct concepts; coverage analysis breaks; reused logic is duplicated. |
| Homonym collision | One name, multiple meanings across layers ("session" meaning a login session in one layer and a work session in another). | Readers conflate distinct concepts; constraints meant for one leak onto the other. |

A third, milder finding: **casing/spelling variance** of the same term (e.g., `taskKey` vs `task_key` vs "task key") — INFO unless the variance implies a real type difference.

## Execution Steps

### Step 1: Extract a term list per artifact

For each artifact, scan for named concepts and record `{surface_name, local_meaning, evidence_quote, location}`. Be generous in extraction — it's cheaper to merge two entries later than to miss a collision. Methods for spotting defined terms in prose are in `../context/comparison-techniques.md`.

### Step 2: Cluster by meaning, then by name

Build two indexes:

1. **By meaning** — group term entries that describe the same underlying concept regardless of surface name. Any cluster containing more than one distinct surface name is a candidate **synonym split**.
2. **By surface name** — group term entries that share a name regardless of meaning. Any cluster containing more than one distinct meaning is a candidate **homonym collision**.

### Step 3: Confirm and classify each candidate

For every candidate, confirm it is a genuine divergence (not just two phrasings of an identical meaning) and classify it as synonym split, homonym collision, or spelling variance. Capture the evidence quote from each layer involved.

### Step 4: Assemble the cross-artifact glossary

Produce one row per underlying concept:

| Concept (canonical) | Epic term | Feature term | Task term | Status |
|---|---|---|---|---|
| The work item being tracked | "task" | "work item" | "task" | ⚠️ synonym split (feature diverges) |
| Login session | "session" | "session" | — | ✅ aligned |
| Work session | — | "session" | "session" | ❌ homonym collision with login session |

The canonical name is your best single label for the concept; note which layer's term you adopted and why (prefer the parent's term as the contract).

### Step 5: Rate severity

- **BLOCKER** — a homonym collision where the two meanings carry different constraints or data shapes (constraints can leak across the conflated concept and cause wrong behavior).
- **WARNING** — a synonym split, or a homonym collision between clearly-separable concepts that nonetheless invites confusion.
- **INFO** — casing/spelling variance with no semantic consequence.

### Step 6: Write the report

Emit the report following `../context/consistency-report-template.md`, scoped to terminology. Include the full glossary, the mismatch list with evidence from each layer, and a remediation per mismatch phrased as "adopt `<canonical>` in `<layer>`" or "split `<name>` into two terms".

## Remediation Guidance (recommend, don't apply)

- **Synonym split** → recommend collapsing to the parent's term across all layers.
- **Homonym collision** → recommend renaming one of the two concepts so the name is unambiguous, then re-checking any constraints attached to the old name.
- **Spelling variance** → recommend a single canonical spelling; low priority.

Never rewrite the artifacts — recommend the canonical choice and let an author make it.

## Success Criteria

1. Every artifact's terms extracted with evidence.
2. Both the by-meaning and by-name indexes built.
3. Each candidate confirmed and classified (synonym / homonym / variance).
4. Glossary produced with a status per concept.
5. Each mismatch carries severity and a concrete remediation.
6. Verdict governed by the highest-severity terminology finding.
