---
feature_key: E40-F04-shark-run-live-progress-and-per-run-log
epic_key: E40
type: test-plan
tier: STANDARD
date: 2026-08-06
---

# Test Plan: E40-F04 — `shark run` live progress and per-run log

**Created:** 2026-08-06
**Feature spec:** [spec.md](spec.md)
**Research report:** [research-report.md](research-report.md)
**Parent epic UAT plan:** [../uat-plan.md](../uat-plan.md)
**Status:** APPROVED (see §"Codex Test-Plan Red-Team" and §"Recommendations")

This is feature-level test planning — no task specs exist yet for E40-F04.
`spec.md`'s Acceptance Criteria table (AC-01..AC-10) is the AC list. There is
no sibling task spec to diff against, so the task-spec-drift steps of the
standard workflow (Step 3, task-spec-specific parts of Step 4) are skipped per
dispatch instructions; this plan instead diffs `spec.md` against the parent
epic PRD (`epic.md`) and architecture (`architecture.md`).

---

## Spec Drift Analysis

### Drift findings

None found. `spec.md` traces every REQ/AC to epic criterion **G3** ("Runs are
observable while in flight" — `epic.md` §2, row G3: "For every run, per-stage
progress is visible during execution (stderr events in `--json` mode) and a
per-run log exists at `.shark/runs/<run_id>/run.log`"). The spec's stderr-NDJSON
+ stage-scoped-heartbeat + unconditional-per-run-log shape is a direct,
non-narrowed, non-expanded restatement of G3. `architecture.md` §"Run liveness
contract" describes the identical mechanism (ticker/Progress-callback gate
defect, stderr NDJSON, unconditional `run.log`) with no field the spec omits or
adds. The two "Added" out-of-scope carve-outs in spec.md (`StageLog.ExitCode`
in the stream; `RunOptions.Verbose` wiring) are architecture-consistent
scope-narrowing decisions recorded with rationale, not undocumented drift.

One decision *is* new relative to the epic-level research report: spec.md
closes research Decision 3 (fix `run.go:275`'s stdout leak vs. carve it out)
in favor of "fix it" (D6 edit 4). This is a scope addition over the epic
report's open question, but it is resolved *in* spec.md with rationale ("REQ-F-003
is a general `shark run` invariant"), not silently introduced — recorded here
for traceability, not as unresolved drift.

### Traceability matrix

| Feature PRD requirement (epic.md G3 / architecture.md) | Task/feature acceptance criterion | Covered? | Notes |
|---|---|---|---|
| Per-stage progress visible during execution, stderr in `--json` mode | REQ-F-001, REQ-F-002, AC-01 | Yes | |
| stdout stays exactly one `RunResult` document | REQ-F-003, AC-02 | Yes | Also X-08's cross-epic promise |
| Plain-mode liveness lines | REQ-F-004, AC-04 | Yes | |
| Correct entity attribution during cascade | REQ-F-005, AC-05 | Yes | |
| Heartbeat ≥ every 10s | REQ-F-006, REQ-N-001, AC-01 | Yes | Vocabulary section prevents conflation with claim/lease heartbeat |
| Unconditional per-run log at `.shark/runs/<run_id>/run.log` | REQ-F-007, AC-06 | Yes | |
| Log path printed once at run start | REQ-F-008, AC-10 | Yes | |
| Stage pairing integrity | REQ-F-009, AC-08 | Yes | |
| Durability across `SIGKILL` | REQ-F-010, AC-07 | Yes | Flush-free-write proxy, no signal delivery needed |
| Fail-soft sinks | REQ-F-011, REQ-N-002, AC-09 | Yes | |
| No change to `RunResult`/`StageLog`/transcript format | REQ-N-004 | Yes (by inspection, not a new test — see below) | |
| Thread safety | REQ-N-003 | Yes | |
| Permissions / no sensitive-field leakage | REQ-N-005 | Yes | |
| I-03 (F04→F02 liveness contract) | Cross-feature interactions §, TC-001 | Yes | Producer side only; F02 not yet decomposed (sequencing, not a gap) |
| X-08 (F04→E22 stdout-purity obligation) | Cross-epic integrations §, TC-002 | Yes | |

### REQ-N-004: deliberately not a new test

D1's own rationale is that REQ-N-004 is "provable by inspection" because F04
touches zero lines of `internal/runner/controller.go`. The codebase's actual
non-regression proof for the `RunResult`/`StageLog`/transcript byte format is
X-07's canary (owned by E40-F02 / E22-F08, ADR-004) — writing a second,
F04-local struct-shape assertion here would be a twin test against the same
surface X-07 already pins. This test plan's verification mechanism for
REQ-N-004 is **code review**: confirm the F04 diff touches no line of
`controller.go`, `transcript.go`'s existing functions, or the `RunResult`/
`StageLog` type definitions. Made concrete (not just "someone looks at it"):
code review runs `git diff --stat <base>..<F04-branch> -- internal/runner/controller.go internal/runner/transcript.go`
and requires it to report **zero changed lines** in those two files (F04 adds
`internal/runner/liveness.go` as a new file and touches `internal/cli/commands/run.go`
only). This is recorded as a deliberate decision, not an omission — writing a
struct-shape assertion here would be a twin test against X-07's canary
(owned by E40-F02/E22-F08, ADR-004), which already pins this exact surface.

---

## Ambiguity findings

None of the ten acceptance criteria are open-ended robustness assertions —
each names a concrete, enumerable behavior (a schema, a pairing rule, a fail-soft
guarantee under named fault classes). No AC required return-to-refinement under
the Step 5 checklist.

One implementation-shape ambiguity surfaced during test design and is recorded
as a **test-plan recommendation to task generation**, not a spec defect (see
"Recommendation: move the `LogPath()` print into `Start()`" under Test
Infrastructure below).

---

## ISTQB Technique Application (per AC)

| AC | Technique(s) applied | Test cases | Rationale |
|---|---|---|---|
| AC-01 | Equivalence Partitioning (event ∈ {stage_start, heartbeat, stage_end}); Boundary Value Analysis (10s tick boundary) | TC-011, TC-012 | Closed 3-value enum, fixed-interval timer |
| AC-02 | Contract surface enumeration (every stdout-writer call-site class, not just one function family) | TC-002 | X-08's invariant is "stdout stays exactly one document" — the surface to enumerate is every way `run.go` can write to stdout, not one literal function name |
| AC-03 | Attack-class enumeration (wrong-writer regression) | TC-013 | Same defect class as AC-02 at one more call site |
| AC-04 | Equivalence Partitioning (json vs. plain mode); Boundary Value Analysis (field-omission threshold) | TC-014 | |
| AC-05 | State Transition (parent stage → child stage → parent stage) | TC-015 | D1's single-slot state machine is explicitly a state machine |
| AC-06 | Decision Table (return-path × recorder-outcome); Attack-class enumeration (nil-capture defer trap) | TC-016, TC-017, TC-018 | |
| AC-07 | State Transition (open stage → heartbeat → read-before-close) | TC-001, TC-019 | Flush-free-write proof, per-event durability |
| AC-08 | Decision Table (reached-action × closed-by × re-dispatch); Boundary Value Analysis (duplicate composite key) | TC-020, TC-021 | Full table in §"Decision Table — AC-08" below |
| AC-09 | Boundary Value Analysis (empty-string project root); Attack-class enumeration (I/O fault injection) | TC-022, TC-023 | |
| AC-10 | State Transition (print-before-first-event ordering); Attack-class enumeration (mode-conditional regression) | TC-024, TC-025 | |
| REQ-N-001 | Contract surface enumeration (constructor signature + source-text scan) | TC-026 | |
| REQ-N-003 | N/A — race-detector stress test, not an ISTQB partitioning technique | TC-027 | Concurrency correctness is proven by `go test -race`, not partitioning |
| REQ-N-005 (perms) | Boundary Value Analysis (permission bits) | TC-028 | |
| REQ-N-005 (leakage) | Contract surface enumeration (closed JSON field set) | TC-029 | |

ACs without a technique annotation: none. Every AC and NFR above has at least
one named technique.

### Decision Table — AC-08

| # | C1: reached `action` phase? | C2: closed by | C3: cascade re-dispatch (dup `(entity_key, iteration)`)? | Expected |
|---|---|---|---|---|
| 1 | yes | next iteration, same entity | no | `stage_end` pairs with its own `stage_start` |
| 2 | no | next iteration, same entity | no | lazy `stage_start` emitted immediately before `stage_end` |
| 3 | yes | next iteration, different entity (cascade descent) | no | parent's `stage_end` fires; child's `stage_start` opens |
| 4 | no | next iteration, different entity (cascade descent) | no | parent's lazy `stage_start` + `stage_end` fire, then child's `stage_start` opens |
| 5 | yes | run end (`Finish`) | no | final `stage_end` fires at `Finish()` |
| 6 | no | run end (`Finish`) | no | lazy `stage_start` + `stage_end` fire at `Finish()` |
| 7 | yes | next iteration, same entity | **yes** (2nd cascade pass) | `stage_end` pairs to the *currently open slot*, unconfused by the earlier same-`(entity_key, iteration)` pair from pass 1 |
| 8 | **n/a — no slot was ever opened** | run end (`Finish`), zero `Observe` calls ever made | no | No stage_start/stage_end is fabricated. Row 8 has two production sub-cases (see TC-016b/TC-017 below): `Finish(result)` where `result` is a valid already-terminal/paused `RunResult` (controller returned before its first `iteration`), and `Finish(nil)` where `controller.Run` itself returned a bare Go `error` before its first `iteration` (e.g. `GetNextStatus` failing at `controller.go:359-361`) |

Rows 1, 2, 5, 6 → TC-020 (non-cascade pairing, including lazy-start). Rows 3, 4
→ TC-015 (already required for AC-05's cascade-labeling state transition; same
synthetic sequence proves both ACs). Row 7 → TC-021 (the AC's explicitly named
tricky case). **Row 8 was missing from the initial draft** — found during
self-review (codex was unavailable, see "Codex Test-Plan Red-Team" below) by
walking `controller.go`'s early-return paths against D1's own table, which
only describes what happens to an *already-open* slot at run end and is silent
on "no slot was ever opened." Two reachable production shapes exist: (a) the
entity is already terminal, or a Question block pauses it, before `Run()`'s
loop executes even once — `runResult` is non-nil (`Outcome`
`already_terminal`/`paused`) but `Stages` is empty; (b) `controller.Run` itself
returns a bare `error` (not a `RunResult`) before the first `iteration` —
`runResult` stays `nil` in `run.go`'s liveness-teardown defer (as opposed to
the `emitRunEnd` defer, which substitutes its own synthetic failed-`RunResult`
and does **not** feed the liveness recorder). Row 8 → TC-016 (sub-case b) and
TC-017 (sub-case a), both revised below to cover this explicitly rather than
leaving it as an accidental byproduct of a general "Finish(x)" call.

