# Test Plan: E40-F02 — Bench harness: run driver and metric collection

**Created:** 2026-08-06
**Feature spec:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/spec.md` (STANDARD 13/27; 28 REQs, 21 ACs, 11 ADRs)
**Epic PRD:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
**Architecture:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md`
**UAT plan:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md`
**Research report:** `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/research-report.md`
**Task specs:** none exist yet — this is feature-level test planning ahead of `task_generation`. `spec.md`'s 21 ACs are treated as the AC list; task-spec drift steps (Step 2/3 of the workflow, task-spec vs. feature spec) have no input and are skipped.
**Status:** APPROVED — see "Red-team review" and "Verdict" at the end of this document

---

## Scope and drift analysis (Steps 1–3)

There is no separate feature PRD document distinct from `spec.md` for E40-F02
— `spec.md` is a `type: combined-spec` (requirements + architecture in one
document), and it is itself already checked for drift against its own
upstream sources (`epic.md`, `architecture.md`, `shark-bench-design.md`,
`E40-interaction-map.md`, `E40-cross-epic-map.md`) inside its own "Context"
and "Traceability" sections. This test plan re-derives that traceability
independently rather than copying it:

- Every REQ-F-/REQ-N-* traces to at least one AC in `spec.md`'s own
  Traceability table (lines 131–160). Cross-checked directly against the
  Requirements section (lines 53–100): no REQ found without a matching AC,
  no AC found that cites no REQ.
- Every epic criterion F02 claims to serve (G2, G4, G7, plus supplying
  F03's G5 inputs) is named in `epic.md` §2 and in `uat-plan.md`'s own
  criterion column — cross-checked against `uat-plan.md` UAT-01/UAT-05/
  UAT-06/UAT-07 (the four Phase-1-exit scenarios that name F02 either
  directly or via I-02/I-03). No scope creep (no REQ claims a capability
  `epic.md` §3 excludes) and no scope narrowing (every "Out of scope"
  bullet in `spec.md` has a corresponding epic-level Phase-2/Phase-3
  deferral, not a silently dropped Phase-1 requirement) found.
- **Q003 is not spec drift — it is a named, scoped, ordering constraint**
  (`spec.md` "Durable unresolved decisions"). REQ-F-021 requires one real
  transcript be captured and inspected *before* the envelope parser is
  written, with a decided fallback (`model` field) and a decided failure
  posture (named parse error, never a zero) already pinned. This test plan
  treats REQ-F-021 as a **task-sequencing dependency**, not an ambiguity:
  see "Sequencing constraint: REQ-F-021 / Q003" below. It is the one place
  this plan cannot pin exact literal values (the confirmed field names)
  before that capture happens, and it says so explicitly rather than
  guessing.
- One finding, not drift but a **precision gap this plan closes**: `spec.md`
  line 88 requires the X-07 canary to run "against a real invocation," and
  ADR-F02-10 says it "needs a scratch project and a dispatch." Neither
  states whether "a real invocation" implies a live LLM API call. Read
  literally, it does not — `internal/runner/claude_dispatcher.go`
  (`ClaudeDispatcher.Dispatch`) resolves the `claude` binary via
  `exec.LookPath("claude")` and execs whatever is first on `PATH`; nothing
  in the dispatch path requires that binary to be the real Anthropic CLI.
  This plan pins the canary test (TC-016) and the driver smoke test
  (TC-014) to a **PATH-stubbed `shark`/`claude`** technique — the same
  external-binary-stub pattern F01's TC-011 already uses for
  `go`/`golangci-lint` — so "a real invocation" means *the real
  `RunController`/`LivenessRecorder`/transcript-writer code path*, not a
  live, metered API call. No spec revision is needed: this is a test-design
  decision consistent with REQ-N-006 and does not narrow what REQ-F-020/
  ADR-F02-10 require.

No BA or architecture refinement is required. No PRD completeness gap
(Step 4 of the generic workflow) applies — `is_first_task_in_feature` has no
task specs to check against; the Component-changes and Interface-contracts
tables in `spec.md`'s Architecture section already name every script this
plan needs a test surface for (`run-one.sh`, `collect-run.sh`,
`canary-runsurface.sh`, `tests/contracts/e40_i02_artifact_contract_test.go`),
with exact file names for the three self-tests already reserved
(`tc014`/`tc015`/`tc016`). This plan does not invent new deliverables the
way F01's plan had to (its PRD completeness gap 1/2) — it only fills in test
design against deliverables `spec.md` already authorizes.

### Sequencing constraint: REQ-F-021 / Q003

REQ-F-021 requires that, **before the envelope parser is implemented**, one
real transcript from F02's own first live run is inspected and the exact
names of `modelUsage`, `num_turns`, and `duration_api_ms` are confirmed and
recorded in `bench/README.md`. This binds task decomposition and this test
plan as follows:

1. A task producing the driver (`run-one.sh`) and a task producing the
   collector's *non-envelope* logic (outcome round-trip, stage/transcript
   reconciliation, LOC, quality gates, oracle) can be built and tested from
   this plan's synthetic fixtures with **no** dependency on Q003 — none of
   those fields appear in the non-envelope parts of the I-02 schema.
2. The task that implements the **envelope parser** (`usage.*` extraction
   inside `stages[].usage`) must be sequenced *after* REQ-F-021's capture.
   That capture is naturally available once task (1)'s driver executes one
   real run — no separate live `claude` invocation is needed solely for
   Q003.
3. TC-015's envelope-shaped sub-cases (15a's `usage` sub-object assertion,
   15c's missing-field parse error) are specified below using the
   design-doc-assumed field names (`modelUsage`, `num_turns`,
   `duration_api_ms`, per `architecture.md`'s "Metric collection and
   artifact schema" section) as **provisional**. The synthetic transcript
   fixtures under `bench/scripts/testdata/run/` that drive these sub-cases
   must be authored (or corrected) using the REQ-F-021-confirmed names, not
   before. If the capture shows `modelUsage` absent, `manifest.model_id_source`
   sub-cases fall back to the envelope's top-level `model` field per the
   spec's named fallback (ADR-F02-05/REQ-F-021) — 15a must exercise both
   branches once the real shape is known.
4. This is a **BLOCKER-class dependency for implementation**, not for this
   test plan: the test *design* below (what to assert, what technique, what
   counter-factual) is fully specified now; only the literal field-name
   strings inside the fixtures are pending. Task decomposition must not
   generate the envelope-parser task ahead of a task that performs the
   REQ-F-021 capture.

### Schema completeness gap: timeout stage attribution has no declared field

**Finding.** AC-03 requires "the record names the stalled stage sourced from
the liveness record." REQ-F-019 requires the same on the DB-status fallback
path (`uat-plan.md` UAT-05: "the stage attribution comes from F04's liveness
record, or failing that from the entity's status in the scratch DB").
`spec.md`'s own Data model changes table (the field inventory this plan's
`TC-001` and every `TC-015` sub-case validate against) has **no field that
carries the stalled stage**, and its `sources` field is documented as
carrying values from a **closed four-value set** — `runresult` /
`transcript` / `scratch_db` / `postrun` — which does not include
`liveness`. Grepped directly against `spec.md`'s "Data model changes"
table (lines 202–236): confirmed, neither exists. Without a fix, `TC-015b`
(AC-03) has no field to assert against, and `TC-001`'s own closed-set check
on `sources` (added below) would reject a correct timeout record the
moment it names `liveness` as a family's source. This is the same class of
gap F01's test plan hit and resolved directly rather than leaving open
(F01 "PRD completeness gaps," `diff-ledgers.sh`/`verify-clean-checkout.sh`).

**Resolution, pinned here for task decomposition and the Go schema
validator to implement (not deferred as an open question):**

- Add a new top-level field `timeout_detail` (object, present **only** when
  `outcome == "timeout"`, otherwise absent — never a zero-valued object on
  a non-timeout record): `{stage_index, status, action, agent_type,
  provider, source}`, where `source` is `"liveness_stream"` or
  `"scratch_db_status_fallback"` (the two sources `architecture.md`'s "Run
  liveness contract" section and `uat-plan.md`'s UAT-05 both name).
- Extend the `sources` closed set from four values to five:
  `runresult` / `transcript` / `scratch_db` / `postrun` / `liveness`. A
  record's `sources.stalled_stage` (or equivalent key naming which family
  `timeout_detail` came from) uses `"liveness"` when resolved from the
  stream, `"scratch_db"` (already in the set) when resolved from the DB
  status fallback.
- `TC-001` gains a **second committed golden record**,
  `tests/contracts/testdata/e40_i02_golden_record_timeout.jsonl`, shaped
  like a real timed-out run: `outcome: "timeout"`, `runresult.*` **absent**
  (not null-filled placeholders — genuinely absent, since no `RunResult`
  was ever delivered), `timeout_detail` populated, and every
  `RunResult`-sourced family (`rejections`, most of `stages[].usage`)
  correspondingly absent or minimally populated from whatever the liveness
  stream + partial transcripts can supply. `TC-001`'s schema check treats
  `runresult.*` as **conditionally required** — required when
  `outcome != "timeout"`, absent when it is — rather than unconditionally
  required, which is itself a schema rule this plan is pinning down that
  `spec.md`'s table left implicit.
- **Owner/trigger, matching F01's own hand-off pattern:** this resolution
  is authoritative for task decomposition and for this plan's own `TC-001`
  design starting now — it is not blocked on a spec edit. The E40-F02
  spec/architecture owner should still add `timeout_detail` and the fifth
  `sources` value to `spec.md`'s Data model changes table before the
  envelope-parser and collector tasks are dispatched, so the schema in
  `spec.md` and the schema this test plan (and, later, the Go contract
  test) enforces do not silently diverge. Interim status: until that
  lands, this section is the authoritative source for the two new fields.

## Acceptance-criteria quality review (Step 5)

Every AC in `spec.md` was checked against: unambiguous (one interpretation),
testable (concrete inputs/outputs), traceable (maps to a REQ, which maps to
an epic criterion), complete (covers the full requirement, not just happy
path), and specifies expected outputs (not "works correctly"). Result: all
21 pass, with one precision gap already resolved above (AC-03's missing
schema field) and one further ambiguity below that this plan resolves by
explicit choice rather than by guessing silently (Rule 7).

**No AC is an open-ended robustness assertion.** The closest candidates and
why each is closed, not open:

- **REQ-N-004 "deterministic... two runs' records differ only where the
  measurements differ"** (AC-14) — closed by requiring **byte-identical**
  stdout across two independent `collect-run.sh` invocations over the same
  synthetic (unchanging) input, which is a concrete, mechanically checkable
  claim, not a vague "is reproducible."
- **REQ-N-005 "fail loud everywhere a measurement could be fabricated"** —
  closed by the enumerated `errors[]` kind set (six named values, all but
  one exercised below) rather than a generic "never lies" assertion; a
  kind not in the closed set is itself a schema violation `TC-001` catches.
- **AC-13's canary "asserts... are unchanged"** — closed by REQ-F-020's own
  concrete failure mode ("aborts, naming the changed field"), not a vague
  "detects drift"; `TC-016b`'s mutated-fixture case pins the exact
  assertion.

**Ambiguity finding — `errors[].kind`'s `transcript_missing` vs.
`stage_join_error` distinction is underspecified.** `spec.md`'s Data model
table (line 235) declares both as separate closed-enum values, but no REQ
or AC names a scenario that produces `transcript_missing` specifically.
AC-07's own text ("with one transcript deliberately removed, the collector
reports a **named join error** stating the expected and observed... counts")
maps unambiguously to `stage_join_error` by name, leaving
`transcript_missing` with zero described scenarios anywhere in `spec.md`.
AC-06's phrase "no missing-transcript error" is the only other prose
mention, and it only proves a **negative** (an `advance_status` stage must
not trigger it), not what positive scenario does.

**Resolution adopted by this plan (not silently guessed):** `stage_join_error`
covers REQ-F-013's two named defects — an aggregate count mismatch and a
per-position filename-component mismatch (both explicitly named in
REQ-F-013's own text as producing "a named join error"). `transcript_missing`
is reserved for a narrower, currently-undescribed case: a stage's own
expected transcript path resolves and is accounted for in the aggregate
count, but the file itself is empty or unreadable at read time (e.g. a
zero-byte file from an interrupted write) — a defect the aggregate count
check alone cannot see. `TC-015m` (added below) gives this kind schema-level
coverage as an edge case; this plan flags the distinction for the spec
owner to confirm or correct before the collector task is dispatched, since
two different implementers could otherwise pick two different splits and
silently disagree on which kind a given defect maps to.

**`usage_unavailable` is deliberately unexercised in this plan.**
`spec.md`'s "Out of scope" section states it applies to codex-dispatched
stages only, and Phase 1 benches Claude-dispatched stages exclusively (no
codex path exists to exercise). Recorded here, not silently dropped, per
Rule 12 — a future Phase-2/3 codex-benching plan must add its test case
rather than discover the kind was always unexercised.

---

## AC Test Matrix

Every AC in `spec.md` has at least one test case below. `TC` values point to
one of: the new Go contract test (`TC-001`), or a sub-case letter inside one
of the three new bash self-tests (`TC-014`/`TC-015`/`TC-016`, matching
`spec.md`'s own reserved file names `tc014_run_one_smoke_test.sh`,
`tc015_collect_run_record_test.sh`, `tc016_canary_runsurface_test.sh`), or
`TC-017` (a small `bench/README.md` content check). Full Caller-Path
Contracts are in their own section below the technique/ISO tables, to avoid
repeating the same entrypoint five times per script.

| AC | Requirement(s) | TC | Technique | Input/setup | Expected outcome; edge and negative case |
|---|---|---|---|---|---|
| AC-01 | REQ-F-001 | TC-014a | Contract-surface enumeration | `run-one.sh --item cart-remove-item-last-match --variant default --rep 1 --timeout 60 --out <tmp>` against a PATH-stubbed `shark` returning a `completed` `RunResult` on the first call | Exits 0 with no stdin read (assert `< /dev/null` doesn't hang it); exactly one file exists at `<tmp>/cart-remove-item-last-match/default/rep-1/record.jsonl`; the file is exactly one line. Negative: running the identical command a second time into the same `--out` does not silently overwrite garbage — it either refuses (non-zero, naming the existing path) or reproduces byte-identically (TC-015j proves the collector half of that; TC-014a asserts the driver does not crash or hang on the second invocation, whichever posture is chosen) — REQ-F-018 pins the path as deterministic and self-identifying, batch-skip logic itself is out of scope (see spec.md "Out of scope"). |
| AC-02 | REQ-F-003, REQ-F-009, REQ-F-011, REQ-F-014, REQ-F-015, REQ-F-016, REQ-F-017 | TC-001, TC-015a, TC-015g, TC-015h, TC-015l | Contract-surface enumeration (six metric families as one interaction surface) | TC-001 validates the committed golden record has every block; TC-015a's `completed`-outcome sub-case (with two `spawn_agent` stages and one `advance_status` stage) asserts `timing`, `stages[].usage`, and `runresult.*` are populated; TC-015g asserts `rejections.*`; TC-015h asserts `oracle.*`/`quality.*`; TC-015l asserts `loc.*`'s prod/test-split arithmetic against a synthetic `numstat.txt` with genuinely non-zero, mixed values | All six families present with concrete, non-placeholder values in the golden record and in the synthetic-fixture-driven collector output. Negative: a record missing any one family (tested via TC-001's malformed-fixture subtests) is rejected by the schema validator, naming the missing block. |
| AC-03 | REQ-F-006, REQ-F-019, REQ-N-003 | TC-015b (record shape), TC-014e (real kill) | State transition (timeout kill path) | TC-015b feeds `collect-run.sh` a synthetic run directory with no `run/stdout.json` (RunResult never delivered), an `exit_status` file recording SIGKILL-by-cap, and a `run/stderr.ndjson` liveness stream whose last `stage_start` never closes; TC-014e runs `run-one.sh` with `--timeout 2` against a stub `shark run` that sleeps 30s ignoring `SIGTERM` | TC-015b: record carries `outcome: "timeout"`; `timeout_detail` (the new field pinned in "Schema completeness gap" above) names the stalled stage's `stage_index`/`status`/`action`/`agent_type`/`provider`, sourced from the liveness stream's last open `stage_start`, with `timeout_detail.source == "liveness_stream"`; `runresult.*` is genuinely **absent** (not null-filled), and no attempt is made to read a nonexistent `run_end` line. A second sub-case (no liveness stream present either) asserts `timeout_detail.source == "scratch_db_status_fallback"`, reading the entity's status directly from the synthetic scratch DB fixture. TC-014e: `run-one.sh` still exits and still writes `record.jsonl` (via the collector) with `outcome: "timeout"`. Negative: a run that completes within the cap never gets `outcome: "timeout"` even if it took most of the budget (boundary case: complete at `timeout - 1s`). |
| AC-04 | REQ-F-006, REQ-N-003 | TC-014e | Attack-class enumeration (orphan-process defensive property) | Same TC-014e stub, but the stub `shark` itself forks a grandchild process (simulating `claude` as `shark`'s child) that also ignores `SIGTERM` and writes a heartbeat file every second | After the cap plus grace period fires, `pgrep -g <the invocation's process group id>` (captured by the test before invoking `run-one.sh`) returns zero PIDs — verified by process-group inspection, not by `run-one.sh`'s own exit code. Negative: a naive `SIGTERM`-to-`shark`-only implementation would leave the grandchild's heartbeat file still growing after the cap; TC-014e polls the heartbeat file's mtime post-cap and asserts it stops advancing. |
| AC-05 | REQ-F-012, REQ-N-005 | TC-015c | Decision table (stage action × envelope presence) | `collect-run.sh` over a synthetic run directory whose sole `spawn_agent` stage's transcript has its `---STDOUT---` JSON envelope missing the `num_turns` key (one axis at a time: also run once each for `modelUsage` and `duration_api_ms` missing) | Record's `errors[]` contains one entry `kind: "envelope_parse_error"`, `detail` naming the missing field and the transcript path; the stage's corresponding `usage.num_turns` (or the relevant field) is **absent**, not `0` or `""`. Negative: a transcript with all three fields present produces zero `envelope_parse_error` entries — proves the check isn't unconditionally firing. |
| AC-06 | REQ-F-012 | TC-015d | Decision table (stage action × envelope presence, the `advance_status` row) | Synthetic run directory whose only stage is `action: "advance_status"` (no transcript file written for it, matching `maybeWriteTranscript`'s scoping to `handleSpawnAgent`/`recordDispatchFailure`) | Zero `envelope_parse_error` and zero `transcript_missing` entries in `errors[]` for that stage. Negative: the same fixture with a stray transcript file accidentally present for the `advance_status` stage index must not be read as that stage's envelope (the collector must key transcripts by `spawn_agent` stage index, not by array position) — TC-015d asserts the stray file is ignored, not misattributed. |
| AC-07 | REQ-F-013 | TC-015e, TC-015m (schema completeness) | Decision table (transcript count reconciliation) | Synthetic run directory with two `spawn_agent` stages in `RunResult.Stages` but only one transcript file present under `run/transcripts/` | Record's `errors[]` contains one `kind: "stage_join_error"` entry whose `detail` names the expected count (2) and observed count (1). Edge case: the *k*-th transcript's filename status/provider component intentionally mismatches the *k*-th agent stage's `status`/`provider` (both files present, count equal) — this is a **second** stage_join_error sub-case, `detail` naming the mismatched component, not the count. Negative: equal count with matching components produces zero `stage_join_error` entries. TC-015m separately exercises `transcript_missing` (a zero-byte/unreadable transcript at an otherwise count- and component-correct position) per this plan's Ambiguity Finding resolution above — not an AC-07 sub-case, added for `errors[].kind` schema completeness. |
| AC-08 | REQ-F-009, REQ-F-010 | TC-015a | Equivalence partitioning (six `outcome` classes) | Six synthetic run directories, one per `RunResult.Outcome` value (`completed`, `paused` with a populated `QuestionBlock`, `failed`, `already_terminal`, `no_action`) plus the harness-assigned `timeout` (TC-015b) | Each record's `outcome` field carries the exact source value unchanged; the `paused` case's `runresult.question_block` is the verbatim `QuestionBlock` object and that run is excluded from any "is this a defect" rollup field the record carries. Negative: a `paused` outcome must never be classified anywhere in the record as `failed` or as `timeout` — TC-015a asserts no cross-contamination between the two boolean-shaped rollup fields. |
| AC-09 | REQ-F-015, REQ-N-005, REQ-N-007 | TC-015h | Contract-surface enumeration (diff-ledgers.sh's stdout is the contract) | A synthetic run directory's `post/test-diff.json` and `post/lint-diff.json` are **generated by a real invocation** of `bench/scripts/diff-ledgers.sh --kind=test`/`--kind=lint` against hand-authored base/post ledgers (not hand-typed record content) — same technique as `bench/scripts/testdata/{lint,test}/*.json`, reused, not re-derived. `collect-run.sh` reads that directory. | `record.oracle.p2p_regressions`/`.p2p_regressions_count`/`.removed`/`.removed_count` and `record.quality.lint_new_issues`/`.lint_new_issues_count` are byte-identical (after JSON key reordering per REQ-N-004's sorted-keys rule, not after any value transformation) to `diff-ledgers.sh`'s own `regressions`/`regressions_count`/`removed`/`removed_count` and `new_issues`/`new_issues_count` fields. Negative: `record.sources.oracle` / `record.sources.quality` name `"postrun"` (copied), never `"collector-recomputed"` — proves the collector performed no independent diff (ADR-F02-06). |
| AC-10 | REQ-F-010, REQ-F-015, REQ-N-005 | TC-015i | Attack-class enumeration (toolchain drift, silent-corruption class) | Synthetic run directory whose `post/` contains a marker recording that `diff-ledgers.sh --toolchain-guard` aborted (mirroring what `run-one.sh` would have recorded had the live guard failed), naming a mismatched Go-version axis, with no `test-diff.json`/`lint-diff.json` present (because the pinned order in REQ-F-015 means the guard runs first and nothing after it ran) | Record's `errors[]` contains one `kind: "postrun_check_aborted"` entry naming the axis; `record.oracle.p2p_regressions`/`record.quality.lint_new_issues` are absent (`null`), never a diff computed anyway. Negative: a run directory with a passing toolchain guard marker never produces a `postrun_check_aborted` entry, even when the diff itself later reports zero new issues. |
| AC-11 | REQ-F-002, REQ-N-002 | TC-014b | Attack-class enumeration (live-repo leak/mutation surface) | This repo's own `.sharkconfig.json` points at a **Turso cloud** backend (`CLAUDE.md`: `skip_migrations: true`, `SHARK_DB_BACKEND=turso`) — there is no local live-DB file to hash, so the primary check is structural, not a file diff. Before invoking `run-one.sh` (stubbed `shark`, `--out <tmp>`), snapshot: SHA-256 of the live repo's `.sharkconfig.json` and `git status --porcelain` (must be unchanged before/after — only pre-existing untracked/dirty state is tolerated, not new dirt). Reuse TC-014c's stub argv/cwd log: every stubbed `shark` invocation the driver makes records its own working directory. | The two file/git snapshots are byte-identical before and after. **Primary DB-integrity check:** every logged invocation's working directory resolves inside the `mktemp` scratch dir (never inside, or an ancestor-walk away from inside, the live repo tree) — since shark's project-root auto-detection walks *up* from cwd, this is what actually prevents it from ever resolving to the live repo's `.sharkconfig.json`/Turso pointer in the first place, independent of what backend is configured. A local-SQLite file-hash comparison is a secondary check, only meaningful when a contributor's environment happens to run local SQLite instead of Turso. Negative: `bench/scripts/run-one.sh` is grepped (code review, not a dynamic assertion — mirrors F01's REQ-NF-003 verification method) for any direct invocation of `shark admin init`/`shark cloud init`/writes to a `.sharkconfig.json` path outside a `mktemp`-created scratch directory; none found is a pass condition recorded here, not re-derived at review time. |
| AC-12 | REQ-F-004, REQ-N-002 | TC-014d | Attack-class enumeration (leak surface, mirrors F01's TC-003 technique) | Real `bench/corpus` item's F2P paths/test names (e.g. `inventory-reserve-rejects-negative-quantity`) are known ahead of dispatch. The stub `shark run` invocation, when called, inspects its own `--workdir` argument and writes a marker file recording whether any F2P path exists in that checkout at that moment. `run-one.sh` runs to a `completed` outcome (stub returns a canned successful `RunResult` with no code changes) so post-run F2P injection fires. | The dispatch-time marker records **zero** F2P paths present. After `run-one.sh` completes, `bench/scripts/verify-clean-checkout.sh`-style grep (or a direct file-existence check) of the same checkout shows **all** F2P paths now present. Negative: if F2P files leaked into the checkout before dispatch, the marker (not the checkout's final state) is what catches it — asserting only the post-run state would miss a dispatch-time leak that got silently overwritten. |
| AC-13 | REQ-F-020 | TC-016a (pass), TC-016b (fail) | Attack-class enumeration (upstream shape-drift / silent-corruption class, ADR-004) | Per AC-21, the canary runs **before** `run-one.sh`'s own provisioning — so it cannot reuse a project `run-one.sh` hasn't created yet. `canary-runsurface.sh` therefore provisions its **own throwaway scratch project** (via `scripts/shark-scratch-env.sh`, the same real provisioning path, not a second implementation of it) with `observability.capture_agent_transcripts` explicitly set `true` before dispatching, and tears the scratch project down on exit regardless of outcome. TC-016a: a real `shark run` invocation in that throwaway project, with a PATH-stubbed `claude` binary emitting a fixed, valid envelope, so a real `spawn_agent` stage really executes and a real transcript is really written (transcript writing requires both `capture_agent_transcripts == true` *and* a non-empty `ProjectRoot` — both are true here because the canary's own scratch project supplies them). TC-016b: same, but `canary-runsurface.sh` is pointed (via an env-var override, mirroring TC-011's `DIFF_LEDGERS_GOLANGCI_CONFIG` override technique) at a **fixture** `RunResult` JSON with `stages_completed` renamed to `stagesCompleted` | TC-016a exits 0, printing the confirmed field set, and confirms the transcript byte format (`COMMAND:`/`EXIT:`/`DURATION:<ms>ms`/`---STDOUT---`/`---STDERR---`) against the real file the throwaway run wrote. TC-016b exits non-zero, naming exactly `stages_completed` (or `stagesCompleted`) as the changed field — never a generic "shape mismatch" message. Negative: a fixture with every field present but reordered (JSON key order, not field renaming) must still pass — the canary checks field *presence and names*, not encoding order. |
| AC-14 | REQ-F-007, REQ-N-004 | TC-015j | State transition (re-run stability, independent invocations over the same input) | `collect-run.sh --run-dir <same synthetic dir>` invoked twice as two fully separate process invocations (not two calls in one script sharing state) | The two stdout outputs are byte-identical, including key order. Negative: a version of the collector that includes a wall-clock "collected_at" timestamp field would fail this TC by construction — proving REQ-N-004 excludes such a field from the schema; TC-001's schema check additionally asserts no non-listed field appears in the golden record. |
| AC-15 | REQ-N-001, REQ-N-006, REQ-N-007 | TC-001 | Contract-surface enumeration | `go test ./tests/contracts/... -run TestTC001_I02ArtifactContract` reading the committed `tests/contracts/testdata/e40_i02_golden_record.jsonl` via `os.ReadFile` | Passes under `make test` with the submodule uninitialized, no scratch project directory created, no network call attempted (verified the same way F01's TC-013 verifies offline-ness is a real property, not an assumption — no live subprocess is spawned by this test at all, so there is nothing to isolate). Negative: the four malformed-fixture subtests (missing `schema_version`, unsupported `schema_version`, `outcome` outside the six-value set, an `errors[]` entry missing `detail`) each fail with that field named. |
| AC-16 | REQ-F-005, REQ-N-005 | TC-014c | Attack-class enumeration (never invent/harness-choose a key, mirrors the cloud-DB key-assignment rule) | Stub `shark create feature ... --json` / `shark create task ... --json` return a canned assigned key (e.g. `E01-F01-003`) different from any value `run-one.sh` might be tempted to guess; the stub also records every argv it was invoked with to a log file | `record.manifest.seeded_keys.*` equals exactly the stub's returned key(s). The stub's argv log shows no `--key`/`--id`/equivalent explicit-key flag on any `create` invocation. Negative: a hypothetical implementation that derives the key from `--item <id>` instead of the create response's `--json` output would still "work" against a single stub call but would diverge the moment the stub returns a key the harness didn't expect — TC-014c's stub deliberately returns a key that does *not* match any input the driver was given, so an implementation reading the wrong source fails loudly rather than by coincidence. |
| AC-17 | REQ-F-021 | TC-017 (content) + TC-015a (record field) | Contract-surface enumeration | `bench/README.md`'s new "Run driver and artifact schema" section (content-only renderer check: the section exists and names `modelUsage`/`num_turns`/`duration_api_ms` or documents the `model`-field fallback, per whichever REQ-F-021's real capture confirms) plus TC-015a's `manifest.model_id_source` field assertion | TC-017 (content-only): the section is present and legible; its stated field names match what TC-015a's fixtures assert. TC-015a: `manifest.model_id_source` is exactly `"modelUsage"` or `"model"` (never any other string), consistent across every fixture in the same test run. This TC's literal expected values are pending REQ-F-021's capture — see "Sequencing constraint" above. |
| AC-18 | REQ-F-015, REQ-F-016, REQ-F-017, ADR-F02-11 | TC-014f (ordering), TC-015l (arithmetic) | Decision table (measurement order × injection order) | `run-one.sh` full run against a corpus item, stub `shark run` returns `completed` with **zero code changes** (the agent's dispatch is a no-op). Real `bench/scripts/build-ledgers.sh`/`diff-ledgers.sh`/`git diff --numstat` execute for real against the real fixture checkout, writing `post/numstat.txt`/`post/lint-diff.json` (this sub-case is not fully synthetic — it drives the real LOC/lint pipeline, still with no API spend since the agent step is stubbed). `collect-run.sh`'s own prod/test-split arithmetic over `post/numstat.txt` is separately and more thoroughly exercised by TC-015l's synthetic fixture (see AC-02), so TC-014f's job is proving the *driver's ordering*, not re-proving the collector's arithmetic. | Post-run: `loc.test_added == 0` and `lint_new_issues_count == 0` in the emitted record, **even though** a direct filesystem check of the checkout shows the item's F2P files ARE present (injected after measurement, per the pinned order). Negative (the exact corruption ADR-F02-11 names): a deliberately mis-ordered test double that injects F2P *before* measuring must show `loc.test_added > 0` and `lint_new_issues_count > 0` purely from the injected files — TC-014f keeps this inverted-order variant as a documented counter-factual check, not a normal test path, to prove the assertion is actually sensitive to ordering rather than vacuously true. |
| AC-19 | REQ-F-014, ADR-F02-09, REQ-N-005 | TC-015g | Decision table (RunResult-inferred rejection count × DB-derived count: agree/disagree) + boundary value analysis (zero rejections vs. N) | A synthetic run directory's `RunResult` implies 1 rejection (one status re-entry after a gate stage); a **committed SQL fixture** (`.sql`, not a binary `.db`) seeds a temp scratch DB's `entity_history` with 2 backward transitions for that entity, plus a `work_sessions` row and an `entity_type='task'` row so the entity_key→entity_id resolution (ADR-F02-09) has something to resolve against | Record's `errors[]` contains one `kind: "crosscheck_disagreement"` entry whose `detail` names both counts (`runresult_inferred=1`, `entity_history_derived=2`); `rejections.crosscheck.agrees == false`. Edge case: a fixture where both sources agree at zero rejections asserts `agrees == true` with **no** error entry (zero is not a special-cased "skip the check" value). Negative: an entity_key that fails to resolve to any `entity_id` in the fixture DB produces a distinct, separately named error (not silently treated as "0 backward transitions" — that would fabricate agreement). |
| AC-20 | REQ-F-008, ADR-F02-02 | TC-015f | Boundary/state enumeration (liveness stream present vs. absent) | Synthetic run directory identical to a clean `completed` fixture except `run/stderr.ndjson` is empty/absent and no `run.log: <path>` line was ever captured; `.shark/runs/` (simulated under the fixture) contains exactly one dated subdirectory | `record.manifest.run_id` equals that subdirectory's name; `record.manifest.run_id_source == "fallback_newest_dir"`. Edge case: two candidate subdirectories present (simulating a race) — the newest by mtime is chosen, and `run_id_source` still records the fallback, not a false "resolved from stream" claim. Negative: when the liveness stream *is* present, `run_id_source` must never read `"fallback_newest_dir"` even if a stale extra `.shark/runs/` subdirectory also exists — the stream takes priority whenever available. |
| AC-21 | REQ-F-020 | TC-014g | Decision table (canary outcome × `--skip-canary` flag, 3 reachable combinations) | `run-one.sh` invoked three ways against a stub `canary-runsurface.sh`: (i) default flags, stub canary exits 0; (ii) default flags, stub canary exits 1 naming a field; (iii) `--skip-canary`, stub canary would exit 1 (but must never be invoked) | (i): provisioning proceeds normally. (ii): `run-one.sh` exits non-zero *before* `scripts/shark-scratch-env.sh` is invoked (asserted by: no scratch directory created, no canned "provisioning" marker written) and the canary's named field appears in `run-one.sh`'s own error output. (iii): provisioning proceeds, the stub canary script is never executed (asserted via an invocation-count file the stub would have written), and `meta.json` records `skip_canary: true`. Negative: `meta.json` must record `skip_canary: false` (not merely omit the key) on path (i)/(ii), so a consumer can't confuse "never asked" with "explicitly ran." |

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-01 | Contract-surface enumeration | TC-014a | The single-run command is F02's own external interface; every argument and the deterministic-path guarantee must be enumerated, not sampled. |
| AC-02 | Contract-surface enumeration | TC-001, TC-015a, TC-015g, TC-015h, TC-015l | The I-02 record is a cross-feature interaction surface (F03 reads nothing else) — every declared block is a contract obligation, including the `timeout_detail`/`sources` extension this plan pins in "Schema completeness gap." |
| AC-03, AC-04 | State transition + Attack-class enumeration | TC-015b, TC-014e | Timeout is a lifecycle transition (running → killed → recorded); "no orphan process" is exactly the defensive property attack-class enumeration exists to force (kill-the-parent-only is the specific attack this AC guards against). |
| AC-05, AC-06, AC-07 | Decision table (stage action × envelope/transcript presence) | TC-015c, TC-015d, TC-015e, TC-015m | Three interacting conditions (action type, envelope field presence, transcript count/component match) determine three distinct error classes — a decision table forces every reachable combination, not just "some transcript is broken." TC-015m adds the fourth, currently-undescribed combination (`transcript_missing`) this plan's Ambiguity Finding resolves rather than leaves untested. |
| AC-08 | Equivalence partitioning | TC-015a | `Outcome` is a closed six-value set (five from `RunResult` + harness `timeout`); EP requires one representative test per class, which is exactly what a naive three-outcome mental model (Finding 3 of the research report) would miss. |
| AC-09, AC-10 | Contract-surface enumeration + Attack-class enumeration | TC-015h, TC-015i | AC-09 is a byte-identity contract against another script's real stdout (not a value the collector invents); AC-10 is the defensive property that a wrong-toolchain diff must never silently compute — the class to enumerate is every way the abort could be papered over. |
| AC-11, AC-12 | Attack-class enumeration | TC-014b, TC-014d | Directly mirrors F01's AC-001/TC-003 rationale: "never leaks/mutates X" is a defensive property, enumerated by leak/mutation surface, not asserted only in the happy path. |
| AC-13 | Attack-class enumeration | TC-016a, TC-016b | ADR-004's own framing: an upstream `RunResult`/`StageLog`/transcript shape change is a silent-corruption attack surface; the canary's job is to convert it into a loud, specific failure. |
| AC-14 | State transition (re-run stability) | TC-015j | "Byte-identical across two independent invocations" is the same re-run-stability claim F01's TC-006/TC-007 test, applied to the collector instead of the admission gate. |
| AC-15 | Contract-surface enumeration | TC-001 | Schema validation over a committed record is a pure interaction-surface check, following F01's TC-001 precedent exactly (and reusing its file/package conventions). |
| AC-16 | Attack-class enumeration | TC-014c | "Never specifies a key" is a defensive property against a specific corruption mode (a harness-invented key silently diverging from the real DB-assigned one) — the class enumerated is every place a key could leak in from the wrong source. |
| AC-17 | Contract-surface enumeration | TC-017, TC-015a | The confirmed field names are a documented contract between the spec's Q003 obligation and the implementation; content-only for the doc half. |
| AC-18 | Decision table (measurement order × injection order) + BVA (LOC arithmetic) | TC-014f, TC-015l | Two binary conditions (measured-before-injection: yes/no) combine to determine whether the metric is corrupted; the counter-factual (inverted-order) row is what proves the test is sensitive to the property it claims to guard, not just a happy-path echo. TC-015l separately applies BVA to the `numstat` split itself (a prod line, a test line, a binary/no-line-count line) so the ordering property isn't the only thing ever exercised at non-zero values. |
| AC-19 | Decision table + Boundary value analysis | TC-015g | Agree/disagree is a decision-table axis; zero-vs-N rejection count is a boundary axis (a naive implementation might special-case "zero" as "nothing to check" and silently skip the crosscheck). |
| AC-20 | Boundary/state enumeration | TC-015f | Directly mirrors F01's TC-002 rationale: stream-present vs. stream-absent are the two real states a completed run directory can be in; both are boundaries, not one representative case. |
| AC-21 | Decision table (canary outcome × skip flag) | TC-014g | Two binary conditions (canary passes/fails × skip flag on/off) combine to three reachable, distinct-behavior combinations (the fourth, skip+irrelevant-outcome, collapses into "skip wins"), which a decision table forces explicitly rather than testing only the two obvious ones. |

ACs without a technique annotation: none. Every AC above has at least one named technique.

## ISO 25010 Coverage Matrix

`N/A` justifications follow `uat-plan.md`'s own framing ("Not a product
concern here — the harness is offline tooling") plus REQ-N-003 (the one
latency claim, F04's heartbeat cadence, is not this feature's to test).

| AC | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-01 | ✅ TC-014a | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-02 | ✅ TC-001, TC-015a/g/h/l | N/A | ✅ TC-001 (I-02 shape) | N/A | N/A | N/A | ✅ TC-001 (schema gate) | N/A |
| AC-03 | ✅ TC-015b | N/A | N/A | ✅ TC-015b (stalled stage named) | ✅ TC-015b, TC-014e | N/A | N/A | N/A |
| AC-04 | ✅ TC-014e | N/A | N/A | N/A | ✅ TC-014e | ✅ TC-014e (no orphaned budget-consuming process) | N/A | N/A |
| AC-05 | ✅ TC-015c | N/A | N/A | ✅ TC-015c (named field) | ✅ TC-015c | N/A | N/A | N/A |
| AC-06 | ✅ TC-015d | N/A | N/A | N/A | ✅ TC-015d | N/A | N/A | N/A |
| AC-07 | ✅ TC-015e | N/A | N/A | ✅ TC-015e (named counts) | ✅ TC-015e | N/A | N/A | N/A |
| AC-08 | ✅ TC-015a | N/A | N/A | N/A | ✅ TC-015a (no misclassification) | N/A | N/A | N/A |
| AC-09 | ✅ TC-015h | N/A | ✅ TC-015h (diff-ledgers.sh contract) | N/A | N/A | N/A | N/A | N/A |
| AC-10 | ✅ TC-015i | N/A | ✅ TC-015i (toolchain pin) | ✅ TC-015i (axis named) | ✅ TC-015i | N/A | N/A | N/A |
| AC-11 | ✅ TC-014b | N/A | N/A | N/A | N/A | ✅ TC-014b (live-repo integrity) | N/A | N/A |
| AC-12 | ✅ TC-014d | N/A | N/A | N/A | N/A | ✅ TC-014d (F2P leak surface) | N/A | N/A |
| AC-13 | ✅ TC-016a/b | N/A | ✅ TC-016a/b (X-07 shape pin) | ✅ TC-016b (field named) | ✅ TC-016a/b | N/A | N/A | N/A |
| AC-14 | ✅ TC-015j | N/A | N/A | N/A | ✅ TC-015j | N/A | ✅ TC-015j (no stray fields) | N/A |
| AC-15 | ✅ TC-001 | N/A | N/A | N/A | N/A | N/A | ✅ TC-001 (lint clean, CI-safe) | N/A |
| AC-16 | ✅ TC-014c | N/A | N/A | N/A | N/A | ✅ TC-014c (no harness-invented key) | N/A | N/A |
| AC-17 | ✅ TC-015a | N/A | N/A | ✅ TC-017 (documented) | N/A | N/A | N/A | N/A |
| AC-18 | ✅ TC-014f, TC-015l | N/A | N/A | N/A | ✅ TC-014f (ordering-sensitive) | N/A | N/A | N/A |
| AC-19 | ✅ TC-015g | N/A | N/A | ✅ TC-015g (named counts) | ✅ TC-015g | N/A | N/A | N/A |
| AC-20 | ✅ TC-015f | N/A | N/A | N/A | ✅ TC-015f | N/A | N/A | N/A |
| AC-21 | ✅ TC-014g | N/A | N/A | N/A | ✅ TC-014g (fails loud, before provisioning) | N/A | N/A | N/A |

No empty cells. Performance and Portability are `N/A` across the board
(uat-plan.md: "Not a product concern here — the harness is offline
tooling"; no cross-OS or cross-container claim is made anywhere in
`spec.md`). No coverage gap: every non-`N/A` cell cites a TC that exists in
the AC Test Matrix above.

## Observability Design (per behavior)

This is offline harness/curator tooling with no production request path
(same posture as F01's own test plan). "Observability" here means the
scripts' own machine-readable output, which is what a curator, a batch
runner (Phase 2/F03), or this collector's own consumer depends on without
re-deriving it.

| Behavior | Log/stdout evidence | Trace/metric | Test assertion |
|---|---|---|---|
| Driver phase progress | `run-one.sh` prints one line per phase to stderr (`provision`, `seed`, `invoke`, `postrun`) in order | N/A — internal, no request path | TC-014a asserts the phase markers appear, in order, on stderr |
| Collector error surfacing | Every named error class (`envelope_parse_error`, `stage_join_error`, `transcript_missing`, `crosscheck_disagreement`, `crosscheck_resolution_error`, `postrun_check_aborted`, `usage_unavailable`) appears in `record.errors[]` with `kind`+`detail`, never as a bare non-zero exit code alone (REQ-N-005) | N/A | TC-015c/e/g/i each assert the specific `kind`+`detail` pair, not just "collect-run.sh exited 0 and something happened" |
| Metric provenance | `record.sources.<family>` names `runresult`/`transcript`/`scratch_db`/`postrun` per REQ-N-007 | N/A | TC-001 (schema requires the field) + TC-015a/g/h/i (each assert the correct source value for their family) |
| Canary result | `canary-runsurface.sh` prints `PASS` or a `FAIL: <field>` line naming the mismatched field, before any provisioning happens | N/A | TC-016a/b |
| Toolchain-guard abort (inherited from F01) | `diff-ledgers.sh --toolchain-guard`'s own stderr (F01-owned, unmodified) names the axis; F02's collector copies that into `postrun_check_aborted.detail`, does not re-derive it | N/A | TC-015i |
| `--skip-canary` audit trail | `meta.json` records `skip_canary: true|false` explicitly on every run, never omitted | N/A | TC-014g |

No new metrics/traces are required or appropriate — this is deliberately not
a runtime-service observability surface (`uat-plan.md` "Not a product
concern here").

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-01 | E40-F01 | E40-F02 | `architecture.md#corpus-and-oracle-contract` | `tests/contracts/e40_i01_corpus_contract_test.go#TC-001` | `TestTC001_I01CorpusAndOracleContract` |
| I-03 | E40-F04 | E40-F02 | `architecture.md#run-liveness-contract` | `tests/contracts/e40_i03_liveness_contract_test.go#TC-001` | `TestTC001_I03LivenessContract` |
| I-02 | E40-F02 | E40-F03 | `architecture.md#metric-collection-and-artifact-schema` | `tests/contracts/e40_i02_artifact_contract_test.go#TC-001` | `TestTC001_I02ArtifactContract` (new) |

**I-01 (consumes).** `spec.md`'s Cross-feature interactions section already
copies the shape source and contract-test pointer verbatim from F01's
`spec.md`, per the interaction map's "Sequencing note on I-01." This test
plan reuses `TestTC001_I01CorpusAndOracleContract` **as-is — no twin test —**
and F02's own **real caller path** for I-01 is `TC-014a`/`TC-014b`/`TC-014d`
(`run-one.sh` actually invoking `bench/scripts/checkout-fixture.sh` and
reading real `corpus.yaml` item fields to seed entities) plus `TC-015h`
(reading `corpus.yaml`-derived ledgers via `diff-ledgers.sh`). Gate mode:
`live` (`E40-interaction-map.md`). This discharges F02's own consumer-side
`I-01: consumes` task obligation the map records.

**I-03 (consumes).** Same treatment: shape source and contract-test pointer
copied verbatim from F04's `spec.md`. `TestTC001_I03LivenessContract`
is reused as-is. F02's **real caller path** for I-03 is `TC-015b`/`TC-015f`
(reading the stderr NDJSON stream / `run.log` for `run_id` resolution and
stalled-stage attribution) and `TC-014e`/`TC-016a` (a real `shark run`
invocation under the PATH-stubbed-`claude` technique genuinely produces a
real liveness stream `run-one.sh` genuinely reads). Gate mode: `live`. Note
from the interaction map's "Sequencing note on I-03": the fallback (DB
status + highest-numbered transcript) satisfies UAT-05 alone, but **not**
transcript *location* — `TC-015f` tests the `run_id_source:
"fallback_newest_dir"` path specifically for that reason (ADR-F02-02),
distinct from the stalled-stage-attribution fallback UAT-05 covers.

**I-02 (produces).** New contract test:
`tests/contracts/e40_i02_artifact_contract_test.go`, function
`TestTC001_I02ArtifactContract`, package `contracts` (matching
`tests/contracts/e40_i01_corpus_contract_test.go`'s and
`e40_i03_liveness_contract_test.go`'s naming convention exactly, so E40-F03
can find it by the pointer `tests/contracts/e40_i02_artifact_contract_test.go#TC-001`,
not by searching the file). It reads **two** committed golden records via
`os.ReadFile` — `tests/contracts/testdata/e40_i02_golden_record.jsonl` (a
`completed` happy path) and
`tests/contracts/testdata/e40_i02_golden_record_timeout.jsonl` (a
`timeout` shape, per "Schema completeness gap" above) — as two `t.Run`
subtests, asserting: required blocks present, **conditionally** on
`outcome` (`runresult.*` required when `outcome != "timeout"`, absent when
it is; `timeout_detail` required exactly when `outcome == "timeout"`, absent
otherwise); `schema_version` is a value the validator supports; `outcome`
is within the closed six-value set (`completed`/`paused`/`failed`/
`already_terminal`/`no_action`/`timeout`); every `errors[].kind` is within
its own closed six-value set (`envelope_parse_error`/`stage_join_error`/
`transcript_missing`/`crosscheck_disagreement`/`postrun_check_aborted`/
`usage_unavailable`) and every entry carries non-empty `kind` and `detail`;
every metric family named in `sources` has a value from the closed
five-value set `runresult`/`transcript`/`scratch_db`/`postrun`/`liveness`
(extended from `spec.md`'s four, per the gap above). It is CI-safe per
REQ-N-001/AC-15: no submodule, no scratch project, no network, no API
spend — the same CI-safety property F01's ADR-F01-05 pins for its own
TC-001. **Consumer-side mirror obligation:** E40-F03 has no task yet; per
`spec.md`'s own note, the obligation to declare `I-02: consumes` and reuse
this exact function is recorded on E40-F03, not discharged here.

## Cross-epic integration tests (X-##)

| X-## | Producer | Consumer | Contract / shape source | Owning feature | Test coverage pointer | TC |
|---|---|---|---|---|---|---|
| X-07 | E22 (E22-F08) | E40 (E40-F02) | `architecture.md#run-lifecycle-and-isolation-contract` / `#metric-collection-and-artifact-schema`; `internal/runner/controller.go` `RunResult`/`StageLog`; `internal/runner/transcript.go` byte format | E40-F02 Bench harness: run driver and metric collection | E40 uat-plan.md X-07 canary scenario; UAT-01, UAT-07 | TC-016a, TC-016b |

The **Owning feature** and **Test coverage pointer** cells above are copied
verbatim from `spec.md`'s "Cross-epic integrations" section (itself mirroring
`E40-cross-epic-map.md` and `docs/product/cross-epic-integration-map.md`
verbatim — cross-checked directly against
`docs/product/cross-epic-integration-map.md` line 21, unaltered there); the
**Integration purpose** and **Contract/shape source** cells are summarized
here for table width, with the full prose staying unaltered in `spec.md`
and both maps — this plan's exit gate only requires the coverage pointer to
match verbatim, and it does. `canary-runsurface.sh` (TC-016) is the
implementation of this row — a harness preflight, deliberately not a
`make test` resident (ADR-F02-10), since it needs a scratch project and a
dispatch. It **is** registered in `bench/scripts/tests/run-all.sh`, per
`spec.md`'s own component-changes table instruction ("Register the three
new tests") and F01's own precedent: F01's `run-all.sh` already mixes
Tier-1b scripts with Tier-2 scripts that need the submodule (`tc004`
through `tc011`, `tc013`) — "Tier" governs whether root `make test` gates a
script, not whether `run-all.sh` includes it. `run-all.sh` remains the
curator's single entry point for the full bench self-test suite; `TC-016`
requires the same submodule/scratch preconditions `tc004`-class scripts
already require, and `bench/README.md`'s Tier 2 curator sequence documents
that precondition, matching how F01 documents it for its own Tier 2
scripts — it is not an *alternative* to registration, it is what makes
registration meaningful. **No X-08 or X-09 row belongs to F02** — `spec.md`
states this explicitly ("F02 produces no X-08 row... X-09 stays proposed
and Phase 2"); this plan does not invent one.

## Integration Scenarios

| Scenario | Boundary | Epic UAT contribution | Test evidence |
|---|---|---|---|
| Corpus (F01) → driver seeding and checkout (I-01) | `bench/corpus/*` file shape → `run-one.sh`'s real reader | UAT-01, UAT-02 (transitively, via a run that only starts from an admitted item) | TC-014a, TC-014b, TC-014d |
| Liveness stream (F04) → run_id resolution and timeout attribution (I-03) | stderr NDJSON / `run.log` → `collect-run.sh`'s real reader | UAT-05, UAT-06 | TC-015b, TC-015f, TC-014e |
| Run driver + collector → F03's aggregation input (I-02) | `record.jsonl` shape → not-yet-built F03 aggregator | UAT-01 (batch report), UAT-07 (replay) | TC-001, TC-015 (all sub-cases), TC-014a |
| `shark run` surface (E22, X-07) → collector's parse assumptions | `RunResult`/`StageLog`/transcript byte format → `canary-runsurface.sh` | UAT-01, UAT-07 (via the canary's "fail loud, not silently wrong" guarantee) | TC-016a, TC-016b |
| F01's ledger-diff tooling → F02's oracle/quality blocks | `diff-ledgers.sh` stdout → `collect-run.sh`'s copy-not-recompute rule | I-02 scenario in `uat-plan.md` ("a record missing a metric family fails aggregation loudly") | TC-015h, TC-015i |
| Scratch DB (`work_sessions`/`entity_history`) → rejection crosscheck | entity-generic DB schema → `collect-run.sh`'s key→id resolution | G4 (complete metrics) | TC-015g |
| Live repo/DB → harness isolation boundary | scratch provisioning + fixture checkout vs. the live repo | Non-functional evidence: "the live repo, its `.sharkconfig.json`, and the live database are untouched" | TC-014b, TC-014d |

Two verification-plan-style items are **not** test cases, mirroring F01's
own plan's treatment of its analogous requirements:

- **REQ-N-002's guardrail-hook clause** ("the shark-config guardrail hook
  stays satisfied") is a Claude-Code-harness-level `PreToolUse` hook that
  protects interactive dev sessions against accidentally running `shark
  admin init`/`cloud init` in this repo; it has no runtime role inside
  `bench/scripts/run-one.sh`'s own execution path (which is a plain bash
  script, not a Claude Code tool invocation). Verified by code review
  (grep `bench/scripts/run-one.sh` for any such invocation; none is a pass
  condition), not by an automated test — matching F01's own REQ-NF-003
  treatment.
- **`bench/README.md`'s new section content** (AC-17's documentation half)
  is a content-only check: the section exists, is legible, and its stated
  field names agree with the record's `manifest.model_id_source` values —
  no decision-table or mutation test simulates the meaning of the prose
  itself.

## Caller-Path Contracts (Step 5.8)

Every runtime test case below has deterministic runtime behavior (bash
scripts and one Go test executing real subprocesses, real file I/O, or real
parsing against real or synthetic files) — `content-only` applies only to
`TC-017`'s documentation-content half.

| TC | Production entrypoint (exact invocation) | Lowest allowed mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `TestTC001_I02ArtifactContract` (the contract test function itself) calling `os.ReadFile` + real JSON unmarshal against `tests/contracts/testdata/e40_i02_golden_record.jsonl` | Real filesystem read of the committed golden record, real parser | Do not substitute an in-memory struct for the golden record; must parse the real committed file (same convention as F01's TC-001) | A validator built against a hand-built in-memory record would stay green even if the real committed golden record were malformed or drifted from the schema |
| TC-014 (a–d, g) | `bench/scripts/run-one.sh --item <id> --variant <id> --rep <n> --timeout <s> --out <dir> [--skip-canary]` | The `shark` executable resolved ahead of the real binary on `PATH` for the duration of the test only (a stub script dispatching on subcommand: `create`/`run`), mirroring TC-011's `go`/`golangci-lint` PATH-stub technique; `scripts/shark-scratch-env.sh` and its real `admin init` run for real (fast, no LLM call) | Do not stub or bypass `run-one.sh`'s own bash control flow (phase ordering, timeout/process-group handling, artifact directory construction, `meta.json` writing) — only the external `shark` binary invocation is replaced | An implementation that hardcodes phase ordering assumptions instead of actually sequencing provision→seed→invoke→postrun would still pass a test that stubs too high (e.g. stubbing `run-one.sh` itself); stubbing only `shark` keeps the driver's own logic on the hook |
| TC-014e/g specifically | Same entrypoint, stub `shark run` that ignores `SIGTERM` (TC-014e) / stub `canary-runsurface.sh` first on `PATH` (TC-014g) | Real process-group signaling (`setsid`/`kill -TERM -<pgid>`) issued by `run-one.sh` itself, not simulated in the test harness | Do not simulate "the process died" by having the test harness kill it directly — `run-one.sh`'s own timeout/cap logic must issue the kill | A `run-one.sh` that only signals its direct child (not the process group) would leave TC-014e's grandchild process alive after the cap — this is the exact defect ADR-F02-03 exists to prevent |
| TC-014f specifically | Same entrypoint, stub `shark run` returns `completed` with zero code changes | Real `bench/scripts/build-ledgers.sh`, `bench/scripts/diff-ledgers.sh`, and real `git diff --numstat` executing against the real fixture checkout | Do not stub the measurement tools (`build-ledgers.sh`/`diff-ledgers.sh`/`git diff`) — the ordering property this AC guards (measure, *then* inject) is only provable if they genuinely run against the checkout's real on-disk state at each point in time, not a canned "0" the test hardcoded | A test that stubs the measurement tools to always return zero would pass regardless of whether `run-one.sh` actually measured before or after injection — the property under test would be unfalsifiable |
| TC-015 (a–l) | `bench/scripts/collect-run.sh --run-dir <dir>` against committed synthetic run directories under `bench/scripts/testdata/run/` (plus, for TC-015g, a temp SQLite DB seeded from a committed `.sql` fixture) | Real filesystem reads of the synthetic run directory; real SQLite read for TC-015g; real invocation of `diff-ledgers.sh` to *produce* TC-015h's fixture files (not to re-verify them at collector-test time) | Do not hand-type `record.oracle`/`record.quality` field values from what the collector is expected to produce — TC-015h's fixtures must come from a real `diff-ledgers.sh` run; do not stub SQLite reads for TC-015g — must query a real (temp, disposable) SQLite file | A collector that recomputes its own diff instead of copying `diff-ledgers.sh`'s stdout could still pass a test whose fixture was hand-typed to match the collector's own (possibly wrong) arithmetic — sourcing the fixture from a real `diff-ledgers.sh` run closes that hole |
| TC-015m specifically | Same entrypoint, a run directory whose `transcripts/` file at the expected count- and component-correct position is a genuine zero-byte file (not a missing file) | Real filesystem `stat`/read of a real zero-byte fixture file | Do not simulate "empty file" as a missing file (that's TC-015e's scenario) — the file must exist and be openable, just contain no bytes | An implementation that only checks file *existence* (not readability/non-emptiness) before attempting to parse would either crash or silently treat a zero-byte file as a `stage_join_error` (wrong kind) instead of `transcript_missing` |
| TC-016a/b | `bench/scripts/canary-runsurface.sh [--corpus <corpus.yaml>]` | Real `RunController`/`LivenessRecorder`/transcript-writer code path via a real `shark run` invocation in the canary's own throwaway scratch project (provisioned by the canary itself, per AC-13's row above); PATH-stubbed `claude` binary only | Do not stub `shark run` itself (unlike TC-014) — the canary's whole purpose is asserting the *real* controller's output shape, so only the outermost LLM CLI binary may be replaced; do not hardcode the expected field list from memory — TC-016 derives "the current shape" from a live invocation's actual JSON keys, and TC-016b's mutated-fixture path is a **separate, explicit** fixture, never the live output edited in place | A canary that hardcodes an expected-field list instead of deriving it from a live invocation would stay green even after `controller.go` silently renamed a field it happens to agree with by coincidence |
| TC-017 | Renderer: direct read of `bench/README.md`'s "Run driver and artifact schema" section | N/A — content-only | N/A — content-only | A section that never gets written, or that states a field name inconsistent with what `bench/scripts/collect-run.sh` actually emits, is exactly what this content-only check catches |

**Implementation hook:** the developer's red-phase tests must drive the
listed entrypoint with the listed argument shape; a test that mocks above
the entrypoint (e.g., a Go unit test that hand-builds a `record` struct
instead of running `collect-run.sh` as a subprocess against a real
directory) is rejected at code review, per REQ-N-006's expectation that
these are real bash self-tests, not simulations of bash behavior in Go.

## Test Infrastructure

### Existing patterns to reuse (with file paths)

- **`tests/contracts/e40_i01_corpus_contract_test.go`** — `package
  contracts`, `TestTC001_...` naming, repo-root-relative artifact reading
  (`filepath.Abs(filepath.Join("..", ".."))` then `os.ReadFile`), and the
  negative-subtest pattern of a hand-authored malformed-fixture Go string
  constant (`e40BrokenManifestYAML`, `e40BrokenToolchainManifestYAML`) fed
  through the *same* real parser as the happy path. `TC-001`
  (`TestTC001_I02ArtifactContract`) follows this exactly: same package,
  same naming convention, same repo-root-relative read helper, and the
  same "malformed fixture through the real parser" technique for its own
  negative subtests (missing `schema_version`, bad `outcome`, etc.) instead
  of a second hand-rolled validator.
- **`tests/contracts/e40_i03_liveness_contract_test.go`** —
  `TestTC001_I03LivenessContract` establishes the exact NDJSON field names
  (`ts`, `run_id`, `event`, `entity_key`, `iteration`, `status`, `action`,
  `agent_type`, `provider`, `stage_elapsed_ms`, `total_elapsed_ms`) and the
  `run.log: <path>` stderr announcement line F02's collector parses;
  TC-015b/TC-015f's synthetic `stderr.ndjson`/`run.log` fixtures must match
  this schema exactly, not an approximation of it.
- **`bench/scripts/tests/tc011_toolchain_guard_test.sh`** — the PATH-stub
  pattern for external binaries (a real, executable stub placed first on
  `PATH`, forwarding every call to the real tool except the one axis under
  test). TC-014's `shark` stub and TC-016's `claude` stub reuse this
  pattern precisely, not a Go-level interface mock — these are bash
  scripts invoking real subprocesses, so the mock seam has to be a real
  executable on `PATH`, exactly as TC-011 already establishes for
  `go`/`golangci-lint`.
- **`bench/scripts/tests/run-all.sh`** — registration pattern (`tests=(...)`
  array, `PASS`/`FAIL` per script, non-zero overall exit on any failure).
  `TC-001` is **not** added here (it is a Go test under `make test`, per
  F01's own precedent of keeping its Go contract test outside this bash
  wrapper); `TC-014`, `TC-015`, and `TC-016` are **all** added, per
  `spec.md`'s literal "Register the three new tests" instruction and F01's
  own precedent of mixing Tier-1b and Tier-2 scripts in the same
  `run-all.sh` (its `tests=(...)` array already includes `tc004`-class
  scripts that need the submodule). `TC-016`'s scratch-project/dispatch
  precondition is documented in `bench/README.md`'s Tier 2 curator
  sequence, the same place F01 documents its own Tier 2 scripts'
  preconditions — registration and documented preconditions are not
  alternatives to each other.
- **`bench/scripts/testdata/{lint,test}/*.json`** — committed synthetic
  ledger convention (small, hand-authored, shaped like real
  `build-ledgers.sh` output). TC-015h's `post/test-diff.json`/
  `post/lint-diff.json` mirror this, except they are generated by a real
  `diff-ledgers.sh` invocation over hand-authored base/post ledgers rather
  than hand-typed directly, per that TC's Caller-Path Contract.

### New test infrastructure needed (this feature's own deliverables)

- **`tests/contracts/e40_i02_artifact_contract_test.go`** — new Go file,
  `TestTC001_I02ArtifactContract` plus its malformed-fixture negative
  subtests (as Go string constants, mirroring `e40BrokenManifestYAML`).
- **`tests/contracts/testdata/e40_i02_golden_record.jsonl`** — one
  committed, hand-authored, realistic I-02 record (a `task` item,
  `completed` outcome, one `advance_status` stage plus two `spawn_agent`
  stages, all six metric families populated, `errors[]` empty) — the
  primary golden record TC-001 validates.
- **`tests/contracts/testdata/e40_i02_golden_record_timeout.jsonl`** —
  second committed golden record (a `timeout` outcome shape: `runresult.*`
  absent, `timeout_detail` populated), per "Schema completeness gap"
  above — proves the schema validator's conditional-required rules, not
  just the happy path.
- **A PATH-stub `shark` executable** (bash, dispatching on `$1`/`$2`
  subcommand: `create feature`/`create task`/`create bug`/`run`),
  env-var-configurable per test scenario (outcome, exit status, whether it
  ignores `SIGTERM`, whether it forks a grandchild, argv logging) — drives
  every `TC-014` sub-case. This requires `run-one.sh` to resolve `shark`
  via `PATH` (or a single overridable variable) rather than a hardcoded
  path to the `shark-scratch-env.sh`-copied binary, so the test can
  substitute it — a testability precondition task decomposition must
  honor, consistent with how `admit.sh`/`diff-ledgers.sh` already resolve
  `go`/`golangci-lint` via `PATH` for TC-011's technique.
- **A PATH-stub `claude` executable** (bash, emitting a fixed valid JSON
  envelope on `--output-format json`, or a mutation of one) — drives
  `TC-016a`/`TC-016b` and (once REQ-F-021 closes) any envelope-shape
  sub-cases inside `TC-015` that need a genuinely-written transcript rather
  than a hand-typed one.
- **`bench/scripts/testdata/run/`** — synthetic run directories driving
  `TC-015`, one subdirectory per sub-case (e.g. `clean-completed/`,
  `clean-paused-question/`, `clean-already-terminal/`, `clean-no-action/`,
  `timeout-with-liveness/`, `timeout-no-liveness-db-fallback/`,
  `missing-envelope-field/`, `missing-transcript-count-mismatch/`,
  `missing-transcript-zero-byte/`, `advance-status-only/`,
  `no-liveness-stream/`, `db-crosscheck-disagree/`, `toolchain-mismatch/`,
  `loc-numstat-mixed/`), each containing the run directory layout
  `spec.md`'s Interface contracts section defines (`run/stdout.json`,
  `run/stderr.ndjson`, `run/run.log`, `run/transcripts/*.log`, `post/*`,
  `meta.json`) as applicable to that sub-case. `loc-numstat-mixed/`'s
  `post/numstat.txt` (TC-015l) carries concrete, non-zero mixed lines —
  e.g. `12\t3\tpkg/pricing/pricing.go` (production),
  `40\t0\tpkg/pricing/pricing_test.go` (test), and `-\t-\ttestdata/blob.bin`
  (binary, no line counts) — so `loc.prod_added`/`.test_added`/
  `.files_touched` are asserted against exact non-zero values, with the
  negative case being a `_test.go` file's lines miscounted into
  `prod_added`.
- **A committed `.sql` fixture** (not a binary `.db` file, for
  diffability and schema-version resilience) seeding `work_sessions` and
  `entity_history` rows for TC-015g's crosscheck-disagreement case, loaded
  into a fresh temp SQLite file at test time via `sqlite3 <tmp>.db <
  fixture.sql`.
- **`bench/README.md` "Run driver and artifact schema" section** — new,
  per `spec.md`'s component-changes table: the run directory layout, the
  I-02 record schema field reference, and the Q003-confirmed envelope
  field names (pending REQ-F-021's capture) — drives TC-017.
- **`bench/scripts/tests/run-all.sh`** — modified to register
  `tc014_run_one_smoke_test.sh`, `tc015_collect_run_record_test.sh`, **and**
  `tc016_canary_runsurface_test.sh` (all three, per `spec.md`'s component
  table and the resolution above).

### Test tiers (mirrors F01's own tiering, extended)

| Tier | Runs | Needs submodule/scratch project? | Where |
|---|---|---|---|
| Tier 1 | `make test` (CI + every dev machine) | No | `tests/contracts/e40_i02_artifact_contract_test.go` (TC-001) |
| Tier 1b | Curator, manually or via `bench/scripts/tests/run-all.sh` | No — synthetic fixtures + PATH-stubbed `shark`/`claude` only | `tc014_run_one_smoke_test.sh`, `tc015_collect_run_record_test.sh` |
| Tier 2 | Curator, via `bench/scripts/tests/run-all.sh`, before every corpus/harness release | Yes — real scratch project + real `shark run` dispatch (stubbed `claude` only) | `tc016_canary_runsurface_test.sh` (TC-016), registered in `run-all.sh` alongside Tier 1b, with its submodule/scratch precondition documented in `bench/README.md`'s Tier 2 curator sequence (mirroring F01's own `tc004`-class scripts) |

---

## Red-team review (Step 7.5)

The dispatch prompt's codex path was unavailable (hard usage limit until
2026-08-07 23:30, per the parent-loop's environment notes); the owner
authorized `gemini-3.1-pro-high` (via `agy --model gemini-3.1-pro-high
--add-dir /home/jwwel/projects/shark-task-manager --print-timeout 20m`) as
the substitute independent red-team reviewer, run against the **revised**
draft (after the internal advisor pass below had already patched the
schema-completeness, run-all.sh-registration, LOC-arithmetic, AC-11/Turso,
TC-014f-seam, TC-016-provisioning, and errors[].kind-closed-set gaps found
before spending the external review budget).

**Verdict: PASS** (`gemini-3.1-pro-high`)

**Checklist results (1–7):** all seven Step-7.5 checks (open-endedness,
ISTQB technique fit, enumeration completeness, ISO 25010 coverage,
observability design, negative cases, Caller-Path Contract/seam
plausibility) returned PASS, with the PATH-stub `claude`/`shark` seam
explicitly confirmed plausible against `internal/runner/claude_dispatcher.go`'s
real `exec.LookPath("claude")` resolution.

**Specific verification (A–E), all VERIFIED:**
- **A.** The `timeout_detail` field and the extended five-value `sources`
  enum are self-consistent across the "Schema completeness gap" section,
  the AC-03 row, the I-02 "produces" section, and the new-test-infrastructure
  deliverables list.
- **B.** `run-all.sh` registration of all three new self-tests
  (`tc014`/`tc015`/`tc016`) matches `spec.md`'s literal "Register the three
  new tests" instruction.
- **C.** `TestTC001_I01CorpusAndOracleContract` and
  `TestTC001_I03LivenessContract` are cited with function names verified
  verbatim against the real files.
- **D.** The X-07 row's "Owning feature" and "Test coverage pointer" cells
  match `E40-cross-epic-map.md` line 13 verbatim (the scoped verbatim claim
  this plan makes, not the abbreviated prose cells).
- **E.** No other AC's required output was found missing from the I-02
  schema as currently described.

**Issues raised:** 2 (both NIT, not BLOCKER/CONCERN)
**Issues addressed before development:** 2 — both were already recorded as
explicit, owned deferrals in the draft gemini reviewed, not new findings:
1. *`spec.md`'s Data model table hasn't been edited yet to add
   `timeout_detail`/`liveness`.* Already the exact Owner/Trigger this plan's
   "Schema completeness gap" section records ("the E40-F02 spec/architecture
   owner should still add [it] to `spec.md`'s Data model changes table
   before the envelope-parser and collector tasks are dispatched... until
   that lands, this section is the authoritative source").
2. *Q003/REQ-F-021's fixture field names are provisional pending live
   capture.* Already the exact dependency this plan's "Sequencing
   constraint: REQ-F-021 / Q003" section states as a "BLOCKER-class
   dependency for implementation... task decomposition must not generate
   the envelope-parser task ahead of a task that performs the REQ-F-021
   capture."

**Issues deferred:** 0 net-new — the two NITs above restate deferrals this
plan already owns explicitly with a named owner/trigger; no additional
deferral was needed.

## Recommendations

- [x] **Ready for development** — every AC in `spec.md` has a named test
  case, ISTQB technique, ISO 25010 row, and Caller-Path Contract; every
  declared I-## has a shared or new contract test matching its declared
  pointer verbatim (I-01, I-03) or a new one this plan fully designs
  (I-02); the one X-## row belongs entirely to this feature and has test
  coverage (TC-016a/b). The one schema gap this plan's own analysis found
  (AC-03's missing `timeout_detail`/`sources.liveness`) is resolved
  directly, not left open, with an explicit owner/trigger for the matching
  `spec.md` edit. The one genuinely open item (REQ-F-021/Q003's exact
  envelope field names) is a task-sequencing dependency this plan states
  explicitly, not an ambiguity in what to test. Independent red-team
  (`gemini-3.1-pro-high`) returned PASS with zero unresolved BLOCKER/CONCERN
  findings.
- [ ] Needs BA refinement.
- [ ] Needs tech refinement.

## Verdict

**APPROVED.**

---

*Last updated*: 2026-08-06
