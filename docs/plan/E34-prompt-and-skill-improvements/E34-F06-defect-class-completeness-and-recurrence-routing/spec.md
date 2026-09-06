---
feature_key: E34-F06-defect-class-completeness-and-recurrence-routing
epic_key: E34
title: Defect-Class Completeness and Recurrence Routing — Specification
---

# E34-F06 Specification

See [Epic PRD](../epic.md) for business context and [architecture.md](../architecture.md)
for the E34-wide I-02/I-03 contract definitions. See
[research-report.md](./research-report.md) for the reuse Capability map this
spec builds on.

This is a **content-only** feature: no new Go code, CLI command, or
persistence type. It adds one bundle workflow file plus reference wiring
inside `internal/sharkdata/default_data/`.

## Requirements (incremental over epic)

Traces to `feature.md` REQ-F-001 through REQ-F-007 and REQ-NF-001 verbatim
(feature.md already states these at full fidelity — see feature.md
"Requirements" section for the complete text). This spec adds only the file-
and section-level implementation detail feature.md does not carry.

### Functional

- **REQ-F-001 (spec)**: Add `internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md`
  as a new workflow file following the two-tier pattern already used by
  sibling files in that directory (`review-code.md`, `qa-testing.md`,
  `validate-tasks.md`, `validate-design.md`, `test-planning.md`,
  `generate-standards.md`). It defines: class naming, search-scope
  declaration, enumeration procedure, zero-result reporting, instance
  evidence shape, guard-selection guidance, closure rule, and the
  three-part re-verification procedure (mirrors, and supersedes as the
  canonical source for, the prose currently duplicated in
  `skills/uat/references/redteam-rubric.md` lines 44-56).
- **REQ-F-002 (spec)**: The new workflow's "Backward-looking rework" section
  instructs the worker to search, before designing a repair: affected code
  and tests (via grep/glob against the touched module), feature/epic decision
  notes (`shark <entity> notes <key>`), tech-debt records (`shark td list`),
  prior review-finding notes (`shark search` / entity notes with
  `--type=review-finding`), relevant spec.md/architecture.md sections, and
  project standards (`docs/standards/` if present). It must preserve existing
  dispositions absent new evidence.
- **REQ-F-003 (spec)**: The workflow's "Structural guard closure" section
  requires: every enumerated instance fixed or dispositioned, `open_count = 0`,
  and a `guard.status = verified` counterfactual (guard catches — flags, fails
  the build, or otherwise blocks — the class when the defect is deliberately
  re-introduced, and does not flag/fail when the defect is absent) before the
  class is reported `complete`. No feasible guard → class stays `open` with a
  linked Shark work item (task, tech-debt, or note), never silently claimed
  closed.
- **REQ-F-004 (spec)**: The workflow's "Full-class re-verification" section
  is invoked identically by code-review, QA, approval, and UAT re-review
  rounds; it re-enumerates the full declared `search_scope`, not just the
  cited fix, and reruns the calling gate's full rubric.
- **REQ-F-005 (spec)**: Recurrence classification: same fingerprint after a
  recorded repair = recurrence; a new fingerprint that belongs to the same
  `class_key` **and** lies inside a previously `status: complete` class's
  `search_scope` = recurrence (both conjuncts required — matching `class_key`
  alone outside the recorded scope, or scope membership alone under a
  different `class_key`, is not recurrence); anything else = normal finding.
  No round-count field or logic is introduced anywhere in the new content.
- **REQ-F-006 (spec)**: Disposition/severity-conflict routing references the
  existing `question-management` skill (`skills/question-management/SKILL.md`)
  for a bounded single-owner conflict, and the Shark Attack council workflow
  (`skills/shark-attack/`) for specialist disagreement, cross-entity
  inconsistency, high blast radius, irreversibility, or no safe evidence path
  — by reference/invocation only, no new escalation table.
- **REQ-F-007 (spec)**: The new workflow file and its edits to
  `code_review.md`, `approval.md`, and `redteam-rubric.md` contain no WWGM
  defect names, no Python tooling references, no test-database variable
  names, and no local filesystem paths. Guard commands and project standards
  are discovered from `{{project guidance}}` render inputs already available
  to those prompts (see `internal/templates/` render context), not hardcoded.

