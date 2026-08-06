---
feature_key: E40-F04-shark-run-live-progress-and-per-run-log
epic_key: E40
type: combined-requirements-architecture-spec
tier: STANDARD
date: 2026-08-06
---

# E40-F04 Specification: `shark run` live progress and per-run log

Business context: see [epic PRD](../epic.md) §"Success Criteria" (G3) and §"Phase 1
scope". System-level decisions: see [epic architecture](../architecture.md)
§"Run liveness contract" and ADR-004. Brownfield capability inventory: see
[feature research report](research-report.md) §"Capability map". None of that is
restated here.

---

## Vocabulary (read this first)

Two unrelated mechanisms in `internal/cli/commands/run.go` are both called
"heartbeat". Conflating them silently breaks REQ-N-001.

| Term | Code | Cadence | Purpose |
|---|---|---|---|
| **Progress ticker** | `run.go:321` `time.NewTicker(10 * time.Second)` | fixed 10s | Operator/machine liveness display. **This is what this spec calls the stage-scoped heartbeat.** |
| **Claim/lease heartbeat** | `startRunHeartbeat`, `run.go:620` | `svc.TTL()/3` — ~5 min at the 15-minute `DefaultClaimTTL` (`internal/services/claim_service.go:22`) | Lease renewal. Never a display channel. |

Implementation must not wire display cadence to `svc.TTL()/3`. Research report
Finding 5 / Decision 6.

"Stage" in this document means one iteration of the run controller's main loop
for one entity — the unit bounded by `RunProgress{Phase: "iteration"}`.

---

## Requirements

Incremental over the epic. Every requirement below traces to epic criterion
**G3** ("Runs are observable while in flight"), which no other Phase 1 feature
owns.

### Functional

| ID | Requirement | Traces to |
|---|---|---|
| REQ-F-001 | While a stage is executing, `shark run` emits liveness events in **both** `--json` and plain mode. Neither mode is silent between stages. | G3; feature.md Scope 1–2 |
| REQ-F-002 | In `--json` mode, liveness events are NDJSON — one JSON object per line, terminated by `\n` — written to **stderr**. | G3; I-03 |
| REQ-F-003 | In `--json` mode, **stdout carries exactly one JSON document**: the `RunResult` produced by `outputRunResult`. No warning, progress, or diagnostic text reaches stdout on any code path, including the `--worktree` cleanup failure path. | X-08; UAT-06 |
| REQ-F-004 | In plain mode, liveness events are human-readable lines on **stderr**, each naming entity key, stage status, action, agent, provider, stage elapsed, and total elapsed. | G3; feature.md Scope 2 |
| REQ-F-005 | Every liveness event names the entity **currently executing**. During a cascade this is the child key, never the parent's. | G3; feature.md Scope 3; UAT-06 |
| REQ-F-006 | The heartbeat event fires at least once every 10 seconds for as long as the run process is alive, whether or not a stage is open. | G3; uat-plan.md:37 |
| REQ-F-007 | Every run writes `<project_root>/.shark/runs/<run_id>/run.log`, unconditionally — not gated on `ObservabilityConfig.Enabled`, `CaptureAgentTranscripts`, `--json`, or `--dry-run`. | G3; feature.md Scope 4 |
| REQ-F-008 | The absolute `run.log` path is printed once at run start, in both modes, on stderr. | G3; feature.md Scope 4 |
| REQ-F-009 | Stage events are balanced in stream order: each `stage_end` pairs with the nearest preceding `stage_start` not already paired. A stage that terminates before its action is resolved emits its `stage_start` lazily, immediately before its `stage_end`. | REQ-F-002 schema integrity |
| REQ-F-010 | Liveness events are durable at the moment they are emitted: after `SIGKILL` mid-stage, `run.log` already contains that stage's `stage_start` and its most recent `heartbeat`. No deferred flush, no buffered writer. | I-03; UAT-05 |
| REQ-F-011 | A liveness sink failure (unwritable path, disk full, missing project root) never fails or aborts the run. The stderr sink and the file sink fail independently. | NFR-002 |

### Non-functional

