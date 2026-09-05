---
feature_key: E34-F10-product-critical-path-guard-for-delivery-workflows
epic_key: E34
title: Product Critical-Path Guard for Delivery Workflows — Specification
---

# E34-F10 Specification

See [Epic PRD](../epic.md) for business context. See
[research-report.md](./research-report.md) for the Capability map and the
four locked decisions (D-F10-01–04) this spec implements. See
[requirements.md Area 10](../requirements.md#area-10-product-critical-path-guard-for-delivery-workflows--e34-f10)
for REQ-F-032/033/034 and REQ-NF-001–003 verbatim — this spec adds only
file-level implementation detail.

This is a **content-only** feature: one new shared prompt partial plus
reference wiring across twelve existing prompts, and two new
human-authored-convention markdown files. No Go code, CLI command, or
persistence change (per D-F10-01).

## Requirements (incremental over epic)

### Functional

- **REQ-F-032 (spec)**: Define the product critical-path artifact's minimal
  shape as plain markdown (matching D01/D02's existing human-authored
  convention, per D-F10-02) at `docs/plan/product-critical-path.md`:
  sections `## Current Gate` (the current roadmap gate name/number), `## Last
  Passing Production Step` (dated, one line, links the executable evidence),
  and `## Next Executable Step`. `docs/plan/product-delivery-roadmap.md`
  gets a minimal shape too: an ordered list of gates, each with a one-line
  description and its D01/D02 traceability pointer. Neither file's schema is
  enforced by a parser — REQ-NF-001 requires only workflow-compatible
  degradation, not machine-validated structure (this is prose guidance
  content, consistent with D01/D02's own unenforced-shape precedent).
- **REQ-F-033 (spec)**: Add
  `internal/sharkdata/default_data/prompts/_partials/_product_critical_path_guard.md`
  defining one Go template block (`{{define "_product_critical_path_guard"}}`)
  that: reads the four source files if present (D01, D02, roadmap,
  critical-path), and instructs the worker to report, before selecting or
  dispatching work: the current gate name, the proposed contribution's
  relationship to that gate, executable advancement evidence (a link to a
  runnable command/test proving progress against the live golden path — see
  REQ-F-034 below for what does not qualify), any unresolved prerequisite
  (missing source file, stale roadmap, or an unclear gate), and an explicit
  disposition for any work item that does not advance the current gate
  ("side quest" — proceed only with a stated reason, e.g. urgent bug fix, or
  defer). When one or more of the four source files is absent, the guard
  reports "unresolved prerequisite: <file> missing" and does not hard-block
  dispatch (per D-F10-03) — it degrades to an advisory note rather than
  stopping work, since most Shark-managed projects (including this
  repository today) have not run the full D01–D14 product-design arc.
  Wire this partial via `{{template "_product_critical_path_guard" .}}` into
  the twelve prompts feature.md's Triage Breadcrumb names:
  `prompts/sprint/{planning,active,closing}.md`,
  `prompts/epic/{assessment,decomposition,active}.md`,
  `prompts/feature/{specification,test_planning,task_generation,task_review,approval}.md`,
  `prompts/task/development.md` (its completion-reporting section only, not
  its whole body).
- **REQ-F-034 (spec)**: The same partial's "executable advancement evidence"
  paragraph enumerates the disqualified evidence classes by name — fixture
  data, a captured/recorded run, a hand-authored test actor standing in for
  a real caller, a contract-only test that never exercises the production
  path, and a component-level test suite that stops short of the live
  golden path — reusing E34-F02's existing evidence-authenticity vocabulary
  (per research-report.md finding 5) rather than inventing parallel terms.
  Only evidence that runs against the live golden path (an executed command,
  a passing end-to-end test, a demonstrated production interaction) advances
  the gate.

### Non-functional

- **REQ-NF-001 (spec)**: A project with none of the four source files present
  renders every wired prompt with the guard reporting all four as
  "unresolved prerequisite" and otherwise proceeding exactly as before this
  feature — no existing prompt output is removed or restructured, the guard
  block is strictly additive. Verified by rendering every wired prompt
  against this repository's own project state (which today has none of the
  four files) and asserting the existing prompt content is byte-identical
  outside the new guard block.
- **REQ-NF-002 (spec)**: The guard partial contains no target-status
  string, outcome name, or dispatch decision of its own — it only reports;
  it never overrides a workflow's `outcomes:` routing or names a status to
  transition to. Rendered-prompt tests assert the partial's content contains
  no bare status-name token from any workflow YAML's `steps:` keys.
- **REQ-NF-003 (spec)**: The partial's instructions are worker-facing
  reporting guidance only — it contains no `shark status`/`shark claim`/
  `shark release`/`shark next-status` command text, consistent with every
  other dispatched-prompt's parent-loop-ownership contract already
  established across this bundle (E34-F06 through F09's prompts all follow
  this convention; this feature does not introduce a new one).

### Acceptance criteria

- AC-1: `_partials/_product_critical_path_guard.md` exists, defines exactly
  one named template, and renders cleanly through the production renderer
  with all four source files absent (this repository's current state).
- AC-2: All twelve prompts named in REQ-F-033 invoke
  `{{template "_product_critical_path_guard" .}}` exactly once; no prompt
  restates the guard's reporting fields inline (mirrors E34-F06's AC-2
  single-source-of-truth pattern and its structural, not exact-string,
  verification approach).
- AC-3: Rendering any of the twelve prompts against this repository's actual
  project state (no D01/D02/roadmap/critical-path files) produces a guard
  block reporting all four prerequisites unresolved, with no error and no
  change to the prompt's pre-existing content.
- AC-4: The guard block names the five disqualified evidence classes
  (REQ-F-034) verbatim in at least the `specification.md`, `test_planning.md`,
  and `approval.md` renders (the three gates most likely to receive a
  fixture/mock-backed claim as "evidence").
- AC-5: No wired prompt's rendered output contains a bare workflow status
  name or a `shark` CLI verb inside the guard block (REQ-NF-002/003,
  grep-verifiable).

### Out of scope

Per feature.md's Triage Breadcrumb and research-report.md's decisions: no
Go command, CLI flag, or database table (D-F10-01); no extension of the
D01–D14 product-design arc's own authoring workflow to produce the roadmap/
critical-path files (D-F10-02 — F10 defines their shape, it does not build
tooling to author them); no runtime sprint-admission enforcement (E19-F10's
scope, preserved verbatim per D-F10-04); no hard dispatch block when
prerequisite files are missing (D-F10-03).

## Architecture

### Component changes

| File | Change |
|---|---|
| `internal/sharkdata/default_data/prompts/_partials/_product_critical_path_guard.md` | NEW — the shared guard template block |
| `internal/sharkdata/default_data/prompts/sprint/planning.md`, `active.md`, `closing.md` | EDIT — add one `{{template}}` invocation each |
| `internal/sharkdata/default_data/prompts/epic/assessment.md`, `decomposition.md`, `active.md` | EDIT — same |
| `internal/sharkdata/default_data/prompts/feature/specification.md`, `test_planning.md`, `task_generation.md`, `task_review.md`, `approval.md` | EDIT — same |
| `internal/sharkdata/default_data/prompts/task/development.md` | EDIT — same, placed in the completion-reporting section only |
| `docs/plan/product-critical-path.md` (this repository's own instance) | NEW (optional, illustrative) — not required by AC-1–5, which all pass with the file absent; not created by this spec, since authoring it is a human/product decision this feature does not make on the project's behalf |

### Data model changes

None. Both new artifact files are plain markdown with no enforced schema, no
Shark entity, and no database table (REQ-NF-001, D-F10-02).

### API / interface contracts

None (no CLI command, no HTTP endpoint, no Go function signature). The
"interface" is the rendered guard-block text every wired prompt now emits.

### Key technical decisions

1. **One shared partial, twelve references** — matches E34-F06's and
   E34-F07's "one canonical source, many bare references" precedent; avoids
   the twelve-copy drift research-report.md finding 3 explicitly warns
   against.
2. **Advisory degradation, not a hard gate** — per D-F10-03, since the
   guard would otherwise block dispatch on every Shark-managed project
   (including this repository) that has not completed the D01–D14 arc.
   REQ-F-033 requires the guard be *consulted*, not that it *block*; a
   hard-block reading would contradict REQ-NF-001's compatibility
   requirement for projects "without overrides" (read here as "without the
   product-design arc completed").
3. **No new Go persistence or enforcement surface** — preserves the
   explicit boundary with E19-F10 (D-F10-04); a second enforcement path
   would create two sources of roadmap-gate truth, which REQ-NF-002/003 and
   the feature's own Triage Breadcrumb rule out.

### Integration with existing code

- `internal/templates/`: the new partial renders through the same
  production renderer already exercised by `internal/templates/includes_test.go`
  for the other `_partials/*.md` files (e.g. `_qa_process.md`); no renderer
  code change is needed.

## Cross-feature interactions

No I-## row in `E34-interaction-map.md` names E34-F10 as a producer or
consumer (grep confirmed empty) — feature.md and `E34-interaction-map.md`
both describe F10 as "an independent pre-dispatch product-alignment guard...
does not produce or consume an I-## payload."

## Cross-epic integrations

None named for E34-F10 in `E34-cross-epic-map.md` or
`docs/product/cross-epic-integration-map.md` (grep confirmed empty).

## Durable unresolved decisions

None material. The four decisions research-report.md already recorded
(D-F10-01–04) resolve every design choice this spec needed to make; no new
Q### is warranted.

*Last Updated*: 2026-09-04
