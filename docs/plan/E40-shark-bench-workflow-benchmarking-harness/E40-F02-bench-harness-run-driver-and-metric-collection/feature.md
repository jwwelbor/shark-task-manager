---
feature_key: E40-F02-bench-harness-run-driver-and-metric-collection
epic_key: E40
title: Bench harness: run driver and metric collection
description: Harness scripts (zero shark Go changes): provision scratch shark project via scripts/shark-scratch-env.sh, create a harness-owned fixture-repo checkout at the corpus base SHA (ADR-001), install a workflow-variant bundle, seed corpus entities via shark CLI (capture assigned keys), invoke 'timeout N shark run <key> --json --workdir <fixture-checkout>' with agent transcripts enabled, then collect RunResult stdout, per-stage transcript envelopes (tokens/cost/model IDs), scratch-DB work_sessions/entity_history, and post-run checks in the fixture checkout (F2P/P2P oracle, fmt/vet/lint/test vs base ledgers, git numstat LOC) into one JSONL artifact per run. Consumes: I-01, I-03. Produces: I-02. Consumes: X-07.
---

# Bench harness: run driver and metric collection

**Feature Key**: E40-F02

See [shark-bench-design.md](../shark-bench-design.md) §1 (run lifecycle) and §4 (per-metric mechanics).

---

## Goal

Execute one benchmark run (corpus task × config variant × rep) unattended and emit one complete JSONL record. Phase 1 constraint: **zero shark Go changes** — everything is collected from what `shark run` already produces (`--json` RunResult, per-stage transcripts with the undecoded claude JSON envelope, scratch-DB tables) plus post-run checks in the harness-owned fixture checkout.

Isolation is **harness-owned, not `--worktree`-owned (ADR-001, Q001)**: `shark run --worktree` force-removes the worktree on every return path before the harness regains control, so a live tree for post-run checks is never available. F02 instead creates its own fixture-repo checkout at the corpus base SHA and passes it to `shark run` via `--workdir`, keeping it alive after the run returns.

## Scope

1. **Provisioning**: scratch shark project (scripts/shark-scratch-env.sh) + a harness-owned fixture-repo checkout at the corpus base SHA (not a `shark run --worktree`); install the variant workflow bundle and point `workflow_config` at it (scratch init defaults to the embedded bundle — must be overridden); enable `CaptureAgentTranscripts`.
2. **Seeding**: create corpus entities via shark CLI from the manifest; capture assigned keys from create responses.
3. **Invocation**: `timeout <cap> shark run <key> --json --workdir <fixture-checkout>`; record harness-level t0/t1 and timeout-as-outcome.
4. **Collection**:
   - RunResult stdout → per-stage `duration_ns`, action, provider, exit codes; rejection inference via status re-entry after gate stages.
   - Stage transcripts (`.shark/runs/<run_id>/`, written under the scratch project root, not the fixture checkout) → parse the claude JSON envelope: `usage.*`, `total_cost_usd`, `duration_api_ms`, `num_turns`, `modelUsage` (exact model IDs → manifest).
   - Scratch DB → `work_sessions` (per-child time windows, outcomes), `entity_history` (backward transitions cross-check).
   - Post-run, in the `--workdir` fixture checkout: inject held-back F2P tests + run; full suite vs base test ledger (`p2p_regressions`); `make fmt`+diff, `go vet`, golangci-lint vs base lint ledger (`lint_new_issues`), `make test`; `git diff --numstat <base_sha>` (LOC, prod/test split).
5. **Artifact**: one JSONL record per run — manifest (task id, variant id, rep, SHAs, model IDs) + per-stage records + rollup. Artifacts are the source of truth; no shark DB tables.

## Acceptance Criteria

- [ ] A single command runs one (task, variant, rep) end-to-end unattended and emits a valid JSONL record
- [ ] All six metric families present per run: time (stage + API + harness wall), tokens/cost per stage, rejections by gate, oracle result + regressions, quality gates, LOC
- [ ] A hung worker is killed by the timeout wrapper and recorded as outcome=timeout, not a harness crash
- [ ] Live repo/DB untouched — everything happens in the scratch project and the harness-owned fixture checkout (guardrail hooks respected); `shark run` is invoked with `--workdir`, never `--worktree`

## Out of Scope

- Core instrumentation G1–G3 (StageLog usage/outcome fields, --stage-timeout) — Phase 2
- Matrix runner / multi-variant orchestration and comparison reports (E40-F03 aggregates; Phase 2 compares)
- Codex-dispatched steps (no usage envelope; G4, Phase 3)

## Success Metric

3 consecutive reps of the same corpus task complete unattended with complete, schema-valid records.

---

*Last Updated*: 2026-08-05