### Non-functional

- **REQ-NF-001 (spec)**: The new workflow's output is nested inside the
  existing I-02 `GateResult.remediation_sweeps` array (`array of I-03` per
  `architecture.md` line 164) via each gate's existing structured-output
  contract. No new Shark note type, table, or lifecycle status is added.

### Acceptance criteria

- AC-1: `internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md`
  exists, renders through the production template renderer with no errors,
  and every section named in REQ-F-001 (spec) is present.
- AC-2: `code_review.md` (line 81 template) and `approval.md` (line 54
  template) reference the new workflow instead of restating sweep prose
  inline; `redteam-rubric.md` "Defect-class sweep" step (lines 44-56)
  references it too. `qa.md` and `development.md` also reference the new
  workflow's "Full-class re-verification" / "Enumeration procedure" sections
  rather than restating them. No duplicated sweep-procedure prose remains in
  any of these five files after the edit.
- AC-3: `skills/quality/SKILL.md` "Workflow Selection" section and
  `skills/README.md` line-21 quality-skill file list both name the new
  workflow file.
- AC-4: The new content contains zero instances of `WWGM`, hardcoded Python
  tool names, or literal local filesystem paths (verified by grep in the
  test plan).
- AC-5: A rendered output sample for each of the seven acceptance scenarios
  in `feature.md` ("Close an enumerated class", "Distinguish recurrence from
  a new finding", "Route a severity conflict", plus the four scenarios named
  in the Verification plan: zero remaining instances, same-fingerprint
  recurrence, new-class/accepted-risk, missing/disabled/ineffective guard)
  demonstrates the workflow content produces the documented decision for
  that scenario.

### Out of scope

Per `feature.md` "Out of scope": arbitrary round-based escalation, a global
project-specific defect-class catalog, automatic lint-config or application
code changes, and replacing the Question/council workflows themselves.

## Architecture

### Component changes

| File | Change |
|---|---|
| `internal/sharkdata/default_data/skills/quality/workflows/defect-class-sweep.md` | NEW — the reusable workflow (REQ-F-001 through REQ-F-006) |
| `internal/sharkdata/default_data/skills/quality/SKILL.md` | EDIT — add a "Defect-Class Sweep" entry to "Workflow Selection", mirroring the existing entries' shape (When/Invoke/Output/Use case) |
| `internal/sharkdata/default_data/skills/README.md` | EDIT — append the new workflow file to the `quality` row's file list (line 21) |
| `internal/sharkdata/default_data/prompts/feature/code_review.md` | EDIT — replace the inline kickback-reason sweep template (line 81) with a reference to `skills/quality/workflows/defect-class-sweep.md`, keeping only the gate-specific kickback-reason string format |
| `internal/sharkdata/default_data/prompts/feature/approval.md` | EDIT — same replacement (line 54) |
| `internal/sharkdata/default_data/skills/uat/references/redteam-rubric.md` | EDIT — replace the "Defect-class sweep" step (lines 44-56) and "ENUMERATE — DO NOT ITERATE" duplication with a reference to the new workflow, keeping UAT-specific framing (red-team stance) |
| `internal/sharkdata/default_data/prompts/feature/qa.md` | EDIT (added during round-1 rework, HIGH-1) — reference the new workflow's "Full-class re-verification" section for QA re-review rounds |
| `internal/sharkdata/default_data/prompts/task/development.md` | EDIT (added during round-1 rework, HIGH-1) — reference the new workflow's "Enumeration procedure" for rework kicked back with a named defect class |

No Go source file changes — this is prompt/skill bundle content only, matching
E34-F05's precedent (also content/schema-only, confirmed via research-report.md
finding that `grep -rln "GateResult\|gate_result_v1" internal/` returns no
Go hits).

### Data model changes

None. I-03 DefectClassSweep v1 is already fully specified in
`architecture.md` from E34-F05's work; this feature implements a workflow
that *produces* that shape, it does not modify the shape itself.

### API / interface contracts

