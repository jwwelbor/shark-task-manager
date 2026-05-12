---
feature_key: E32-F07-dispatch-instrumentation-file-jsonl-exporter-shark
epic_key: E32
title: Dispatch instrumentation — file_jsonl exporter + shark next/advance spans
description: Add an OTel file-JSONL span exporter and wrap shark next / shark advance in spans so dispatch tokens, latency, and quality are measurable per call. Enables side-by-side efficiency comparison between Shark 1.x and Shark 2.0.
size: M
---

# Dispatch instrumentation — file_jsonl exporter + shark next/advance spans

**Feature Key**: E32-F07

---

## Epic

- **Epic**: [E32 — Shark 2.0 — Single-Artifact Consolidation](../../epic.md)
- **External spec** (full picture across engine + harness + analyzer):
  `~/.claude/plans/let-s-do-all-three-cheerful-mitten.md`. The engine
  portion is reproduced below; harness and analyzer pieces are already
  built on the harness side (`hooks/shark2-trial-recorder.sh`,
  `scripts/shark2_stats.py`).

---

## Goal

### Problem

The Shark 2.0 rework changes the dispatch path: per-entity YAML
workflows, inline-rendered prompts, unified `shark-data/` layout. We need
to verify the rework actually improves efficiency — but we have no
per-call telemetry. The May 2026 trial on `wormwoodGM` (see
`docs/plan/changes-needed-for-shark2.md`) revealed three blocking
defects, including a class of bug (unresolved `<…>` placeholders in
rendered prompts) that would be invisible without per-dispatch
instrumentation.

### Solution

Emit one OTel **span** per `shark next` / `shark advance` invocation,
carrying the per-call attributes the harness and analyzer need: prompt
bytes, agent type, action, unresolved-placeholder count, status
transition source/target. Export those spans to a JSONL file at
`<project>/shark-data/.stats/events.jsonl` via a new `file_jsonl`
exporter wired into the existing observability subsystem. Surface
zero-tolerance defects (unresolved placeholders > 0) to stderr inline
so they cannot be missed during a trial.

### Impact

- **Tokens**: avg prompt bytes / dispatch directly measurable, with
  per-action breakdown.
- **Speed**: `shark next` latency per call; pairs with harness
  `subagent_start`/`subagent_stop` events for worker wall-time per task.
- **Quality**: surviving `<…>` placeholders count is a hard pass/fail
  defect signal; status reversals via `shark.advance` spans + history
  give rework / block / human-intervention rates.

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: file_jsonl exporter is registered and writes spans**

- **Given** a project with `.sharkconfig.json` containing
  `observability: {enabled: true, tracing_enabled: true, exporter:
  "file_jsonl"}`
- **When** the user runs any `shark next <key>` against that project
- **Then** `<project>/shark-data/.stats/events.jsonl` contains a
  well-formed JSON object on its own line
- **And** that object has `span_name: "shark.next"`, a non-empty
  `trace_id`, a numeric `duration_ms`, and an `attrs` block with at
  least `entity_key`, `entity_type`, `status`, `action`, `prompt_bytes`

**Scenario 2: zero-tolerance defects are surfaced inline**

- **Given** a `shark next <task-key>` invocation
- **When** the rendered prompt contains at least one
  `<task-id>`/`<epic>`/`<feature>`-style placeholder
- **Then** the JSONL span records `attrs.unresolved_placeholders` > 0
- **And** the CLI also writes `[shark-stats] WARN: <key> has N
  unresolved placeholders` to stderr

**Scenario 3: shark advance spans capture status transitions**

- **Given** an entity in any non-terminal status
- **When** `shark status advance <key>` succeeds
- **Then** `events.jsonl` has a new line with `span_name:
  "shark.advance"` containing `from_status`, `to_status`, `actor`,
  `forced`, `had_rejection_note`

**Scenario 4: failure modes are fail-soft**

- **Given** `shark-data/.stats/` is read-only or non-existent
- **When** any shark command runs with `exporter: "file_jsonl"`
- **Then** the command still succeeds (exit 0)
- **And** no telemetry write breaks the CLI flow

---

## Requirements

### Functional Requirements

**REQ-F-001**: New file_jsonl OTel span exporter
- Implements `sdktrace.SpanExporter`. One JSON line per span.
- Path resolution via existing `FindProjectRoot()` at
  `internal/cli/root.go:268`. Target file:
  `<projectRoot>/shark-data/.stats/events.jsonl` (create dir if
  missing).
- Append-only. Errors swallowed (fail-soft — never break the CLI). Use
  the zero-value-safe pattern from `internal/observability/metrics.go:14`
  `CommandMetrics`.
- Maps to task **T-E32-F07-001**.

