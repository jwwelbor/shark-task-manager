---
research_schema: 2
entity_key: E40-F04
entity_type: feature
recipe: universal
rigor: standard
categories:
  - backend
  - workflow_operations
related_work: true
---

# Research report: shark run live progress and per-run log

## Scope

E40-F04 is the epic's sole Phase 1 Go change: `shark run` gives no liveness
signal in the paths that matter. It touches one command file,
`internal/cli/commands/run.go`, in four ways — stderr NDJSON progress events
in `--json` mode (stdout stays exactly one `RunResult` document), a
stage-scoped heartbeat in plain mode, correct child-entity labeling during
cascades, and an unconditional per-run log at
`.shark/runs/<run_id>/run.log`. It produces **I-03** (consumed by E40-F02,
not yet decomposed) and **X-08** (E40's only cross-epic seam into E22, whose
activation owner is F04 itself per `E40-cross-epic-map.md`).

Complexity is pre-scored **STANDARD (11/27)** — recorded as a decision note
on E40-F04, 2026-08-06. This research corroborates that score directly: the
defect is a display bug confined to two print statements plus a missing
`--json`-mode gate in a single file, and the new work is one small,
fail-soft writer with two directly reusable precedents already in the same
package tree (`internal/runner`). No new capability crosses a service or
data boundary the way F01–F03's harness work does.

