---
feature_key: E34-F07-state-space-planning-and-decision-propagation
epic_key: E34
title: State-Space Planning and Decision Propagation — Specification
---

# E34-F07 Specification

See [Epic PRD](../epic.md) for business context and [architecture.md](../architecture.md)
for the E34-wide I-02/I-04 contract definitions. See
[research-report.md](./research-report.md) for the Capability map this spec
builds on. See [feature.md](./feature.md) for the full requirements text
(REQ-F-001–007, REQ-NF-001) — this spec adds only file- and section-level
implementation detail feature.md does not carry.

This is a **near-content-only** feature: it adds guidance/templates to
prompt/skill bundle files, plus one thin new CLI command
(`shark impact record`) required because ADR adoption has no existing
Shark-tracked resolution event to hook I-04 persistence onto (unlike
Question resolution, which already has one).

## Requirements (incremental over epic)

Traces to feature.md REQ-F-001 through REQ-F-007 and REQ-NF-001 verbatim.
This spec adds file/section-level detail only.

### Functional

- **REQ-F-001 (spec)**: Add a new reusable workflow file
  `internal/sharkdata/default_data/skills/quality/workflows/state-space-coverage.md`
  defining: the lifecycle-field detection heuristic (a field is
  behavior-bearing if a documented transition determines dispatch routing,
  gate outcome, or persisted disposition — not every enum), the closed-table
  shape (value, meaning, allowed entry transitions, allowed exit transitions,
  terminal/no-exit marker, invalid-transition list, failure/recovery
  behavior), and the review rule that a prose-only progression or an
  "other state" catch-all fails specification review. `feature/specification.md`
  references this workflow's table-shape section instead of restating it
  (mirrors E34-F06's canonical-workflow-plus-reference pattern — no
  restated procedure at the call site).
- **REQ-F-002 (spec)**: Add a "Technique selection from state shape" section
  to the same workflow file: when specification.md declares a closed
  lifecycle table for a field, `feature/test_planning.md` must select
  state-transition/decision-table technique for that field's test cases,
  covering every value, every relevant cross-entity axis pulled from I-##/X-##
  rows naming that field, and explicit invalid-transition/recovery/state-addition
  regression cases. `test_planning.md` gains one reference line, not a
  restated technique-selection algorithm.
- **REQ-F-003 (spec)**: Add a "Dependency discovery by interaction and
  caller path" section to the same workflow file, listing discovery sources
  in priority order: I-##/X-## interaction-map rows naming the field or
  entity, production caller paths (per the existing Caller-Path Contract
  concept already used in test-plan.md's TC format), persistence readers
  (grep for the field name in non-test `.go` files), deduplication/short-circuit
  logic, and named downstream obligations recorded in prior spec.md
  "Cross-feature interactions" sections. Require recording, per candidate
  axis considered, one line of inclusion/exclusion rationale.
  `specification.md`'s READ list gains one line pointing at this section
  instead of the current unstated "grep for related services" heuristic.
- **REQ-F-004 (spec)**: Add a "Shipped consumer re-verification" section:
  when a feature adds or widens a state value read by an already-completed
  feature's implemented consumer (discovered per REQ-F-003's search), the
  specification must list the consumer's caller path, owning feature key,
  affected AC IDs, and a regression-test pointer (existing or newly assigned).
  If the current feature cannot safely amend the consumer directly, it must
  create a linked task or tech-debt record naming the consumer feature — a
  staged handoff following E34-F03's existing declaration contract, never
  presented as already-verified.
- **REQ-F-005 (spec)**: Extend `prompts/feature/task_review.md` with one
  paragraph: compare every shared field/state/event/contract name the task
  touches against the owning specification and interaction map verbatim;
  report unexplained drift as a contract finding (blocking) even when the
  local name compiles/passes tests.