**REQ-F-002**: Provider switch case
- `internal/observability/provider.go:177` `buildTracerProvider`: add a
  `case "file_jsonl"` branch that constructs the new exporter and wraps
  it in `sdktrace.NewSimpleSpanProcessor` (immediate flush, no batching).
- `ObservabilityConfig.Exporter` (`internal/config/config.go:204`) is
  free-form `string`, no struct change required — just document the new
  valid value alongside `"stdout"` and `"otlp"`.
- Maps to task **T-E32-F07-002**.

**REQ-F-003**: shark.next span instrumentation
- `internal/cli/commands/next.go:101` `runNext` — wrap body in
  `tracer.Start(ctx, "shark.next")` / `span.End()`.
- Span attributes set after `outputNextJSON`: `entity_key`,
  `entity_type`, `status`, `action`, `agent_type`, `model`,
  `prompt_bytes` (= `len(resp.Prompt)`), `unresolved_placeholders` (count
  of `<[a-zA-Z][a-zA-Z0-9_-]*>` matches in `resp.Prompt`), `exit_status`.
- When `unresolved_placeholders > 0` also emit `[shark-stats] WARN:
  <entity_key> has N unresolved placeholders` to stderr.
- Maps to task **T-E32-F07-003**.

**REQ-F-004**: shark.advance span instrumentation
- `internal/cli/commands/status_group.go:362` `runStatusAdvance` — wrap
  in `tracer.Start(ctx, "shark.advance")` / `span.End()`.
- Span attributes: `entity_key`, `entity_type`, `from_status`,
  `to_status`, `actor` (env var `SHARK_ACTOR`, default `"cli"`),
  `forced`, `had_rejection_note`.
- Maps to task **T-E32-F07-004**.

### Non-Functional Requirements

**REQ-NF-001**: Performance overhead
- **Target**: dispatch overhead from the exporter < 5ms p95 per call.
- **Measurement**: existing `shark.cli.command.duration` histogram
  (`internal/observability/metrics.go:23`) before/after this feature
  lands.
- **Justification**: telemetry must not become the thing it's measuring.

**REQ-NF-002**: Fail-soft on telemetry failure
- Telemetry write errors never propagate to CLI exit code.
- Pattern: every write wrapped in `defer recover()` or explicit
  error-discard with `slog.Debug` (not `slog.Error`).

**REQ-NF-003**: Test isolation
- Tests using `t.TempDir()` + `shark-data/` layout (see
  `internal/cli/commands/next_test.go:17`) naturally isolate
  `.stats/events.jsonl` to the tmpdir. No global state pollution.

---

## Out of Scope

### Explicitly Excluded

1. **OTel metrics file exporter** — only span (tracing) exporter for
   now. Spans give us per-call data; metrics aggregate. The existing
   `CommandMetrics`/`DBMetrics` stay on their current exporter
   (`stdout`/`otlp`).
2. **A new `shark stats` CLI subcommand** — display is handled by the
   harness-side analyzer at `~/.claude/scripts/shark2_stats.py`. Adding
   a CLI frontend later is a follow-up if usage warrants.
3. **Retention / log rotation** — `events.jsonl` grows append-only. A
   `shark stats prune --older-than 30d` is a follow-up.
4. **Sampling** — default `sample_rate: 1.0` (every span). Tune later
   if trial volume gets noisy.
5. **`CaptureAgentTranscripts`** (already in `ObservabilityConfig` at
   `internal/config/config.go:212`) — out of scope here; worker output
   bytes are captured harness-side via the trial hook.

### Alternative Approaches Rejected

**Alternative 1: Use OTLP exporter to a local collector that writes JSONL**
- **Why rejected**: adds a collector binary as a runtime dependency.
  Heavy for trial telemetry. The file_jsonl exporter is ~80 lines of Go.

**Alternative 2: Skip OTel; build a parallel side-channel JSONL writer**
- **Why rejected**: duplicates the existing observability infra.
  Engineers would have to learn two telemetry stacks. OTel spans
  naturally model "one dispatch step = one event with attributes".

**Alternative 3: Use OTel metrics histograms (`shark.cli.command.duration` + new attributes)**
- **Why rejected**: metrics aggregate over windows. We need per-call
  granularity for the analyzer's join with harness events on
  `entity_key`. Spans fit; histograms don't.

---

## Tasks

| Key | Title | Size |
|---|---|---:|
| **T-E32-F07-001** | Add file_jsonl OTel span exporter + unit test | S |
| **T-E32-F07-002** | Wire file_jsonl into buildTracerProvider exporter switch | XS |
| **T-E32-F07-003** | Wrap shark next in shark.next span with unresolved_placeholders check | M |
| **T-E32-F07-004** | Wrap shark status advance in shark.advance span | S |