| ID | Requirement |
|---|---|
| REQ-N-001 | Heartbeat interval is a fixed 10 seconds, independent of `claim_ttl_seconds` and of `svc.TTL()`. |
| REQ-N-002 | Fail-soft. Sink errors are logged at `slog.Debug` and swallowed, matching `internal/observability/file_jsonl_exporter.go`. A run's exit code and `RunResult` are unchanged by liveness behavior. |
| REQ-N-003 | Thread-safe. The ticker goroutine and the controller's `Progress` callback touch shared stage state concurrently and must be mutex-guarded. |
| REQ-N-004 | No change to `RunResult`, `StageLog`, or the transcript byte format. These are X-07's pinned surface; F04 must leave them provably untouched. |
| REQ-N-005 | Security: `run.log` inherits transcript permissions — directory `0o755`, file `0o644` (`internal/runner/transcript.go`). Liveness events carry no prompt bodies, agent output, or credentials — only keys, statuses, action metadata, and durations. |

### Acceptance criteria

| ID | Criterion | Verified by |
|---|---|---|
| AC-01 | `shark run <key> --json` writes ≥1 NDJSON line to stderr per 10s of stage execution; every line unmarshals into the REQ-F-002 schema. | `internal/runner/liveness_test.go` |
| AC-02 | With `--json` set, the complete stdout byte stream decodes as exactly one `RunResult` and a second decode returns `io.EOF`; **and** every `fmt.Print*` call site in `internal/cli/commands/run.go` lies inside `outputRunResult`. Both halves per D7. | `tests/contracts/e40_i03_liveness_contract_test.go#TC-002` |
| AC-03 | The `--worktree` cleanup failure path writes its warning to stderr, not stdout; AC-02 holds when worktree removal fails. | `internal/cli/commands/run_worktree_test.go` |
| AC-04 | Plain-mode stage and heartbeat lines each contain entity key, status, action, agent, provider, stage elapsed, total elapsed. | `internal/runner/liveness_test.go` |
| AC-05 | Given a synthetic `Progress` sequence in which a parent's cascade stage is followed by child iterations, every event emitted between the first child `iteration` and the parent's next `iteration` carries the child's key. | `internal/runner/liveness_test.go` |
| AC-06 | After any run — success, failure, pause, `--dry-run`, or `SIGKILL` — `.shark/runs/<run_id>/run.log` exists and is non-empty. | `internal/cli/commands/run_test.go` |
| AC-07 | With the recorder driven into an open stage and at least one heartbeat, `run.log` read from disk **before `Finish()` is ever called** already contains that stage's `stage_start` and latest `heartbeat` with entity, status, agent, provider. This is the flush-free proxy for `SIGKILL` and needs no signal delivery. | `internal/runner/liveness_test.go`; `tests/contracts/e40_i03_liveness_contract_test.go#TC-001` |
| AC-08 | Scanning an emitted stream in order and pairing each `stage_end` with the nearest preceding unpaired `stage_start` leaves no unpaired `stage_end` — including for stages that never reached the `action` phase, and for a child re-dispatched by a second cascade pass (whose `iteration` restarts at 1, so `(entity_key, iteration)` is **not** a unique key). | `internal/runner/liveness_test.go` |
| AC-09 | A recorder constructed with an empty project root emits to stderr and writes no file, without error. A recorder whose file sink returns `EACCES` on every write still emits every stderr event and returns no error to the run. | `internal/runner/liveness_test.go` |
| AC-10 | The `run.log` absolute path appears once on stderr before the first stage event, in both modes. | `internal/cli/commands/run_test.go` |

### Out of scope

Inherited verbatim from feature.md, plus two carve-outs this spec adds.

- Streaming agent stdout/stderr live to the terminal.
- Progress UI: spinners, TTY rewriting, color. Append-only lines only.
- G2 outcome instrumentation (Phase 2 `StageLog` work).
- **Added:** per-stage exit codes in the liveness stream. `StageLog.ExitCode` is
  not available at the loop boundary this design observes (see D1), and it
  already reaches consumers on `RunResult`. A killed run has no exit code for
  its open stage by definition — the open `stage_start` plus growing
  `stage_elapsed_ms` *is* the timeout signal.
- **Added:** `RunOptions.Verbose` is dead — set at `run.go:311`, read nowhere
  (`grep -rn "\.Verbose" internal/runner/` returns no consumer). F04 does not
  wire it up and does not remove it. Rule 3.