None (no CLI command, no HTTP endpoint). The "interface" this feature
produces is the I-03 markdown/JSON shape nested in GateResult, covered under
Cross-feature interactions below.

### Key technical decisions

1. **New file, not an edit to an existing workflow file** — no existing
   `skills/quality/workflows/*.md` file owns defect-class-sweep semantics
   (confirmed via research-report.md directory listing); a new file avoids
   overloading `review-code.md` or `qa-testing.md` with cross-cutting concerns
   they don't otherwise own.
2. **Reference, not duplicate, from every call site** — `code_review.md`,
   `approval.md`, and `redteam-rubric.md` originally each carried independent
   copies of similar sweep prose; `qa.md` and `development.md` were added as
   call sites during this feature's own rework (round 1, HIGH-1). REQ-F-001/
   REQ-F-002 require consolidating to one canonical source all five reference,
   eliminating the exact drift risk the feature's Problem statement describes.
3. **No new Shark schema or Go type** — REQ-NF-001 and the E34-F05 precedent
   both point to reusing `remediation_sweeps: array of I-03` inside the
   existing GateResult envelope rather than adding storage.

### Integration with existing code

- `internal/templates/` renderer: the new workflow file must render cleanly
  through the same production renderer exercised by
  `internal/templates/includes_test.go` (which already covers
  `review-code.md`); no new renderer code is needed, only a new input file.
- `skills/quality/SKILL.md` "Usage Pattern" section documents how agents
  reference workflow files by path — the new file follows that existing
  convention, requiring no change to the reference mechanism itself.

## Cross-feature interactions

### Consumes

- **I-02** — GateResult v1. Producer: E34-F05. Shape source:
  [architecture.md#i-02-gateresult-v1](../architecture.md#i-02-gateresult-v1).
  Contract test: **GAP (verified 2026-09-04)** — `E34-interaction-map.md`
  names `E34-F05-structured-gate-results-and-parent-owned-persisten/test-plan.md#TC-I-02-GATERESULT-PARITY`
  as "the planned shared contract test" that "F05 test planning must create,"
  but E34-F05 is already completed and that directory contains no
  `test-plan.md` and no such pointer exists anywhere in the repo (confirmed
  by directory listing and grep). This feature does not create that upstream
  test — it is out of E34-F06's scope, since I-02 is consumed, not produced,
  here. E34-F06 instead verifies its own I-02 consumption structurally via
  TC-010 (no new Go persistence layer; nests inside the existing envelope).
  The missing upstream `TC-I-02-GATERESULT-PARITY` pointer remains an open
  gap for E34-F05's owner to close.
  This feature's new workflow nests its I-03 output inside I-02's
  `remediation_sweeps` array; it does not modify I-02 itself.

### Produces

- **I-03** — DefectClassSweep v1. Consumer: E34-F08. Shape source:
  [architecture.md#i-03-defectclasssweep-v1](../architecture.md#i-03-defectclasssweep-v1).
  Contract test:
  `E34-F06-defect-class-completeness-and-recurrence-routing/scenario-review-TC-005-TC-009.md#tc-i-03-defect-class-closure-cross-reference`
  (the `TC-I-03-DEFECT-CLASS-CLOSURE` anchor lives in this file, not in
  `test-plan.md` — `test-plan.md` only names the pointer in its
  cross-feature-contract table; the actual walkthrough evidence is here).

Both IDs and their shape sources/contract-test pointers are taken verbatim
from `E34-interaction-map.md` (rows I-02, I-03) — no new interaction ID is
introduced by this spec.

## Cross-epic integrations

None. `E34-cross-epic-map.md` and `docs/product/cross-epic-integration-map.md`
contain no X-## row naming E34-F06 (grep confirmed empty during research).

## Durable unresolved decisions

None material. The one open design choice — whether `defect-class-sweep.md`
lives directly under `skills/quality/workflows/` versus a new
`skills/quality/workflows/defect-class/` subdirectory — is a non-material
naming/placement call: this spec adopts the flat placement (Key technical
decision 1) to match every existing sibling file in that directory with no
subdirectories precedent. No Q### is warranted for a naming convention already
demonstrated by six sibling files.

*Last Updated*: 2026-09-04
