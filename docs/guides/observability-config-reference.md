# Observability Configuration Reference

Complete reference for the `observability` section of `.sharkconfig.json`.

---

## Schema

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
    "sample_rate": 0
  }
}
```

The `observability` key is optional. When absent, all observability is disabled with zero overhead.

---

## Field Reference

### `enabled`

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| JSON key | `enabled` |

Master on/off switch for the entire observability subsystem.

When `false`:
- No OTel SDK is initialized.
- No `TracerProvider` or `MeterProvider` is created.
- No network connections are made.
- No goroutines are started.
- The structured logger is replaced with a discard handler.
- All `tracing_enabled`, `metrics_enabled`, and other signal fields are ignored.

When `true`, the subsystem initializes based on the remaining fields. At least one of `tracing_enabled` or `metrics_enabled` must also be `true` to produce any exported signal data.

---

### `tracing_enabled`

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| JSON key | `tracing_enabled` |

Enables span creation and export via the configured exporter. Requires `enabled: true`.

When `true`, a `TracerProvider` is initialized and set as the global OTel tracer provider. CLI command spans and internal operation spans are created and exported.

---

### `metrics_enabled`

| Property | Value |
|----------|-------|
| Type | `bool` |
| Default | `false` |
| JSON key | `metrics_enabled` |

Enables metric recording and export via the configured exporter. Requires `enabled: true`.

When `true`, a `MeterProvider` is initialized and set as the global OTel meter provider. The following instruments are registered:

| Instrument | Type | Unit | Description |
|-----------|------|------|-------------|
| `shark.cli.command.duration` | Float64Histogram | ms | CLI command execution time |
| `shark.cli.command.invocations` | Int64Counter | — | Total CLI command invocations |
| `shark.cli.command.errors` | Int64Counter | — | CLI command error count |
| `shark.db.query.duration` | Float64Histogram | ms | Database query execution time |
| `shark.db.query.errors` | Int64Counter | — | Database query error count |

---

### `log_level`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"info"` |
| JSON key | `log_level` |
| Env override | `SHARK_LOG_LEVEL` |

Minimum severity level for the structured logger. Records below this level are discarded.

Accepted values (case-insensitive):

| Value | Numeric Level |
|-------|--------------|
| `debug` | -4 |
| `info` (default) | 0 |
| `warn` or `warning` | 4 |
| `error` | 8 |

Any unrecognized value falls back to `info`.

---

### `log_format`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"json"` |
| JSON key | `log_format` |
| Env override | `SHARK_LOG_FORMAT` |

Format of structured log records written to stderr.

Accepted values (case-insensitive):

| Value | Description |
|-------|-------------|
| `json` (default) | Newline-delimited JSON objects. Machine-parseable. |
| `text` | Human-readable `key=value` pairs. |

Any unrecognized value falls back to `json`.

---

### `log_file`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `""` (empty — log to stderr) |
| JSON key | `log_file` |
| Env override | `SHARK_LOG_FILE` |

Optional file destination for structured log records. When empty (the default), logs are written to stderr. When non-empty and `enabled` is `true`, structured log records are written to this file in addition to the normal initialization path.

**Path resolution:**

- **Absolute paths** (e.g., `/var/log/shark.log`) are used as-is.
- **Relative paths** (e.g., `./logs/shark.log` or `logs/shark.log`) are resolved against the **project root** (the directory containing `.sharkconfig.json`), not the current working directory. This keeps log file locations consistent regardless of where `shark` is invoked from.

**File behavior:**

- The file is opened in **append mode** (`O_APPEND | O_CREATE | O_WRONLY`) with permissions `0644`. Existing content is preserved across runs.
- If the **parent directory does not exist**, it is created with permissions `0755` before the file is opened.
- If the file cannot be opened (for example, a directory path, a permission denied error, or a path containing `\x00`), the logger **falls back to stderr** and the CLI continues running. The fallback is a safety measure — diagnostic observability must never prevent shark from functioning. The fallback condition is visible in the returned logger configuration.

**Lifecycle:**

- When `log_file` is set and observability is enabled, the opened file handle is owned by the observability container and closed during CLI shutdown via `ShutdownObservability`. Closing is idempotent — subsequent `Close()` calls return `os.ErrClosed` without side effects.
- When observability is **disabled** (`enabled: false`), the field is ignored and no file handle is opened — even if `log_file` is set. This preserves the zero-overhead contract for users who have not opted in.

**Security:**

- Because logs may contain sensitive contextual data, place log files in directories with appropriate access controls (for example, outside world-readable paths).
- The absolute path of a successfully opened log file is surfaced by `InitLoggerWithRoot`; avoid logging that path to stdout or including it in `--json` command output.

---

### `exporter`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"stdout"` |
| JSON key | `exporter` |

Destination for trace spans and metrics. Applies to both signals when both are enabled.

Accepted values (case-insensitive):

| Value | Description |
|-------|-------------|
| `stdout` (default) | Writes to stderr (never stdout). Uses `SimpleSpanProcessor` for traces (immediate output) and `PeriodicReader` for metrics. |
| `otlp` | Sends to an OTLP/gRPC endpoint. Uses `BatchSpanProcessor` for traces (asynchronous, lower latency impact) and `PeriodicReader` for metrics. |

