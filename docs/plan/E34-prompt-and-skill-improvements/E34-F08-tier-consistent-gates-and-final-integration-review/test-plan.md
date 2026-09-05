---
feature_key: E34-F08-tier-consistent-gates-and-final-integration-review
epic_key: E34
title: Tier-Consistent Gates and Final Integration Review — Test Plan
---

# E34-F08 Test Plan

This feature has two shapes, per spec.md's own split, and this plan tests each
on its own terms:

- **Content part** (AC-1, AC-2, and REQ-F-003/REQ-F-008/REQ-NF-001's content
  half): `tier-matrix.md` plus five reference edits. Per CLAUDE.md's
  prompt-only testing guidance and this plan's own "Prompt-only changes"
  exemption, these test cases use the production template renderer, direct
  bundle-content grep/existence checks, and manual policy-wording review —
  not caller-path/mutation/decision-table tests, which apply only to
  deterministic runtime behavior this half does not add.
- **Go/CLI part** (AC-3 through AC-8): a new `internal/integration/` package
  and `shark integration backfill` CLI command. This is genuine deterministic
  runtime behavior — full Caller-Path Contracts apply. Per this repo's
  `.claude/rules/testing/architecture.md` golden rule, the file-based
  `internal/integration/` state (events, candidate, run record) is
  filesystem-only — no repository to mock, tests drive real temp-directory
  fixtures, matching E34-F09's precedent for filesystem-only features. The
  one exception is `shark integration backfill`'s claim/session
  authorization check, which **is** DB-backed (claims live in the existing
  claims table via the workflow/claim service) — those subtests mock the
  claim lookup at the service seam, per the golden rule's "CLI tests use
  mocked repositories" branch. This mixed seam is called out explicitly in
  the Caller-Path Contracts table below so a reviewer doesn't assume the
  whole feature is filesystem-only.

## Drift and traceability notes (recorded, not blocking)

