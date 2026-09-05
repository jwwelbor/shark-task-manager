---
feature_key: E34-F08-tier-consistent-gates-and-final-integration-review
epic_key: E34
title: Tier-Consistent Gates and Final Integration Review — Specification
---

# E34-F08 Specification

See [Epic PRD](../epic.md) and [architecture.md](../architecture.md) for the
E34-wide I-02/I-03/I-04/I-05 contracts this feature consumes and produces.
See [research-report.md](./research-report.md) for the Capability map. See
[feature.md](./feature.md) for the full requirements text (REQ-F-001–008,
REQ-NF-001) — this spec adds file- and mechanism-level detail feature.md does
not carry.

This feature has two parts with very different shapes:

- **Content part** (REQ-F-001, REQ-F-003 partial, REQ-F-008): the tier
  matrix, executable-evidence contract, and I-03/I-04 consumption are bundle
  content — one new shared reference plus edits to consuming prompts.
- **Go/CLI part** (REQ-F-002, REQ-F-004, REQ-F-005 partial, REQ-F-006,
  REQ-F-007): a new immutable, crash-safe integration-event log; a new
  `integration_review` epic workflow step; a new `shark integration backfill`
  command; and I-05 production. This is genuine COMPLEX-tier systems design,
  not content-only, and is where this spec concentrates its detail.

## Requirements (incremental over epic)

Traces to feature.md REQ-F-001 through REQ-F-008 and REQ-NF-001 verbatim.

### Functional — content part

- **REQ-F-001 (spec)**: Add
  `internal/sharkdata/default_data/skills/quality/context/tier-matrix.md`
  containing feature.md's tier-contract table verbatim (SIMPLE/STANDARD/
  COMPLEX × planning source/test source/same-model gate/separate QA/final
  UAT) plus one paragraph: "missing artifacts are failures only when the
  selected tier requires them." `feature/assessment.md` (tier selection),
  `feature/task_generation.md`, `feature/code_review.md`, `feature/qa.md`,
  and `feature/approval.md` each reference this file instead of restating
  the matrix (mirrors E34-F06's AC-2 single-source-of-truth pattern —
  structural test, not exact-string blacklist, per that feature's own
  round-4 lesson).
- **REQ-F-002 (spec)**: Add an "Executable gate evidence" section to
  `tier-matrix.md`: require, per gate report, the exact command, working
  directory, exit status, runner-native pass/fail/error/skip counts, an
  expected-skip comparison (against a project-declared expected-skip list if
  one exists, otherwise "no expected-skip list declared" is itself a
  reportable fact, not silently ignored), and a bounded log/artifact
  pointer. A prose-only total, omitted exit status, missing declared test
  case, or unexplained unexpected skip fails the gate. Commands themselves
  are discovered from project guidance (`docs/architecture/tech-stack.md`
  Quality Gate section or equivalent), never hardcoded to a project-specific
  tool.
- **REQ-F-003 (spec, content half)**: `code_review.md`/`qa.md`/`approval.md`
  each add one paragraph: "consume E34-F06's I-03 DefectClassSweep and
  E34-F07's I-04 ChangeImpactSet evidence for prior blocking defect classes
  and material decisions in scope" — referencing, not restating, those two
  features' own workflow files.
- **REQ-F-008 (spec)**: Add a "Pinned E40 benchmark scenarios" note (not a
  gate) to `tier-matrix.md` naming four scenario categories (tier routing,
  evidence fidelity, defect-class recurrence, integration closure) as
  benchmark follow-up work for E40, explicitly non-blocking for this
  feature's acceptance (REQ-F-008's own text).

### Functional — Go/CLI part