---

## Architecture

### D1 — Derive stage boundaries in a recorder, not in the controller

`RunProgress` has no stage-end phase, and `result.Stages` is appended at eight
scattered sites across `internal/runner/controller.go`. Two options existed:
add stage-end emissions to the controller, or derive boundaries from the
existing `Progress` stream.

**Decision: derive.** A `LivenessRecorder` holds a **single open-stage slot**:

| Input | Slot action | Event emitted |
|---|---|---|
| `RunProgress{Phase: "iteration"}` | close any open slot, open a new one bound to `update.EntityKey` | `stage_end` for the closed slot (lazy `stage_start` first if it never emitted one) |
| `RunProgress{Phase: "action"}` | enrich the open slot with action / agent / provider | `stage_start` |
| ticker tick (10s) | none | `heartbeat` from current slot state |
| run end | close the open slot | `stage_end` |
| other phases | ignored | none |

A single slot is correct because **cascade children run strictly sequentially**:
`handleCascade`'s child loop (`controller.go:600-608`) is a plain `for` over
`childrenState.Children` calling `c.runChild` inline, and `controller.go`
contains no `go func`, `errgroup`, or `WaitGroup` (verified by grep, not
assumed). A child's `iteration` event therefore closes the parent's cascade
stage and opens the child's, which is the semantically right display: the
parent is not executing, the child is. That is exactly REQ-F-005. When the
cascade returns, the parent's next `iteration` closes the last child's stage.

Rationale for preferring this over controller emissions:

- **REQ-N-004 is provable by inspection.** `controller.go` is untouched, so
  `RunResult`/`StageLog` — X-07's pinned surface, actively owned by E22-F08 —
  cannot drift. Adding emissions would put F04 in the same file E22-F08 is
  editing, for display-only benefit.
- The consumption side (`RunOptions.Progress`) has exactly one production call
  site system-wide (`run.go:337`), so the whole change stays contained.
- No new plumbing for `run_id`: `controller.go:601` `childOpts := opts` already
  propagates it unchanged to every cascade child, so one `run.log` naturally
  spans a whole cascade tree (research Decision 7).

Deviation noted: this is a state machine in a helper type rather than an
explicit protocol, which is slightly less direct than a controller callback.
Accepted for the X-07 isolation above.

### D2 — Progress goes to stderr in both modes

Today plain mode prints progress to **stdout** (`run.go:328`, `run.go:348` —
`fmt.Printf`). This spec moves it to stderr in both modes.

Rationale: one sink for one concern. stdout becomes "the result" in every mode
(`RunResult` JSON, or the human summary from `outputRunResult`), stderr becomes
"the running commentary". This is standard CLI behavior, keeps REQ-F-003 a
single invariant rather than a mode-conditional one, and is invisible to a human
at a terminal. `internal/cli/commands/run_test.go` has no assertions on
plain-mode progress stdout (verified by grep), so nothing regresses.

### D3 — Two output formats, one event model

Both sinks render the same three-value event struct. Neither is a subset of the
other. The `run.log` file additionally carries one **summary line that is not an
event-struct render** (see below).

**stderr in `--json` mode — NDJSON. This table is the I-03 machine contract.**

| Key | Type | Presence | Example |
|---|---|---|---|
| `ts` | string, RFC3339 with nanoseconds, UTC | always | `2026-08-06T14:03:11.482913Z` |
| `run_id` | string (UUIDv4, from `generateRunID`) | always | `9f1c…` |
| `event` | string enum, **closed at exactly three values**: `stage_start`, `heartbeat`, `stage_end` | always | `heartbeat` |
| `entity_key` | string — the entity of the open stage; the top-level normalized key when no stage is open | always | `T-E40-F04-003` |
| `iteration` | int, 1-based loop counter for that entity's own `Run()` | always; `0` when no stage is open | `2` |
| `status` | string, the workflow status driving the stage | always; `""` when no stage is open | `in_development` |
| `action` | string, the resolved `OrchestratorAction` | always; `""` before the `action` phase | `spawn_agent` |
| `agent_type` | string | omitted when empty | `developer` |
| `provider` | string | omitted when empty | `anthropic` |
| `stage_elapsed_ms` | int64 — ms since the open stage began, or since run start when none is open | always | `74213` |
| `total_elapsed_ms` | int64 — ms since `runStart` | always | `181940` |

