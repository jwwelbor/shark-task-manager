---
research_schema: 2
entity_key: E40
entity_type: epic
recipe: universal
rigor: complex
categories:
  - backend
  - data
  - workflow_operations
  - documentation
related_work: true
---

# E40 research report: Shark Bench — workflow benchmarking harness

## Lifecycle v2 addendum (2026-08-11)

This report records the Phase 1 brownfield investigation. It remains the source
for E40-F01 through E40-F04, but its branch states and its claim that `shark run`
is the only execution path are historical snapshots. Verify those states again
inside each new feature workflow.

Lifecycle v2 adds E40-F05 through E40-F10 and reuses these current product
contracts:

| Contract | Current owner | E40 consumer | Research obligation |
|---|---|---|---|
| Provider usage and session metadata | E27-F15 | E40-F06/E40-F08 through X-09 | Verify the current decoder and real provider envelopes; fail closed on missing identity |
| Product-design action and derived progress | E36-F02 | E40-F07 through X-10 | Invoke the existing action and methodology; research only the benchmark replay seam |
| Keyed Rider execution, prompt provenance, and resume | E38-F07/E38-F09 | E40-F08 through X-11 | Verify the live `next`/claim/heartbeat/outcome/release procedure and preserve the rendered prompt unchanged |
| Canonical installed Shark-data content | E32-F04 | E40-F09 through X-12 | Determine a deterministic digest over installed workflows, prompts, skills, and agents |
| Durable Question lifecycle | E39-F04 | E40-F08 through X-13 | Verify replay-authorized response and unresolved-gate behavior without transcript-only decisions |

New feature research must also validate the controlled Python fixture, the
three-root evaluator-isolation model, stage snapshot schema, LLM-judge
calibration method, comparison-identity fields, and operator spend gates. The
accepted boundaries and stable shapes are I-04 through I-08 in
[architecture.md](architecture.md) and
[E40-interaction-map.md](E40-interaction-map.md).

The execution-path decision is phase-specific: Phase 1 keeps `shark run` for
the shipped task/bug baseline, while lifecycle v2 uses a host-side controller
over public Shark and Rider contracts so it can replay D01-D05 inputs and freeze
every stage. This controller must not create another workflow engine, claim
store, prompt assembler, or Question store.

## Scope

