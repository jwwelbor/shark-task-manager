# Shark Bench — Design

**Epic**: E40 · **Status**: agreed approach (architect consultation, 2026-08-05)

> **Isolation corrected to match ADR-001, Q001 (resolved):** the original
> design below prescribed `shark run --worktree`; the per-run steps in §1 now
> read `--workdir` against a harness-owned fixture-repo checkout instead,
> because `--worktree` force-removes the tree before the harness regains
> control, which post-run checks need alive. [architecture.md](architecture.md#run-lifecycle-and-isolation-contract)
> and E40-F02's feature.md remain the authoritative sources; this document is
> otherwise retained as the historical record of the original design.

Benchmark harness that measures the effectiveness of a shark workflow configuration and detects the effect of config changes (model, effort, prompt, per-step assignments). Modeled on the transferable mechanics of SWE-bench / Aider polyglot / terminal-bench: execution-based test oracles, fresh isolation per attempt, per-step cost/latency capture, repetition with variance reporting, and paired per-task A/B comparison.

---

## 1. Architecture

**Shark bench is a harness *around* shark, not a feature inside it.** `shark run` is the execution engine — it already:

- drives every entity type (task, feature, epic, bug, change-card) through its workflow, cascading feature→tasks and epic→features via nested run controllers, each child holding its own claim + work_session;
- dispatches agents headlessly (`claude ... --output-format json`, `internal/runner/claude_dispatcher.go`);
- supports `--worktree` (isolated git worktree per top-level run, force-removed on return — rejected for bench post-run checks by ADR-001), `--workdir` (agent-process cwd override — the path bench actually uses), and `--json` (RunResult with per-stage `StageLog{status, action, agent_type, provider, duration_ns, exit_code}`), with run_id correlation;
- blocks workers from self-advancing status via `--disallowedTools` — measurement integrity for free.

**A "config variant" is a workflow YAML bundle** (per-step `provider`/`model`/`effort`/`skills`/`prompt` already exist in the schema). No Go changes are needed to *express* variants.

### Per run (task × variant × rep)

1. Provision a fresh scratch shark project (`scripts/shark-scratch-env.sh`) + a harness-owned fixture-repo checkout at a pinned commit (not a `shark run --worktree`). Install the variant workflow bundle (scratch init uses the embedded bundle by default — bench must point `workflow_config` at the variant).
2. Seed the corpus task's entities via shark CLI (capture assigned keys from create responses — never specify keys).
3. `timeout <cap> shark run <key> --json --workdir <fixture-checkout>` with `CaptureAgentTranscripts: true` (ADR-001: harness-owned isolation via `--workdir`, not `--worktree` — see [architecture.md](architecture.md#run-lifecycle-and-isolation-contract)).
4. Collect: RunResult stdout, per-stage transcripts (`.shark/runs/<run_id>/<n>-<status>-<provider>.log`), `work_sessions`/`entity_history` from the scratch DB, and post-run checks in the `--workdir` fixture checkout (oracle tests, quality gates, LOC).
5. Emit one JSONL artifact per run (manifest: task id, variant id, rep, commit SHAs, exact model IDs from `modelUsage`; records: per-stage + rollup). Artifacts are the source of truth; a small aggregator produces paired-comparison reports. No new shark DB tables.

### Instrumentation gaps in core (Phase 2; Phase 1 needs zero Go changes)

Phase 1 works untouched because the claude JSON envelope (with `usage`, `total_cost_usd`, `duration_api_ms`, `num_turns`, `modelUsage`) is already persisted per stage to the transcript files — it is received but never decoded.

- **G1 (S)** — decode the envelope in ClaudeDispatcher; extend StageLog with `tokens_in/out`, `cache_read/creation`, `cost_usd`, `api_duration_ms`, `num_turns`, `model_id`, `worker_session_id`. Reuse the unmerged `E27-F15-cross-session-usage-tracking` branch (already parses `usage`/`total_cost_usd`) — audit before landing.
- **G2 (S)** — record per-stage released outcome + `outcome_source ∈ {marker, default_pass, exit_fail}` in StageLog. The worker's `recommended outcome:` marker is parsed then discarded today (`controller.go:894-906`). `outcome_source` is the scaffold-compliance column (Aider's "well-formed rate" analog) — separates harness-contract failures from reasoning failures.
- **G3 (S)** — `--stage-timeout` flag; no timeout exists today, only ctx cancellation. Interim: bench wraps in `timeout(1)`, records outcome=timeout.
- **G4 (deferred, P3)** — codex usage parity; `codex exec` is dispatched as plain text with no usage envelope.
- **Verify before P2 rollups**: whether cascade children's StageLogs surface in the parent's `--json` RunResult or need run_id-correlated collection.

**Resolved blocker (2026-08-05)**: `shark run TD-###` stalled at `in_progress` — `check_or_resume` was grouped with pause in the run controller but dispatched by `shark next` (**B051**, now completed). Tech-debt benching is unblocked; it remains Phase 2 by phasing, not by blocker.

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
| **Change-card** | machine-checkable acceptance predicate (required at admission; CCs without one are excluded) | task metrics; LOC reported, not judged | |
| **Tech-debt** | P2P only (behavior preservation) + structural predicate ("debt is gone": lint-rule count drop, dependency removed, complexity threshold) | time/tokens; LOC expected net-negative | Phase 2 (B051 fixed) |
| **Feature** | union of child oracles + feature-level integration set | planning-step tokens/time; gate rejections (task_review, code_review, qa, approval); tasks_generated | child metrics summed; planning quality proxied by artifacts + downstream rejection rate (no LOC for planning steps) |
| **Epic** | feature rollup + epic acceptance set | refinement/design/decomposition tokens/time; feature_review rejections | Phase 3 (cost) |

**Feature modes**: **Mode A** — pre-seeded child tasks, start at `active`: isolates cascade + gates from planning variance. **Mode B** — start at `draft`; the task_generation agent creates children: measures decomposition quality. A first; B once A's noise band is known.

**Per-child attribution in cascades**: each child has its own claim/work_session (time windows) and its own StageLogs/transcripts — tokens/time/rejections attribute exactly. LOC is exact at feature level (`git diff` base→final); per-child LOC is best-effort by mapping commit timestamps into child session windows.

---

## 4. Per-metric mechanics

All post-run checks execute in the run's worktree; results land in the run's JSONL record.

| Metric | How, concretely |
|---|---|
| **Execution time** | (a) `stages[].duration_ns` from `shark run --json`; (b) envelope `duration_api_ms` (API vs harness overhead); (c) harness monotonic t0/t1 around the invocation; cross-check `shark task sessions --json`. |
| **Tokens / cost** | Parse the JSON envelope from each stage transcript (`.shark/runs/<run_id>/…`, between the STDOUT markers) → `usage.*`, `total_cost_usd`, `modelUsage` (exact model IDs into the manifest). Exact-in-StageLog after G1. |
| **Rejections** | *Definition: gate routes work backward (caught in-workflow).* Status re-entry after a review-type stage in `stages[]` = one rejection attributed to that gate; `rework_loops` = re-entry count. Cross-check `entity_history` backward transitions + `work_sessions.outcome`. Exact after G2. |
| **Defects** | *Definition: escapes to terminal status.* At terminal: inject held-back F2P tests → `go test -run '<F2P regex>' -count=1` (still-failing = unresolved); full suite diffed against the base-SHA test ledger → newly-red = `p2p_regressions[]`. P2: `-race` sweep. P3: adversarial reviewer files bugs → `defects_posthoc`. |
| **Code quality** | `make fmt && git diff --exit-code` → `fmt_clean`; `go vet` → `vet_ok`; `golangci-lint --out-format json` diffed against base-SHA lint ledger → `lint_new_issues` (only *new* issues count); `make test` → `tests_pass`. Rubric reviewer is P3, reported alongside, never instead. |
| **LOC** | `git add -A -N && git diff --numstat <base_sha>` → added/deleted, prod vs `*_test.go` split, `files_touched`. |

---

## 5. Comparison method

- **3 reps** per task×config (1 is untrustworthy — order-of-magnitude run-to-run variance is documented for agentic harnesses; 10 is cost-prohibitive). Raise reps only for decisions that matter.
- **Paired per-task comparison** (same task, variant vs baseline), never cross-aggregate. The Phase 1 baseline's run-to-run spread *is* the deliverable: it defines the noise band any config delta must clear. Deltas inside the band are reported as "no detectable effect."
- Pin exact model IDs in every manifest; never compare across unpinned model versions.

---

## 6. Phasing

- **Phase 1 — baseline (M total; harness itself needs zero Go changes)**: fixture repo + ~10 screened tasks/bugs with oracles (E40-F01, M); harness — provision, seed, invoke, parse RunResult + transcripts + worktree checks → JSONL (E40-F02, M); baseline report with noise band (E40-F03, S); `shark run` liveness — stderr progress events in `--json` mode, stage-scoped heartbeats, per-run log at `.shark/runs/<run_id>/run.log` (E40-F04, S — the one Phase 1 item touching Go; today `--json` runs are silent until completion, which unattended bench batches can't tolerate).
- **Phase 2 — config matrix (M)**: G1 (via E27-F15 branch) + G2 + G3 (S each); variant bundles + matrix runner with per-run budget caps (S); paired comparison report (M); feature Mode A then B; tech-debt (B051 fixed); optional `entity_history` `outcome`+`session_id` columns (S, severable).
- **Phase 3 (L — break down before starting)**: epics; SWE-bench Verified slice; rubric reviewer calibrated against the deterministic gate; codex parity (G4); post-hoc defect window; corpus rotation policy.

---

## 7. Risks

1. **Variance swamps config effects** (top risk): 10×3 detects coarse deltas (~20pp pass rate, large token shifts), not subtle prompt tweaks. Mitigate with paired comparison, published noise bands, targeted corpus/rep growth.
2. **Oracle quality**: weak tests convert "wrong code" into "pass". Mitigate with the reference-solution admission gate + mandatory P2P set.
3. **Cost of repetition**: multi-step × matrix × reps compounds. Mitigate with step/turn budget caps, cheap-model smoke tier before full-matrix runs, matrix runs only on knobs under active decision.
4. **Scaffold-compliance confound**: a worker failing to emit the outcome marker is not a reasoning failure — track `outcome_source` as its own column or comparisons misattribute harness bugs to models.
5. **Model-version drift**: record exact model IDs per run; treat unpinned comparisons as invalid.