- **REQ-F-004 (spec) — Integration-event log and workflow step**:
  - Add `integration_review` to
    `internal/sharkdata/default_data/workflow/epic.yaml` as a new
    non-terminal step between `active` and `completed`: `active`'s
    `outcomes.pass` changes from `completed` to `integration_review`;
    `integration_review` adds `outcomes: {pass: completed, fail: active,
    blocked: blocked, on_hold: on_hold}`. `action: spawn_agent`, dedicated
    prompt `epic/integration_review.md`.
  - New package `internal/integration/` (not `internal/sharkdata` — this is
    epic-run state, not bundle content) with:
    - `type IntegrationRun struct { EpicRunID, EpicKey, BaseCommit string;
      CreatedAt time.Time }` — captured once, at first feature dispatch under
      the epic (the `active` step's cascade action, on its first invocation
      per epic run, calls `integration.CaptureBase(epicKey)` if no run
      record exists yet for this epic; a second call for the same epic is a
      no-op returning the existing run, never overwriting `BaseCommit`).
    - `type IntegrationEvent struct { EventID, EpicRunID, FeatureKey,
      FeatureCommit string; TrackedPaths, UntrackedPaths []string;
      RecordedAt time.Time }` — one immutable JSON file per feature
      completion, written to
      `.shark/runs/<epic-run-id>/integration-events/<event-id>.json`.
      `EventID` is derived deterministically from
      `sha256(epic_run_id + feature_key + feature_commit)` (hex, first 16
      chars) so a retried write of the *same* completion is idempotent
      (same bytes, same path) while a genuinely different completion for the
      same feature (e.g. a re-opened, re-completed feature) gets a new
      `EventID` because `FeatureCommit` differs.
    - `type IntegrationCandidate struct { EpicRunID, BaseCommit, HeadCommit
      string; EventIDs []string; Digest string }` — one atomic file,
      `.shark/runs/<epic-run-id>/integration-candidate.json`, holding the
      current accumulated view. `Digest` is `sha256` over the struct's
      canonical JSON with `Digest` itself excluded (matches I-03's guard
      digest convention already established — no new hashing scheme).
      Written via write-to-temp-file-then-`os.Rename` (atomic on POSIX
      filesystems), with an `fsync` on the temp file before rename and a
      compare-and-swap on the *previous* candidate's `Digest` (read current
      file, verify its digest matches the writer's expected prior digest,
      write only if it matches; otherwise the writer re-reads and retries
      once, then reports a conflict rather than looping — see Durable
      unresolved decision Q-F08-01 below for the retry-count boundary).
    - One idempotent epic-level `reference` note
      (`shark create note <epic-key> ... --type=reference`) registers
      `{EpicRunID, BaseCommit}` so a restarted parent loop or a second
      concurrent orchestrator discovers the existing run instead of
      capturing a second base. A second attempt to register a *different*
      `EpicRunID` for the same epic while a nonterminal one is registered is
      rejected (the note-creation path checks for an existing unresolved
      registration note first).
  - `shark integration backfill <epic-key> --epic-run-id=<run-id>
    --base=<full-commit> --events-file=<bounded-v1-json>
    --session=<authorized-session-id> [--dry-run]` (new command,
    `internal/cli/commands/integration_cmd.go`, thin wrapper delegating to
    `integration.Backfill(...)`): for an epic with no pre-execution
    `IntegrationRun` record (i.e., already active before this feature
    shipped), requires an active claim on `<epic-key>` matching
    `--session`, validates `--base` is a reachable commit and
    `--events-file` parses as a bounded array of `IntegrationEvent`-shaped
    entries with no duplicate `EventID` and no digest/path mismatch against
    the named commits, then performs the identical atomic
    capture-then-append sequence `CaptureBase`/event-write/candidate-update
    would perform in steady state — no separate code path, so backfill and
    steady-state capture share one correctness argument. `--dry-run`
    performs every validation and reports what would be written without
    writing. Any invalid input leaves every sidecar and note unchanged
    (validate fully before the first write).
- **REQ-F-005 (spec) — Integration closure checks (`epic/integration_review.md`)**:
  reads the candidate's full accumulated diff (`git diff <BaseCommit>..<HeadCommit>`
  plus `UntrackedPaths` from every event) and verifies: every I-##/X-## row
  whose producer or consumer feature is in this epic is `accounted`/closed
  per that row's own contract-test pointer; every completed E34-F06 defect-class
  sweep referenced by an in-scope finding has `status: complete`; every
  E34-F07 I-04 `ChangeImpactSet` referenced is `accounted`; open
  review-finding notes, ADRs, and project-standards references naming a
  changed path are cross-checked for disposition. Adds a new structural
  guard, `CheckInteractionMapCompleteness` (in
  `internal/sharkdata/embed_test.go`, mirroring the existing
  `TestEmbedded_*` bundle-index-completeness pattern), reading
  `E34-interaction-map.md`'s Interaction Contracts table (the actual
  producer/consumer/shape-source/payload/style source for I-01 through
  I-05 — architecture.md's per-interaction sections are field-meaning
  tables only, e.g. I-01's `assessor_verdict`/`owner_decision`/... list, and
  are not this checker's row source) and asserting every row names a
  producer, consumer(s), a shape-source link, a payload, and a style. This
  is a distinct check from the already-existing, already-shipped
  `TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY` in
  `internal/cli/commands/interaction_prompts_test.go` (E34-F02/F03's I-01
  ReadinessEvidence field-symmetry contract test, per
  `E34-interaction-map.md:32-33`) — this feature does not re-create or
  duplicate that test.