E40 builds a harness *around* `shark run`, not a feature inside it, to measure
whether a workflow configuration (per-step `provider`/`model`/`effort`/`skills`/
`prompt`) is effective and whether a change to it is an improvement, a
regression, or noise. Phase 1 (E40-F01–F04) must produce a screened fixture
corpus with execution-based oracles, a run driver that provisions/seeds/invokes
`shark run --json` against a harness-owned `--workdir` fixture checkout
(ADR-001, superseding this report's original `--worktree` assumption) and
collects per-run metrics into JSONL, a
baseline report whose deliverable is a published **noise band**, and the one
Phase-1 Go change — `shark run` liveness in `--json` mode.

The Phase 1 work was **COMPLEX** research: it spans four features with declared
producer/consumer handoffs (F01 corpus → F02 harness → F03 aggregation, plus
F04 as an orthogonal core-runner fix), it makes one Go-side change to a
still-actively-developed parent capability (`shark run`, owned by epic E22,
itself not yet closed), and it depends on external, undecoded process output
(the `claude --output-format json` envelope) whose exact shape this research
found to be **only partially corroborated** by existing code.

Terms used in this report: **StageLog** (per-stage record inside
`runner.RunResult`: status, action, agent_type, provider, duration_ns,
exit_code), **transcript** (the on-disk `.shark/runs/<run_id>/<n>-<status>-<provider>.log` file written when `CaptureAgentTranscripts` is enabled),
**envelope** (the raw JSON object `claude --output-format json` prints to
stdout, captured verbatim inside a transcript's `---STDOUT---` block),
**config variant** (an alternate workflow YAML bundle pointed to by
`workflow_config`), **oracle** / **admission gate** (execution-based FAIL_TO_
PASS + PASS_TO_PASS test check that admits or rejects a corpus item), **noise
band** (the Phase 1 baseline's own run-to-run spread per metric, over 3 reps —
the threshold any future config delta must clear), and **check_or_resume**
(the tech-debt `in_progress` orchestrator action implicated in B051).

## Research checklist

- [x] `scope_vocabulary` — Evidence: `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`, `shark-bench-design.md`, and the four feature files (`E40-F01`…`E40-F04`) define the corpus/harness/report/liveness vocabulary and the Phase 1/2/3 boundary.
- [x] `affected_implementation_or_contract` — Evidence: `internal/runner/controller.go` (StageLog/RunResult, action routing, `ActionCheckOrResume` now dispatched alongside `spawn_agent`), `internal/cli/commands/run.go` (`--worktree`/`--json` flags, the `Progress` callback and heartbeat ticker), `internal/runner/transcript.go` (the exact on-disk transcript byte format), `internal/runner/dispatcher.go` (`DefaultDisallowedTools`), and `internal/runner/claude_dispatcher.go` (the raw, currently-undecoded `--output-format json` invocation).
- [x] `related_work` — Evidence: epic **E22** (External Orchestration Runner, status `active` — the parent capability E40 harnesses), branch `E27-F15-cross-session-usage-tracking` (unmerged usage-envelope decoder, G1's reuse candidate), branch `origin/fix/B051-shark-run-tech-debt-check-resume` (unmerged fix for the tech-debt dispatch stall), and prior memory `project_shark_bench_e40.md`.
- [x] `pattern_contract` — Evidence: `internal/config/action/orchestrator.go` (`OrchestratorAction` step schema: `provider`/`model`/`effort`/`skills`/`instruction_template`), `internal/sharkdata/default_data/workflow/{task,feature,epic}.yaml` (`effort: low|high|xhigh` in production steps), and `scripts/shark-scratch-env.sh` (the mandatory scratch-provisioning + live-repo-guardrail pattern).
- [x] `dependency_impact` — Evidence: `internal/db/db.go` (`work_sessions`, `entity_history` schemas — already `entity_type`/`entity_key` generic), root `Makefile` (`fmt`/`vet`/`lint`/`test` targets the per-metric mechanics table depends on), and `internal/reporting/{scan_report,schema,formatters}.go` (existing report infrastructure, and why it does not apply here).
- [x] `cross_boundary_risks` — Evidence: branch `E27-F15-cross-session-usage-tracking`'s `claudeJSONResult` struct and `testdata/claude-usage-result.json` fixture (decodes `type/session_id/request_id/result/model/total_cost_usd/usage{...}` only — no `modelUsage`, `num_turns`, or `duration_api_ms`, leaving the design doc's and E40-F02's assumed field names unverified either way); branch `origin/fix/B051-shark-run-tech-debt-check-resume` (code-reviewed, DB status `completed`, but absent from `main` and from this `E40-shark-bench` branch); epic **E22** `active` status (foundation not frozen).
- [x] `alternatives` — Evidence: `shark-bench-design.md` §2 (curated fixture repo + admission gate vs. mining own OSS issues, and SWE-bench Verified deferred to Phase 3) and §3 (feature-benching Mode A pre-seeded children vs. Mode B agent-generated children).

## Capability map

| Capability | Brownfield evidence | Decision | E40 responsibility |
|---|---|---|---|
| `shark run` execution engine (dispatch, cascade, claim/work_session, `--workdir`, `RunResult`) | `internal/runner/controller.go`; `internal/cli/commands/run.go` | REUSE | Drive `shark run <key> --json` unmodified, against a harness-owned fixture checkout passed via `--workdir` (ADR-001) rather than `shark run --worktree`; the epic's own constraint forbids forking or reimplementing the controller. |
| Per-stage transcript capture | `internal/runner/transcript.go` (documented `COMMAND:`/`EXIT:`/`DURATION:`/`---STDOUT---`/`---STDERR---` byte format); `internal/config/config.go` `CaptureAgentTranscripts` | REUSE | Enable in the scratch config; parse the stable on-disk format harness-side rather than inventing a new capture path. |
| Claude JSON usage envelope | `internal/runner/claude_dispatcher.go` (appends `--output-format json`, does not decode it); branch `E27-F15-cross-session-usage-tracking` (decodes a narrower, test-verified field set) | EXTEND, with a field-name caveat | Phase 1 harness must verify exact envelope field names against a live captured transcript before relying on `modelUsage`/`num_turns`/`duration_api_ms` — no in-repo code corroborates those three names. Phase 2's G1 should reuse and audit E27-F15 rather than hand-roll a second decoder, and reconcile any field-name gap found here. |
| Workflow config-as-YAML step schema (`provider`/`model`/`effort`/`skills`/`prompt`) | `internal/config/action/orchestrator.go`; `internal/sharkdata/default_data/workflow/*.yaml` | REUSE | "Config variant" = an alternate workflow YAML bundle pointed to by `workflow_config`; no schema change is needed to express one. |
| Scratch-project provisioning + live-repo guardrail | `scripts/shark-scratch-env.sh`; `.sharkconfig.json` guardrail hook | REUSE | Every run provisions here; scratch `admin init` defaults to the embedded workflow bundle, so the harness must explicitly repoint `workflow_config` at the variant after provisioning — it will not happen automatically. |
| Self-advancement lockout (`DefaultDisallowedTools`) | `internal/runner/dispatcher.go` | REUSE | Corroborates "measurement integrity for free": dispatched workers cannot run `shark status advance`/`set-status`/`next-status` regardless of bench configuration. |
| `--json`-mode liveness / per-run log | `internal/cli/commands/run.go` (progress ticker + `Progress` callback both gated behind `if !cli.GlobalConfig.JSON`, lines ~320–351; both print sites use the closure's `normalizedKey` instead of `update.EntityKey`, lines 328 and 348) | NEW (an extension of `shark run` itself) | F04 is the one Phase-1 Go change: stderr NDJSON progress events in `--json` mode, corrected child-entity labeling at both print sites, and an unconditional `.shark/runs/<run_id>/run.log`. |
| Tech-debt `check_or_resume` dispatch (B051) | Branch `origin/fix/B051-shark-run-tech-debt-check-resume` (code-reviewed PASS, shark DB status `completed`) vs. `internal/runner/controller.go` on `main`/this branch (still groups `ActionCheckOrResume` with `pause`) | EXTEND, pending merge | Confirms tech-debt benching is unblocked in principle for Phase 2, but the fix exists only on an unmerged remote branch today. Phase 1 excludes tech-debt regardless of this; Phase 2 planning should track the merge, not assume it landed. |
| `work_sessions` / `entity_history` tables | `internal/db/db.go` | REUSE | Already `entity_type`/`entity_key` generic (not task-only); F02's collection step can query them directly with no migration. |
| `internal/reporting` package (`ScanReport`) | `internal/reporting/scan_report.go`, `schema.go`, `formatters.go` | CONTRADICTS (not applicable) | This is a fixed schema for docs/plan filesystem-scan reports, unrelated to benchmark rollups. F03's aggregator/report is new, bench-only tooling — consistent with the epic's "no new shark DB tables, JSONL is the store" constraint, not a repurposing of this package. |

## Findings

1. `shark run` already provides everything Phase 1 needs without Go changes
   except F04: cascade dispatch with per-child claim/work_session, isolation
   (later resolved to harness-owned `--workdir`, not `--worktree`, by ADR-001 —
   see architecture.md), a `--json` `RunResult` carrying `StageLog[]`, and a hard
   self-advancement lockout via `DefaultDisallowedTools`
   (`internal/runner/dispatcher.go:20-26`) — which, read directly, is the exact
   same command set this research task's own dispatch contract forbids running
   against E40 (`shark status advance*`, `*task next-status*`, `*status set*`,
   `*task set-status*`, `*feature next-status*`, `*epic next-status*`). The
   design doc's "measurement integrity for free" claim holds up.

2. The claude JSON envelope is genuinely undecoded today —
   `internal/runner/claude_dispatcher.go` only appends `--output-format json`
   to the command line and returns the raw stdout unmodified — and the on-disk
   transcript format is a documented, stable byte contract
   (`COMMAND:`/`EXIT:`/`DURATION:<ms>ms`/`---STDOUT---`/`---STDERR---`,
   `internal/runner/transcript.go:10-23`) that a harness-side parser can rely
   on without touching Go code.

3. The design doc's and E40-F02's assumed envelope field names —
   `modelUsage`, `num_turns`, `duration_api_ms` — are **unverified against a
   real envelope** by anything in this codebase. The only real parsing code
   and test fixture (`E27-F15-cross-session-usage-tracking` branch,
   `internal/runner/claude_dispatcher.go` diff and
   `testdata/claude-usage-result.json`) decode a different, narrower field
   set for its own purposes: `type`, `session_id`, `request_id`, `result`,
   `model`, `total_cost_usd`, and
   `usage{input_tokens, cache_read_input_tokens, cache_creation_input_tokens,
   output_tokens, reasoning_output_tokens}`. That branch not decoding
   `modelUsage`/`num_turns`/`duration_api_ms` is evidence of what its author
   needed at the time, not evidence that the live envelope lacks those
   fields — it simply cannot serve as corroboration either way. F02's
   harness-side parser should independently confirm the real field names
   (e.g., against one live captured transcript) before depending on the
   design doc's names as fact.

4. Two capabilities the epic's own phasing leans on exist only as **unmerged
   branches**, not committed history: `E27-F15-cross-session-usage-tracking`
   (G1's reuse candidate) and `origin/fix/B051-shark-run-tech-debt-check-resume` (the tech-debt `check_or_resume` dispatch fix). Both carry
   `completed`/reviewed status in the shark database, but neither is present
   on `main` nor on this `E40-shark-bench` branch (which branches from the
   current `main` HEAD, `d2d90955`). Phase 2 planning that treats either
   capability as "already fixed in the codebase" would be planning against
   code that does not yet exist on any branch this epic builds from.

5. F04's described defect is reproduced exactly in the current source:
   `internal/cli/commands/run.go` gates both the 10s progress ticker and the
   `opts.Progress` callback behind `if !cli.GlobalConfig.JSON` (line 320), and
   both print sites reference the outer `normalizedKey` closure variable
   rather than the per-update `update.EntityKey` (lines 328 and 348) — so a
   cascading run's child-task progress prints the parent's key even once
   stderr liveness ships. A correct F04 fix must change both the JSON-mode
   gate and both print sites.

6. `work_sessions` and `entity_history` (`internal/db/db.go`) are already
   entity-type-generic (`entity_type`/`entity_key` columns, not task-only) —
   exactly what F02's collection step needs to cross-check per-child claim
   windows and backward-transition rejections, with no database migration.

7. The workflow step schema (`internal/config/action/orchestrator.go`'s
   `OrchestratorAction`) already carries every "config variant" knob the epic
   needs — `provider`, `model`, `effort` (`low`/`medium`/`high`/`xhigh`),
   `skills`, and the prompt template — and production workflow YAML already
   varies `effort` per step. A config variant is only an alternate YAML tree;
   this corroborates the design doc's central "zero Go changes" claim for
   F01–F03.

8. `shark run`'s home epic, **E22**, is itself still `active` (6 features
   `draft`, 1 `assessment`, 7 `completed`), not a closed, stable capability.
   E40 is building a harness "around" a foundation that can still change
   shape; the design doc's implicit assumption of a frozen `RunResult`/
   `StageLog` contract is a risk to monitor rather than a settled fact.

## Decisions

1. **Proceed as designed.** `shark run` is confirmed as the sole execution
   engine; F01–F03 require zero Go changes and F04 is the sole Phase-1 Go
   change — both borne out by direct inspection of `controller.go`, `run.go`,
   `transcript.go`, and `dispatcher.go`.

2. **Verify the envelope field names before writing F02's parser.** Do not
   copy `modelUsage`/`num_turns`/`duration_api_ms` from the design doc as
   given; confirm them against one live-captured transcript (or the current
   `claude` CLI's own documentation) and reconcile against
   `E27-F15-cross-session-usage-tracking`'s narrower, test-verified field set
   before F02 is implemented.

3. **Track the two unmerged branches as external dependencies, not landed
   capabilities.** `E27-F15-cross-session-usage-tracking` and
   `origin/fix/B051-shark-run-tech-debt-check-resume` should be named
   explicitly in Phase 2 planning (G1, tech-debt benching) with their merge
   state as a precondition, and the design doc's "audit before landing" note
   on E27-F15 should be extended to reconcile the field-name gap found here.

4. **F04 must fix both halves of the confirmed defect together.** Unwrap the
   heartbeat/`Progress` reporting from the `if !cli.GlobalConfig.JSON` gate
   (routing to stderr, not stdout, in JSON mode) and replace `normalizedKey`
   with `update.EntityKey` at both print sites
   (`internal/cli/commands/run.go:328,348`) in the same change — fixing only
   the stderr-emission half would still mislabel cascade children.

5. **F02's collection step queries `work_sessions`/`entity_history` directly;
   it does not touch `internal/reporting`.** No schema migration is needed for
   collection, and the existing `ScanReport` infrastructure is a fixed,
   unrelated domain object — F03's aggregator and report generator are new,
   bench-only tooling by design, consistent with the epic's "no new shark DB
   tables" constraint.

6. **Treat E22 as an active, not-yet-closed foundation.** Design/
   implementation for F02 should pin the exact `RunResult`/`StageLog` fields
   the harness parses (a small canary/compatibility check against a real
   `shark run --json` invocation) so a future E22 change fails loud in the
   bench harness rather than silently corrupting collected metrics.

## Sources

- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/epic.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/shark-bench-design.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F01-benchmark-corpus-v1-fixture-repo-and-screened-task/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F02-bench-harness-run-driver-and-metric-collection/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F03-baseline-report-and-noise-band/feature.md`
- `docs/plan/E40-shark-bench-workflow-benchmarking-harness/E40-F04-shark-run-live-progress-and-per-run-log/feature.md`
- `internal/sharkdata/default_data/research/recipes.yaml`
- `internal/runner/controller.go` (StageLog, RunResult, action routing, `ActionCheckOrResume`)
- `internal/runner/transcript.go` (on-disk transcript format)
- `internal/runner/dispatcher.go` (`DefaultDisallowedTools`)
- `internal/runner/claude_dispatcher.go` (raw `--output-format json` invocation)
- `internal/cli/commands/run.go` (`--worktree`, `--json`, `Progress` callback, heartbeat ticker)
- `internal/config/action/orchestrator.go` (`OrchestratorAction` step schema)
- `internal/sharkdata/default_data/workflow/task.yaml`, `feature.yaml`, `epic.yaml`, `tech-debt.yaml`
- `scripts/shark-scratch-env.sh`
- `internal/db/db.go` (`work_sessions`, `entity_history` schemas)
- `internal/reporting/scan_report.go`, `schema.go`, `formatters.go`
- `Makefile` (`fmt`/`vet`/`lint`/`test` targets)
- `shark get B051 --json` (status `completed`, code-review note); branch `origin/fix/B051-shark-run-tech-debt-check-resume` diff against `main`
- branch `E27-F15-cross-session-usage-tracking` (`internal/runner/claude_dispatcher.go` diff, `testdata/claude-usage-result.json`)
- `shark epic get E22 --json` (status `active`)
- `docs/guides/route-based-workflow.md`
- Memory: `project_shark_bench_e40.md`

RECOMMENDED OUTCOME: pass
