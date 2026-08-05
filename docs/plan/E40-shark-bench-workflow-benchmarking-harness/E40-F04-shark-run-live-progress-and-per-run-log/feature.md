---
feature_key: E40-F04-shark-run-live-progress-and-per-run-log
epic_key: E40
title: shark run live progress and per-run log
description: shark run gives no liveness signal in the paths that matter: --json mode (used by all agent/skill/bench invocations) suppresses the heartbeat ticker and per-stage progress entirely and emits nothing until the final RunResult; the plain-mode heartbeat lacks stage/agent context; cascade child progress prints the parent key (run.go closure uses normalizedKey instead of update.EntityKey); and run events go only to the global shark.log, so there is no per-run log to tail. Add stderr progress events in --json mode (stdout stays a single clean RunResult), stage-scoped heartbeats (entity, stage status, agent/provider, stage elapsed + total elapsed), correct child-entity labeling, and a per-run log file under .shark/runs/<run_id>/ written unconditionally.
---

# shark run live progress and per-run log

**Feature Key**: E40-F04

---

## Goal

A `shark run` can take many minutes per stage with no way to tell "working" from "hung". Fix liveness in all invocation modes and leave a per-run log on disk. This also serves the bench (E40-F02/F03): long unattended bench batches need a tailable signal per run, and B051-style silent stalls become visible instead of looking like slow runs.

## Current behavior (verified)

- **`--json` mode is fully silent until completion** — `internal/cli/commands/run.go` wraps both the 10s heartbeat ticker and the `opts.Progress` callback in `if !cli.GlobalConfig.JSON`; stdout gets one RunResult at the end. This is the mode every agent/skill/bench invocation uses.
- **Plain mode** (since PR #134): per-stage "Processing …" lines + a 10s "[processing] <key> still running (elapsed)" ticker — but the ticker has no stage/agent context, and stage lines print the top-level `normalizedKey`, so cascade child progress is mislabeled as the parent (children inherit `opts.Progress`; the closure ignores `update.EntityKey`).
- **Run events** (run.start / stage / run.end slog) go only to the global observability log (`./shark.log`, config-gated); per-stage transcripts only with `CaptureAgentTranscripts`. No per-run log exists.

## Scope

1. **Liveness in `--json` mode via stderr**: emit progress events (NDJSON, one object per line: ts, run_id, entity_key, stage status, action, agent/provider, event ∈ {stage_start, heartbeat, stage_end}, stage_elapsed_ms, total_elapsed_ms) to **stderr** while stdout remains exactly one RunResult document. Machine-consumers (bench, skills) get liveness without stdout corruption.
2. **Stage-scoped heartbeat in plain mode**: ticker line includes the entity currently executing (child key during cascades), stage status, agent/provider, stage elapsed and total elapsed.
3. **Correct child labeling**: progress printing uses `update.EntityKey`, not the closure's top-level key.
4. **Per-run log**: append human-readable stage events to `.shark/runs/<run_id>/run.log`, written unconditionally (not gated on observability config); print the path once at run start in both modes so users know what to tail.

## Acceptance Criteria

- [ ] `shark run <key> --json` emits stderr heartbeat/stage events at least every 10s while a stage runs; stdout parses as a single JSON document (existing consumers unbroken)
- [ ] Plain-mode heartbeat identifies entity, stage, agent, stage elapsed, total elapsed
- [ ] During a feature cascade, progress lines show the child task key being worked
- [ ] `.shark/runs/<run_id>/run.log` exists after every run (including failures/timeouts) with stage start/end and outcomes; path printed at run start
- [ ] A hung agent process is distinguishable at the prompt: heartbeats continue with growing stage-elapsed and unchanged stage

## Out of Scope

- Streaming agent stdout/stderr live to the terminal (transcripts already capture it post-stage; revisit if heartbeats prove insufficient)
- Progress UI (spinners, TTY rewriting) — plain append-only lines only
- G2 outcome instrumentation (Phase 2 StageLog work; this feature is display + logging only)

## Success Metric

An observer watching a multi-stage run (plain or --json) can answer "which entity/stage is running and for how long" at any moment from the terminal or by tailing the run log.

---

*Last Updated*: 2026-08-05