Terms used in this report, matching the epic report's vocabulary: **progress
ticker** (the 10-second `time.NewTicker` at `run.go:321`, display-only,
gated by `!cli.GlobalConfig.JSON`) and **claim/lease heartbeat**
(`startRunHeartbeat`, `run.go:620`, interval `svc.TTL()/3`, purpose is lease
renewal). Both are called "heartbeat" somewhere in this codebase and must
not be conflated — see Findings 5 and Decisions 6.

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/feature.md` (Goal, Scope items 1–4, Acceptance Criteria, Out of Scope) and `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` §"Run liveness contract" define the stderr-NDJSON / stage-scoped-heartbeat / per-run-log vocabulary and the boundary against streaming agent output (explicitly out of scope).
- [x] `affected_implementation_or_contract` — Evidence: `internal/cli/commands/run.go` full read (823 lines) — confirmed defect at lines 320 (`if !cli.GlobalConfig.JSON` gate), 328 (ticker print uses `normalizedKey`), 337–349 (`opts.Progress` callback, print at 348 uses `normalizedKey`), plus a newly found second stdout-purity instance at line 275 (`fmt.Printf` warning inside the `--worktree` cleanup defer). `internal/runner/controller.go` (`RunOptions`, `RunProgress`, `Run()` method, cascade `childOpts` construction at line 601) corroborates that the controller side already binds `EntityKey` correctly per call.
- [x] `related_work` — Evidence: epic-level `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (Capability map row on `--json`-mode liveness, Finding 5, Decision 4 — reproduces the same 320/328/348 defect at epic scope); `E40-interaction-map.md` §"Sequencing note on I-03" (F04 is F02's primary stalled-stage source, not a hard dependency); `E40-cross-epic-map.md` row X-08 and its "No E23 row" note (F04 is the activation owner; the per-run log is a deliberate departure from E23's slog path); `E40-F01-.../research-report.md` (sibling report, read for front-matter/section-structure precedent).
- [x] `pattern_contract` — Evidence: `internal/runner/transcript.go` (existing `.shark/runs/<run_id>/` path owner, `transcriptDirMode 0o755` / `transcriptFileMode 0o644`, documented byte contract, fail-soft-by-caller-convention); `internal/observability/file_jsonl_exporter.go` (`O_APPEND|O_CREATE|O_WRONLY`, `json.NewEncoder(f).Encode` per line, fail-soft via `slog.Debug` + swallow); `internal/runner/logging.go` (existing `run.stage.*` slog vocabulary — `entity_key`/`run_id`/`status`/`phase`/`agent_type`/`provider` bound per-call via struct fields, config-gated by `obs.Enabled`).
- [x] `dependency_impact` — Evidence: `grep -rn "\.Progress\s*=" internal/` — `RunOptions.Progress` has exactly one production call site (`run.go:337`), so the callback contract itself needs no change; `internal/runner/controller.go:601` (`childOpts := opts`) — confirms `RunID` propagates unchanged to cascade children with no additional plumbing; `E40-interaction-map.md` (E40-F02 as I-03's declared, not-yet-implemented consumer).

## Capability map

| Capability | Brownfield evidence | Decision | E40-F04 responsibility |
|---|---|---|---|
| `shark run` progress ticker + `Progress` callback print sites | `internal/cli/commands/run.go:320` (JSON gate), `:328` and `:348` (both print `normalizedKey`, ignoring `update.EntityKey`) | EXTEND | Confirmed defect, independently re-verified at feature scope (matches epic report Finding 5 / Decision 4). Fix requires touching the gate and both print sites together. |
| `RunProgress` emission inside the controller loop | `internal/runner/controller.go:425-432,481-491` — `RunProgress{EntityKey: key, ...}` bound from the `Run()` method's own `key` parameter, not a closure | REUSE | Controller-side binding is already correct for both parent and cascade-child invocations (each `Run()` call gets its own `key`). No controller change needed — the bug is entirely local to run.go's consumption of the callback. |
| `RunID` propagation to cascade children | `internal/runner/controller.go:601` (`childOpts := opts`, only `EntityType` overridden); `run.go`'s `runChild` closure (only `EntityType`, `SessionID` overridden) | REUSE | `run_id` is already shared across a whole cascade tree with no extra plumbing — `.shark/runs/<run_id>/run.log` is naturally one file per run including children, once the EntityKey fix above lands. |
| Per-dispatch transcript writer (`.shark/runs/<run_id>/` owner) | `internal/runner/transcript.go` — documented dir/file modes, byte-contract format, path construction (`relTranscriptPath`) | EXTEND | New `run.log` writer should live beside this file (not inline in `run.go`), reuse its directory conventions, and must MkdirAll independently since `writeTranscript`'s directory creation is conditional on `CaptureAgentTranscripts`. `run.log` and `<n>-<status>-<provider>.log` filenames do not collide. |
| Fail-soft JSONL append pattern | `internal/observability/file_jsonl_exporter.go` — `O_APPEND\|O_CREATE\|O_WRONLY`, encode-per-line, errors logged and swallowed, never returned to caller | REUSE | Directly reusable mechanics for both the stderr NDJSON stream and the `run.log` file: a disk-full or permission error must not fail a `shark run` invocation. |
| Config-gated `run.stage.*` slog vocabulary | `internal/runner/logging.go` — `emitStageStart/Dispatch/Complete/Transition/Error`, all no-op when `obs.Enabled` is false, all land only in `shark.log` | REUSE field vocabulary, CONTRADICTS as transport | Field names (`entity_key`, `run_id`, `status`, `phase`, `agent_type`, `provider`) are worth mirroring in the new NDJSON schema for legibility, but this path cannot be F04's transport — it is config-gated and never reaches stderr. `architecture.md`'s liveness contract section and the cross-epic map's "No E23 row" note already commit to this as a deliberate departure, not an open question. |
| Claim/lease heartbeat (`startRunHeartbeat`) | `run.go:620-645` — interval `svc.TTL()/3` (≈5 min at the 15-min default), purpose is lease renewal, prints only a stderr warning on failure | REUSE, do not conflate | A second, unrelated thing named "heartbeat" in the same file. F04's "stage-scoped heartbeat" (feature.md Scope item 2) is the progress ticker, not this — a spec that reuses this interval would silently violate the "≥10s" liveness claim in `uat-plan.md:37`. |
| `RunOptions.Progress` callback contract | `internal/runner/controller.go` (`type RunOptions`, field `Progress func(RunProgress)`) | REUSE | Exactly one production caller (`run.go`) system-wide (`grep -rn "\.Progress\s*=" internal/`). The callback signature and its controller-side invocation need no change. |
| E40-F02 bench harness (I-03 consumer) | `architecture.md` §"Run liveness contract"; `E40-interaction-map.md` §"Sequencing note on I-03" | NEW (forward dependency) | F02 is not yet decomposed, so nothing today parses F04's stream, but architecture.md names it F02's *primary* stalled-stage source for `outcome=timeout` runs (scratch-DB status is the documented fallback). F04's output shape should stay stable once F02 begins consuming it, but F04 does not need to wait for F02 to exist. |

## Findings

1. The epic report's Finding 5 (`internal/cli/commands/run.go` gates both the
   progress ticker and the `Progress` callback behind `if
   !cli.GlobalConfig.JSON` at line 320, and both print sites at lines 328 and
   348 reference the outer `normalizedKey` closure variable instead of
   `update.EntityKey`) is independently re-confirmed by a full read of the
   file at feature scope. No new ambiguity was found in the defect itself.