Field names `entity_key`, `run_id`, `status`, `agent_type`, `provider` mirror
`internal/runner/logging.go`'s `run.stage.*` vocabulary so the two channels read
alike (research Decision 5). **`phase` is deliberately absent** — `logging.go`
uses `phase` for a different concept (`context`, `placeholders`,
`action_lookup`, …) and carrying both keys would invite a consumer to conflate
them. `event` is the only state key.

**Consumer rule (I-03):** stderr is a shared channel. Pre-existing warning
lines — claim-release failure (`run.go:137`), claim-heartbeat failure
(`run.go:638`), and after this change the worktree-cleanup warning — are plain
text, not NDJSON. A consumer must skip any stderr line that does not parse as a
JSON object carrying an `event` key. F04 does not attempt to convert those
warnings to NDJSON; doing so would change unrelated error-reporting behavior.

**stderr in plain mode, and `run.log` in both modes — one text line per event:**

```
2026-08-06T14:03:11.482913Z  heartbeat  T-E40-F04-003  status=in_development action=spawn_agent agent=developer provider=anthropic stage=1m14s total=3m01s
```

Fixed leading columns (`ts`, two spaces, `event`, two spaces, `entity_key`),
then `key=value` pairs. Empty-valued keys are omitted. Human-tailable per
feature.md Scope 4, and still greppable field-by-field so an operator recovering
a killed run does not need the stderr capture.

`run.log` gets one additional trailing line at run end, sourced from
`RunResult` when one exists, satisfying the "and outcomes" half of feature.md's
Scope 4 acceptance criterion:

```
2026-08-06T14:06:02.110004Z  run_end  E40-F04  outcome=completed final_status=completed stages=4 total=3m52s
```

**`run_end` is a file-only summary line, not a fourth `event` value.** It is
rendered by `Finish` directly, never through the event struct, and **never
appears on stderr in any mode**. The `event` enum in the table above stays
closed at three values, exactly as feature.md declares it, so an I-03 consumer
can treat any unrecognized `event` value as a hard schema violation rather than
as forward compatibility. The tradeoff — a `run.log` reader must tolerate one
trailing line whose second column is outside the enum — is accepted because
`run.log` is the human-tailable sink and the machine contract is stderr.

### D4 — Per-event durability, no buffering

REQ-F-010 exists because the case I-03 was built for is `timeout` killing the
process. Deferred closers do not run on `SIGKILL`. Therefore:

- `run.log` writes are open-append-write-close per event
  (`os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)`), the exact
  shape of `internal/observability/file_jsonl_exporter.go`. No `bufio.Writer`
  anywhere in the path.
- stderr is `os.Stderr` directly. It is unbuffered in Go — **do not wrap it**.
- "Flush at run end" is a completeness nicety for the final open stage, never
  the durability mechanism.

The per-event `open`/`close` costs one syscall pair per 10 seconds. That is
irrelevant against multi-minute agent stages, and it is the only shape that
survives `SIGKILL`.

### D5 — Component changes

| File | Change |
|---|---|
| `internal/runner/liveness.go` | **New.** `LivenessRecorder` — the D1 state machine, both D3 renderers, the D4 writers, and the 10s ticker. Constructor `NewLivenessRecorder(projectRoot, runID, topLevelKey string, jsonMode bool, start time.Time) *LivenessRecorder`. Methods: `Observe(runner.RunProgress)` (matches the `RunOptions.Progress` signature), `Start()` / `Stop()` for the ticker goroutine, `Finish(*runner.RunResult)`, `LogPath() string`. Mutex-guarded per REQ-N-003. Empty `projectRoot` disables the file sink only — mirroring `NewFileJSONLExporter("")`. |
| `internal/runner/liveness_test.go` | **New.** AC-01, AC-04, AC-05, AC-07 (writer-level), AC-08, AC-09. Drives `Observe` with synthetic `RunProgress` sequences against `t.TempDir()` — no controller, no database, matching `internal/runner/transcript_test.go`. |
| `internal/cli/commands/run.go` | **Modified.** Five edits, detailed below. |
| `internal/cli/commands/run_test.go` | **Modified.** AC-06, AC-10. |
| `internal/cli/commands/run_worktree_test.go` | **Modified.** AC-03. |
| `tests/contracts/e40_i03_liveness_contract_test.go` | **New.** TC-001 (I-03 producer shape), TC-002 (X-08 stdout purity). Execution mechanism specified in D7 — both run unattended in CI with no spawned agent, no scratch project, and no database. |