- **REQ-F-006 (spec) — Gate authority**: `integration_review`'s outcome
  routing (`outcomes.fail: active`) is additive — a `fail` reopens the epic
  to `active` without touching any individual feature's own status; the
  prompt text states explicitly: "This review adds a gate; it never
  overrides or supersedes an existing feature verdict. A currently-rejected
  or currently-in-development feature blocks epic completion through its
  own status, not through this step." No global owner-approval config
  change is introduced (REQ-F-006's own "do not introduce" clause). WWGM's
  historical E04-F02 shipped-while-rejected record is out of this feature's
  scope — tracked under E34-F09 per feature.md's own routing.
- **REQ-F-007 (spec)**: `epic/integration_review.md`'s output follows the
  same `GateResult` markdown/JSON envelope convention E34-F05/F06 already
  established (content convention, no new Go type). It additionally
  produces one I-05 `CanonicalAdoptionManifest` entry (shape:
  architecture.md#i-05-canonicaladoptionmanifest-v1) listing every changed
  canonical bundle path in the reviewed diff, its workflow-compatibility
  note, any required override action, and the validation commands run —
  nested in the same `GateResult` envelope as a new `adoption_manifest`
  field (sibling to `remediation_sweeps`/`change_impacts`, matching the
  established sibling-array convention).

### Non-functional

- **REQ-NF-001 (spec)**: `tier-matrix.md` and `integration_review.md`
  contain zero WWGM script names, Python environment variables, specific
  LLM names, or host-only commands — commands are discovered from
  `{{project guidance}}` render inputs, matching every prior E34 feature's
  posture.

### Acceptance criteria

- AC-1: `tier-matrix.md` exists, renders cleanly, and every one of
  `assessment.md`/`task_generation.md`/`code_review.md`/`qa.md`/`approval.md`
  references it rather than restating the matrix (structural test).
- AC-2: A rendered-prompt test proves the SIMPLE, STANDARD, and COMPLEX
  routes each require exactly their matrix-defined artifacts and gates (no
  route requires an artifact the matrix doesn't assign it).
- AC-3: `integration.CaptureBase` called twice for the same epic returns the
  identical `BaseCommit` both times (idempotent capture); a second call from
  a concurrent process (simulated via two goroutines racing the same epic
  key against a temp `.shark/` directory) results in exactly one
  `IntegrationRun` record and both callers observing the same `BaseCommit`.
- AC-4: Two concurrent `IntegrationEvent` writes for two different features
  under the same epic run both survive (two files exist, both readable,
  neither overwrites the other) and the candidate's `EventIDs` list contains
  both after both complete — proven with two goroutines racing real file
  writes in a temp directory (no live network/DB dependency, matching this
  repo's CLI-test-uses-mocks-or-temp-fs convention).
- AC-5: A candidate-head compare-and-swap write against a stale expected
  prior digest is rejected (returns a typed conflict error), not silently
  overwritten; the file on disk after a rejected write is byte-identical to
  before the attempt.
- AC-6: `shark integration backfill --dry-run` against a valid, complete
  base/events input writes nothing to disk (verified via directory listing
  and note-count before/after) and reports what it would create.
  `backfill` (non-dry-run) with the same input creates exactly one
  `IntegrationRun`, one `IntegrationEvent` per input entry, one
  `IntegrationCandidate`, and one epic reference note. A second `backfill`
  attempt against an epic that already has a registered run is rejected
  with no mutation.
- AC-7: `integration_review`'s report explicitly states, for a fixture where
  one in-scope feature is currently rejected/in-development, that epic
  completion remains blocked by that feature's own status — the review
  itself does not report an overriding PASS that ignores it.
- AC-8: `CheckInteractionMapCompleteness` passes against the current
  `E34-interaction-map.md` Interaction Contracts table and fails when a row
  is missing one of its five required fields — producer, consumer(s),
  shape-source link, payload, or style (fixture-injected via a test-local
  copy of the table with one field removed).
- `make fmt && make lint && make test` pass with the new files included.

### Out of scope

Per feature.md: separate QA for every STANDARD feature, numeric round-based
escalation, WWGM validation scripts/DB setup/lint config/model assignments,
rewriting historical E04 lifecycle records, and provider-backed E40
comparison as a delivery prerequisite.

## Architecture

### Component changes

| File | Change |
|---|---|
| `internal/sharkdata/default_data/skills/quality/context/tier-matrix.md` | NEW — tier contract, executable-evidence contract, I-03/I-04 consumption note, E40 scenario note |
| `internal/sharkdata/default_data/prompts/feature/assessment.md`, `task_generation.md`, `code_review.md`, `qa.md`, `approval.md` | EDIT — reference tier-matrix.md; code_review/qa/approval also reference I-03/I-04 consumption |
| `internal/sharkdata/default_data/prompts/epic/integration_review.md` | NEW — the epic-level final-review gate prompt |
| `internal/sharkdata/default_data/workflow/epic.yaml` | EDIT — insert `integration_review` step between `active` and `completed` |
| `internal/integration/run.go` | NEW — `IntegrationRun`, `CaptureBase` |
| `internal/integration/event.go` | NEW — `IntegrationEvent`, event-ID derivation, atomic write |
| `internal/integration/candidate.go` | NEW — `IntegrationCandidate`, digest computation, CAS update |
| `internal/integration/backfill.go` | NEW — `Backfill` (shared by the CLI command and, structurally, by steady-state capture) |
| `internal/integration/lock.go` | NEW — run-scoped advisory lock guarding the registration-note write sequence only (task-owned by T-E34-F08-012, closing the archived-head/run-lock durability gap named in architecture.md) |
| `internal/integration/history.go` | NEW — rebase/squash/interleaved-commit detection and dirty/untracked path digests (task-owned by T-E34-F08-013) |
| `internal/integration/*_test.go` | NEW — concurrency, idempotency, CAS-rejection, backfill validation, restart/failure-injection, and history-edge tests |
| `internal/cli/commands/integration_cmd.go`, `integration_cmd_test.go` | NEW — `shark integration backfill` thin wrapper + mocked-repo-free CLI test (filesystem-only, temp dir) |
| `internal/sharkdata/embed_test.go` | EDIT — add `TestI01ReadinessSymmetry` |

### Data model changes

No SQLite schema change (`CurrentSchemaVersion` untouched) — all new state is
file-based under `.shark/runs/<epic-run-id>/`, following the existing
`.shark/` convention for run-scoped artifacts (confirmed present in this
repo's `.gitignore`/tooling for other run-scoped state). One new Shark note
type is *not* introduced — the registration record uses the existing
`--type=reference` note, matching E34-F05/F07's precedent.

### API / interface contracts

```
shark integration backfill <epic-key> --epic-run-id=<id> --base=<commit> \
  --events-file=<path> --session=<id> [--dry-run] [--json]
```

```go
package integration

func CaptureBase(epicKey string) (*IntegrationRun, error)
func RecordEvent(epicRunID, featureKey, featureCommit string, tracked, untracked []string) (*IntegrationEvent, error)
func UpdateCandidate(epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error) // CAS internally
func Backfill(epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool) (*IntegrationCandidate, error)
```

### Key technical decisions

1. **File-based, not database-backed, run state** — matches this feature's
   own requirement that no new DB schema is introduced, and matches the
   existing `.shark/` run-artifact convention rather than inventing a
   second one.
2. **Backfill shares the steady-state write path** — a single
   `UpdateCandidate`/event-write implementation used by both the cascading
   `active` step and the explicit `backfill` command, so there is one
   correctness argument for atomicity/CAS, not two independently-maintained
   ones.
3. **CAS over a mutex** — epic feature completions can originate from
   independent parent-loop processes (not threads in one process), so an
   in-memory lock cannot serialize them; a file-based CAS (read-verify-write
   with digest comparison, atomic rename) is the mechanism that works
   across processes without a new coordination service.

### Integration with existing code

- `internal/sharkdata/default_data/prompts/epic/active.md`: gains one line
  invoking `integration.CaptureBase` semantics (described in prose, not a
  Go call from the prompt itself — the *cascade action*, i.e. the Go code
  backing `action: cascade` for the `active` step, is what actually calls
  `CaptureBase`; this is a small addition to the existing cascade handler in
  `internal/cli/commands/` or the workflow-cascade service, not a new
  standalone command).
- `internal/workflow/`: no schema change to the workflow loader itself — a
  new step in `epic.yaml` is data, not a loader code change (route-based
  workflow already supports arbitrary new steps per Epic E35's design).

## Cross-feature interactions

### Consumes

- **I-03** — DefectClassSweep v1. Producer: E34-F06 (completed). Consumed by
  `code_review.md`/`qa.md`/`approval.md`'s new reference paragraph and by
  `integration_review.md`'s closure check. Contract test (verbatim, per
  E34-interaction-map.md):
  `E34-F06-defect-class-completeness-and-recurrence-routing/test-plan.md#TC-I-03-DEFECT-CLASS-CLOSURE`.
- **I-04** — ChangeImpactSet v1. Producer: E34-F07 (now completed). Same
  consumption points. Contract test (verbatim, per E34-interaction-map.md):
  `E34-F07-state-space-planning-and-decision-propagation/test-plan.md#TC-I-04-CHANGE-IMPACT-CLOSURE`.
- **I-02** — GateResult v1. Producer: E34-F05. Contract test pointer
  (mirrored verbatim per `E34-interaction-map.md:44-46`, required of every
  consumer regardless of upstream readiness):
  `E34-F05-structured-gate-results-and-parent-owned-persisten/test-plan.md#TC-I-02-GATERESULT-PARITY`.
  Status/context: this is an upstream gap already documented by E34-F06
  (TD-198/TD-199) — E34-F05 has no `test-plan.md` yet, so the pointer target
  does not exist on disk; this feature mirrors the pointer string per the
  parent map's requirement without re-litigating F05's own test debt.
  `integration_review.md` is the concrete task-owned consumer of this shape
  (reads prior features' `GateResult` envelopes for closure evidence and
  nests `adoption_manifest` inside its own), per T-E34-F08-010's Integration
  Contracts.

### Produces

- **I-05** — CanonicalAdoptionManifest v1. Consumer: E34-F09. Shape source:
  architecture.md#i-05-canonicaladoptionmanifest-v1. Contract test: **N/A**,
  per `E34-interaction-map.md`'s I-05 row — E34-F09's Go/CLI surface does
  not parse the manifest, so there is no cross-feature Go-level contract
  test (the earlier pointer naming a test-plan.md#TC in F09 or here was
  reconciled away; it is not carried forward). This feature separately
  writes **TC-I-05-ADOPTION-MANIFEST** in its own test-plan.md as an
  internal producer-side structural test (rendered `adoption_manifest`
  field list matches architecture.md's I-05 table) — that internal test is
  not a cross-feature contract pointer and does not conflict with the N/A
  above.
- **`CheckInteractionMapCompleteness`** — the structural guard over
  `E34-interaction-map.md`'s Interaction Contracts table (REQ-F-005), living
  in `internal/sharkdata/embed_test.go` per this spec's Component-changes
  table. Not to be confused with the pre-existing, separately-owned
  `TestI01ReadinessContract_TC_I_01_READINESS_SYMMETRY` in
  `internal/cli/commands/interaction_prompts_test.go` (E34-F02/F03's I-01
  ReadinessEvidence contract test).

## Cross-epic integrations

None named for E34-F08 in `E34-cross-epic-map.md` or
`docs/product/cross-epic-integration-map.md` (grep confirmed empty).

## Durable unresolved decisions

- **Q-F08-01**: Exact CAS retry policy on `UpdateCandidate` conflict (retry
  once then fail, vs. bounded exponential backoff with N retries) is not
  fully specified here — feature.md's verification plan asks for "reject a
  stale CAS writer" without prescribing a retry count. This spec adopts
  "read-verify-write once, retry exactly once on conflict, then report a
  typed conflict error to the caller" as the simplest policy consistent with
  "concurrent feature completions are additive" (two legitimately-concurrent
  writers should both eventually succeed via their own retry, not loop
  indefinitely) — but the exact retry count is a tuning decision, not an
  architectural one, and may be revisited during implementation without a
  spec amendment if the single-retry policy proves insufficient under real
  concurrent load in testing (task_generation should flag if TC evidence
  during implementation suggests the fixed single-retry is inadequate,
  rather than silently widening it).

*Last Updated*: 2026-09-04
