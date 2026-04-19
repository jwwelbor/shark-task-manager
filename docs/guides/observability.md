# Observability Developer Guide

Shark Task Manager includes an OpenTelemetry (OTel) observability subsystem that provides structured logging, distributed tracing, and metrics. The subsystem is **disabled by default** — existing users without the `observability` key in their `.sharkconfig.json` are entirely unaffected.

---

## What Is Available

### Structured Logging

When observability is enabled, Shark replaces the default logger with a structured `slog` handler that writes to `stderr`. Output can be JSON (default) or plain text. Every log record carries a `service.name` attribute.

When observability is disabled, the logger is set to a discard handler that suppresses all output below a level above `ERROR`, so no log noise reaches the terminal.

### Distributed Tracing

Shark wraps CLI command execution and key internal operations in OpenTelemetry spans. Spans carry standard semantic convention attributes (`service.name`, `service.version`) and are exported via the configured exporter.

The W3C Trace Context and Baggage propagators are installed globally, so trace context can be propagated across process boundaries if needed.

### Metrics

Two sets of metric instruments are available:

**CLI command metrics** (`shark.cli.command.*`):
- `shark.cli.command.duration` (histogram, ms) — execution time per command
- `shark.cli.command.invocations` (counter) — total invocations, tagged with `command` and `status` (ok/error)
- `shark.cli.command.errors` (counter) — error count, tagged with `command` and `error_type`

**Database query metrics** (`shark.db.query.*`):
- `shark.db.query.duration` (histogram, ms) — query execution time, tagged with `operation` and `table`
- `shark.db.query.errors` (counter) — query error count

All metric instruments have a safe zero value: if the meter provider is a no-op, all `Record`/`Add` calls are no-ops with no allocation overhead.

---

## Configuration

Observability is configured via the `observability` key in `.sharkconfig.json`. The zero value of every field means "use the default", so the minimal enabling configuration is:

```json
{
  "observability": {
    "enabled": true
  }
}
```

### Full Configuration Schema

```json
{
  "observability": {
    "enabled": false,
    "tracing_enabled": false,
    "metrics_enabled": false,
    "log_level": "info",
    "log_format": "json",
    "log_file": "",
    "exporter": "stdout",
    "otlp_endpoint": "",
    "otlp_protocol": "grpc",
    "service_name": "shark-task-manager",
    "sample_rate": 0,
    "capture_agent_transcripts": false,
    "log_truncate_bytes": 4096
  }
}
```