Any value other than `stdout` or `otlp` causes `InitProvider` to return an error.

---

### `otlp_endpoint`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"localhost:4317"` |
| JSON key | `otlp_endpoint` |
| Env override | `OTEL_EXPORTER_OTLP_ENDPOINT` |

Address of the OTLP receiver in `host:port` format. Used only when `exporter` is `"otlp"`.

When the field is empty and `exporter` is `"otlp"`, the default `localhost:4317` is used.

The OTLP exporter connects without TLS (`WithInsecure()`). For TLS in production, place a TLS-terminating proxy in front of the collector and point this field at the proxy address.

---

### `otlp_protocol`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"grpc"` |
| JSON key | `otlp_protocol` |
| Env override | `OTEL_EXPORTER_OTLP_PROTOCOL` |

Transport protocol for the OTLP exporter. Currently only `"grpc"` is implemented. This field is reserved for future HTTP/protobuf support.

---

### `service_name`

| Property | Value |
|----------|-------|
| Type | `string` |
| Default | `"shark-task-manager"` |
| JSON key | `service_name` |
| Env override | `OTEL_SERVICE_NAME` |

Service name embedded in all telemetry signals as the `service.name` resource attribute (OTel semantic convention). Also added as a `service.name` attribute on every structured log record.

Use a unique value per deployment to distinguish telemetry from different environments or projects.

---

### `sample_rate`

| Property | Value |
|----------|-------|
| Type | `float64` |
| Default | `0` |
| JSON key | `sample_rate` |

Trace sampling ratio. Controls what fraction of traces are exported.

| Value | Behavior |
|-------|----------|
| `0` (default) | Sample all traces (`AlwaysSample`). |
| `0 < rate < 1.0` | `TraceIDRatioBased` sampler at the given ratio. For example, `0.1` samples approximately 10% of traces. |
| Any value outside `(0, 1.0)` | Sample all traces (`AlwaysSample`). |

This field has no effect when `tracing_enabled` is `false`.

---

## Environment Variable Reference

All environment variables are applied at runtime after reading `.sharkconfig.json`. They take precedence over file-based configuration.

| Environment Variable | Overrides Field | Accepted Values |
|---------------------|----------------|-----------------|
| `SHARK_OTEL_ENABLED` | `enabled` | `true`, `false`, `1`, `0` (any value accepted by `strconv.ParseBool`) |
| `SHARK_LOG_LEVEL` | `log_level` | `debug`, `info`, `warn`, `warning`, `error` |
| `SHARK_LOG_FORMAT` | `log_format` | `json`, `text` |
| `SHARK_LOG_FILE` | `log_file` | Absolute or project-root-relative file path; empty string is ignored |
| `OTEL_SERVICE_NAME` | `service_name` | Any string |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `otlp_endpoint` | `host:port` string |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `otlp_protocol` | `grpc` |

`OTEL_EXPORTER_OTLP_ENDPOINT` and `OTEL_EXPORTER_OTLP_PROTOCOL` conform to the [OpenTelemetry Environment Variable Specification](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/).

---

## Example Configurations

### Local development — stdout tracing

Inspect spans during development without running a collector. Output goes to stderr.

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "exporter": "stdout",
    "log_level": "debug",
    "log_format": "text",
    "service_name": "shark-dev"
  }
}
```

### CI pipeline — observability disabled

No changes needed. The default configuration (no `observability` key) disables all observability with zero overhead.

```json
{}
```

To enable minimal logging in CI without tracing overhead:

```json
{
  "observability": {
    "enabled": true,
    "log_level": "warn",
    "log_format": "json"
  }
}
```

Or via environment variables only, with no config file changes:

```bash
SHARK_OTEL_ENABLED=true SHARK_LOG_LEVEL=warn shark task list
```

### Production — OTLP with sampling

Send traces and metrics to a collector with 10% sampling to limit volume.

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "metrics_enabled": true,
    "exporter": "otlp",
    "otlp_endpoint": "otel-collector.internal:4317",
    "service_name": "shark-prod",
    "log_level": "info",
    "log_format": "json",
    "sample_rate": 0.1
  }
}
```

Override the endpoint via environment variable in deployments where the collector address is injected at runtime:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=otel-collector.svc.cluster.local:4317 shark status
```

### Full sampling — debugging a production issue

Temporarily sample all traces to capture every span. Restore `sample_rate` afterward.

```json
{
  "observability": {
    "enabled": true,
    "tracing_enabled": true,
    "exporter": "otlp",
    "otlp_endpoint": "otel-collector.internal:4317",
    "service_name": "shark-prod",
    "sample_rate": 0
  }
}
```

### File logging — persistent log history

Write structured log records to a file at a project-root-relative path. The parent directory is created automatically, and logs accumulate across runs.

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

Override the destination per-run without editing the config:

```bash
SHARK_LOG_FILE=/var/log/shark/today.log shark task list
```

If the configured file cannot be opened, shark automatically falls back to stderr and continues running — the CLI is never blocked by log-file failures.

---

## Related Documentation

- [Observability Developer Guide](./observability.md) — usage examples and troubleshooting
- [Workflow Profiles Guide](./workflow-profiles.md) — workflow configuration
- [CLI Reference](../cli-reference/README.md) — full command reference