- **REQ-F-006 (spec)**:
  - Add an "I-04 propagation" section to the state-space-coverage.md workflow:
    resolving a Question, tech-debt item, change card, or ADR that changes
    already-implemented or already-specified behavior must produce one I-04
    `ChangeImpactSet` (shape: architecture.md#i-04-changeimpactset-v1) naming
    every invalidated spec/test-plan/task/interaction-map/standards artifact
    and every affected shipped consumer AC, each amended in the same change
    or linked to explicit follow-up work — never a completion record that
    omits an affected artifact without a stated disposition.
  - `prompts/tech_debt/resolved.md` (currently a single-line template) gains
    a second line invoking the I-04 propagation section when the resolved
    item changed accepted behavior (a bug fix or the like still exits with
    the existing one-line template unchanged when no I-04 conditions apply).
  - `skills/question-management/SKILL.md`'s existing resolution-service step
    gains one reference line: persist I-04 through the same validated
    reference-note path Question resolution already writes through — no new
    persistence mechanism, reusing the existing `--type=reference` typed note
    convention.
  - **`shark impact record <entity-key> --source-kind=<kind>
    --source-key=<key> --source-pointer=<path> --impact-file=<bounded-I-04-json>`**
    satisfies this feature's ADR-adoption hook. It is implemented by
    `internal/cli/commands/impact.go` (shipped by E34-F05, PR #211) rather
    than a new command of this feature's own — F05 had already delivered the
    exact flag-based boundary architecture.md's "Compatibility and migration"
    section declares for ADR adoption, validated against
    `gateresult.ValidateChangeImpactSet` and persisted through
    `gatepersist`'s bounded reference-note path. F07 originally planned a
    second, independent implementation (`impact_cmd.go` /
    `services.ImpactService`); that duplicate was found and removed during
    the E34→main merge (2026-09-06) once F05's prior implementation was
    discovered — see the merge commit for the resolution rationale. This
    feature reuses F05's command as-is; no new Go surface is added here.
- **REQ-F-007 (spec)**: Add a "Design divergence" section to
  state-space-coverage.md: rework departing from an accepted fix design must
  cite the original decision pointer, new evidence, affected consumers, and
  resulting artifact/test amendments; absence of a cited divergence means the
  recorded compatible design remains controlling (this generalizes
  defect-class-sweep.md's "Backward-looking rework" section from E34-F06 to
  the broader planning-decision context — REQ-F-007 here reuses that
  section's language rather than redefining divergence handling twice;
  state-space-coverage.md references defect-class-sweep.md's "Backward-looking
  rework" section directly instead of restating it).

### Non-functional

- **REQ-NF-001 (spec)**: All of the above except `shark impact record` is
  implemented in bundle content (prompts/skills/workflows) with zero new
  Shark database columns, tables, or relationship types. `shark impact record`
  itself adds no new persistence — it calls the existing note-creation code
  path, so no schema migration is introduced (`CurrentSchemaVersion` in
  `internal/db/db.go` is untouched).

### Acceptance criteria

- AC-1: `skills/quality/workflows/state-space-coverage.md` exists, renders
  cleanly, and contains all six sections named in REQ-F-001–004/006/007
  (closed-table, technique-selection, dependency-discovery, shipped-consumer
  re-verification, I-04 propagation, design-divergence).
- AC-2: `specification.md`, `test_planning.md`, and `task_review.md` each
  reference the relevant state-space-coverage.md section rather than
  restating its procedure (mirrors E34-F06's AC-2 pattern and reuses its
  `TestDefectClassSweepConsolidatedNotDuplicated`-style structural test
  approach rather than an exact-string blacklist).
- AC-3: `tech_debt/resolved.md` and `question-management/SKILL.md` each
  reference the I-04 propagation section.
- AC-4: `shark impact record <key> --source-kind=<kind> --source-key=<key>
  --source-pointer=<path> --impact-file=<path>` writes exactly one
  `--type=reference` note on `<key>` with the merged content (impact-file
  JSON plus the three source flags), exits non-zero on a target key that
  doesn't exist, and exits non-zero when the merged content fails minimal
  I-04 shape validation (missing `source_kind`, `source_key`, or
  `affected_artifacts`) or when any required flag or `--impact-file` is
  missing/unreadable.
- AC-5: A rendered-output sample demonstrates each of the three acceptance
  scenarios in feature.md ("Plan a multi-entity lifecycle", "Propagate a
  ratified decision", "Reject naming drift").

### Out of scope

Per feature.md "Out of scope": a runtime state-machine engine or generated
application code, a fixed foreign-key-hop dependency rule, automatic
rewriting of specs/tests from a decision record, and any new Shark entity or
relationship type (including an ADR entity type — ADRs remain plain markdown
files; `shark impact record` targets an *existing* entity key, it does not
create or track ADRs themselves). The flag-based CLI shape itself
(`--source-kind`/`--source-key`/`--source-pointer`/`--impact-file`) is fixed
by architecture.md's ADR-adoption boundary and is not this spec's own design
choice to alter.

## Architecture

### Component changes

| File | Change |
|---|---|
| `internal/sharkdata/default_data/skills/quality/workflows/state-space-coverage.md` | NEW — closed-table shape, technique-selection trigger, dependency-discovery priority order, shipped-consumer re-verification, I-04 propagation, design-divergence reference (REQ-F-001–004, F006, F007) |
| `internal/sharkdata/default_data/prompts/feature/specification.md` | EDIT — reference the closed-table section; replace the unstated "grep for related services" READ item with a pointer to the dependency-discovery section |
| `internal/sharkdata/default_data/prompts/feature/test_planning.md` | EDIT — reference the technique-selection section |
| `internal/sharkdata/default_data/prompts/feature/task_review.md` | EDIT — add the shared-naming-drift paragraph (REQ-F-005) |
| `internal/sharkdata/default_data/prompts/epic/feature_review.md` | EDIT — reference the shipped-consumer re-verification section for cross-feature epic-level review |
| `internal/sharkdata/default_data/prompts/tech_debt/resolved.md` | EDIT — add the conditional I-04 propagation line |
| `internal/sharkdata/default_data/skills/question-management/SKILL.md` | EDIT — reference the I-04 propagation section in the existing resolution-service step |
| `internal/cli/commands/impact.go` | ALREADY SHIPPED by E34-F05 — `shark impact record` reused as-is by this feature; no new file (see "Cross-feature interactions" note above) |

### Data model changes

None. `shark impact record` writes through the existing `EntityNoteRepository`
(`note_type = reference`) — no new table, column, or migration.

### API / interface contracts

```
shark impact record <entity-key> --source-kind=<kind> --source-key=<key> \
  --source-pointer=<path> --impact-file=<bounded-I-04-json> [--json]
```

This is the exact boundary architecture.md's "Compatibility and migration"
section declares for ADR adoption. `--impact-file` points to a JSON file
containing the bounded I-04 content the caller owns (at minimum a non-empty
`affected_artifacts` array). `--source-kind`, `--source-key`, and
`--source-pointer` are top-level I-04 fields (architecture.md's I-04 field
table) supplied as flags rather than embedded in the file; the CLI merges
them into the impact-file's JSON, overriding any value already present there,
before validation and persistence. All four flags are required — a missing
or unreadable `--impact-file`, or any missing flag, exits non-zero before any
repository call. Minimal content validation (not full schema enforcement,
applied after the merge): `source_kind`, `source_key`, and
`affected_artifacts` (non-empty array) must be present; a project-agnostic
decision, consistent with the epic's REQ-NF posture across F05/F06/F09.
There is no positional `<content-or-@file>` form — the flag-based shape
replaces it entirely.

### Key technical decisions

1. **One new command, not a new entity type or table** — ADRs stay plain
   markdown; `shark impact record` is a thin persistence hook onto the
   existing note-creation path, chosen over a new `adr` entity because
   Question/tech-debt/change-card resolution already have a Shark-tracked
   status transition to hook I-04 persistence onto, and only ADR adoption
   lacks one. Adding a full ADR entity type to solve that single gap would
   violate REQ-NF-001 and this epic's established "no new entity/relationship
   type" precedent (E34-F05, E34-F06, E34-F09) for a problem a five-line CLI
   wrapper solves.
2. **Reuse the existing typed reference-note convention** rather than a new
   note type — matches E34-F05's `--type=review-finding` precedent; I-04
   nests as note content, not a new column.
3. **state-space-coverage.md as one new workflow file**, not edits scattered
   across five prompt files with restated procedures — mirrors E34-F06's
   "one canonical source, many bare references" decision (Key technical
   decision 2 in that spec) to avoid recreating the exact duplication-drift
   risk E34-F06 spent seven UAT rounds converging on.

### Integration with existing code

- `internal/cli/commands/impact.go` (E34-F05) already follows the same
  thin-wrapper pattern as every other command file; this feature has no
  further integration work here.
- `internal/templates/`: the new workflow file renders through the existing
  production renderer, no renderer code changes needed.

## Cross-feature interactions

### Consumes

- **I-02** — GateResult v1. Producer: E34-F05. Shape source:
  [architecture.md#i-02-gateresult-v1](../architecture.md#i-02-gateresult-v1).
  Contract test: same upstream gap E34-F06 already documented (E34-F05 has no
  test-plan.md; `TC-I-02-GATERESULT-PARITY` does not exist anywhere in the
  repo — tracked as TD-198/epic-owner scope, not re-documented per-feature).
  This feature nests I-04 inside the existing `remediation_sweeps`-sibling
  slot in `GateResult` (a new `change_impacts` array alongside
  `remediation_sweeps`, both arrays inside the same envelope — see
  architecture.md line 164's table for the sibling-array precedent).

### Produces

- **I-04** — ChangeImpactSet v1. Consumer: E34-F08 (final integration
  review). Shape source:
  [architecture.md#i-04-changeimpactset-v1](../architecture.md#i-04-changeimpactset-v1).
  Contract test:
  `E34-F07-state-space-planning-and-decision-propagation/test-plan.md#TC-I-04-CHANGE-IMPACT-CLOSURE`
  (created when this feature's test-plan.md is written in the next pipeline
  step).

Both IDs are taken verbatim from `E34-interaction-map.md`.

## Cross-epic integrations

None. Neither `E34-cross-epic-map.md` nor
`docs/product/cross-epic-integration-map.md` names an X-## row for E34-F07
(grep confirmed empty).

## Durable unresolved decisions

None material. `shark impact record`'s exact minimal-validation field list
(source_kind/source_key/affected_artifacts) is a non-material implementation
choice consistent with I-04's architecture.md shape table — no Q### is
warranted for a validation-strictness call this spec has full authority and
context to make.

*Last Updated*: 2026-09-05
