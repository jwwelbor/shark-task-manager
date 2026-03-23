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

---

## Related Documentation

- [Observability Developer Guide](./observability.md) — usage examples and troubleshooting
- [Workflow Profiles Guide](./workflow-profiles.md) — workflow configuration
- [CLI Reference](../cli-reference/README.md) — full command reference