### Configuration Fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `enabled` | bool | `false` | Master switch. When false, no OTel SDK is initialized and no network connections are made. |
| `tracing_enabled` | bool | `false` | Enable span creation and export. Requires `enabled: true`. |
| `metrics_enabled` | bool | `false` | Enable metric recording and export. Requires `enabled: true`. |
| `log_level` | string | `"info"` | Minimum log level. Accepted values: `debug`, `info`, `warn`/`warning`, `error`. Unrecognized values fall back to `info`. |
| `log_format` | string | `"json"` | Log output format. `"json"` (default) or `"text"`. |
| `log_file` | string | `""` | Optional file destination for structured logs. Empty string logs to stderr. Relative paths are resolved against the project root. Missing parent directories are created (0755); the file is opened append-mode (0644). On open failure, logs fall back to stderr automatically. The file handle is closed during CLI shutdown. Ignored when `enabled: false`. |
| `exporter` | string | `"stdout"` | Export destination. `"stdout"` writes to stderr; `"otlp"` sends to an OTLP/gRPC endpoint. |
| `otlp_endpoint` | string | `"localhost:4317"` | OTLP receiver address (host:port). Used only when `exporter` is `"otlp"`. |
| `otlp_protocol` | string | `"grpc"` | OTLP transport protocol. Currently only `"grpc"` is supported. |
| `service_name` | string | `"shark-task-manager"` | Service name embedded in all telemetry. |
| `sample_rate` | float64 | `0` | Trace sampling ratio between 0.0 and 1.0 exclusive. `0` (or any value outside that range) means sample all traces (`AlwaysSample`). |
| `capture_agent_transcripts` | bool | `false` | Opt-in forensic capture of full agent stdout/stderr under `.shark/runs/{run_id}/`. See [Agent Transcript Capture](#agent-transcript-capture-capture_agent_transcripts). |
| `log_truncate_bytes` | int | `4096` | Cap on `stderr` and `stdout_tail` bytes emitted on `run.stage.error` events. See [Error-Event Truncation](#error-event-truncation-log_truncate_bytes). |

---

## `/run` Command Observability

When `enabled: true`, the `shark /run` command emits structured slog events (`run.start`, `run.stage.start`, `run.stage.dispatch`, `run.stage.complete`, `run.stage.transition`, `run.stage.error`, `run.end`) so that every invocation produces a grep-friendly record of what ran, what it returned, and how long it took. Two `observability` keys control the forensic depth of those events without changing the default event set: `capture_agent_transcripts` and `log_truncate_bytes`.

### Agent Transcript Capture (`capture_agent_transcripts`)

**Default**: `false` (disabled). No transcript files are written and no `transcript_path` attribute appears on any event.

**When to enable**: Turn this on while diagnosing a flaky agent, when the truncated stderr/stdout tail in `shark.log` is not enough, or when you need an after-the-fact audit trail of exactly what a `claude` or `codex` subprocess produced.

**Directory layout and file format** (REQ-F-012):

When enabled, every agent dispatch in a `/run` invocation writes one file at:

```
.shark/runs/{run_id}/{stage_n}-{status}-{provider}.log
```

- `{run_id}` is the per-invocation identifier that also appears as `run_id` on every slog event from that run (Story 5). Each run gets its own subdirectory.
- `{stage_n}` is the 1-based stage counter within the run (so the first dispatch is `1-…`, the second is `2-…`, etc.).
- `{status}` is the current status that drove the dispatch (for example `in_development`).
- `{provider}` is the agent provider key (for example `anthropic` or `codex`).

File contents are written verbatim in the following format (no trailing newline after `<stderr>`):

```
COMMAND: <cmd>
EXIT: <code>
DURATION: <ms>ms
---STDOUT---
<stdout>
---STDERR---
<stderr>
```

**Permissions**: the run directory is created with mode `0755`; each transcript file with mode `0644`.

**Event correlation**: on success, `run.stage.complete` (and, on dispatch failures, `run.stage.error`) carries a `transcript_path` attribute whose value is the project-relative path of the file. Cross-reference the `run_id` attribute on the slog event with the directory name to locate the forensic record for a given stage.

**Non-fatal write failures**: if the transcript directory or file cannot be created (permissions, read-only filesystem, disk full), the `/run` invocation is **not** aborted. The runner emits a single `run.transcript.warning` event the first time a write fails in the run, then sets a run-scoped latch that suppresses all further transcript write attempts for the remainder of that run. Subsequent `run.stage.complete` and `run.stage.error` events for that run therefore omit `transcript_path`. Each new `/run` invocation starts fresh with the latch cleared.

**Storage and privacy**: `.shark/runs/` is git-ignored by default (see the project's shipped `.gitignore`). Transcripts may contain source code, internal file paths, and the full instruction argv passed to the agent — treat the directory as local-only forensic data. There is no automatic rotation; operators should prune old run directories manually or via external tooling.

### Error-Event Truncation (`log_truncate_bytes`)

**Default**: `4096` bytes. A value of `0` (or negative) is treated as the default.

**Scope — error events only**: this setting caps only the `stderr` and `stdout_tail` attributes emitted on `run.stage.error` events (REQ-F-011). `stderr` is head-truncated (the first N bytes are retained so the first error message is visible); `stdout_tail` is tail-truncated (the last N bytes are retained so the output immediately preceding the exit is visible). When truncation actually occurs on an event, a `truncated: true` attribute is added; when the captured bytes already fit, the attribute is omitted.

**Does not apply to the successful-dispatch `command` field**: `run.stage.dispatch` events are emitted on every dispatch, including successful ones, and carry a `command` attribute containing the full shell-equivalent invocation. That attribute is capped by a separate, spec-baked 1024-byte limit (`dispatchCommandMaxBytes`) under REQ-F-010 — **not** by `log_truncate_bytes`. The 1024-byte cap is intentional and not configurable: the successful-dispatch event is on a hot path and must stay within a ~1 KB envelope regardless of agent-instruction size. Raising or lowering `log_truncate_bytes` has no effect on that field.

**Tuning**: raise `log_truncate_bytes` (for example to `16384`) when the default truncation is cutting off the root-cause portion of an agent's stderr. Lower it (for example to `1024`) in log-volume-sensitive deployments. If you need the full, untruncated output, enable `capture_agent_transcripts` instead — the on-disk transcript is never truncated.

### Example — Forensic `/run` debugging

Enable transcript capture and raise the error-path cap while diagnosing:

```json
{
  "observability": {
    "enabled": true,
    "log_level": "info",
    "log_format": "json",
    "capture_agent_transcripts": true,
    "log_truncate_bytes": 16384
  }
}
```

See [Observability Configuration Reference — `capture_agent_transcripts`](./observability-config-reference.md#capture_agent_transcripts) and [`log_truncate_bytes`](./observability-config-reference.md#log_truncate_bytes) for the full field reference and additional example configurations.

---

## Environment Variable Overrides

Environment variables take precedence over values in `.sharkconfig.json`. They are applied at runtime before any provider or logger is initialized.

| Environment Variable | Config Field Overridden | Example |
|---------------------|------------------------|---------|
| `SHARK_OTEL_ENABLED` | `enabled` | `SHARK_OTEL_ENABLED=true` |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otlp_endpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT=collector:4317` |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `otlp_protocol` | `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` |
| `OTEL_SERVICE_NAME` | `service_name` | `OTEL_SERVICE_NAME=my-shark` |
| `SHARK_LOG_LEVEL` | `log_level` | `SHARK_LOG_LEVEL=debug` |
| `SHARK_LOG_FORMAT` | `log_format` | `SHARK_LOG_FORMAT=text` |
| `SHARK_LOG_FILE` | `log_file` | `SHARK_LOG_FILE=/var/log/shark.log` |

`OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_EXPORTER_OTLP_PROTOCOL` follow the standard OpenTelemetry environment variable specification, making it straightforward to integrate Shark into existing OTel pipelines.

---

## Default Behavior

When the `observability` key is absent from `.sharkconfig.json`, or when `enabled` is `false`:

- No OTel SDK (`TracerProvider`, `MeterProvider`) is initialized.
- No goroutines are started, no file handles are opened, no network connections are attempted.
- The global logger is set to a discard handler — no log output is produced.
- The `ShutdownFunc` returned by `InitProvider` is a no-op that returns `nil` immediately.
- All metric `Record`/`Add` calls are no-ops via the OTel no-op providers.

There is zero runtime overhead for users who do not opt in.

---

## Usage Examples

### Enable stdout tracing for local debugging

Write spans directly to stderr. This is the simplest way to verify instrumentation is working without running a collector.

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "exporter": "stdout",
    "log_level": "debug",
    "log_format": "text"
  }
}
```

Run any command and span data will appear on stderr:

```bash
shark task get E07-F01-001
```

Spans are written by `SimpleSpanProcessor`, so output appears immediately when each span ends (no buffering).

### Enable OTLP exporter for production

Send traces and metrics to an OTLP-compatible collector (Jaeger, Grafana Tempo, Honeycomb, etc.):

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "metrics_enabled": true,
    "exporter": "otlp",
    "otlp_endpoint": "otel-collector.internal:4317",
    "service_name": "shark-prod",
    "sample_rate": 0.1
  }
}
```

Setting `sample_rate` to `0.1` samples 10% of traces, which is appropriate for high-frequency CLI usage. Set it to `0` (or omit it) to sample everything.

The OTLP exporter uses `BatchSpanProcessor` for traces, which batches and sends spans asynchronously to minimize latency impact on CLI commands.

### Override endpoint via environment variable

When running in CI or a container where the collector address varies, override the endpoint without modifying `.sharkconfig.json`:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 shark task list
```

### Configure log levels

For verbose debug logging during development:

```bash
SHARK_LOG_LEVEL=debug shark status E07-F01
```

For production, keep the default `info` level or set `warn` to reduce volume:

```json
{
  "observability": {
    "enabled": true,
    "log_level": "warn"
  }
}
```

### Enable observability via environment variable only

Enable observability without modifying `.sharkconfig.json` at all:

```bash
SHARK_OTEL_ENABLED=true SHARK_LOG_LEVEL=info shark task list
```

This is useful in CI environments where config files should not be modified.

### Write structured logs to a file

To persist structured log output across runs, configure a `log_file` destination. Shark writes newline-delimited JSON records in append mode, so multiple invocations accumulate in the same file.

```json
{
  "observability": {
    "enabled": true,
    "log_level": "info",
    "log_format": "json",
    "log_file": "./logs/shark.log"
  }
}
```

Run any command and the structured log records are written to the file:

```bash
shark task list
cat logs/shark.log
```

Example output (one JSON object per line):

```json
{"time":"2026-04-18T15:04:05.123456789Z","level":"INFO","msg":"command started","service.name":"shark-task-manager","command":"task list"}
{"time":"2026-04-18T15:04:05.234567890Z","level":"INFO","msg":"command completed","service.name":"shark-task-manager","command":"task list","duration_ms":111}
```

Key behaviors:

- **Relative paths** (`./logs/shark.log`) are resolved against the project root (the directory containing `.sharkconfig.json`), not the process working directory.
- **Missing parent directories** are created automatically with permissions `0755`.
- **Append mode** (`O_APPEND | O_CREATE | O_WRONLY`, perms `0644`) means existing log content is preserved across runs.
- **Fallback to stderr** — if the file cannot be opened (directory path, permission denied, NUL byte in path, etc.), shark logs to stderr instead and continues running. The CLI is never blocked by log-file failures.
- **Lifecycle** — the file handle is owned by the observability container and closed during CLI shutdown. `Close()` is idempotent.
- **Disabled short-circuit** — when `enabled: false`, `log_file` is ignored entirely and no file handle is opened, preserving the zero-overhead contract.

Override the destination per-run without editing the config:

```bash
SHARK_LOG_FILE=/var/log/shark/$(date +%F).log shark status
```

An empty `SHARK_LOG_FILE` is treated as "not set" and does not override the config value — use a different env var setting or edit `.sharkconfig.json` to explicitly disable the file destination.

---

## Viewing Telemetry Output

All observability output writes exclusively to **stderr**, never stdout. This preserves the integrity of `--json` output and shell pipelines.

### Quick Start — See Traces in Your Terminal

The fastest way to see what shark emits:

```bash
# One-liner: enable everything, see it on stderr
SHARK_OTEL_ENABLED=true ./bin/shark task list
```

To capture telemetry separately from normal output:

```bash
# Normal CLI output on stdout, traces + logs on stderr redirected to file
SHARK_OTEL_ENABLED=true ./bin/shark task list 2>otel.log
cat otel.log
```

To see structured JSON logs with debug detail:

```bash
SHARK_OTEL_ENABLED=true SHARK_LOG_LEVEL=debug SHARK_LOG_FORMAT=json ./bin/shark status 2>debug.log
cat debug.log | python3 -m json.tool  # pretty-print
```

### Viewing Traces with Jaeger

[Jaeger](https://www.jaegertracing.io/) provides a web UI for exploring distributed traces.

**With Podman:**

```bash
# Start Jaeger all-in-one
podman run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  docker.io/jaegertracing/all-in-one:latest
```

**With Docker:**

```bash
docker run -d --name jaeger \
  -p 4317:4317 \
  -p 16686:16686 \
  jaegertracing/all-in-one:latest
```

Then configure shark to send traces there:

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "metrics_enabled": true,
    "exporter": "otlp",
    "otlp_endpoint": "localhost:4317",
    "log_level": "info",
    "log_format": "json"
  }
}
```

Or via environment variables (no config file changes):

```bash
SHARK_OTEL_ENABLED=true \
  OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
  ./bin/shark task list
```

Open **http://localhost:16686** to browse traces in the Jaeger UI. Select "shark-task-manager" from the Service dropdown.

To stop the collector:

```bash
podman stop jaeger && podman rm jaeger   # or docker stop/rm
```

### Viewing Traces with Grafana + Tempo

For a more complete stack with metrics dashboards:

**With Podman:**

```bash
# Grafana Tempo (trace backend)
podman run -d --name tempo \
  -p 4317:4317 \
  -p 3200:3200 \
  docker.io/grafana/tempo:latest \
  -config.file=/etc/tempo/tempo.yaml

# Grafana (UI)
podman run -d --name grafana \
  -p 3000:3000 \
  docker.io/grafana/grafana:latest
```

**With Docker:**

```bash
docker run -d --name tempo \
  -p 4317:4317 -p 3200:3200 \
  grafana/tempo:latest -config.file=/etc/tempo/tempo.yaml

docker run -d --name grafana \
  -p 3000:3000 \
  grafana/grafana:latest
```

Open **http://localhost:3000**, add Tempo as a data source (`http://tempo:3200`), then explore traces.

### Running an OTel Collector

For production or when you need to fan out telemetry to multiple backends, use the OpenTelemetry Collector:

**With Podman:**

```bash
podman run -d --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  docker.io/otel/opentelemetry-collector-contrib:latest
```

**With Docker:**

```bash
docker run -d --name otel-collector \
  -p 4317:4317 \
  -p 4318:4318 \
  otel/opentelemetry-collector-contrib:latest
```

Point shark at it:

```bash
SHARK_OTEL_ENABLED=true \
  OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317 \
  ./bin/shark status
```

The collector can be configured to export to Jaeger, Zipkin, Prometheus, Grafana Cloud, Honeycomb, Datadog, or any OTLP-compatible backend via its config file.

---

## Troubleshooting

### No output when `exporter` is `stdout`

Verify that `tracing_enabled` (or `metrics_enabled`) is also set to `true`. Setting `enabled: true` alone does not start any signal export; each signal type must be individually enabled.

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true
  }
}
```

### OTLP exporter fails to connect

Check that the collector is reachable at the configured endpoint:

```bash
# Test connectivity (replace host:port as needed)
nc -zv localhost 4317
```

The OTLP exporter uses `WithInsecure()` and does not perform TLS by default. For production deployments requiring TLS, configure a TLS-terminating proxy in front of the collector endpoint.

If `InitProvider` returns an error, Shark falls back to the no-op provider rather than crashing. Check application logs or stderr for the error message.

### Log output not appearing

Confirm `enabled: true` is set. When `enabled` is `false`, the logger discards all output. Also verify the log level is not filtering out the messages you expect — if `log_level` is `error`, only error-level records appear.

### Spans appear out of order

The `stdout` exporter uses `SimpleSpanProcessor`, which writes spans immediately when they end. Nested child spans end before their parents, so output appears in reverse nesting order. This is expected behavior for the stdout exporter.

### Environment variable not taking effect

Environment variables are read once at provider initialization, which happens at CLI startup. Changing them in the same shell session after a command has started has no effect. Verify the variable is set in the environment that runs the `shark` binary:

```bash
printenv SHARK_OTEL_ENABLED
printenv SHARK_LOG_LEVEL
```

---

## Related Documentation

- [Observability Configuration Reference](./observability-config-reference.md) — complete field reference and example configurations
- [Configuration Reference](../cli-reference/README.md) — full `.sharkconfig.json` documentation
