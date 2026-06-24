---
feature_key: E02-F01-extract-craft-from-scaffolding-per-workflow-file
epic_key: E02
title: Extract craft from scaffolding (per-workflow-file)
description: Split each in-scope skill workflow into craft (intrinsic to the activity) and workflow scaffolding (gates, fetches, mutations, advancements). Craft stays in skill files with declared inputs/outputs frontmatter; scaffolding moves to sidecar files awaiting prompt placement in F4.
size: L
---

# Extract craft from scaffolding (per-workflow-file)

**Feature Key**: E02-F01

---

## Epic

- **Epic PRD**: [Epic](../epic.md)

---

## Goal

### Problem

Today's shark-referenced skills (specification-writing, quality, architecture, research, implementation, test-driven-development, debugging, assessment, uat) embed shark's workflow contract directly into their content. They:

- Fetch entity context (`shark get`, `shark context`)
- Store notes via `shark --field`
- Gate transitions inline (e.g., "do not advance until codex red-team passes")
- Advance status (`shark status set ready_for_*`)

A "strip the shark vocabulary" pass would leave a skill that *looks* clean but still assumes a shark-shaped host (entity ID, parent context, status names). An audit found 401 shark-vocab hits in `specification-writing` alone across 4,078 lines, with state mutations woven into the craft.

### Solution

Apply the **craft-vs-scaffolding** decision test to every line of every in-scope skill workflow file. For each line:

> *If you removed this, would the activity still be that activity?*

- **Yes** → craft, stays in the skill workflow file.
- **No** → scaffolding, moves to a sidecar at `_extracted/<filename>.md`.

Each craft file declares an `inputs:` (and `outputs:` where meaningful) frontmatter block — the materialized host contract. Sidecars preserve the extracted scaffolding with tagged blocks (`# fetch`, `# gate`, `# mutate`, `# advance`, `# preflight`) so F4 can group recurring patterns into prompt partials.

### Impact

- Each craft skill workflow is **standalone-readable** by a stranger with no shark context.
- Every craft file declares its `inputs:` contract — prompts in F4 promise to provide these inputs.
- A `_partials_inventory.md` lists recurring scaffolding patterns (e.g., "fetch parent context" appears in ~15 sidecars), feeding F4's `_partials/` design.

---

## Reality check on effort

The original "grep-and-move" framing estimated 2–3 days. Realistic effort for the corpus is **4–6 days** of focused work after the pilot validates the rule. `specification-writing` alone has 17 workflow files; `quality` has 13; estimated ~30–50 files in scope across the 9 in-scope skills.

Unit of work is **per-workflow-file**, not per-skill. Each maps to a different shark status with its own scaffolding fingerprint.

---

## Scope — three sub-passes

### F1.a — Pilot one file end-to-end (~½ day)

Pick `quality/workflows/qa-testing.md`:
- High-leverage (you know what good looks like).
- Medium-coupled (78 hits — representative).
- Produces a reusable template.

Do the **full** split *plus* a sketched prompt mapping:

- `quality/workflows/qa-testing.md` ← craft only, with `inputs:` frontmatter
- `quality/_extracted/qa-testing.md` ← scaffolding sidecar (transient)
- Draft `shark-templates/feature/in_qa.tmpl` showing where extracted scaffolding lands

**Output**: working template for the bulk pass + concrete evidence the rule is sound. If the rule breaks somewhere, surface it here, not 30 files in.

---

### F1.b — Extract-only pass on the corpus (~3 days)

For each remaining in-scope skill workflow file: same split, but stop at the sidecar. **Don't design prompts yet** — F1.c discovers the partial structure organically.

**Decision rule per block**: *if removing it changes the activity, it's craft; otherwise it's scaffolding.*

**Heuristic shortcut** — any line touching the following is scaffolding by default:
- `shark` CLI invocations
- Status names (`ready_for_*`, `in_*`)
- Parent-context fetching (`epic context get`, `feature context get`)
- Note storage (`note add`, `context set`)
- File-path manipulation tied to shark conventions
- Status advancement (`status set`, `status advance`)

