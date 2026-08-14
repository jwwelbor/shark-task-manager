# Shark Bench — Design

**Epic**: E40 · **Status**: Phase 1 delivered; lifecycle v2 boundaries assigned
2026-08-13

> **Isolation corrected to match ADR-001, Q001 (resolved):** the original
> design below prescribed `shark run --worktree`; the per-run steps in §1 now
> read `--workdir` against a harness-owned fixture-repo checkout instead,
> because `--worktree` force-removes the tree before the harness regains
> control, which post-run checks need alive. [architecture.md](architecture.md#run-lifecycle-and-isolation-contract)
> and E40-F02's feature.md remain the authoritative sources; this document is
> otherwise retained as the historical record of the original Phase 1 design.

> **Lifecycle v2 extension (2026-08-11):** E40-F05 through E40-F10 extend the
> benchmark from task/bug execution into reproducible feature, bug,
> change-card, and tech-debt lifecycles. The authoritative contracts are I-04
> through I-08 in [architecture.md](architecture.md) and
> [E40-interaction-map.md](E40-interaction-map.md).

> **Value-attribution extension (2026-08-13):** lifecycle v2 also separates
> provider work from coordination time, pins the exact reviewed candidate and
> gate policy, records artifact use and structured review findings, and compares
> feature QA with finish-feature deep review under independent and sequential
> policies. The full PR/CI/merge tail remains deferred.

Benchmark harness that measures the effectiveness of a shark workflow configuration and detects the effect of config changes (model, effort, prompt, per-step assignments). Modeled on the transferable mechanics of SWE-bench / Aider polyglot / terminal-bench: execution-based test oracles, fresh isolation per attempt, per-step cost/latency capture, repetition with variance reporting, and paired per-task A/B comparison.

---

## 1. Architecture

**Shark bench is a harness around Shark, not a second workflow engine.** Phase 1
uses `shark run` as its execution engine. It already:

- drives every entity type (task, feature, epic, bug, change-card) through its workflow, cascading feature→tasks and epic→features via nested run controllers, each child holding its own claim + work_session;
- dispatches agents headlessly (`claude ... --output-format json`, `internal/runner/claude_dispatcher.go`);
- supports `--worktree` (isolated git worktree per top-level run, force-removed on return — rejected for bench post-run checks by ADR-001), `--workdir` (agent-process cwd override — the path bench actually uses), and `--json` (RunResult with per-stage `StageLog{status, action, agent_type, provider, duration_ns, exit_code}`), with run_id correlation;
- blocks workers from self-advancing status via `--disallowedTools` — measurement integrity for free.

**A "config variant" is a workflow YAML bundle** (per-step `provider`/`model`/`effort`/`skills`/`prompt` already exist in the schema). No Go changes are needed to *express* variants.

### Phase 1 per run (task × variant × rep)

1. Provision a fresh scratch shark project (`scripts/shark-scratch-env.sh`) + a harness-owned fixture-repo checkout at a pinned commit (not a `shark run --worktree`). Install the variant workflow bundle (scratch init uses the embedded bundle by default — bench must point `workflow_config` at the variant).
2. Seed the corpus task's entities via shark CLI (capture assigned keys from create responses — never specify keys).
3. `timeout <cap> shark run <key> --json --workdir <fixture-checkout>` with `CaptureAgentTranscripts: true` (ADR-001: harness-owned isolation via `--workdir`, not `--worktree` — see [architecture.md](architecture.md#run-lifecycle-and-isolation-contract)).
4. Collect: RunResult stdout, per-stage transcripts (`.shark/runs/<run_id>/<n>-<status>-<provider>.log`), `work_sessions`/`entity_history` from the scratch DB, and post-run checks in the `--workdir` fixture checkout (oracle tests, quality gates, LOC).
5. Emit one JSONL artifact per run (manifest: task id, variant id, rep, commit SHAs, exact model IDs from `modelUsage`; records: per-stage + rollup). Artifacts are the source of truth; a small aggregator produces paired-comparison reports. No new shark DB tables.

### Lifecycle v2 per scenario

1. Load an admitted I-04 scenario package and provision the scratch Shark
   project, controlled fixture checkout, and inaccessible evaluator root.
2. For a feature scenario, run D01-D05 through the existing Shark Rider
   product-design action with the I-06 replay bundle. Other entity families
   record those stages as non-applicable.
3. Start from the scenario's created Shark entity. For each keyed dispatch,
   preserve the response, apply the recorded scheduling policy, claim the
   concrete entity, pass the rendered prompt unchanged, heartbeat, persist the
   semantic outcome, transition, and release.
4. Freeze an I-05 snapshot after every applicable stage and an I-07 lifecycle
   run record across the scenario root and every eligible descendant.
5. Evaluate structural correctness, calibrated artifact quality, held-back
   execution truth, and comparison identity into I-08.
6. Retain raw evidence and publish only after the explicit spend, pilot,
   identity, and oracle gates pass.

Lifecycle v2 does not call `shark run` as one opaque top-level operation because
replay, per-stage snapshots, and deterministic descendant scheduling require
host control. It reuses the same canonical Shark and Rider contracts and does
not reconstruct routing, prompts, claims, Questions, or workflow state.

### Phase 1 instrumentation gaps and lifecycle v2

Phase 1 works untouched because the claude JSON envelope (with `usage`, `total_cost_usd`, `duration_api_ms`, `num_turns`, `modelUsage`) is already persisted per stage to the transcript files — it is received but never decoded.

- **G1 (S)** — decode the envelope in ClaudeDispatcher; extend StageLog with `tokens_in/out`, `cache_read/creation`, `cost_usd`, `api_duration_ms`, `num_turns`, `model_id`, `worker_session_id`. E40-F06 owns the lifecycle-v2 evidence shape and X-09 reuse decision; E40-F08 writes the runtime values. Verify the current E27-F15 state before reuse.
- **G2 (S)** — record per-stage released outcome + `outcome_source ∈ {marker, default_pass, exit_fail}` in StageLog. The worker's `recommended outcome:` marker is parsed then discarded today (`controller.go:894-906`). `outcome_source` is the scaffold-compliance column (Aider's "well-formed rate" analog) — separates harness-contract failures from reasoning failures.
- **G3 (S)** — `--stage-timeout` flag; no timeout exists today, only ctx cancellation. Interim: bench wraps in `timeout(1)`, records outcome=timeout.
- **G4 (deferred, P3)** — codex usage parity; `codex exec` is dispatched as plain text with no usage envelope.
- **Verify before P2 rollups**: whether cascade children's StageLogs surface in the parent's `--json` RunResult or need run_id-correlated collection.

These StageLog improvements remain useful to the Phase 1 `shark run` surface,
but lifecycle v2 does not wait for an opaque parent rollup. E40-F08 records each
keyed dispatch directly in I-07. Missing required usage or model identity still
fails closed through I-05/I-08 rather than being guessed.

**Resolved blocker (2026-08-05)**: `shark run TD-###` stalled at `in_progress` — `check_or_resume` was grouped with pause in the run controller but dispatched by `shark next` (**B051**, now completed). Tech-debt benching is unblocked and enters lifecycle v2 through E40-F05 and E40-F08.

---

## 2. Corpus & oracles

- **Curated fixture repo** (small Go service with a real test suite) + corpus manifest: per task, an issue-style prompt, entity seed spec, held-back FAIL_TO_PASS tests, and PASS_TO_PASS set. Held-back tests live in the corpus dir, never in the repo the agent sees.
- **Admission gate** (SWE-bench Verified lesson — ~30% of raw mined instances are broken): every task validated by running the base commit (F2P red, P2P green) and a reference patch (both green) before admission.
- **Discriminative band** (Aider lesson): drop tasks every config aces or every config fails.
- **OSS realism deferred, not rejected**: Phase 3 adopts ~10–20 pre-screened SWE-bench Verified instances rather than mining our own issues. Contamination is constant across configs, so paired deltas stay valid.

---

## 3. Per-entity-type measurement matrix

| Entity | Oracle | Native metrics | Notes |
|---|---|---|---|
| **Task** | held-back F2P + P2P suite | all metrics per stage | rolls up → feature |
| **Bug** | author-written repro test as F2P + P2P | task metrics + `repro_confirmed` | |
| **Change-card** | machine-checkable acceptance predicate (required at admission; CCs without one are excluded) | task metrics; LOC reported, not judged | Lifecycle v2 |
| **Tech-debt** | P2P only (behavior preservation) + structural predicate ("debt is gone": lint-rule count drop, dependency removed, complexity threshold) | time/tokens; LOC expected net-negative | Lifecycle v2; B051 fixed |
| **Feature** | union of child oracles + feature-level integration set | D01-D05 and planning-stage quality, tokens/time, gate rejections, generated-task count, full lifecycle outcome | Lifecycle v2 starts with agent-generated children; pre-seeded Mode A remains a diagnostic option |
| **Epic** | feature rollup + epic acceptance set | refinement/design/decomposition tokens/time; feature-review rejections | Deferred after lifecycle v2 |

**Feature modes**: **Mode A** — pre-seeded child tasks, start at `active`: isolates cascade + gates from planning variance. **Mode B** — start at `draft`; the task_generation agent creates children: measures decomposition quality. A first; B once A's noise band is known.

**Per-child attribution:** the Phase 1 `shark run` surface has known flattened
StageLog and sibling-transcript collision limits for cascades (ADR-005). In
lifecycle v2, E40-F08 records each concrete dispatch, lease, transition, and
I-05 snapshot directly, so child time, usage, outcomes, and rework are explicit.
LOC remains exact at scenario level; per-child LOC is diagnostic only unless a
feature workflow proves a stable attribution method.

---

## 4. Per-metric mechanics

All Phase 1 post-run checks execute in the harness-owned `--workdir` checkout;
results land in the run's JSONL record. Lifecycle v2 execution checks use the
scenario adapter and write evaluator access plus results into I-08.

| Metric | How, concretely |
|---|---|
| **Execution time** | (a) `stages[].duration_ns` from `shark run --json`; (b) envelope `duration_api_ms` (API vs harness overhead); (c) harness monotonic t0/t1 around the invocation; cross-check `shark task sessions --json`. |
| **Tokens / cost** | Parse the JSON envelope from each stage transcript (`.shark/runs/<run_id>/…`, between the STDOUT markers) → `usage.*`, `total_cost_usd`, `modelUsage` (exact model IDs into the manifest). Exact-in-StageLog after G1. |
| **Rejections** | *Definition: gate routes work backward (caught in-workflow).* Status re-entry after a review-type stage in `stages[]` = one rejection attributed to that gate; `rework_loops` = re-entry count. Cross-check `entity_history` backward transitions + `work_sessions.outcome`. Exact after G2. |
| **Defects** | *Definition: escapes to terminal status.* At terminal: inject held-back F2P tests → `go test -run '<F2P regex>' -count=1` (still-failing = unresolved); full suite diffed against the base-SHA test ledger → newly-red = `p2p_regressions[]`. P2: `-race` sweep. P3: adversarial reviewer files bugs → `defects_posthoc`. |
| **Code quality** | `make fmt && git diff --exit-code` → `fmt_clean`; `go vet` → `vet_ok`; `golangci-lint --out-format json` diffed against base-SHA lint ledger → `lint_new_issues` (only *new* issues count); `make test` → `tests_pass`. Rubric reviewer is P3, reported alongside, never instead. |
| **LOC** | `git add -A -N && git diff --numstat <base_sha>` → added/deleted, prod vs `*_test.go` split, `files_touched`. |
| **Time attribution** | I-05 records non-overlapping provider-active, tool/test, queue/claim, replay/human-gate, retry/backoff, and unclassified intervals. Stage and lifecycle totals must reconcile before publication. |
| **Candidate identity** | For each code-producing or review stage, hash the base commit, candidate tree, binary diff, changed-path set, dirty and untracked manifest, and test suite. A matching branch or `HEAD` is not sufficient. |
| **Review findings** | Preserve raw `review-finding` fields in I-07; normalize and confirm them in I-08. Report emitted, unique, duplicate, recurrent, confirmed, unconfirmed, and downstream-escape findings by gate, severity, and defect class. Publish precision and recall only against seeded defects or another retained truth set. |
| **Artifact use** | Record typed producer and downstream consumer/access edges plus artifact size. An explicit empty consumer set means orphaned; a missing set means incomplete telemetry. |
| **Replayed human burden** | Record request/response counts and sizes, revisions, replay wait class, and unresolved gates for D01-D05. These are reproducible interaction proxies, not observed human minutes. |

---

## 5. Comparison method

### Phase 1 paired task comparison

- **3 reps** per task×config (1 is untrustworthy — order-of-magnitude run-to-run variance is documented for agentic harnesses; 10 is cost-prohibitive). Raise reps only for decisions that matter.
- **Paired per-task comparison** (same task, variant vs baseline), never cross-aggregate. The Phase 1 baseline's run-to-run spread *is* the deliverable: it defines the noise band any config delta must clear. Deltas inside the band are reported as "no detectable effect."
- Pin exact model IDs in every manifest; never compare across unpinned model versions.

### Lifecycle v2 evaluation and identity

- Freeze every applicable stage from a real Shark-generated input. Use the
  stage view to diagnose where a lifecycle regressed, not as a separate product
  baseline.
- Run deterministic structural checks and a calibrated, versioned LLM judge for
  applicable planning and decomposition artifacts. Keep judge rationale, score,
  prompt, model, configuration, usage, and cost.
- Treat the held-back execution oracle as the implementation-quality authority.
  Terminal status and worker self-report are not correctness evidence.
- Require uniform scenario and replay, fixture and adapter, Shark binary,
  installed content, rendered prompt, provider/model/effort, judge, reference,
  and resource-policy identity. Reject and retain every incompatible run.
- Require uniform candidate and workflow-policy identity for review comparisons:
  exact candidate/test snapshot, enabled gates, order, reviewer configuration,
  full review-bundle digest, and whether fixes are allowed.
- Compare QA and finish-feature deep review in two modes: independently against
  one frozen candidate with no intervening fixes, and sequentially in real gate
  order with every intervening candidate retained. Only the independent or a
  controlled policy comparison supports a causal gate-value claim.
- Keep quality, elapsed time, and cost as separate outcomes. Report paired
  deltas, unique confirmed finding yield, overlap, recurrence, downstream
  escapes, rework, and artifact use; do not publish one efficiency score.

---

## 6. Phasing

- **Phase 1 — completed:** E40-F01 through E40-F04 delivered the Go fixture
  corpus, single-run collector, baseline/noise-band report, and `shark run`
  liveness surface.
- **Lifecycle v2 — active:** E40-F05 establishes the scenario and adapter
  contract; E40-F06 and E40-F07 then specify dependency-independent contracts
  at live execution orders 6 and 7; E40-F08 drives the
  lifecycle; E40-F09 evaluates and validates identity; E40-F10 owns safe
  operator execution and retained publication.
- **After lifecycle v2:** epic roots, D06-D14, a SWE-bench Verified slice,
  codex usage parity where still missing, a post-hoc adversarial defect window,
  sprint scenarios, the PR/CI/merge delivery tail, and corpus rotation remain
  deferred.

---

## 7. Risks

1. **Variance swamps config effects** (top risk): 10×3 detects coarse deltas (~20pp pass rate, large token shifts), not subtle prompt tweaks. Mitigate with paired comparison, published noise bands, targeted corpus/rep growth.
2. **Oracle quality**: weak tests convert "wrong code" into "pass". Mitigate with the reference-solution admission gate + mandatory P2P set.
3. **Cost of repetition**: multi-step × matrix × reps compounds. Mitigate with step/turn budget caps, cheap-model smoke tier before full-matrix runs, matrix runs only on knobs under active decision.
4. **Scaffold-compliance confound**: a worker failing to emit the outcome marker is not a reasoning failure — track `outcome_source` as its own column or comparisons misattribute harness bugs to models.
5. **Model-version drift**: record exact model IDs per run; treat unpinned comparisons as invalid.
6. **Replay drift or leakage**: hash every response and evaluator reference;
   block live inputs and inspect actual dispatch-time visibility.
7. **Lifecycle truncation**: execute every eligible generated task or stop and
   invalidate the entire scenario at a named safety ceiling.
8. **Judge overreach**: calibrate against human scores and keep structural and
   execution-oracle truth separate.