2. `internal/runner/controller.go`'s `RunProgress` emission (lines 425–432 and
   481–491) already binds `EntityKey: key` correctly for every call, because
   `key` is the `Run()` method's own parameter (`func (c *RunController)
   Run(ctx context.Context, key string, opts RunOptions)`), bound fresh for
   each parent or cascade-child invocation. The mislabeling defect is entirely
   local to `run.go`'s two `fmt.Printf` call sites, not a controller-side bug
   — the fix is a targeted two-line change plus the JSON-mode gate, not a
   plumbing change.

3. **New finding, not in the epic report**: `run.go:275`, inside the
   `--worktree` cleanup `defer`, uses `fmt.Printf("warning: failed to remove
   worktree %s: %v\n", ...)` — stdout, ungated by JSON mode — while the
   structurally identical claim-release warning (line 137) and
   claim-heartbeat warning (line 638) both correctly use
   `fmt.Fprintf(os.Stderr, ...)`. This means `shark run <key> --json
   --worktree` can already emit non-JSON text on stdout today if worktree
   removal fails, independent of any F04 change. ADR-001 in
   `architecture.md` means the bench harness itself never exercises
   `--worktree` (it uses harness-owned `--workdir`), so this does not block
   E40's own UAT-6/I-03 path — but F04's own acceptance criterion ("stdout
   parses as a single JSON document ... existing consumers unbroken") is a
   general `shark run` invariant, not one scoped to bench, and this is a
   second, independently discovered instance of the same defect class F04
   already owns fixing at 328/348.

4. `RunID` propagates unchanged to cascade children:
   `controller.go:601`'s `childOpts := opts` copies the parent's full
   `RunOptions` (including `RunID`) before overriding only `EntityType`;
   `run.go`'s `runChild` closure (lines 196–262) further overrides only
   `EntityType` (redundantly) and `SessionID`, never `RunID`. Every entity in
   a cascade therefore already shares one `run_id`, so
   `.shark/runs/<run_id>/run.log` is naturally a single file spanning the
   whole cascade tree, as feature.md's Scope item 4 implies — but every
   line's correctness depends entirely on Finding 1's `update.EntityKey` fix:
   without it, every line in that one shared file would print the parent's
   key regardless of which child is actually executing.

5. Two unrelated mechanisms share the word "heartbeat" in this file and must
   not be conflated by a spec or implementation: (a) the **progress ticker**
   at line 321 (`time.NewTicker(10 * time.Second)`, display-only, gated by
   `!cli.GlobalConfig.JSON`) — what feature.md's "stage-scoped heartbeat" and
   `uat-plan.md:37`'s "heartbeats appear at least every 10 seconds" describe;
   and (b) the **claim/lease heartbeat** via `startRunHeartbeat` (line 620),
   whose interval is `svc.TTL()/3` (a fraction of `claim_ttl_seconds`,
   15-minute default → ~5-minute interval) and whose purpose is lease
   renewal, not operator display. `internal/services/claim_service.go:22`
   confirms `DefaultClaimTTL = 15 * time.Minute` in code (not just in docs),
   so `svc.TTL()/3` is a ~5-minute interval by default — reusing (b)'s
   cadence for F04's display heartbeat would silently violate the ≥10s
   liveness claim.

6. No implementation of `.shark/runs/<run_id>/run.log` exists anywhere in the
   codebase today (`grep -rn "run\.log" internal/` returns nothing outside
   comments naming the future path). This is greenfield work. Its natural
   home is beside `internal/runner/transcript.go`, which already owns
   `.shark/runs/<run_id>/` path construction and documents the directory's
   permission modes. `writeTranscript`'s own `os.MkdirAll` runs only when
   `CaptureAgentTranscripts` is true, so an unconditional `run.log` writer
   must perform its own `MkdirAll` and tolerate the transcript writer
   creating the same directory concurrently in the same run — filenames do
   not collide (`run.log` vs `<n>-<status>-<provider>.log`).

7. A structured, per-call slog event vocabulary already exists for exactly
   this content: `internal/runner/logging.go`'s `run.stage.start` /
   `run.stage.dispatch` / `run.stage.complete` / `run.stage.transition` /
   `run.stage.error` events, each carrying `entity_key`, `run_id`, `status`,
   `phase`/`action`, `agent_type`, `provider` bound from per-call struct
   fields (`stageStartParams`, `stageDispatchParams`, etc.) — the in-repo
   proof that Finding 2's correct binding pattern is a known, working
   convention elsewhere in the same package. Every one of these emitters
   no-ops when `ObservabilityConfig.Enabled` is false (default) and, when
   enabled, writes only to the global slog handler (`shark.log`), never to
   stderr or a per-run file. `architecture.md`'s liveness-contract section
   and `E40-cross-epic-map.md`'s "No E23 row" note already commit to NOT
   reusing this transport for F04 — this research corroborates that decision
   rather than reopening it.

8. `internal/observability/file_jsonl_exporter.go` is the codebase's other
   close JSONL precedent: `O_APPEND|O_CREATE|O_WRONLY` file open,
   `json.NewEncoder(f).Encode` per record, and fail-soft error handling
   (`slog.Debug` + swallow, never propagated to the caller) so a disk-full or
   permission error cannot break a `shark run` invocation. This is a directly
   reusable mechanics precedent for F04's new writer, even though its own
   output target (`events.jsonl`) and trigger (OTel span export) are
   unrelated.

9. `RunOptions.Progress` has exactly one production call site across the
   whole codebase (`grep -rn "\.Progress\s*=" internal/` → only
   `run.go:337`). F04's fix is therefore fully contained to one file: the
   callback's signature and controller-side invocation need no change.

## Decisions

1. **Rigor stays STANDARD**, matching the pre-recorded complexity score
   (11/27). The defect is a display bug confined to two print statements and
   a missing gate in one file; the new work is one small, fail-soft writer
   with two directly reusable precedents already in `internal/runner`. The
   cross-boundary and alternatives analysis a COMPLEX-tier feature would
   need already exists upstream in `architecture.md` and
   `E40-cross-epic-map.md` (ADR-001, ADR-004, X-08) and is cited here rather
   than re-derived.

2. **Fix the JSON-mode gate and both print sites together**, reaffirming the
   epic report's Decision 4 at feature scope: unwrap the ticker and
   `opts.Progress` from `if !cli.GlobalConfig.JSON` (routing to stderr, not
   stdout, in JSON mode) and replace `normalizedKey` with `update.EntityKey`
   at both `run.go:328` and `:348` in the same change. Finding 2 confirms no
   controller-side change is needed to support this.

3. **Extend F04's scope to also fix `run.go:275`'s stdout leak** (Finding 3),
   or explicitly carve it out with a stated reason if it is deferred — same
   file, same command, same defect class (a warning message ungated by JSON
   mode), and directly relevant to F04's own "stdout parses as a single JSON
   document" acceptance criterion even though ADR-001 keeps bench itself off
   the `--worktree` path.

4. **Build the new writer beside `internal/runner/transcript.go`**, not
   inline in `internal/cli/commands/run.go`, reusing its `.shark/runs/<run_id>/`
   path convention and directory/file permission modes, and reusing
   `file_jsonl_exporter.go`'s open/encode/fail-soft mechanics. This keeps
   `run.go` a thin command wrapper per `.claude/rules/architecture.md` and
   avoids inventing a third JSONL-writing convention alongside the two that
   already exist in this codebase.

5. **Do not route the new channel through `internal/runner/logging.go`'s
   slog events.** Reuse only their field-naming vocabulary (`entity_key`,
   `run_id`, `status`, `phase`, `agent_type`, `provider`) for the new NDJSON
   schema so the two channels stay legible side by side, but keep the
   transport separate and unconditional — this is the epic's own already-
   recorded architecture decision (Finding 7), not a new one this report is
   proposing.

6. **Name both "heartbeats" explicitly in any downstream spec** (Finding 5):
   the stage-scoped heartbeat is the progress ticker (`run.go:321`, 10s),
   never the claim/lease heartbeat (`startRunHeartbeat`, `svc.TTL()/3`).
   Stating this in the spec's own vocabulary section prevents an
   implementation from silently wiring display cadence to lease-renewal
   cadence.

7. **No additional RunID-threading work is needed.** Because `run_id` already
   propagates unchanged to cascade children (Finding 4), the "one log file
   per run, including cascade children" property in feature.md's Scope item
   4 requires only the EntityKey fix in Decision 2 — call this out
   explicitly in the spec so an implementer doesn't add unnecessary
   plumbing.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/research-report.md` (epic-level report)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/architecture.md` (§"Run liveness contract", ADR-001, ADR-004)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-interaction-map.md` (I-03 row and "Sequencing note on I-03")
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-cross-epic-map.md` (row X-08, "No E23 row" note)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/uat-plan.md` (UAT-05, UAT-06, line 37 latency claim)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md` (line 101, Phase 1 sizing)
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/research-report.md` (sibling report, style precedent)
- `internal/cli/commands/run.go` (full read — lines 275, 320, 328, 337–349, 396–430 claim lease, 620–645 heartbeat)
- `internal/cli/commands/run_logging.go` (`emitRunStart`/`emitRunEnd`)
- `internal/runner/controller.go` (`RunOptions`, `RunProgress`, `RunResult`, `StageLog`, `Run()` method lines 351+, cascade `childOpts` at line 601)
- `internal/runner/logging.go` (`run.stage.*` slog event vocabulary)
- `internal/runner/transcript.go` (`.shark/runs/<run_id>/` path owner, byte-contract format)
- `internal/observability/file_jsonl_exporter.go` (JSONL append + fail-soft pattern)
- `internal/config/config.go` (`ObservabilityConfig`, `CaptureAgentTranscripts`, `GetLogTruncateBytes`)
- `internal/services/claim_service.go` (`DefaultClaimTTL = 15 * time.Minute`, line 22 — code-level confirmation for Finding 5)
- `grep -n "fmt.Printf\|fmt.Fprintf" internal/cli/commands/run.go` (confirms line 275 is the only ungated stdout warning beyond the human-readable `outputRunResult` path, which is itself correctly gated behind `!cli.GlobalConfig.JSON`)
- `internal/sharkdata/default_data/research/recipes.yaml` — the actual v2 universal recipe catalog read; the path named in the dispatch prompt (`shark-data/research/recipes.yaml`) does not exist in this repo (per project memory, the embedded copy under `internal/sharkdata/default_data/` is canonical and `shark-data/` is a deployed, overwritable copy)
- `shark feature get E40-F04 --json` / `shark feature notes E40-F04` (decision note: COMPLEXITY STANDARD, score 11/27, 2026-08-06)
- `grep -rn "\.Progress\s*=" internal/` (single production call site for `RunOptions.Progress`)
- `grep -rn "run\.log" internal/` and `grep -rn "\.shark/runs" internal/` (no prior `run.log` writer exists)

RECOMMENDED OUTCOME: standard