---

## ISO 25010 Coverage Matrix

| AC / NFR | Functional | Performance | Compat | Usability | Reliability | Security | Maintainability | Portability |
|---|---|---|---|---|---|---|---|---|
| AC-01 | ✅ TC-011,012 | ✅ TC-012 (10s cadence) | N/A | N/A | N/A | N/A | ✅ same-package white-box, no new abstraction | N/A |
| AC-02 | ✅ TC-002 | N/A | ✅ TC-002 (existing `--json` consumers unbroken) | N/A | N/A | N/A | ✅ AST guard runs in ms, no fixture | N/A |
| AC-03 | ✅ TC-013 | N/A | N/A | N/A | N/A | N/A | N/A | N/A |
| AC-04 | ✅ TC-014 | N/A | N/A | ✅ TC-014 (human-tailable, field-labeled) | N/A | N/A | N/A | N/A |
| AC-05 | ✅ TC-015 | N/A | N/A | ✅ TC-015 (operator sees the entity actually running) | N/A | N/A | N/A | N/A |
| AC-06 | ✅ TC-016,017,018 | N/A | N/A | N/A | ✅ TC-016 (recovers a log even from a paused/never-completed run) | N/A | N/A | N/A |
| AC-07 | ✅ TC-001,019 | N/A | N/A | N/A | ✅ TC-001,019 (SIGKILL-survival proxy) | N/A | N/A | N/A |
| AC-08 | ✅ TC-020,021 | N/A | N/A | N/A | ✅ TC-021 (doesn't corrupt under a documented non-unique-key scenario) | N/A | N/A | N/A |
| AC-09 | ✅ TC-022,023 | N/A | N/A | N/A | ✅ TC-022,023 (fail-soft under named fault classes) | N/A | ✅ TC-023 (isolated fault injection, no shared fixture state) | N/A |
| AC-10 | ✅ TC-024,025 | N/A | N/A | ✅ TC-024 (operator knows what to tail) | N/A | N/A | N/A | N/A |
| REQ-N-001 | N/A | ✅ TC-026 (interval provably fixed) | N/A | N/A | N/A | N/A | N/A | N/A |
| REQ-N-002 | N/A | N/A | N/A | N/A | ✅ TC-023 (slog.Debug evidence) | N/A | N/A | N/A |
| REQ-N-003 | N/A | N/A | N/A | N/A | ✅ TC-027 (`-race` clean under concurrent Observe + tick) | N/A | N/A | N/A |
| REQ-N-004 | N/A | N/A | ✅ code review, not a test (see rationale above) | N/A | N/A | N/A | N/A | N/A |
| REQ-N-005 | N/A | N/A | N/A | N/A | N/A | ✅ TC-028 (perm bits), TC-029 (no-leakage) | N/A | ✅ TC-028 (Windows-skip pattern, matches `transcript_test.go`) |

No blank cells. `N/A` cells are justified inline in the row (e.g., a display
schema has no Performance dimension beyond the fixed-cadence check already
covered under Functional/Performance for AC-01).

### Coverage gaps

None identified. Every AC has at least Functional coverage, and every
characteristic that plausibly applies to a display/logging feature (Reliability
under fault injection, Security under field-leakage, Maintainability under
white-box test cost) is covered. Usability is covered narrowly (field
labeling, tailability) because this is a CLI/log feature, not a UI — a fuller
usability pass belongs to UAT-06's manual/exploratory verification, not this
plan's automated suite.

---

## Observability Design (per behavior)

| Behavior | Metric | Log | Trace span | Alert threshold | Test assertion |
|---|---|---|---|---|---|
| Liveness stream itself (`stage_start`/`heartbeat`/`stage_end` on stderr, `run.log`) | N/A — the events *are* the observability artifact for `shark run`, not a thing needing its own meta-metric | N/A — same reason | N/A | N/A | TC-011, TC-014, TC-015, TC-020, TC-021 assert the artifact's own content directly |
| File sink failure (unwritable path, `EACCES`, disk full) | N/A | `slog.Debug` matching `internal/observability/file_jsonl_exporter.go`'s pattern (`"...: failed to ..."`, path + err attrs) | N/A | N/A | TC-023 asserts a `slog.Debug` record is emitted on write failure |
| `run.log` path announcement | N/A | The printed stderr line itself is the operator-facing signal (REQ-F-008) | N/A | N/A | TC-024 |

Rules applied: every behavior has at least one of metric/log/trace or an
explicit `internal — no observability` justification. The liveness stream and
the path announcement are marked "the artifact is its own evidence" rather than
`internal — no observability`, because they are not internal — they are the
feature's entire external contract; a "log the log" meta-instrumentation would
be redundant with what TC-011/014/015/020/021/024 already assert directly. The
one case needing separate instrumentation (sink failure) already has a
directly reusable precedent (`file_jsonl_exporter.go`'s `slog.Debug` pattern)
that D5/Decision 4 already commits the implementation to reusing.

**Implementation hook:** TC-023's `slog.Debug` assertion becomes a hard
requirement on the task spec — the `LivenessRecorder`'s fail-soft write path
must call `slog.Debug` (not silently `return nil`) on every sink write failure,
matching `file_jsonl_exporter.go`'s convention exactly.

---

## Caller-Path Contracts (Step 5.8, full table)

| TC | Production entrypoint | Lowest mock seam | Forbidden mocks | Counter-factual |
|---|---|---|---|---|
| TC-001 | `runner.RunController.Run(ctx, key, RunOptions{Progress: rec.Observe, ProjectRoot: t.TempDir(), RunID: ...})` — D7's exact mechanism, stub transitioner/action-service/dispatcher fixtures from `controller_test.go` | `MockTransitioner`, `MockActionService`, `MockDispatcher`, `MockCascadeChildrenService` (all already exist in `controller_test.go`) | Do not stub `LivenessRecorder` itself — it is the surface under test | An implementation that batches writes (buffers instead of open-append-write-close per event) passes every liveness_test.go unit test yet fails this test's read-before-`Finish()` assertion |
| TC-002 | Runtime half: `outputRunResult(result)` with `os.Stdout` swapped for an `os.Pipe`, `cli.GlobalConfig.JSON=true`. Source-guard half: `go/parser` over `internal/cli/commands/run.go`'s AST | Runtime half: none below `outputRunResult` — it is the entrypoint. Source-guard half: n/a (static analysis) | Do not narrow the source guard's enumerated call-site class to `fmt.Print/Printf/Println` alone (see rationale below) | A future change adds `cli.Warning(...)` to the dispatch path (E22-F08's forward obligation touches the same file); a guard scoped to `fmt.Print*` text stays green while `cli.Warning`'s `pterm.Warning.Println`/`fmt.Println` fallback pollutes stdout unconditionally (verified: `internal/cli/root.go:498-543` — `Success`/`Warning`/`Info` have no `GlobalConfig.JSON` gate, unlike `Error`) |
| TC-011 | `(*runner.LivenessRecorder).Observe(runner.RunProgress{...})` — this is the literal production callback bound as `opts.Progress = rec.Observe` (D6 edit 3); no adapter sits between controller and recorder | None — writes go through the recorder's real file/stderr sinks against `t.TempDir()` | Do not replace the recorder's internal writer with an in-memory buffer that bypasses the real `os.OpenFile`/append path — matches `transcript_test.go`'s real-file convention | A buffered/batched writer would still satisfy "eventually correct content" while violating REQ-F-010's per-event durability, which TC-019 (not TC-011) specifically catches — TC-011 alone would miss it, which is why both exist |
| TC-012 | Internal — same-package (`package runner`) call to the recorder's tick-handling method (task spec must expose one; see Test Infrastructure) | n/a | Do not make the tick interval configurable to work around a real 10s sleep — that reads as a REQ-N-001 violation ("fixed 10 seconds, independent of config") to a reviewer even if the test-only constructor path is unreachable from production | An implementation that only emits `heartbeat` from inside `Start()`'s ticker `select` loop (no separately callable method) forces either a real 10s sleep per test run or an interval parameter that risks becoming a de facto config knob |
| TC-013 | Internal/source-guard — `go/parser` AST check on the specific statement inside the `--worktree` cleanup `defer` in `run.go`, asserting its `fmt.Fprintf` call's first argument is `os.Stderr` | n/a (static analysis; the runtime half of this path is already covered system-wide by TC-002's widened guard) | n/a | An implementation that reverts to `fmt.Printf` (dropping `Fprintf(os.Stderr, ...)`) at this one call site would already be caught by TC-002's whole-file guard; this TC additionally proves the *specific* writer target, not just "not a bare stdout write" |
| TC-014 | `(*runner.LivenessRecorder).Observe(...)` in plain mode (`jsonMode=false`) | None | Same as TC-011 | An implementation that omits a field from the plain-mode renderer (e.g. drops `provider=`) passes AC-01's JSON-schema check yet silently degrades AC-04's human-tailable line |
| TC-015 | `(*runner.LivenessRecorder).Observe(...)` driven with a synthetic parent→child→parent `RunProgress` sequence matching `controller.go`'s actual cascade shape (`handleCascade`'s sequential `for` loop, `controller.go:600-608`) | None | Do not hand-construct already-labeled events; the sequence must go through `Observe` exactly as the controller would call it, so a labeling bug in `Observe` itself is caught | An implementation that labels heartbeats during a cascade child's stage with the *parent's* key (the exact Finding 1 defect this feature fixes) fails this test's mid-cascade assertions while still passing a non-cascade AC-01 test |
| TC-016 | `(*runner.LivenessRecorder).Finish(nil)` after driving one open, unclosed stage | None | n/a | An implementation whose `run.log` writer only fires from a fully-populated `RunResult` (nil-dereferences or early-returns on `nil`) leaves `run.log` empty for every paused/`no_action`/early-terminated run — exactly AC-06's "pause" arm |
| TC-017 | `(*runner.LivenessRecorder).Finish(&runner.RunResult{...})` | None | n/a | An implementation that never renders the `run_end` summary line, or renders it as a fourth `event` value instead of D3's file-only summary line, fails either "trailing line exists" or "not a 4th enum value" |
| TC-018 | Internal/source-guard — `go/parser` AST check on `runRun`'s teardown-defer statement | n/a | n/a | `defer rec.Finish(runResult)` (direct method-value defer, arguments evaluated at registration time, captures `nil` forever) passes every liveness_test.go test yet silently loses the run-end outcome in production — this is D6's own named trap, quoting `run.go:112-127`'s existing `emitRunEnd` pattern |
| TC-019 | `(*runner.LivenessRecorder).Observe(...)` (open a stage, emit one heartbeat) then read `run.log` from disk via `os.ReadFile` **before** calling `Finish()` | None | Do not call `Finish()` before the disk read — that would silently convert this into a testing an entirely different (flush-at-end) property | An implementation using `bufio.Writer` or deferring all writes to `Finish()` passes TC-016/017 (which call `Finish`) yet fails this test, which is the actual SIGKILL proxy |
| TC-020 | `(*runner.LivenessRecorder).Observe(...)` driven through Decision Table rows 1/2/5/6 | None | n/a | An implementation that requires an explicit `action`-phase event before ever emitting `stage_start` drops the lazy-start guarantee, producing an unpaired `stage_end` for any stage that pauses/fails before dispatch |
| TC-021 | `(*runner.LivenessRecorder).Observe(...)` driven through two full cascade passes reusing the same child `entity_key` with `iteration` restarting at 1 | None | Do not use a `(entity_key, iteration)` composite map key inside the test's own verification logic — that would just re-implement the bug under test | An implementation whose internal pairing state is keyed by `(entity_key, iteration)` instead of stream-order slot identity silently merges the two passes' events, corrupting the pairing for the second pass |
| TC-022 | `runner.NewLivenessRecorder("", runID, key, jsonMode, start)` | None | n/a | An implementation that panics or returns an error on empty `projectRoot` (rather than mirroring `NewFileJSONLExporter("")`'s silent-skip) breaks every dry-run/read-only invocation outside a project root |
| TC-023 | `(*runner.LivenessRecorder).Observe(...)` with `projectRoot` pointing at a directory pre-`chmod`'d `0o555` (blocks file creation) | None | n/a | An implementation that returns an error from `Observe` (rather than swallowing it) would fail a live `shark run` invocation on a read-only filesystem — exactly REQ-F-011 |
| TC-024 | `(*runner.LivenessRecorder).Start()` (recommended relocation of the `LogPath()` print — see Test Infrastructure) or, in the AST-guard fallback, the `run.go` print statement | None (runtime half); n/a (AST fallback) | n/a | An implementation that prints the path only in plain mode (still gated by a residual `if !cli.GlobalConfig.JSON`) silently violates "in both modes" |
| TC-025 | Internal/source-guard — `go/parser` AST position check: the recorder-start statement in `runRun` (or, in the relocated design, the call to `rec.Start()`) occurs before the first preflight/lease-acquisition statement | n/a | n/a | An implementation that constructs the recorder correctly but starts/prints after `acquireRunLeaseForRunnableAction` would still pass every `liveness_test.go` test while violating "a run that pauses at a Question block still leaves a log" (D6 edit 2's own stated reason for the ordering) |
| TC-026 | Internal — `reflect` signature check on `runner.NewLivenessRecorder`, plus a source-text scan of `internal/runner/liveness.go` for `svc.TTL()` / `claim_ttl_seconds` / `DefaultClaimTTL` | n/a | n/a | An implementation that threads a TTL-derived interval into the recorder (reusing `startRunHeartbeat`'s cadence, as research Finding 5 explicitly warns against) silently violates the documented "≥10s" liveness claim at the default 15-minute TTL (~5 minute actual cadence) |
| TC-027 | `(*runner.LivenessRecorder).Observe(...)` called concurrently from N goroutines plus a live ticker, run under `go test -race ./internal/runner/...` | None | n/a | An unguarded read/write of shared slot state under `-race` is exactly REQ-N-003's named risk ("the ticker goroutine and the controller's Progress callback touch shared stage state concurrently") |
| TC-028 | `(*runner.LivenessRecorder).Observe(...)` then `os.Stat` on the written `run.log` and its parent dir | None | n/a | A `run.log` written with default `os.Create` semantics (0666 minus umask) rather than an explicit `0o644` open would drift from the documented, transcript-matching permission contract |
| TC-029 | `json.Unmarshal` each emitted stderr line into `map[string]json.RawMessage` and assert the key set is exactly D3's closed field list | None | n/a | An implementation that adds a debug field carrying, e.g., the rendered prompt or raw agent stdout would violate REQ-N-005's "carries no prompt bodies, agent output, or credentials" without any single test asserting the schema is closed |

---

## Cross-feature contract tests (I-##)

| I-## | Producer | Consumer(s) | Shape source | Contract test pointer | TC |
|---|---|---|---|---|---|
| I-03 | E40-F04 (this feature) | E40-F02 (Bench harness: run driver and metric collection) | `architecture.md#run-liveness-contract` | `tests/contracts/e40_i03_liveness_contract_test.go#TC-001` (Go function name: `TestTC001_I03LivenessContract`) | TC-001 |

- **Gate mode:** `live`, as declared in `E40-interaction-map.md` — not staged, not
  `contract-only`. The staged-edge field set (activation owner, closure key,
  review basis) is a conditional requirement for staged rows only; recorded here
  as explicitly checked and not applicable, per `E40-interaction-map.md` line
  17 ("All three rows use the default `live` gate mode. No `contract-only` row
  is declared").
- **Counterpart status, read live from Shark (2026-08-06):**
  `shark feature get E40-F02 --json --field status` → `draft`. E40-F02 has not
  entered `task_generation`; per `E40-interaction-map.md` §"Sequencing note on
  I-03" this is build-order sequencing (`execution_order` 4 vs. F04's 2), not an
  open gap, and does not change the row's `live` gate mode.
- **Shared-contract evidence:** `TestTC001_I03LivenessContract` is written by
  this feature's task generation (producer side) and reused **verbatim** —
  same function, same file — by E40-F02's own task generation when it declares
  `I-03: consumes`. No twin test is written.
- **Review basis:** `architecture.md#run-liveness-contract` (elaborated by this
  spec's D3 NDJSON field table, which is the elaboration, not a replacement,
  per spec.md's own closing note on Cross-feature interactions).

TC-001 asserts the producer side: `RunController.Run` driven in process with
stub fixtures (D7 mechanism), captured stderr matches the D3 schema, and
`run.log` read before `Finish()` already names the open stage — the durability
property F02 depends on for UAT-05 (stalled-stage attribution when stdout
never delivers a `RunResult`).

---

## Cross-epic integration tests (X-##)

| X-## | Producer | Consumer(s) | Shape / contract source | Test coverage pointer | TC |
|---|---|---|---|---|---|
| X-08 | E40-F04 (this feature) | E22 — External Orchestration Runner (no owning feature; see below) | `architecture.md#run-liveness-contract`; `internal/cli/commands/run.go` progress callback and ticker | `tests/contracts/e40_i03_liveness_contract_test.go#TC-002` (Go function name: `TestTC002_X08StdoutPurity`), plus `uat-plan.md` UAT-06 | TC-002 |

- **Activation owner:** E40-F04 itself, per both `E40-cross-epic-map.md` and
  `docs/product/cross-epic-integration-map.md` (rows verified verbatim-identical
  on 2026-08-06 — no drift between the epic-local and global maps). `shark run
  --json` stdout consumers are dispersed system-wide with no single owning E22
  feature; accountability sits with F04's own non-regression proof.
- **Forward obligation:** E22-F08 ("Simplify RunController to match
  architecture v2") must keep TC-002's stdout assertion green when it next
  changes `controller.go`'s dispatch path. TC-002's source-guard half is
  designed specifically to be **inheritable**: no agent, no scratch project, no
  database — it runs in E22-F08's own CI unchanged.
- **Status:** `assigned` (both maps).
- **Review basis:** `architecture.md#run-liveness-contract`; the CX review note
  in `E40-cross-epic-map.md` ("X-08 is the only row with a user-facing
  handoff... a long run stops being indistinguishable from a hung one").

**Widened source guard, recorded as a deliberate strengthening of D7's literal
text.** D7 specifies the source-guard half as "assert every `fmt.Print`/
`fmt.Printf`/`fmt.Println` call site lies inside `outputRunResult`." Verified
during test design (`internal/cli/root.go:454-543`) that `cli.Success`,
`cli.Warning`, and `cli.Info` write to stdout **unconditionally** — no
`GlobalConfig.JSON` gate, unlike `cli.Error` — via `pterm.*.Println` or, in
`--no-color` mode, bare `fmt.Println`. A guard scoped only to literal
`fmt.Print*` text in `run.go` would not see a future `cli.Warning(...)` call
added to the same file (exactly the shape of E22-F08's forward-obligation
change). TC-002's guard is therefore widened to also flag: `fmt.Fprint*` with
`os.Stdout`, direct `os.Stdout.Write`, and any call expression naming
`cli.Success`, `cli.Warning`, `cli.Info`, `cli.OutputTable`, or `cli.Title`
inside `run.go`. This is flagged per Rule 7 (surface conflicts, don't
average them) rather than silently narrowed to D7's literal wording — the
wider guard is what actually defends X-08's stated invariant.

---

## Acceptance Test Cases

Test cases TC-001 and TC-002 are the fixed-name cross-feature/cross-epic
contract tests (D7); all others are this plan's local IDs. All are runtime
Go tests except the explicitly marked source-guard (AST) cases, which are
static but still committed, CI-gated Go tests — not manual review steps.

### TC-001: I-03 producer shape + per-event durability

**Feature Requirement:** REQ-F-002 (NDJSON schema), REQ-F-010 (durability); I-03.
**Task Acceptance Criterion:** AC-07 (contract half).
**Technique Applied:** State Transition (open stage → heartbeat → read-before-close).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.

**Caller-Path Contract:** see table above.

**Preconditions:** Stub `MockTransitioner`/`MockActionService`/`MockDispatcher`/
`MockCascadeChildrenService` fixtures from `controller_test.go`, configured for
one entity that opens a dispatch stage and does not return before the test
reads `run.log`. `t.TempDir()` as `ProjectRoot`.

**Input:** `RunOptions{Progress: rec.Observe, ProjectRoot: <tmp>, RunID: <uuid>}`
driven through `RunController.Run` with `os.Stdout`/`os.Stderr` swapped for
`os.Pipe`s to capture stderr.

**Expected Output:** Every captured stderr line unmarshals into the D3 schema
with `event` in `{stage_start, heartbeat, stage_end}`. `<tmp>/.shark/runs/<run_id>/run.log`,
read from disk **before** `Finish()` is called, already contains the open
stage's `stage_start` and its latest `heartbeat` line naming entity, status,
agent, provider.

**Observability Evidence:** N/A — the captured stream is the assertion target itself.

**Edge Cases:**
- Stage never reaches the `action` phase before the test reads the file: lazy `stage_start` must already be present.

**Negative Cases:**
- No line on stderr fails to parse as JSON with an `event` key (I-03's consumer rule).

---

### TC-002: X-08 stdout purity (two halves)

**Feature Requirement:** REQ-F-003; X-08.
**Task Acceptance Criterion:** AC-02, AC-03 (source-guard half also covers AC-03).
**Technique Applied:** Contract surface enumeration (widened — see rationale above).
**ISO 25010 Characteristic(s):** Functional Suitability, Compatibility.

**Caller-Path Contract:** see table above.

**Preconditions (runtime half):** `cli.GlobalConfig.JSON = true`; `os.Stdout`
swapped for an `os.Pipe`.

**Input (runtime half):** A synthetic `runner.RunResult` passed to
`outputRunResult(result)`, driven over a stage sequence produced by the
recorder.

**Expected Output (runtime half):** The captured byte stream decodes via one
`json.Decoder.Decode` into exactly one `runner.RunResult`; a second `Decode`
call returns `io.EOF`.

**Input (source-guard half):** `go/parser` AST of `internal/cli/commands/run.go`.

**Expected Output (source-guard half):** Every call expression matching the
widened class (`fmt.Print`/`Printf`/`Println`; `fmt.Fprint*` targeting
`os.Stdout`; `os.Stdout.Write`; `cli.Success`/`Warning`/`Info`/`OutputTable`/
`Title`) lies lexically inside `outputRunResult`'s function body.

**Observability Evidence:** N/A.

**Edge Cases:**
- Worktree-cleanup failure path (AC-03): the guard must confirm the
  `--worktree` cleanup `defer`'s warning call is `fmt.Fprintf(os.Stderr, ...)`,
  not any stdout-writer class member.

**Negative Cases:**
- A `cli.Warning(...)` call added anywhere in `run.go` outside `outputRunResult` fails the guard (this is the counter-factual this widening exists to catch).

---

### TC-011: NDJSON schema — field presence and closed enum

**Feature Requirement:** REQ-F-002, REQ-F-001; AC-01.
**Technique Applied:** Equivalence Partitioning (event type; field-presence class).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table.

**Preconditions:** Fresh `LivenessRecorder` in `jsonMode=true` against `t.TempDir()`.

**Input:** A sequence of `Observe(RunProgress{...})` calls covering: `Phase:
"iteration"` (opens/closes slots), `Phase: "action"` (enriches open slot), and
a synthetic tick equivalent (see TC-012's internal seam) producing `heartbeat`.

**Expected Output:** Every emitted stderr line unmarshals with `ts` (RFC3339Nano,
UTC), `run_id`, `event` ∈ {stage_start, heartbeat, stage_end}, `entity_key`,
`iteration` (int, `0` when no stage open), `status` (`""` when none open),
`action` (`""` before action phase), `stage_elapsed_ms`, `total_elapsed_ms`
always present; `agent_type`/`provider` omitted (not empty-valued) when unset.

**Observability Evidence:** N/A.

**Edge Cases:**
- No stage open (before first `iteration`): `entity_key` = top-level key, `iteration: 0`, `status: ""`, `action: ""` (per spec's "decisions the implementer does not get to make" table).
- `agent_type`/`provider` both unset: keys absent from JSON, not present with empty string.

**Negative Cases:**
- No emitted line ever carries a `phase` key (D3: "`phase` is deliberately absent").
- No `event` value outside the closed 3-value set ever appears on stderr (the `run_end` summary line is file-only, asserted separately by TC-017).

---

### TC-012: Heartbeat cadence at the 10-second boundary

**Feature Requirement:** REQ-F-006, REQ-N-001; AC-01.
**Technique Applied:** Boundary Value Analysis.
**ISO 25010 Characteristic(s):** Functional Suitability, Performance Efficiency.

**Caller-Path Contract:** see table — internal same-package tick-method call, not a real sleep.

**Preconditions:** Recorder with an open stage.

**Input:** N tick-method invocations simulating elapsed stage execution across
multiple 10s windows.

**Expected Output:** At least one `heartbeat` line emitted per simulated 10s
window; no window with an open stage produces zero heartbeat lines.

**Observability Evidence:** N/A.

**Edge Cases:** Tick occurs with no stage open (boundary case already covered under TC-011's edge case).

**Negative Cases:** Heartbeat cadence does not vary when a `runClaimServicer`
with a non-default `TTL()` is present elsewhere in the process — proven
structurally by TC-026 (the recorder has no TTL input at all), not by this TC.

---

### TC-013: Worktree-cleanup warning targets `os.Stderr`

**Feature Requirement:** REQ-F-003; AC-03.
**Technique Applied:** Attack-class enumeration (wrong-writer regression).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table.

**Preconditions:** `go/parser` AST of `run.go`, locating the `--worktree`
cleanup `defer`'s error-handling branch (`if removeErr := ...; removeErr !=
nil { ... }`).

**Input:** Static AST inspection.

**Expected Output:** The branch's warning call is `fmt.Fprintf` with first
argument `os.Stderr` (matching the structurally identical warnings already at
`run.go:137` and `:638`).

**Observability Evidence:** N/A.

**Negative Cases:** A bare `fmt.Printf` (no `Fprintf`, no `os.Stderr`) at this call site fails.

---

### TC-014: Plain-mode line rendering — required fields, omitted-when-empty

**Feature Requirement:** REQ-F-004; AC-04.
**Technique Applied:** Equivalence Partitioning (mode); Boundary Value Analysis (field omission).
**ISO 25010 Characteristic(s):** Functional Suitability, Usability.

**Caller-Path Contract:** see table.

**Preconditions:** Recorder in `jsonMode=false`.

**Input:** Same `Observe` sequence class as TC-011, driven in plain mode.

**Expected Output:** Each stderr line matches D3's fixed-column text format:
`<ts>  <event>  <entity_key>  key=value...`, containing entity key, status,
action, agent, provider, stage elapsed, total elapsed when non-empty; empty-valued
keys omitted entirely (not printed as `key=`).

**Observability Evidence:** N/A.

**Edge Cases:** `agent_type`/`provider` unset → `agent=`/`provider=` tokens absent, not empty.

**Negative Cases:** No stray key-value pair for an empty field (e.g., no `action=` before the action phase).

---

### TC-015: Cascade parent→child→parent labeling (state transition)

**Feature Requirement:** REQ-F-005; AC-05; AC-08 rows 3-4.
**Technique Applied:** State Transition.
**ISO 25010 Characteristic(s):** Functional Suitability, Usability.

**Caller-Path Contract:** see table.

**Preconditions:** Recorder with `topLevelKey` = parent key.

**Input:** `Observe` sequence: parent `iteration(1)` → parent `action` → tick
(heartbeat) → child `iteration(1)` → child `action` → tick (heartbeat) →
parent `iteration(2)` (closes child's stage, reopens parent's) — mirroring
`controller.go`'s actual sequential cascade-child loop shape.

**Expected Output:** Every event between the child's first `iteration` and the
parent's next `iteration` carries the **child's** `entity_key`. The parent's
`stage_end` for its cascade stage is emitted (lazily if needed) exactly when
the child's `iteration` event arrives. The child's `stage_end` is emitted when
the parent's next `iteration` arrives.

**Observability Evidence:** N/A.

**Negative Cases:** No heartbeat during the child's open window ever carries the parent's key (the exact Finding 1 defect this feature exists to fix).

---

### TC-016: `Finish(nil)` still leaves a non-empty `run.log` (two sub-cases)

**Feature Requirement:** REQ-F-007; AC-06; AC-08 decision-table row 8.
**Technique Applied:** Decision Table (row: run end, no `RunResult` available — both with and without a previously-open stage).
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.

**Caller-Path Contract:** see table.

**Sub-case a — stage was opened, never closed** (SIGKILL-style abrupt cutoff):

**Preconditions:** Recorder driven through one open, unclosed stage (no `action` phase reached, i.e., paused before dispatch).

**Input:** `rec.Stop(); rec.Finish(nil)`.

**Expected Output:** `run.log` exists and is non-empty: it contains the lazily-emitted `stage_start` and a final `stage_end` for the open stage, even though no `RunResult` was available to source a `run_end` summary line from.

**Sub-case b — zero `Observe` calls ever made** (decision-table row 8b: `controller.Run` returns a bare Go `error`, e.g. `GetNextStatus` failing at `controller.go:359-361`, before its first `iteration` event; `run.go`'s liveness-teardown defer receives `runResult == nil` because only `emitRunEnd`'s *separate* defer substitutes a synthetic failed `RunResult` — the liveness recorder is never given one):

**Preconditions:** Freshly constructed recorder, zero `Observe` calls.

**Input:** `rec.Stop(); rec.Finish(nil)`.

**Expected Output:** `run.log` still exists and is non-empty — this is the case AC-06 most directly targets with the word "failure," and it is the one shape the initial draft of this test plan missed (found on self-review, not by codex — see red-team section). The recorder must write *something* identifiable (at minimum, its own `run_end`-shaped line with an empty/unknown outcome, or an explicit "no stages" marker) rather than silently writing zero bytes. **This is flagged as an explicit, must-resolve implementation decision for task generation** — the spec's D3 table does not define what the `run_end` line looks like when no `RunResult` is available, and this test plan does not prescribe the exact format, only the invariant: non-empty file, no fabricated `stage_start`/`stage_end` pair.

**Negative Cases:** `Finish(nil)` must not panic in either sub-case. Neither sub-case may fabricate a `stage_start` or `stage_end` line for a slot that was never opened (sub-case b only).

---

### TC-017: `Finish(result)` appends the `run_end` file-only summary line

**Feature Requirement:** D3 "one additional trailing line... not an event-struct render"; AC-06; AC-08 decision-table row 8a.
**Technique Applied:** Decision Table (row: run end, `RunResult` available — with and without any prior stage).
**ISO 25010 Characteristic(s):** Functional Suitability.

**Caller-Path Contract:** see table.

**Input (primary case — a stage ran):** `rec.Finish(&runner.RunResult{Outcome: "completed", FinalStatus: "completed", StagesCompleted: 4, TotalDuration: ...})` after driving at least one full stage through `Observe`.

**Input (decision-table row 8a — no stage ever ran):** `rec.Finish(&runner.RunResult{Outcome: "already_terminal", FinalStatus: "completed", StagesCompleted: 0, Stages: nil})` with **zero prior `Observe` calls** — the shape `controller.Run` actually returns when an entity is already terminal or a Question block pauses it before the loop's first `iteration` (`controller.go:366-381`; also the top-level/cascade Question-block early returns in `run.go` at lines 168-171 and 178-181, which never touch the controller at all).

**Expected Output:** `run.log`'s final line matches D3's `run_end` format
(`<ts>  run_end  <key>  outcome=... final_status=... stages=... total=...`) in
both inputs above — including the zero-stage case, where the file's *only*
content is this one line (no stray `stage_start`/`stage_end`). This line is
**never** written to stderr in either mode (assert stderr capture from the
same driven sequence contains no `run_end` line).

**Negative Cases:** The `run_end` line's second column value is not treated as a valid `event` enum member anywhere else in the codebase's D3-schema parsing (documented tradeoff, not re-tested — `run.log` readers must special-case one trailing line). The zero-stage case must not emit a fabricated `stage_start`/`stage_end` pair to "make the file look normal."

---

### TC-018: Teardown defer is a closure, not a direct method-value capture

**Feature Requirement:** D6 edit 2 ("Not `defer rec.Finish(runResult)`"); AC-06.
**Technique Applied:** Attack-class enumeration (argument-capture-at-registration-time bug class).
**ISO 25010 Characteristic(s):** Reliability, Maintainability.

**Caller-Path Contract:** see table.

**Preconditions:** `go/parser` AST of `runRun`.

**Input:** Static AST inspection of the statement registering the recorder's teardown defer.

**Expected Output:** The `defer` statement's call expression is a `*ast.FuncLit`
(`defer func() { rec.Stop(); rec.Finish(runResult) }()`), never a direct
`defer rec.Finish(runResult)` value-capture form.

**Negative Cases:** `defer rec.Finish(runResult)` (captures `nil` forever per Go's argument-evaluated-at-registration-time semantics) fails this check even though it would pass every `liveness_test.go` unit test in isolation.

---

### TC-019: Flush-free per-event durability (read-before-`Finish`)

**Feature Requirement:** REQ-F-010; AC-07.
**Technique Applied:** State Transition.
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Input:** `Observe(iteration)` → `Observe(action)` → one tick-method call
(heartbeat), then `os.ReadFile` the `run.log` path directly — **before**
calling `Finish()` or `Stop()`.

**Expected Output:** The file already contains that stage's `stage_start` line
and its latest `heartbeat` line, with entity, status, agent, provider present.

**Negative Cases:** A `bufio.Writer`-backed implementation (buffers pending a later flush) fails this test even if it passes TC-016/017 (which call `Finish`).

---

### TC-020: Stage pairing — lazy start, in-run and run-end closes (Decision Table rows 1/2/5/6)

**Feature Requirement:** REQ-F-009; AC-08.
**Technique Applied:** Decision Table.
**ISO 25010 Characteristic(s):** Functional Suitability, Reliability.

**Caller-Path Contract:** see table.

**Input:** Four driven sequences, one per row 1/2/5/6 of the AC-08 decision table above.

**Expected Output:** Scanning each sequence's emitted stream and pairing each
`stage_end` with the nearest preceding unpaired `stage_start` (emitting one
lazily first when row's C1 = no) leaves zero unpaired `stage_end`s in every
row.

**Negative Cases:** Any row producing an unpaired `stage_end` fails.

---

### TC-021: Stage pairing under a non-unique `(entity_key, iteration)` re-dispatch (Decision Table row 7)

**Feature Requirement:** REQ-F-009 ("not a unique key"); AC-08.
**Technique Applied:** Boundary Value Analysis (duplicate composite key), Decision Table row 7.
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Input:** Two full cascade passes over the same child entity key, `iteration`
restarting at 1 in the second pass (a second cascade dispatch of the same
child, per Q004's documented shape).

**Expected Output:** Pairing correctness holds across both passes — the
second pass's `stage_end` pairs with the second pass's `stage_start`, not the
first pass's.

**Negative Cases:** An implementation keying its pairing state by `(entity_key,
iteration)` merges the two passes' events; this test's assertion is
specifically designed to fail against that implementation (see counter-factual
in the caller-path table).

---

### TC-022: Empty `projectRoot` — file sink disabled, no error, stderr unaffected

**Feature Requirement:** REQ-F-011; AC-09.
**Technique Applied:** Boundary Value Analysis (empty-string boundary).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Input:** `NewLivenessRecorder("", runID, key, jsonMode, start)`, then a full `Observe` sequence.

**Expected Output:** stderr events still emitted for every `Observe` call;
`LogPath()` returns an empty value or one that resolves to no file being
created; no error returned from any recorder method; no panic.

**Negative Cases:** Any write attempt under an empty root fails the test (must be a complete no-op, matching `NewFileJSONLExporter("")`).

---

### TC-023: File sink `EACCES`-class failure — fail-soft, `slog.Debug` evidence

**Feature Requirement:** REQ-F-011, REQ-N-002; AC-09.
**Technique Applied:** Attack-class enumeration (I/O fault injection).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Preconditions:** `t.TempDir()` with `.shark/runs/` pre-created and
`os.Chmod`'d to `0o555` (blocks file/directory creation inside it); skipped on
`runtime.GOOS == "windows"` (matches `transcript_test.go`'s existing
permission-bit skip convention); `t.Cleanup` restores permissions before
`t.TempDir()`'s own cleanup runs.

**Input:** Full `Observe` sequence against the unwritable root, with `slog`
output captured (matches `transcript_test.go`'s `captureSlog(t)` helper).

**Expected Output:** Every stderr event still fires for every `Observe` call;
no error is returned to the run; at least one `slog.Debug` record is captured
describing the write failure (implementation hook from Observability Design).

**Observability Evidence:** `slog.Debug` record present, matching `file_jsonl_exporter.go`'s pattern.

**Negative Cases:** An error propagating out of `Observe`/`Finish` to the caller fails the test — REQ-F-011 is explicit that a sink failure "never fails or aborts the run."

**Scope note on REQ-F-011's fault-class list.** REQ-F-011 names three faults:
"unwritable path, disk full, missing project root." This plan maps them as:
*missing project root* → TC-022 (empty `projectRoot`, matching D6 edit 1's
"an empty root disables the file sink"); *unwritable path* → this TC's
`chmod 0o555` injection; *disk full* → **not independently tested**. `ENOSPC`
and `EACCES` surface identically to the recorder — both are an error return
from the same `os.OpenFile`/`os.MkdirAll` call, handled by the same fail-soft
branch — so a dedicated disk-full test would exercise the identical code path
as this TC with a harder-to-simulate portable fixture, for no additional
branch coverage. This matches the existing precedent's own scope:
`file_jsonl_exporter_test.go` (the codebase's other fail-soft-JSONL test)
carries no disk-full test either. Recorded as a deliberate, cost-justified
scope decision, not a silent omission.

---

### TC-024: Run-log path printed on stderr exactly once, before the first event, in both modes

**Feature Requirement:** REQ-F-008; AC-10.
**Technique Applied:** State Transition (ordering).
**ISO 25010 Characteristic(s):** Functional Suitability, Usability.

**Caller-Path Contract:** see table — recommended entrypoint `Start()`; AST fallback if task generation keeps the print in `run.go` (see Test Infrastructure recommendation).

**Input:** Table-driven over `jsonMode ∈ {true, false}`: construct recorder, capture stderr via pipe swap, call `Start()`, then drive one `Observe` sequence.

**Expected Output:** Exactly one line on stderr containing the absolute `LogPath()` value, appearing before any `stage_start`/`heartbeat`/`stage_end` line, in **both** `jsonMode` values.

**Negative Cases:** Two prints (once per mode-branch) or zero prints in either mode fails.

---

### TC-025: Recorder start precedes preflight/lease acquisition (wiring order)

**Feature Requirement:** D6 edit 2 ("a run that pauses at a Question block still leaves a log"); AC-10.
**Technique Applied:** Attack-class enumeration (ordering regression).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Preconditions:** `go/parser` AST of `runRun`.

**Input:** Static AST position comparison: the recorder-construction-and-start
statement(s) vs. the first call to `acquireRunLeaseForRunnableAction` /
`preflightCascadeQuestionBlock`.

**Expected Output:** Recorder start occurs strictly before both preflight calls.

**Negative Cases:** An implementation that moves recorder construction after preflight (e.g., to have `entityType` available "for free") silently breaks liveness for every Question-paused run — passes all of liveness_test.go, fails only this ordering check.

---

### TC-026: Heartbeat interval is a literal constant, independent of any TTL/config input

**Feature Requirement:** REQ-N-001.
**Technique Applied:** Contract surface enumeration (constructor signature + source-text scan).
**ISO 25010 Characteristic(s):** Performance Efficiency, Maintainability.

**Caller-Path Contract:** see table.

**Input:** `reflect.TypeOf(runner.NewLivenessRecorder)` signature inspection (5 params: projectRoot, runID, topLevelKey string; jsonMode bool; start time.Time — no `time.Duration` parameter); plus `os.ReadFile("internal/runner/liveness.go")` grepped for `TTL(`, `claim_ttl_seconds`, `DefaultClaimTTL`.

**Expected Output:** Constructor signature has no duration/interval parameter; source text contains none of the three claim/lease-TTL identifiers.

**Negative Cases:** Any match against those identifiers fails the test — this is the Finding-5/Decision-6 "two heartbeats" conflation guard, made mechanical rather than relying on code review alone.

---

### TC-027: Concurrent `Observe` + tick under the race detector

**Feature Requirement:** REQ-N-003.
**Technique Applied:** N/A (race-detector stress, not a partitioning technique — noted in the ISTQB table).
**ISO 25010 Characteristic(s):** Reliability.

**Caller-Path Contract:** see table.

**Input:** N goroutines calling `Observe` with distinct iterations concurrently with repeated tick-method invocations, run via `go test -race ./internal/runner/...`.

**Expected Output:** No data race reported; final emitted-line count is internally consistent (no torn writes, no lost/duplicated close events beyond what the interleaving itself would legitimately produce).

**Negative Cases:** Any `-race` report fails the test; this is the mechanical enforcement of REQ-N-003's mutex-guard requirement.

---

### TC-028: File and directory permission bits match the transcript convention

**Feature Requirement:** REQ-N-005.
**Technique Applied:** Boundary Value Analysis (permission bits).
**ISO 25010 Characteristic(s):** Security, Portability.

**Caller-Path Contract:** see table.

**Input:** `Observe` sequence producing at least one `run.log` write, then `os.Stat` on the file and its parent directory. Skipped on `runtime.GOOS == "windows"` (matches `transcript_test.go`).

**Expected Output:** File mode `0o644`; parent directory mode `0o755`.

**Negative Cases:** Default `os.Create`-derived permissions (umask-dependent) fail this test.

---

### TC-029: Closed JSON field set — no prompt/output/credential leakage

**Feature Requirement:** REQ-N-005 ("carries no prompt bodies, agent output, or credentials").
**Technique Applied:** Contract surface enumeration (closed field set).
**ISO 25010 Characteristic(s):** Security.

**Caller-Path Contract:** see table.

**Input:** `json.Unmarshal` every emitted stderr line into `map[string]json.RawMessage`.

**Expected Output:** The key set for every line is a subset of D3's closed
eleven-field list (`ts`, `run_id`, `event`, `entity_key`, `iteration`,
`status`, `action`, `agent_type`, `provider`, `stage_elapsed_ms`,
`total_elapsed_ms`) — no extra keys under any driven scenario, including the
dispatch-triggering `action` phase (which is where a prompt-body leak would
most plausibly be introduced).

**Negative Cases:** Any unexpected key present (e.g., `command`, `prompt`, `stdout`) fails the test.

---

## Integration Scenarios

| Scenario | Components | Boundary verified | Epic UAT reference |
|---|---|---|---|
| `--json` run under normal completion | `run.go` → `LivenessRecorder` → stderr + `run.log` → `outputRunResult` → stdout | stderr/stdout separation holds end-to-end | UAT-06 |
| Cascade dispatch (parent→children→parent) | `RunController.handleCascade` → `LivenessRecorder.Observe` | Child labeling correctness during cascade | UAT-06 ("During a cascade the child key is shown, not the parent's") |
| Run killed mid-stage (SIGKILL proxy) | `LivenessRecorder` file sink, no `Finish()` call | `run.log` already durable before any deferred cleanup could run | UAT-05 ("the stage attribution comes from F04's liveness record") |
| `--worktree` cleanup failure | worktree-removal `defer` → stderr | AC-02's stdout invariant survives a failure path, not just the happy path | UAT-06 (stdout purity) |
| Read-only / disk-full filesystem | `LivenessRecorder` file sink → fail-soft swallow | A `shark run` invocation never fails because of a liveness-logging fault | (NFR evidence — no numbered UAT scenario; covered under `uat-plan.md` §"Non-functional evidence" performance/cost claims) |

This feature contributes to **UAT-05** and **UAT-06** per `uat-plan.md`'s own
cross-feature note ("I-03 (F04 → F02): covered by UAT-05 and UAT-06 together").
No other epic UAT scenario depends on F04.

---

## Test Infrastructure

### Existing patterns to follow

| Pattern | File | Reused for |
|---|---|---|
| Same-package (`package runner`), no-controller, no-DB writer test against `t.TempDir()` | `internal/runner/transcript_test.go` | Direct structural precedent for `liveness_test.go` |
| Permission-bit assertion with Windows skip | `internal/runner/transcript_test.go:126-138` | TC-023, TC-028 |
| Stub `MockTransitioner`/`MockActionService`/`MockDispatcher`/`MockCascadeChildrenService` fixtures | `internal/runner/controller_test.go` | TC-001 (reused verbatim per D7, not reimplemented) |
| Fail-soft JSONL append + `slog.Debug` on error | `internal/observability/file_jsonl_exporter.go` | TC-023's expected behavior baseline |
| `TestTC00N_...` naming + `package contracts`, real-file reads, no DB | `tests/contracts/e40_i01_corpus_contract_test.go` | TC-001, TC-002 naming and file placement |
| `mockWorktreeCreator` test double | `internal/cli/commands/run_worktree_test.go` | Existing coverage of `setupWorktree`/`RemoveWorktree`; TC-013 is additive, not a replacement |
| Override-var injection pattern (`runClaimSvcOverride`, `withRunClaimSvcOverride`) | `internal/cli/commands/run_test.go` | Precedent exists if task generation later wants a runtime (not just AST) test for run.go wiring; not required by this plan's design |
| `captureSlog(t)` helper | `internal/runner/transcript_test.go` (referenced, not re-read in full) | TC-023 |

### New test helpers needed

- **Permission-denied fixture** (`chmod 0o555` on a pre-created directory,
  `t.Cleanup` restoring permissions, `runtime.GOOS != "windows"` guard). No
  existing precedent in this repo for this specific fault class (checked:
  `grep -rn "0o000\|Chmod(0" internal/` finds none); TC-023 introduces it,
  following the standard Go idiom already established by
  `transcript_test.go`'s Windows-skip convention.
- **Tick-forcing method on `LivenessRecorder`** (hard requirement — see below).

### Considered and rejected: a full DI-seam refactor of `runRun`

Before settling on the AST-guard + same-package-recorder-test split, this plan
considered extending the existing `runClaimSvcOverride`/`withRunClaimSvcOverride`
package-var-injection pattern (`run_test.go`) to `buildTransitioner`,
`cli.GetActionService`, `cli.GetWorkflowService`, `cli.GetCascadeService`,
`cli.GetQuestionBlocker`, and `cli.FindProjectRoot`, so that `runRun` itself
could be driven end-to-end with mocks and AC-06/AC-10's wiring half could be
proven at runtime instead of statically. Rejected: `buildTransitioner` and
friends are plain functions, not override-able vars today, so this would mean
refactoring six-plus call sites across `run.go` into injectable seams — a
scope and risk level disproportionate to a feature research itself scored
STANDARD (11/27) specifically because the defect is "confined to two print
statements plus a missing gate in one file" (research Decision 1). It would
also duplicate what `RunController.Run`'s stub-fixture pattern (already reused
verbatim by TC-001 per D7) already proves about controller/recorder
integration, just one layer higher and at much higher implementation cost. The
AST-guard approach is the proportionate choice for a STANDARD feature; a
future feature that touches `runRun`'s wiring more substantially may want to
revisit this trade-off.

### Hard requirement on task generation: a directly callable tick method

REQ-N-001 requires a fixed, non-configurable 10-second interval. TC-012 and
TC-019 (heartbeat cadence, flush-free durability) must not depend on a real
10-second sleep per test run. The task spec **must** expose the ticker's
per-tick action as a separately callable **unexported** method on
`LivenessRecorder` (exact name is an implementation choice; e.g., an internal
`heartbeatNow()` or `tick()` method) that both `Start()`'s `select`-loop
goroutine and `liveness_test.go` (same package: `package runner`) call
directly. This must **not** become a configurable interval parameter on the
constructor or a struct field settable from outside the package — that would
read as a REQ-N-001 violation to a reviewer even if unreachable from
production, and TC-026 explicitly checks the constructor signature carries no
duration parameter.

### Recommendation: move the `LogPath()` print into `Start()`

D6 edit 2 places the `LogPath()` stderr print in `run.go`, immediately after
recorder construction. This plan recommends the task spec instead place the
print inside `LivenessRecorder.Start()`, for two reasons:

1. **Testability.** `runRun` cannot be driven end-to-end in a unit test under
   this repo's testing rules (`.claude/rules/testing/cli-tests.md`: CLI
   command tests never use a real database; `runRun` resolves services via
   `cli.Get*Service()` singletons with no override seam for the entity
   services it needs). Moving the print into `Start()` converts AC-10 from an
   AST-only guard into a real `os.Pipe`-captured runtime assertion
   (TC-024), table-driven over `jsonMode`, matching how every other
   liveness-stream assertion in this plan is tested.
2. **Consistency with the spec's own stated layering principle.** D6's closing
   paragraph: "All formatting, state, and I/O live in `internal/runner`, beside
   `transcript.go`." The `LogPath()` print is stderr I/O; under that stated
   principle it belongs in `liveness.go`, not `run.go`.

This is a **recommendation, not a spec override** — flagged explicitly per
Rule 7 rather than silently assumed. If task generation keeps the print in
`run.go` as D6 literally states, TC-024's runtime half degrades to the TC-025
AST-guard technique only (ordering + unconditional-on-mode, proven statically
rather than by capturing real output), and this plan's exit gate is still met
either way — TC-024/TC-025 as written already state both the recommended
entrypoint and the AST fallback.

---

## Caller-Path Contracts (summary)

Every runtime test case above has a Caller-Path Contract in the table under
"Caller-Path Contracts (Step 5.8, full table)". No test case in this plan uses
an `internal-only` opt-out without justification: TC-012's tick-method call and
TC-018/TC-013/TC-025's AST-guard cases are explicitly justified inline (real
10s wait is infeasible for a unit test; wiring-order/writer-target properties
are syntactic, not behavioral, and this repo's own D7 already establishes the
source-invariant-validator convention for exactly this class of property).

---

## Codex Test-Plan Red-Team

**Verdict:** FAILED — codex unavailable (not a content verdict).
**Issues raised:** 0 (codex did not run)
**Issues addressed before dev:** N/A
**Issues deferred:** N/A

Two `codex exec -m gpt-5.6-terra -s workspace-write` invocations were attempted
against the drafted test plan, each with a 590-second timeout (exceeding the
580s minimum), per this workflow's Step 7.5. Both failed **immediately** (not
a timeout) with:

```
ERROR: You've hit your usage limit. Visit https://chatgpt.com/codex/settings/usage
to purchase more credits or try again at Aug 7th, 2026 11:30 PM.
```

Per Step 7.5's stated policy ("After two failures, log 'Codex test-plan
review: FAILED — [error]' as a non-blocking note in the test plan and proceed
— do not gate on codex availability, but document the gap"): **Codex
test-plan review: FAILED — usage limit exhausted, retry available after
2026-08-07 23:30.** This is a non-blocking gap, documented rather than hidden
per Rule 12 (fail loud).

**Mitigation applied in place of codex:** a self-red-team pass was run against
this plan using the identical evaluation criteria that would have been given
to codex (open-endedness, technique fit, enumeration completeness, ISO
coverage, observability, negative cases, caller-path soundness — see the
codex prompt preserved at
`/tmp/claude-1000/-home-jwwel-projects-shark-task-manager/3a15eae6-c465-4b17-a93c-fc3faa466fc5/scratchpad/e40_f04_test_plan_codex_prompt.md`
for the exact instructions that would have been sent). This pass, cross-checked
directly against `controller.go`'s source rather than assumed, found and fixed
one real BLOCKER-class gap and recorded two deliberate scope decisions:

- **BLOCKER, fixed:** AC-08's decision table (originally 7 rows) was missing
  the "zero stages ever opened at run end" case — reachable via two distinct
  production paths (already-terminal/Question-block-paused entities that
  never reach the controller's first `iteration`, and a bare `error` return
  from `controller.Run` before its first `iteration`). Added as row 8; TC-016
  and TC-017 were both revised to test it explicitly rather than leaving it as
  an accidental byproduct of a general "Finish(x)" call. Verified against
  `controller.go:359-381` and `run.go`'s Question-block early-return paths
  directly, not assumed.
- **CONCERN, resolved with a scope decision:** REQ-F-011 names three fault
  classes (unwritable path, disk full, missing project root); the plan tested
  two directly and mapped the third (missing project root) onto the existing
  empty-`projectRoot` case. "Disk full" is recorded as a deliberate,
  cost-justified non-test (identical code path to the tested `EACCES` case;
  the codebase's own `file_jsonl_exporter_test.go` precedent — verified by
  direct read — carries no disk-full test either).
- **CONCERN, resolved with a scope decision:** whether to extend the
  `runClaimSvcOverride` injection pattern to make `runRun` fully mockable
  (avoiding the AST-guard fallback for AC-06/AC-10's wiring half) was
  evaluated and explicitly rejected as disproportionate to a STANDARD-scored
  feature — recorded under Test Infrastructure above with the concrete reason
  (six-plus call sites would need refactoring into injectable seams).

The two items already flagged as open implementation-shape recommendations
(`LogPath()` relocation, the tick-forcing method) were re-checked and stand as
originally written — both are testable either way, and neither is a BLOCKER.

No item found during self-review reached FAIL-level (i.e., nothing that would
by itself justify sending this plan back for refinement rather than proceeding
with a documented note). **This plan proceeds as APPROVED. A follow-up codex
pass should be run once quota resets (2026-08-07 23:30) before or during task
generation, as a cheap additional check — not a gate.**

## Recommendations

- [x] Ready for development (no drift, spec is clear, every AC has a technique
      + ISO matrix entry + caller-path contract, observability designed;
      codex confirmation unavailable — self-red-team substituted and one
      BLOCKER-class gap was found and fixed in this same pass, see above)
- [ ] Needs BA refinement
- [ ] Needs tech refinement

Three implementation-shape items are flagged for task generation to resolve
explicitly (not silently): (1) the `LogPath()` print relocation recommendation,
(2) the tick-method requirement, and (3) the exact `run_end`-equivalent shape
`Finish(nil)` writes when zero stages ever opened (TC-016 sub-case b). None
block approval — all are testable either way.