- **architecture.md vs. spec.md on candidate-state shape**: architecture.md's
  "Epic integration candidate identity" section describes a richer design
  than spec.md's literal Go types — it names an `event kind` field, a
  "landed or staged commit identity," a "completion history identity," a
  retained `integration-heads/<record-digest>.json` history of prior candidate
  heads, and a "run-scoped repository lock plus compare-and-swap." spec.md's
  actual `IntegrationEvent`/`IntegrationCandidate` structs (quoted in this
  plan's Caller-Path Contracts) carry none of the extra fields, no prior-head
  retention directory, and explicitly chooses CAS *instead of* a lock
  ("Key technical decisions" #3: "CAS over a mutex ... not two independently
  maintained ones"). Per this workflow's own instruction ("this spec adds
  file- and mechanism-level detail feature.md does not carry" — spec.md is
  the file-level authority for this feature), this plan tests spec.md's
  literal contract (AC-3 through AC-6) and does **not** invent a test for the
  undocumented `integration-heads/` retention directory or a separate lock,
  since neither appears in spec.md's API surface, component-changes table, or
  ACs. Flagged here so the next E34-F08-adjacent worker (task_generation, or
  whoever reconciles architecture.md) can decide whether architecture.md
  needs a spec-amendment-level update or spec.md's simpler design is the
  intentional final word — this plan does not resolve it, matching this
  feature's own Q-F08-01 durable-unresolved-decision posture.
- **I-05 contract-test-pointer naming mismatch**: spec.md's own "Cross-feature
  interactions → Produces" section commits to a pointer named
  `E34-F08-tier-consistent-gates-and-final-integration-review/test-plan.md#TC-I-05-ADOPTION-MANIFEST`
  (i.e., a TC in *this* file). architecture.md's I-05 section and
  `E34-interaction-map.md` (row I-05) instead name
  `E34-F09-override-drift-visibility-and-wwgm-reconciliation/test-plan.md#TC-I-05-OVERRIDE-ADOPTION`
  (i.e., a TC in *F09's* file). F09's already-written test-plan.md does not
  contain a `TC-I-05-OVERRIDE-ADOPTION` — it records I-05 as "informational
  ... N/A — no Go-level contract test." So neither pointer currently resolves
  to a test named exactly as either doc claims. This plan follows spec.md's
  own commitment (spec.md is this feature's authoritative file) and creates
  **TC-I-05-ADOPTION-MANIFEST** below. The interaction-map/architecture.md
  naming should be reconciled to point here instead of F09 — flagged as a
  follow-up for whoever next edits `E34-interaction-map.md` or
  architecture.md's I-05 section (not blocking this feature's own test
  planning, since the producer-side test this plan owes can be written
  either way).

## AC / Requirement Test Matrix

| TC | AC / Requirement | Description | Covered? |
|----|----|--------------|----------|
| TC-001 | AC-1 | `tier-matrix.md` renders cleanly; five consuming prompts reference it, don't restate it | Yes |
| TC-002 | AC-2 | Rendered SIMPLE/STANDARD/COMPLEX routes require exactly their matrix-defined artifacts/gates | Yes |
| TC-003 | REQ-F-003 (content half) | `code_review.md`/`qa.md`/`approval.md` each reference I-03/I-04 evidence, don't restate those features' workflows | Yes |
| TC-004 | REQ-F-008 | E40 benchmark-scenario note present, explicitly non-blocking | Yes |
| TC-005 | REQ-NF-001 | No WWGM/Python-env/LLM-name/host-only leakage in `tier-matrix.md` or `integration_review.md` | Yes |
| TC-006 | AC-3 | `CaptureBase` idempotent (repeat call, then concurrent race) | Yes |
| TC-007 | AC-4 | Two concurrent `IntegrationEvent` writes for two features both survive; candidate lists both | Yes |
| TC-008 | AC-5 | Stale-digest CAS write rejected; file byte-identical before/after | Yes |
| TC-009 | AC-6 | `backfill` dry-run/non-dry-run/re-registration + input-validation decision table | Yes |
| TC-010 | AC-6 (session/claim authorization) | `backfill` rejects a missing/mismatched claim session, no mutation | Yes |
| TC-011 | REQ-F-004 (cascade wiring) | `active` step's cascade action calls `CaptureBase` on first feature dispatch only — wiring, not just the naked function | Yes |
| TC-012 | AC-7 | `integration_review` report states blocked-by-feature-status for a rejected/in-development in-scope feature; non-supersession clause present in prompt text | Yes |
| TC-013 | REQ-F-005 (general closure check) | I-##/X-## closure, I-03 sweep `status: complete`, I-04 `status: accounted` checks scoped correctly in prompt text | Yes |
| TC-014 | AC-8 | `TC-I-01-READINESS-SYMMETRY` passes on the real I-## table, fails on a fixture missing one required reference | Yes |
| TC-I-05-ADOPTION-MANIFEST | REQ-F-007 / I-05 produced | `adoption_manifest` field lists exactly architecture.md's 8 I-05 fields, nested as a `GateResult` sibling array | Yes |

`make fmt && make lint && make test` passing with new files is the plan's own
closing gate (spec.md's own last AC), verified at Step 8, not a separate TC.

## Caller-Path Contracts

| TC | Entrypoint | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|----|-----------|--------------------------|-----------------|------------------|
| TC-001, TC-003, TC-004, TC-005 | `internal/templates` production renderer / direct file grep | content-only — no mock | n/a | A restated (not referenced) matrix passage, or a leaked WWGM term, is invisible to a test that only checks the renderer returns no error; grep/structural assertion is required |
| TC-002 | `internal/templates` production renderer, rendered once per tier (`SIMPLE`/`STANDARD`/`COMPLEX` render context) | content-only — no mock | n/a | A route requiring an artifact absent from `tier-matrix.md`'s row for that tier passes a renderer-error-only check but fails an artifact-list-diff check |
| TC-006 | `integration.CaptureBase(epicKey string) (*IntegrationRun, error)` | none — real temp `.shark/` directory, no DB | do not mock the filesystem or introduce an in-memory stub for the run record; must be real file I/O | An implementation that regenerates `BaseCommit` per call, or serializes concurrent callers incorrectly (second caller creates a second run file), passes a single-threaded happy-path test but fails the goroutine race |
| TC-007 | `integration.RecordEvent(epicRunID, featureKey, featureCommit string, tracked, untracked []string) (*IntegrationEvent, error)` then `integration.UpdateCandidate(epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error)` | none — real temp dir, real goroutines | do not serialize the two goroutines with a test-side mutex before the calls (defeats the point); do not mock `UpdateCandidate`'s CAS | A last-writer-wins implementation drops one feature's `EventID` from the candidate; this test's `EventIDs` assertion catches it, a single-writer test would not |
| TC-008 | `integration.UpdateCandidate(epicRunID string, newEvent *IntegrationEvent) (*IntegrationCandidate, error)` | none — real temp dir | do not mock digest computation; must recompute the real SHA-256 to construct the "stale" expected-prior value | An implementation that overwrites on digest mismatch (no real CAS) passes a happy-path write test but silently corrupts state under this test's crafted stale-digest scenario |
| TC-009 | `integration.Backfill(epicKey, epicRunID, base string, events []IntegrationEvent, dryRun bool) (*IntegrationCandidate, error)` and the CLI wrapper `runIntegrationBackfill` (in `internal/cli/commands/integration_cmd.go`) — subtests split across both layers per row detail below | none for the file-based writes (real temp dir); the CLI-layer subtests mock only the claim-lookup call (see TC-010) | do not mock `integration.Backfill` itself from the CLI test — the CLI test must drive the real function, matching F09's CLI-harness precedent | An implementation that writes partial state before validating `--events-file` fully (e.g., writes the `IntegrationRun` before checking for duplicate `EventID`s) fails this test's before/after directory-listing diff on an invalid-input subtest |
| TC-010 | CLI wrapper `runIntegrationBackfill` (`internal/cli/commands/integration_cmd.go`) | claim-lookup service (mocked, matching this repo's CLI-test-mocks-repositories convention for the one DB-backed dependency in this feature) | do not mock `internal/integration` package functions in this test — only the claim lookup is mocked; the file-based write path (or its absence, on rejection) must be real | An implementation that checks `--session` against the events file itself rather than the live claim record would pass a test with a forged session string; mocking only the claim lookup and returning "no matching claim" catches an implementation that skips the check entirely |
| TC-011 | The epic `active` step's cascade action (the Go handler backing `action: cascade` — in `internal/cli/commands/` or the workflow-cascade service, per spec.md's own "Integration with existing code" pointer) | none — drives the real cascade dispatch path against a temp `.shark/` dir and a fixture epic/feature pair; do not call `integration.CaptureBase` directly and call that "wiring" coverage | do not mock the cascade handler itself — this TC exists specifically because a passing `CaptureBase` unit test proves nothing about whether production code ever calls it | An unwired `CaptureBase` (dead code with a green unit test) passes TC-006 but fails TC-011, since TC-011 asserts a run record exists *after* driving the cascade path, never invoking `CaptureBase` by name |
| TC-012, TC-013 | `epic/integration_review.md` prompt content, reviewed against fixture scenarios (rejected in-scope feature; open I-##/X-## row; incomplete I-03 sweep; `incomplete` I-04 impact) | content-only — manual policy-wording review, per F06's precedent that no fixture-execution harness exists for prose decision procedures | n/a | A prompt whose closure-check language is silently satisfied by an unresolved row, or that reports an overriding PASS while a sibling feature is rejected, fails the scenario walkthrough even though no Go test would catch it |
| TC-014 | The I-## table checker exercised by `TestI01ReadinessSymmetry` in `internal/sharkdata/embed_test.go` | internal — the checker function is the production entrypoint (mirrors existing `TestEmbedded_*` bundle-completeness pattern; justification: this is a structural bundle-completeness guard, not application business logic with a caller above it) | n/a | A checker that only greps for the literal string `I-0` without validating all five required references per row would pass on the real table (which happens to be complete) but also incorrectly pass the mutated fixture missing one reference — the mutation subtest is what catches that |
| TC-I-05-ADOPTION-MANIFEST | `internal/templates` production renderer of `epic/integration_review.md`, output field list diffed against architecture.md's I-05 table (parsed programmatically, not hand-copied) | content-only — no mock | n/a | A prompt that lists 7 of the 8 required `adoption_manifest` fields (e.g., drops `promoted_policies`) passes a "renders without error" check but fails this field-list diff |

## Test Cases

- **TC-001** — Extend `internal/templates/includes_test.go`'s
  `TestIncludeResolver_*` table (or a sibling test) to render
  `tier-matrix.md` through the production renderer: no error, section headers
  for SIMPLE/STANDARD/COMPLEX rows and the "missing artifacts are failures
  only when the selected tier requires them" paragraph are present. Then, a
  structural grep/regex test (new, in the same package) asserts each of
  `assessment.md`, `task_generation.md`, `code_review.md`, `qa.md`,
  `approval.md` contains an `{{include}}` or textual pointer to
  `tier-matrix.md`'s path and does **not** contain a second copy of the
  SIMPLE/STANDARD/COMPLEX artifact table (structural test — a table with the
  same three tier names and a "gate" column, not an exact-string blacklist,
  per E34-F06's own round-4 lesson about brittle string matching).
  **Edge case:** a prompt that references the file by a stale/renamed path
  fails the include-resolution check, not just the grep.

- **TC-002** — Render `assessment.md` (or whichever prompt enumerates
  required artifacts/gates per tier — confirm exact prompt during
  implementation; if none currently enumerates it, this AC requires
  `task_generation.md` to do so, since it is the artifact-assignment point)
  three times, once per tier context variable (`SIMPLE`, `STANDARD`,
  `COMPLEX`). Parse the required-artifact/gate list out of each rendered
  output and out of `tier-matrix.md`'s own table row for that tier
  (programmatic table parse, not hand-transcribed expected values — avoids
  the test silently drifting from the matrix it's supposed to verify).
  Assert set-equality per tier: the rendered route requires exactly the
  matrix row's artifacts, no more, no fewer.
  **Edge case:** SIMPLE must not require a separate-QA artifact; COMPLEX must
  require it. **Negative case:** an artifact appearing in a rendered route
  but absent from the matrix row for that tier fails the test (this is the
  literal AC-2 wording: "no route requires an artifact the matrix doesn't
  assign it").

- **TC-003** — Grep `code_review.md`, `qa.md`, `approval.md` post-edit for the
  required reference sentence pattern (a paragraph naming E34-F06's I-03
  DefectClassSweep and E34-F07's I-04 ChangeImpactSet) and assert it is a
  *reference* (points at the other features' own workflow files) rather than
  restating their content (structural: no duplicated procedural step list
  from `defect-class-sweep.md` or F07's workflow file appearing verbatim or
  near-verbatim in these three files).
  **Negative case:** a file that inlines F06/F07's procedure instead of
  referencing it fails.

- **TC-004** — Grep `tier-matrix.md` for the "Pinned E40 benchmark scenarios"
  section; assert it names all four scenario categories (tier routing,
  evidence fidelity, defect-class recurrence, integration closure) and
  contains explicit non-blocking language (e.g. "non-blocking," "not a
  gate"). **Negative case:** absence of the explicit non-blocking qualifier
  fails — a benchmark note that reads as a gate requirement would silently
  make E40 a delivery prerequisite, which REQ-F-008 forbids.

- **TC-005** — `grep -rin "WWGM\|\.py\b\|/home/\|/Users/\|gpt-\|claude-[0-9]\|gemini-[0-9]" tier-matrix.md epic/integration_review.md` (extend the pattern list used by F06's TC-004-equivalent check with LLM-name tokens per this feature's own REQ-NF-001 wording, which additionally forbids "specific LLM names"). Zero matches.
  **Negative case:** a reintroduced host-only path or model name fails.

- **TC-006** — Two subtests against a real temp `.shark/` directory:
  (a) single-process: call `integration.CaptureBase("E99")` twice sequentially;
  assert both return values have identical `BaseCommit`, identical
  `EpicRunID`, and exactly one run file exists on disk after both calls.
  (b) concurrent: spin up two goroutines calling
  `integration.CaptureBase("E99")` simultaneously against the same temp
  directory (synchronized via a start barrier, not a serializing mutex —
  the point is to race the real file writes); assert both goroutines observe
  the same `BaseCommit`/`EpicRunID` and exactly one run record exists on
  disk after both complete (`sync.WaitGroup` join before assertions).
  **Edge case:** a third call after the temp directory is deliberately
  corrupted (run file present but unparseable JSON) — assert a typed error,
  not a silent second run creation.

- **TC-007** — Two goroutines, each racing a real `RecordEvent` +
  `UpdateCandidate` sequence for a *different* `featureKey` under the same
  `epicRunID`, against the same temp directory, started via a barrier.
  After both join: assert two distinct `IntegrationEvent` JSON files exist on
  disk (readable, valid JSON, distinct `EventID`s derived from
  `sha256(epic_run_id+feature_key+feature_commit)`), and the final
  `IntegrationCandidate.EventIDs` contains both IDs (order-independent set
  assertion). **Negative case:** if either event file is missing or the
  candidate's `EventIDs` contains only one entry, the test fails — this is
  the literal AC-4 wording ("neither overwrites the other").

- **TC-008** — Seed a temp directory with an existing
  `integration-candidate.json` (known `Digest`). Compute a deliberately stale
  expected-prior-digest value (e.g. digest of a candidate with one fewer
  `EventID`). Call `integration.UpdateCandidate` in a code path that supplies
  this stale expected digest (test seam: either a lower-level internal
  function taking an explicit expected-digest parameter, or by mutating the
  on-disk file between an internal read and write via a test hook — whichever
  the implementation's actual CAS entrypoint shape turns out to be; the test
  plan records the intent, the task spec/implementation fixes the exact
  seam). Assert: a typed conflict error is returned (not a generic `error`),
  and a byte-for-byte comparison (`os.ReadFile` before/after) of
  `integration-candidate.json` shows no change.
  **Edge case (Q-F08-01):** a second call retries once per spec.md's adopted
  policy; assert the retry succeeds if the second attempt's expected digest
  is now current (i.e., exactly one retry closes a race that resolved between
  attempts), and assert a conflict that persists through the single retry
  reports the typed conflict error rather than looping a third time — this
  directly tests the "retry once then fail" policy Q-F08-01 adopts, so if
  implementation evidence during development shows single-retry is
  insufficient under real concurrent load, this test (and Q-F08-01) is the
  place to revisit, not a silent widening of the retry count.

- **TC-009** — Four subtests, decision-table style (BVA + equivalence
  partitioning over valid/invalid `--events-file` shapes):
  (a) `--dry-run` against a valid, complete base/events-file input: assert
  the temp `.shark/` directory tree and epic note count are byte-for-byte
  identical before/after (directory listing + note-count snapshot diff, not
  just "no error"), and the command's reported output describes what it
  would create.
  (b) same input, non-dry-run: assert exactly one `IntegrationRun` file, one
  `IntegrationEvent` file per input array entry, one
  `IntegrationCandidate` file, and one epic `--type=reference` note exist
  after the call.
  (c) a second `backfill` call against the same epic (now already
  registered): assert rejection and a before/after diff showing zero new
  files/notes.
  (d) four malformed-input variants, each independently: `--base` names an
  unreachable commit; `--events-file` contains a duplicate `EventID`;
  `--events-file` contains an entry whose digest doesn't match the named
  commit's actual content; `--events-file` exceeds the bounded-array size
  limit (or is not valid JSON). Each variant: assert rejection and a
  before/after diff showing **zero** files/notes written (validate-fully-
  before-first-write, per spec.md's explicit ordering requirement).

- **TC-010** — Mock the claim-lookup seam only. Two subtests: (a) no active
  claim exists on `<epic-key>` at all — assert rejection, zero mutation.
  (b) an active claim exists but its session id does not match `--session` —
  assert rejection naming the mismatch, zero mutation. **Negative case:** an
  implementation that reads `--session` only as an opaque string written into
  the produced records (never checked against a live claim) would pass a
  naive "records the session id" test but fail both of these subtests, since
  neither supplies a valid matching claim.

- **TC-011** — Build a fixture epic with two features under it, no prior
  `IntegrationRun` record. Drive the real cascade dispatch path for the
  `active` step's first feature (not calling `integration.CaptureBase`
  directly by name in the test) against a temp `.shark/` dir; assert a run
  record now exists with `BaseCommit` matching the fixture's expected base.
  Drive the same cascade path for the second feature dispatch; assert the
  run record is unchanged (`BaseCommit` identical, no second run file) —
  proving the "first dispatch only" no-op behavior end-to-end through the
  real wiring point, not just through `CaptureBase`'s own idempotency (which
  TC-006 already covers in isolation).

- **TC-012** — Scenario walkthrough (recorded in this test plan, following
  F06's precedent that no fixture-execution harness exists for prose
  decision procedures): fixture epic where feature A is `completed` and
  feature B is `rejected` (or `in_development`). Render `integration_review.md`
  against this fixture's accumulated diff. Assert the rendered/produced
  report text explicitly names feature B's own current status as the reason
  epic completion remains blocked, and does not report an overriding PASS.
  Backed by a grep regression guard (new, in `embed_test.go` alongside the
  existing `TestDefectClassSweepBackwardLookingReworkRequiresCompatOrDivergence`-
  style checks) asserting the prompt's literal text contains the
  non-supersession sentence from REQ-F-006 ("This review adds a gate; it
  never overrides or supersedes an existing feature verdict...").
  **Negative case:** a prompt that reports epic-level PASS without
  qualification when an in-scope feature is rejected fails the scenario
  walkthrough, even if the grep regression guard alone would pass (the
  sentence could be present but ignored by the actual review logic the
  prompt instructs) — this is why both a grep guard and a scenario
  walkthrough are needed, not either alone.

- **TC-013** — Scenario walkthrough, four fixtures: (a) an I-## row with
  producer/consumer both in this epic and an unresolved contract-test
  pointer — assert the review reports it as not-accounted. (b) an E34-F06
  defect-class sweep referenced by an in-scope finding with `status: open`
  — assert reported as not-closed. (c) an E34-F07 `ChangeImpactSet` with
  `status: incomplete` — assert reported as not-accounted. (d) all three
  fully resolved — assert the review reports closure for all three
  categories. **Edge case:** an I-## or X-## row whose producer/consumer are
  both *outside* this epic must not be reported as in-scope (out-of-scope
  rows are excluded, not silently treated as closed).

- **TC-014** — In `internal/sharkdata/embed_test.go`, add
  `TestI01ReadinessSymmetry`: (a) run the checker against the real, current
  architecture.md I-## table (parsed from the file) — assert it passes (no
  missing-reference errors) for every row. (b) construct a test-local copy of
  the table with one required reference removed (parameterized subtest,
  once per required-reference field: producer, consumer, Rider verb where
  applicable, embedded skill reference, interaction-map entry) — assert the
  checker fails, naming the specific missing field and row. **Edge case:** a
  row where "Rider verb" is legitimately not applicable (per the field's own
  "where applicable" qualifier) must not be flagged as missing — the checker
  distinguishes "not applicable" from "missing."

- **TC-I-05-ADOPTION-MANIFEST** — Parse architecture.md's I-05
  `CanonicalAdoptionManifest v1` table programmatically (8 field names:
  `schema_version`, `source_commit`, `bundle_digest`, `changed_paths`,
  `workflow_changes`, `promoted_policies`, `override_actions`,
  `validation_evidence`). Render `epic/integration_review.md` and extract the
  `adoption_manifest` field list it instructs the worker to produce. Assert
  set-equality against the 8 architecture.md fields, and assert the prompt
  text places `adoption_manifest` as a sibling to `remediation_sweeps`/
  `change_impacts` in the same `GateResult` envelope (structural check: same
  nesting level in the documented JSON shape, not restating a whole second
  envelope schema). **Negative case:** a prompt that nests `adoption_manifest`
  as a top-level sibling of the outer worker-control envelope instead of
  inside the `gate_result` object fails this test (matches ADR-E34-01's
  "outer envelope exclusively owns outcome and executable evidence" rule).

## Integration Scenarios

- **tier-matrix.md → five consuming prompts** (TC-001/TC-002): the
  single-source-of-truth boundary — TC-001 proves the reference exists and
  isn't restated; TC-002 proves the *behavior* each route produces still
  matches the matrix. Mirrors E34-F06's TC-001/TC-002 split.
- **`active` cascade → `CaptureBase`** (TC-011): the wiring boundary between
  "the function works" (TC-006) and "production ever calls it" (TC-011) —
  this is the QA persona's "verify wiring, not just behavior" rule applied
  directly; a regression here would be dead code with 100% green unit tests.
- **`shark integration backfill` CLI → `integration.Backfill` → claim
  service** (TC-009/TC-010): the two-seam boundary — file-based writes are
  real (no mock), the one DB-backed check (claim/session) is mocked at its
  own seam, not conflated with the file-based path.
- **`integration_review` → I-03/I-04/I-##/X-## closure** (TC-012/TC-013):
  the boundary this feature was built to close — per UAT-06 in the parent
  epic's `uat-plan.md`, "Its pass result cannot replace a failed required
  feature verdict," directly exercised by TC-012.
- **E34-F08 → E34-F09 via I-05** (TC-I-05-ADOPTION-MANIFEST): see Cross-feature
  contract tests below.

## Test Infrastructure

- **Existing to reuse**: `internal/templates/includes_test.go`'s
  `TestIncludeResolver_*` table pattern (TC-001, TC-002, TC-I-05-ADOPTION-MANIFEST);
  `internal/sharkdata/embed_test.go`'s existing `TestEmbedded_*` structural
  patterns (TC-005 grep-style regression guards, TC-014's new
  `TestI01ReadinessSymmetry`); `internal/cli/commands`' existing claim-mock
  test harness pattern used by other claim-gated commands (TC-010); F09's
  `internal/cli/commands/overrides_cmd_test.go`-style filesystem-only CLI
  test pattern (TC-009).
- **New test helpers needed**:
  - A small `internal/integration` test-fixture builder (temp `.shark/`
    directory scaffolding, sha256 digest helper for constructing "stale"
    expected digests in TC-008) — this package is entirely new, so its test
    infrastructure is new by definition, not reused from elsewhere in the
    repo (no prior atomic-rename-plus-CAS Go code exists in this repo to
    borrow a helper from; confirmed via `internal/runner/transcript.go`'s
    simpler non-atomic `os.WriteFile` pattern, which this feature
    deliberately does not reuse since it lacks CAS).
  - A programmatic architecture.md I-## table parser and I-05 table parser
    (TC-002, TC-014, TC-I-05-ADOPTION-MANIFEST) — avoids hand-transcribed
    expected-value drift, the same rationale F06's codex red-team flagged
    against raw hand-copied assertions.
  - A cascade-dispatch test harness driving the real `active` step handler
    against a fixture epic/feature pair (TC-011) — new, since no existing
    test exercises the cascade action end-to-end for this purpose; the
    handler itself is existing code (this feature only adds one call inside
    it), so the harness may already partially exist for other cascade
    behaviors — confirm during implementation and extend rather than
    duplicate if so.

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-02 | E34-F05 | E34-F06, E34-F07, E34-F08 | architecture.md#i-02-gateresult-v1 | **Inherited gap** — TD-198/TD-199, no `TC-I-02-GATERESULT-PARITY` exists (E34-F05 has no test-plan.md); not re-litigated by this feature per E34-F06's own precedent. `integration_review.md`'s own envelope nesting (adoption_manifest as a `gate_result` sibling) is checked structurally by TC-I-05-ADOPTION-MANIFEST's negative case, which is the closest coverage this feature can responsibly add without owning I-02's parity test. | (owned by E34-F05, not created here) |
| I-03 | E34-F06 | E34-F08 | architecture.md#i-03-defectclasssweep-v1 | `E34-F06-defect-class-completeness-and-recurrence-routing/scenario-review-TC-005-TC-009.md#tc-i-03-defect-class-closure-cross-reference` (pointer owned by F06, per F06's own test-plan.md) | **TC-I-03-DEFECT-CLASS-CLOSURE** (F06's TC-005+TC-008+TC-009 combined) proves the I-03 shape; this feature's **TC-013** proves *consumption* — that `integration_review.md` correctly reports a sweep whose `status` is not yet `complete` as not-closed. Same pointer, not a twin test — TC-013 references F06's pointer rather than re-deriving the I-03 shape. |
| I-04 | E34-F07 | E34-F08 | architecture.md#i-04-changeimpactset-v1 | Owned by E34-F07's test-plan.md (not yet written at the time of this pass — F07's spec.md and test-plan.md are concurrent artifacts per the git status at authoring time) | **TC-013** consumes the same shape from the consumer side; when F07's test-plan.md is written, its contract-test pointer should reference TC-013 as the consumer-side proof rather than F07 writing a duplicate consumer test. |
| I-05 | E34-F08 | E34-F09 | architecture.md#i-05-canonicaladoptionmanifest-v1 | `E34-F08-tier-consistent-gates-and-final-integration-review/test-plan.md#TC-I-05-ADOPTION-MANIFEST` (this file — see Drift note above re: naming mismatch with architecture.md/interaction-map.md, which currently point at F09 instead) | **TC-I-05-ADOPTION-MANIFEST** (this plan). F09's test-plan.md already records I-05 as consumed informationally with "no Go-level contract test... verified outside this test suite" — that stance is consistent with this producer-side test existing without F09 needing a twin. |

## Cross-epic integration tests (X-##)

None. spec.md confirms `E34-cross-epic-map.md` and
`docs/product/cross-epic-integration-map.md` name no X-## row for E34-F08
(grep-confirmed, per spec.md's own "Cross-epic integrations" section).

## Codex Test-Plan Red-Team

**Attempt 1:** `codex exec -s read-only -c model_reasoning_effort=high
--skip-git-repo-check` with a prompt instructing codex to evaluate this test
plan's TC-001 through TC-014 and TC-I-05-ADOPTION-MANIFEST against spec.md's
AC-1 through AC-8, per Step 7.5's seven criteria (open-endedness, ISTQB
technique fit, enumeration completeness, ISO 25010 coverage, observability
design, negative cases, Caller-Path Contract completeness), applying the
content-only exemption to TC-001–005/012/013 and full Caller-Path rigor to
TC-006–011/014/I-05. Timeout 595s.

Command run (redacted prompt body, full text supplied inline to codex):

```
codex exec -s read-only -c model_reasoning_effort=high --skip-git-repo-check "<red-team prompt>"
```

Result: the run exceeded the available tool-invocation budget for this pass
and was not completed within this session before the test plan needed to be
handed back (a single long-running foreground `codex exec` at a 595s budget
did not return before this dispatch's own working window closed). Per Step
7.5's degrade guidance, this is logged as:

**Codex test-plan review: FAILED — run did not complete within this session's
available time; not retried a second time in this pass because the retry
budget (2×595s ≈ 20 minutes) did not fit the remaining dispatch window.**

**Verdict:** NOT RUN (documented gap, not silently skipped)
**Issues raised:** 0 (not executed)
**Issues addressed before dev:** 0
**Issues deferred:** Full codex red-team pass, owner: next worker to touch
this test plan before development begins on E34-F08, or the parent loop if it
elects to run the codex pass itself before advancing past test_planning.
Given this feature's COMPLEX tier and its history of two prior stalled
specification dispatches, the codex pass is high-value here specifically —
recommend it run before task_generation, not skipped.

This plan proceeds on internal self-review only (the drift/traceability notes
above, the wiring-specific TC-011, and the mixed-seam Caller-Path table)
rather than gating on codex availability, per Step 7.5's "do not gate on
codex availability, but document the gap" instruction.

## Recommendations

- [x] Ready for development — every spec.md AC (AC-1 through AC-8) has at
      least one TC; every runtime TC has a Caller-Path Contract (or an
      explicit content-only/internal-only justification); the I-05 producer
      test is created and cross-referenced against F09's consumer-side
      stance; the I-03/I-04 consumer-side tests reference F06's existing
      pointer and flag F07's not-yet-written one rather than inventing a
      duplicate. Two items are recorded as open but non-blocking: the
      architecture.md/spec.md candidate-shape drift (documented, deferred per
      this feature's own Q-F08-01 posture) and the codex red-team pass (not
      run this session, logged per Step 7.5's degrade path, recommended
      before task_generation rather than before test-plan approval).
- [ ] Needs BA refinement
- [ ] Needs tech refinement

*Last Updated*: 2026-09-05