**No data model change.** No migration, no `CurrentSchemaVersion` bump, no new
config field. `.shark/runs/` is already gitignored (`.gitignore:57`).

**No API contract change.** `RunOptions`, `RunProgress`, `RunResult`,
`StageLog`, and `writeTranscript`'s byte format are all unchanged (REQ-N-004).
`run.log` and transcripts share `.shark/runs/<run_id>/` without collision:
`run.log` vs `<n>-<status>-<provider>.log`. The recorder performs its own
`os.MkdirAll` because `writeTranscript`'s directory creation is conditional on
`CaptureAgentTranscripts` (research Finding 6); `MkdirAll` on an existing
directory is a no-op, so concurrent creation in the same run is safe.

### D6 — Integration with `internal/cli/commands/run.go`

Five precise edits.

1. **Resolve the project root early and fail-soft.** `cli.FindProjectRoot()`
   currently runs at `run.go:299`, after preflight, and returns a hard error.
   Add a *separate*, non-fatal lookup immediately after `runID := generateRunID()`
   (`run.go:90`) whose error is discarded — an empty root disables the file sink
   per D5. **Do not move or soften the existing hard-error call at line 299**:
   moving it above `ParseGetArgs` would change which error a malformed key
   produces.

2. **Construct and start the recorder** immediately after the `emitRunStart`
   call (`run.go:99-107`) and before the preflight at `run.go:165`, so a run
   that pauses at a Question block still leaves a log. It cannot go earlier than
   `emitRunStart`: the constructor takes `topLevelKey`, which does not exist
   until `ParseGetArgs` at `run.go:93`. Print `LogPath()` once to stderr
   (REQ-F-008 / AC-10). Register the teardown as:

   ```go
   defer func() { rec.Stop(); rec.Finish(runResult) }()
   ```

   **Not** `defer rec.Finish(runResult)`. A deferred call evaluates its
   arguments at registration time and would capture `nil` forever — the run-end
   summary line would silently lose its outcome while AC-06 (file exists,
   non-empty) still passed. `run.go:112-127` already documents this exact trap
   for `emitRunEnd`; follow the same closure pattern. `Stop()` precedes
   `Finish()` so the ticker cannot race a final `stage_end`.

3. **Replace the `if !cli.GlobalConfig.JSON` block at `run.go:320-351`
   entirely** with:

   ```go
   opts.Progress = rec.Observe
   ```

   This single line resolves all three original defects at once: the JSON gate
   disappears (REQ-F-001/002), the inline ticker is replaced by the recorder's
   own (REQ-F-006), and `normalizedKey` is no longer in scope at either print
   site so `update.EntityKey` is used by construction (REQ-F-005). The
   `Phase != "action"` filter at `run.go:338` is superseded by D1's phase table.

4. **Fix the stdout leak at `run.go:275`.** `fmt.Printf("warning: failed to
   remove worktree %s: %v\n", …)` inside the `--worktree` cleanup `defer`
   becomes `fmt.Fprintf(os.Stderr, …)`, matching the structurally identical
   warnings already at `run.go:137` and `run.go:638`. In scope per research
   Decision 3: same file, same command, same defect class, and REQ-F-003 is a
   general `shark run` invariant rather than a bench-scoped one. ADR-001 keeps
   bench itself off the `--worktree` path, so this is correctness for other
   consumers, not for E40's own harness.

5. **`outputRunResult` is unchanged.** It remains the sole stdout writer.

Layering: `run.go` stays a thin wrapper (`.claude/rules/cli/patterns.md`) — it
constructs the recorder and hands `rec.Observe` to the controller. All
formatting, state, and I/O live in `internal/runner`, beside `transcript.go`,
per research Decision 4.

### D7 — How the contract tests actually run