Suggested order: 001 → 002 → 003 → 004 (each builds on the previous).
001 and 002 together enable end-to-end span emission with no spans yet
being recorded. 003 and 004 are independent of each other after 002
lands.

---

## JSONL event schemas

### `shark.next` span line

```json
{
  "ts": "2026-05-11T14:32:01.123Z",
  "span_name": "shark.next",
  "trace_id": "...",
  "span_id": "...",
  "duration_ms": 47,
  "exit_status": "ok",
  "attrs": {
    "entity_key": "T-E01-F41-003",
    "entity_type": "task",
    "status": "in_development",
    "action": "spawn_agent",
    "agent_type": "developer",
    "model": "sonnet",
    "prompt_bytes": 27800,
    "unresolved_placeholders": 19,
    "shark_version": "dev (c057a68)"
  }
}
```

### `shark.advance` span line

```json
{
  "ts": "2026-05-11T14:35:10.001Z",
  "span_name": "shark.advance",
  "trace_id": "...",
  "span_id": "...",
  "duration_ms": 22,
  "exit_status": "ok",
  "attrs": {
    "entity_key": "T-E01-F41-003",
    "entity_type": "task",
    "from_status": "in_development",
    "to_status": "completed",
    "actor": "cli",
    "forced": false,
    "had_rejection_note": false
  }
}
```

---

## Existing functions / utilities to reuse

| Path | Why |
|---|---|
| `internal/cli/root.go:268` `FindProjectRoot()` | Resolve `<projectRoot>/shark-data/.stats/` consistently. |
| `internal/observability/provider.go:38` `InitProvider` | The OTel SDK init we hook into. No signature change. |
| `internal/observability/noop.go:18` `NoopProvider` | Fallback when stats disabled or init fails. Means failures never break the CLI. |
| `internal/observability/metrics.go:14` `CommandMetrics` | Zero-value-safe pattern — copy for the new exporter's nil guards. |
| `internal/templates/orchestrator_renderer.go:289` `GetOrchestratorEngine` | Already used by `next.go:202` — no change needed here, but note that the renderer is what produces the prompt being measured. |

---

## Verification

### Unit / package

1. **`internal/observability/file_jsonl_exporter_test.go`** — round-trip:
   construct exporter pointing at tmpdir, emit two synthetic spans,
   reread JSONL, assert two non-empty lines, each parseable as JSON,
   each containing the expected key set.
2. **Provider test** — `.sharkconfig.json` containing
   `observability.exporter=file_jsonl` initializes successfully via
   `InitProvider` and produces a tracer that records spans.

### Integration smoke (manual)

```bash
cd /tmp && rm -rf shark-stats-test && mkdir shark-stats-test && cd shark-stats-test
shark init
cat > .sharkconfig.json <<'JSON'
{"observability":{"enabled":true,"tracing_enabled":true,"exporter":"file_jsonl"}}
JSON
shark validate
shark next E01 --json 2>/dev/null || true   # error is fine; we want a span
cat shark-data/.stats/events.jsonl          # expect ≥ 1 JSON line, well-formed
```

### Cross-stack with the harness analyzer

```bash
# After this feature lands and a trial runs:
python3 ~/.claude/scripts/shark2_stats.py --project /home/jwwel/projects/wormwoodGM --feature E01-F41
```
Expect:
- `Dispatches captured` reflects the count of `shark.next` spans for
  that feature's entities
- `Avg prompt bytes / dispatch` and `Avg shark next latency (ms)` are
  populated (not `—`)
- `Unresolved placeholders (total)` is 0 once B2 in
  `docs/plan/changes-needed-for-shark2.md` is also fixed

### Full pipeline trial

Re-run the wormwoodGM trial after this feature and the Shark 2.0
dispatch fixes (separate feature, see
`docs/plan/changes-needed-for-shark2.md`) both land. Compare against
the F40-on-1.x baseline already captured in
`~/.claude/dev-artifacts/shark2-trial/e01-f40-*.log`. The deliverable
is the comparison table produced by `shark2_stats.py --compare`.

---

## Dependencies & Integrations

- **E23 — OpenTelemetry Observability**: completed. This feature
  extends, doesn't modify, the OTel SDK already wired in
  `internal/observability/`.
- **Harness side** (already shipped, `~/.claude` branch `shark2`):
  - `hooks/shark2-trial-recorder.sh` — captures harness events to
    `~/.claude/dev-artifacts/shark2-trial/hooks.jsonl`.
  - `dev-artifacts/shark2-trial/config.json` — gates the trial hook.
  - `scripts/shark2_stats.py` — analyzer that joins engine spans (this
    feature's output) + harness events + entity history.
- **Blocks**: efficiency comparison reporting for the Shark 2.0 epic
  rollout. Without this feature, the rework cannot be quantitatively
  evaluated.

---

*Last updated*: 2026-05-11