**Grey-zone resolution** (cases the simple rule won't decide):
- Codex red-team — *how to run one productively* is craft; *required before advancing* is gate (scaffolding).
- "Run linter, fix issues" — *what good linting looks like* is craft; *cannot advance with lint errors* is gate.
- Codex red-team being **mandatory** — that's a workflow-level invariant. The mandate goes in the prompt; the methodology stays in the skill.

**Output per file**:
- Craft skill workflow file with `inputs:` / `outputs:` frontmatter
- Sidecar at `_extracted/<filename>.md` capturing what was removed
- Each block in the sidecar tagged: `# fetch`, `# gate`, `# mutate`, `# advance`, `# preflight`

---

### F1.c — Corpus review + stale-ref cleanup (~½ day)

Read the `_extracted/*.md` sidecars across all in-scope skills as a single corpus. Patterns will emerge:

- "Fetch parent context" appears in ~15 sidecars → candidate for `_partials/_read_parent_context.md` in F4.
- "Run codex red-team and gate on PASS" appears in ~6 → `_partials/_codex_gate.md`.
- "Advance to next status with rationale note" appears nearly everywhere → `_partials/_advance.md`.

**Output**: `_partials_inventory.md` listing the candidate partials with which sidecars feed each. F4 turns this into the actual `_partials/` directory.

Also in this sub-pass:
- Resolve stale workflow refs in `.sharkworkflow.json`: `discovery` → `research` (4 refs); `build` → drop or replace with `devops` (1 ref). Update any `LOAD:` lines accordingly.

---

## Supplemental Batch A Extraction (2026-06-22)

The jaunty-panda extraction plan added a broader Batch A beyond the original E32-F01 in-scope shark-coupled skills.

Batch A is now complete and present in `shark-data/skills/`:

- `brownfield-analysis`
- `frontend-design`
- `product-design`

These skills were originally listed as out of scope for E32 because they were not shark-coupled workflow skills. Treat them as **supplemental skill-library content**, not as new blockers for the original F1 exit gate.

F1 acceptance still applies to the original shark-coupled skills:

- `specification-writing`
- `architecture`
- `research`
- `quality`
- `implementation`
- `test-driven-development`
- `debugging`
- `assessment`
- `uat`

Supplemental Batch A skills must follow the same layer rule before being treated as canonical shipped content:

- Methodology belongs in `shark-data/skills/`.
- Workflow scaffolding belongs in `shark-data/prompts/` or `shark-data/workflow/`.
- Skill purity audits should not count transient `_extracted/` sidecars as craft, but F4 must either consume or relocate those sidecars before final E32 acceptance.

---

## Inputs contract format

Every craft skill workflow declares its inputs in frontmatter:

```markdown
---
inputs:
  - spec_path: absolute path to the feature spec markdown
  - scope_summary: 1-paragraph description of what's being QA'd
outputs:
  - defect_report: structured markdown
---
```

Prompts (in F4) promise to provide these inputs; skills promise to operate on them without further assumptions.

---

## Acceptance Criteria / Exit gate

1. Every in-scope skill workflow is **standalone-readable** — a stranger with no shark context can execute it.
2. Every craft file declares `inputs:` (and `outputs:` where meaningful) in frontmatter.
3. Every workflow file has a corresponding tagged sidecar in `_extracted/`.
4. `_partials_inventory.md` exists, capturing recurring scaffolding patterns.
5. Stale workflow refs resolved (`discovery` → `research`; `build` dropped or replaced with `devops`).
6. Existing dispatch is **broken on this branch by design** — skills no longer carry scaffolding. F1 is **not independently mergeable**; it lands together with F4.

---

## Verification

- (a) Each craft skill workflow standalone-readable by a stranger with no shark context.
- (b) Every craft file has `inputs:` (and `outputs:` where meaningful) frontmatter.
- (c) Sidecars present in `_extracted/` with tagged blocks (`# fetch`, `# gate`, `# mutate`, `# advance`).
- (d) `_partials_inventory.md` exists.
- (e) Branch dispatch is **expected to fail** — F1 is not independently mergeable; lands with F4.
- (f) `grep -rE "shark|ready_for_|in_" skills/<scope>/` is a *signal*, not a gate — passing the grep with the host contract still embedded means the extraction was lexical, not structural.

---

## Out of Scope (for F1)

- Designing prompts (F4).
- Building partials (F4).
- Moving files to `shark-data/` (F4).
- Writing the host-contract spec (F4).
- Engine work (F2/F3).

---

## Branch hygiene

- Dedicated branch (e.g., `shark-skill-extraction`) off `shark-tuneup`.
- One commit per workflow file so individual reverts are clean.
- Branch stays open until F4 lands and re-enables dispatch.

---

## Dependencies

- **Blocks**: F4 (which moves the decoupled skills into `shark-data/skills/` and re-enables dispatch).
- **Blocked by**: none. Can run in parallel with F2.

---

## Risks

- **Grey-zone calls.** The rule won't always decide cleanly. Document the call in a `# decision:` comment in the sidecar so F4 can revisit.
- **Sidecar drift.** Sidecars are transient (consumed by F4). If F1 lingers, sidecars get stale relative to skill content updates. Mitigation: keep F1 to ~1 week and merge with F4 promptly.
- **The 4,078-line, 401-hit reality.** Don't plan as if this is mechanical find-and-replace. Budget for genuine reading and judgment.

---

*Last Updated*: 2026-06-22