The X-08 obligation is only discharged if E22-F08 can *inherit a running gate*.
Neither contract test may depend on a spawned agent, a scratch project, or the
live database. `scripts/shark-scratch-env.sh` is the sanctioned route for a
live-ish environment (never `admin init` against this repo), but it is not
needed here, and the `tests/contracts/` precedent
(`e40_i01_corpus_contract_test.go`) spawns nothing.

**TC-001 — I-03 producer shape.** Drive `runner.RunController.Run` **in
process** with the stub transitioner / action service / dispatcher fixtures that
`internal/runner/controller_test.go` already provides, passing
`RunOptions{Progress: rec.Observe, ProjectRoot: t.TempDir(), RunID: …}`. Capture
stderr through an `os.Pipe` swap. Assert every captured line unmarshals into the
D3 table with the required keys present and `event` within the closed enum, and
that `<tmp>/.shark/runs/<run_id>/run.log` contains the corresponding stage
lines. **Read `run.log` before calling `Finish()`** — that is what proves
REQ-F-010 per-event durability rather than flush-at-end, and it needs no signal
delivery.

**TC-002 — X-08 stdout purity.** Two halves, because the runtime half alone
cannot see a stdout write added on a path the fixture does not exercise.

- *Runtime:* swap `os.Stdout` for an `os.Pipe`, set `cli.GlobalConfig.JSON`,
  drive the recorder over a synthetic stage sequence, then call
  `outputRunResult(result)`. Assert the captured byte stream decodes via
  `json.Decoder` into exactly one `runner.RunResult` and that a second
  `Decode` returns `io.EOF` — no trailing bytes, no interleaved text.
- *Source guard:* parse `internal/cli/commands/run.go` with `go/parser` and
  assert every `fmt.Print`/`fmt.Printf`/`fmt.Println` call site lies inside
  `outputRunResult`. This is the half E22-F08 durably inherits: it is
  environment-free, runs in milliseconds, and fails loud the moment anyone adds
  an ungated stdout write to the dispatch path — which is precisely the defect
  class F04 is fixing at `run.go:275`. It follows the codebase's existing
  source-invariant validator convention rather than inventing one.

An end-to-end `shark run --json` invocation is deliberately **not** the gate.
Should a future integration test want one, `--dry-run` is the agent-free lever:
it drives the full loop — `iteration` and `action` progress events fire, stages
append, statuses advance simulated (`controller.go:726`, `:782`, `:1028`) — and
short-circuits only the agent spawn and the real transition. That is noted for
F02's benefit, not required by F04.

### Decisions the implementer does not get to make

| Question | Decision | Rationale |
|---|---|---|
| Is `run.log` written under `--dry-run`? | **Yes.** REQ-F-007 says unconditional and means it. | A dry run is still a run whose stage sequence an operator may want to inspect. Carving it out would add a conditional whose only effect is a missing file. |
| Ticker fires with no stage open (before the first `iteration`, during cascade child lookup)? | **Emit a heartbeat** with `entity_key` = top-level key, `iteration: 0`, `status: ""`, `action: ""`, and `stage_elapsed_ms` measured from run start. | Those windows are precisely when a stall is invisible today. Suppressing here would make liveness go dark in the gap it exists to cover. |
| Heartbeat interval source? | **Literal 10s constant in `liveness.go`.** Never `svc.TTL()`, never config. | REQ-N-001. See Vocabulary. |
| `--json --verbose`? | No effect on the stream. `RunOptions.Verbose` has no consumer. | Out of scope above. |

---

## Cross-feature interactions

- **Produces**: I-03 — Run liveness contract. Consumer feature: **E40-F02**
  (Bench harness: run driver and metric collection).
- **Shape source**: `architecture.md#run-liveness-contract`
- **Contract tests**: `tests/contracts/e40_i03_liveness_contract_test.go#TC-001`
- **Gate mode**: `live`, as declared in
  [E40-interaction-map.md](../E40-interaction-map.md). No `contract-only`
  declaration is made or claimed.

The D3 NDJSON field table is the *elaboration* of that shape source, not a
replacement for it. F02 must cite `architecture.md#run-liveness-contract`
verbatim.

**Consumer-side mirror obligation.** E40-F02 is `execution_order` 3 and not yet
decomposed, so no F02 task exists today to declare `I-03: consumes`. This
mirrors the map's "Sequencing note on I-01" precedent exactly: it is build-order
sequencing, not an open gap. F02's own `task_generation` must create at least
one task that declares `I-03: consumes`, copies the shape source
`architecture.md#run-liveness-contract` and the contract test
`tests/contracts/e40_i03_liveness_contract_test.go#TC-001` **verbatim**, and
owns the real caller path that reads the liveness record to attribute a
timed-out run's stalled stage (UAT-05). The gate mode stays `live`.

TC-001 asserts the producer side by driving `RunController.Run` in process with
stub fixtures against `t.TempDir()` (mechanism in D7): captured stderr matches
the D3 schema, and `run.log` read before `Finish()` already names the open
stage — the durability property F02 depends on when the timeout cap kills a run
and stdout never delivers a `RunResult`.

---

## Cross-epic integrations

- **Produces**: X-08 — Extend `shark run` observability without changing its
  stdout contract. Consumer epic: **E22 — External Orchestration Runner**; no
  E22 feature owns this surface, because `shark run --json` stdout consumers
  (skills, agents, CLI callers) are dispersed system-wide.
- **Contract / shape source**: E40 architecture "Run liveness contract";
  `internal/cli/commands/run.go` progress callback and ticker.
- **UX / CX handoff notes**: Every `shark run` user gains liveness, not only
  bench; stdout stays exactly one `RunResult` document so skills and agents
  parsing it are unbroken; the per-run log path is printed once at run start so
  users know what to tail. **Activation owner: E40-F04 itself**, accountable via
  its own stdout-preservation check. Forward obligation: E22-F08, as the next
  scheduled change to the same `controller.go` dispatch path, must keep that
  stdout assertion green when it lands.
- **Test coverage**:
  `tests/contracts/e40_i03_liveness_contract_test.go#TC-002`, plus uat-plan.md
  UAT-06.

TC-002 is the **durable artifact of the X-08 activation-owner obligation**. The
cross-epic map places a forward obligation on E22-F08 to "keep that stdout
assertion green"; a manual UAT step is not something E22-F08 can inherit or run,
so the obligation is discharged as a committed Go test in `tests/contracts/`
(AC-02). Its second half — the `go/parser` source guard asserting every
`fmt.Print*` in `run.go` sits inside `outputRunResult` — is what makes the
obligation genuinely inheritable: it needs no agent, no database, and no scratch
project, so it runs in E22-F08's own CI unchanged and fails loud if that feature
reintroduces a stdout write on the dispatch path. It lives in `tests/contracts/`
rather than `internal/cli/commands/` precisely so a cross-epic consumer can find
and depend on it.

X-08's status in the map is `assigned` and is unchanged by this spec. F04
produces no X-07 or X-09 obligation: REQ-N-004 forbids touching X-07's pinned
`RunResult`/`StageLog`/transcript surface, and X-09 is Phase 2 work gated on
Q003.

---

## Durable unresolved decisions

**No new Q### is required for this feature.** Recorded deliberately rather than
left as an absence of `TBD` text.

Every decision this feature depends on is either closed upstream or closed here:

- **Q001** (isolation mechanism) and **Q002** (G6 phase placement) are
  **resolved** upstream and applied throughout `architecture.md` and `epic.md`.
- **Q003** (agent usage envelope field names) constrains F02's transcript
  parser. F04 emits no token, cost, or model-ID field, so Q003 cannot block it.
- **Q004** (Phase 2 cascade attribution — sibling children sharing a `run_id`
  while restarting their stage counter, so transcripts collide on filename)
  constrains Phase 2 feature benching. It does **not** affect F04: `run.log` is
  one file per `run_id` and is append-only, so a shared `run_id` across cascade
  siblings is the desired property here (research Decision 7), not a collision.
  D1's single-slot design plus REQ-F-005 makes each line self-identifying by
  `entity_key`, which is strictly more attribution than the transcript
  filenames Q004 describes.
- The one decision research left genuinely open — whether to fix
  `run.go:275`'s stdout leak in F04 or carve it out with a stated reason
  (research Decision 3) — is **closed in this spec**: in scope, D6 edit 4, on
  the grounds that REQ-F-003 is a general `shark run` invariant.

---

*Last Updated*: 2026-08-06
