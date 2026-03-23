# E23-F01: Observability Foundation Package — Test Plan

**Feature Key:** E23-F01-observability-foundation-package
**Epic:** E23 — OpenTelemetry Observability (tracing, metrics, structured logging)
**Date:** 2026-03-22
**Status:** Draft

---

## 1. Scope and Objectives

This test plan covers all unit tests, integration tests, edge cases, and coverage targets for the `internal/observability/` package and the `ObservabilityConfig` additions to `internal/config/`. It is organized by file, matching the implementation structure defined in the specification.

### Test Files to Create

| Implementation File | Test File |
|---|---|
| `internal/observability/provider.go` | `internal/observability/provider_test.go` |
| `internal/observability/logger.go` | `internal/observability/logger_test.go` |
| `internal/observability/noop.go` | `internal/observability/noop_test.go` |
| `internal/observability/metrics.go` | `internal/observability/metrics_test.go` |
| `internal/config/config.go` (ObservabilityConfig) | `internal/config/config_observability_test.go` |

### Coverage Targets

| Package | Target | Rationale |
|---|---|---|
| `internal/observability` — happy paths | 80%+ | Service logic coverage gate |
| `internal/observability` — error paths | 100% | All error returns must be tested |
| `internal/config` — ObservabilityConfig parsing | 100% | All nine fields and zero-value defaults |
| `applyEnvOverrides` (all branches) | 100% | Each env var override is a critical behavior contract |

---

## 2. Testing Rules and Conventions

1. **No real database.** The observability package has no database dependency — this is straightforward.
2. **No real OTLP collector.** All tests must pass without a running collector. OTLP exporter tests verify lazy-connect behavior, not data delivery.
3. **Table-driven tests** for multi-scenario coverage (log level parsing, env override branches, metric attribute logic).
4. **Use `testify/assert` and `testify/require`** throughout.
5. **Stdout capture required** for stdout-protection tests: redirect `os.Stdout` to a `bytes.Buffer` and assert zero bytes after `InitProvider`/`InitLogger` calls.
6. **Stderr capture required** for logger format tests: use a `bytes.Buffer` as the writer target instead of `os.Stderr` directly. Design `InitLogger` to accept an `io.Writer` in its internal helpers (or use `slog.NewJSONHandler(buf, opts)` in tests).
7. **OTel global state cleanup:** Every test that calls `InitProvider`, `NoopProvider`, or `otel.SetTracerProvider` must restore a clean noop state via `t.Cleanup(func() { NoopProvider() })` to prevent state leakage between tests.
8. **Race detection:** The full test suite must pass with `go test -race ./internal/observability/...`.
9. **Repeated execution:** All tests must pass with `go test -count=2 ./internal/observability/...` (idempotency requirement from REQ-NF-004).

---

## 3. Test Data Requirements

### 3.1 Config Fixtures

```go
// A fully-populated ObservabilityConfig for positive tests
var fullConfig = config.ObservabilityConfig{
    Enabled:        true,
    TracingEnabled: true,
    MetricsEnabled: true,
    LogLevel:       "debug",
    LogFormat:      "json",
    Exporter:       "stdout",
    OTLPEndpoint:   "localhost:4317",
    OTLPProtocol:   "grpc",
    ServiceName:    "test-shark",
}

// The zero value (all fields false/empty) — disables all telemetry
var disabledConfig = config.ObservabilityConfig{}
```

### 3.2 JSON Config Files for Config Tests

The config tests use in-memory JSON strings, not real `.sharkconfig.json` files, to remain portable and isolated. Each test creates a temporary directory with a `sharkconfig.json` file via `t.TempDir()`.

### 3.3 Metric Test Data

| Scenario | Command | Duration | Error |
|---|---|---|---|
| Success case | `"task list"` | `150ms` | `nil` |
| Error case | `"task get"` | `50ms` | `errors.New("not found")` |
| Zero duration | `"task create"` | `0` | `nil` |
| Long duration | `"epic get"` | `5000ms` | `nil` |

---

## 4. Unit Test Specifications

### 4.1 `provider_test.go`

**Package:** `package observability_test`

**Imports required:** `testify/assert`, `testify/require`, `go.opentelemetry.io/otel`, `context`, `bytes`, `os`, `testing`

---

#### `TestInitProvider_DisabledReturnsNoop`

**Purpose:** Verify that `InitProvider` with `Enabled=false` returns a valid no-op shutdown function and does not initialize any SDK components.

**Inputs:** `cfg = config.ObservabilityConfig{Enabled: false}`

**Expected Outputs:**
- Returned `ShutdownFunc` is non-nil
- No error returned
- After calling `ShutdownFunc`, no panic occurs
- `otel.Tracer("test").Start(ctx, "noop-span")` returns a span without panicking

**Test Steps:**
```go
func TestInitProvider_DisabledReturnsNoop(t *testing.T) {
    cfg := config.ObservabilityConfig{Enabled: false}
    shutdown, err := InitProvider(cfg)
    require.NoError(t, err)
    require.NotNil(t, shutdown)

    // Tracer must not panic
    tracer := otel.Tracer("test")
    ctx, span := tracer.Start(context.Background(), "noop-span")
    require.NotNil(t, span)
    require.NotNil(t, ctx)
    span.End()

    // Shutdown must not error or panic
    err = shutdown(context.Background())
    assert.NoError(t, err)

    t.Cleanup(func() { NoopProvider() })
}
```

---

#### `TestInitProvider_StdoutExporter_WritesToStderr`

**Purpose:** Verify that the stdout exporter writes to `os.Stderr`, never `os.Stdout` (REQ-F-009).

**Inputs:** `cfg = config.ObservabilityConfig{Enabled: true, Exporter: "stdout", ServiceName: "test"}`

**Expected Outputs:**
- `InitProvider` returns no error
- After starting and ending a span and calling shutdown, stdout buffer contains zero bytes
- Span lifecycle (start + end + flush) completes without panic

**Test Steps:**
```go
func TestInitProvider_StdoutExporter_WritesToStderr(t *testing.T) {
    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    cfg := config.ObservabilityConfig{
        Enabled:     true,
        Exporter:    "stdout",
        ServiceName: "test-provider",
    }
    shutdown, err := InitProvider(cfg)
    require.NoError(t, err)

    tracer := otel.Tracer("test")
    ctx, span := tracer.Start(context.Background(), "stdout-test-span")
    span.End()
    _ = ctx

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    _ = shutdown(shutdownCtx)

    w.Close()
    os.Stdout = oldStdout

    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    assert.Equal(t, 0, buf.Len(), "stdout must be empty; OTel output must go to stderr only")

    t.Cleanup(func() { NoopProvider() })
}
```

---

#### `TestInitProvider_OTLPExporter_LazyConnect`

**Purpose:** Verify that `InitProvider` with OTLP exporter does not fail at initialization even when no collector is reachable (lazy connection, REQ-F-001).

**Inputs:**
```go
cfg = config.ObservabilityConfig{
    Enabled:      true,
    Exporter:     "otlp",
    OTLPEndpoint: "localhost:19999", // port chosen to be unreachable
    OTLPProtocol: "grpc",
    ServiceName:  "lazy-test",
}
```

**Expected Outputs:**
- `InitProvider` returns no error
- `ShutdownFunc` call does not panic (may return a timeout/flush error, which is acceptable)

```go
func TestInitProvider_OTLPExporter_LazyConnect(t *testing.T) {
    cfg := config.ObservabilityConfig{
        Enabled:      true,
        Exporter:     "otlp",
        OTLPEndpoint: "localhost:19999",
        OTLPProtocol: "grpc",
        ServiceName:  "lazy-test",
    }
    shutdown, err := InitProvider(cfg)
    require.NoError(t, err, "InitProvider must not fail when collector is unreachable")
    require.NotNil(t, shutdown)

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    // Shutdown may return an error (flush timeout) — that is acceptable
    _ = shutdown(shutdownCtx)

    t.Cleanup(func() { NoopProvider() })
}
```

---

#### `TestInitProvider_ShutdownIdempotent`

**Purpose:** Calling `ShutdownFunc` twice must not panic or error on the second call (REQ-F-001).

**Inputs:** `cfg = config.ObservabilityConfig{Enabled: true, Exporter: "stdout", ServiceName: "idempotent-test"}`

**Expected Outputs:** Both shutdown calls complete without panic. Second call may return an error but must not panic.

```go
func TestInitProvider_ShutdownIdempotent(t *testing.T) {
    cfg := config.ObservabilityConfig{
        Enabled:     true,
        Exporter:    "stdout",
        ServiceName: "idempotent-test",
    }
    shutdown, err := InitProvider(cfg)
    require.NoError(t, err)

    ctx := context.Background()
    err1 := shutdown(ctx)
    assert.NotPanics(t, func() {
        _ = shutdown(ctx)
    }, "second shutdown call must not panic")
    _ = err1

    t.Cleanup(func() { NoopProvider() })
}
```

---

#### `TestApplyEnvOverrides` (table-driven)

**Purpose:** Verify each environment variable override independently and in combination (REQ-F-006). All six override branches plus "unset does not override" case.

**Test Cases:**

| Name | Env Var | Env Value | Initial Config Field | Expected Field Value |
|---|---|---|---|---|
| `SHARK_OTEL_ENABLED overrides false` | `SHARK_OTEL_ENABLED` | `"true"` | `Enabled: false` | `Enabled: true` |
| `SHARK_OTEL_ENABLED=false overrides true` | `SHARK_OTEL_ENABLED` | `"false"` | `Enabled: true` | `Enabled: false` |
| `OTEL_EXPORTER_OTLP_ENDPOINT overrides` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `"collector:4317"` | `OTLPEndpoint: "original"` | `OTLPEndpoint: "collector:4317"` |
| `OTEL_EXPORTER_OTLP_PROTOCOL overrides` | `OTEL_EXPORTER_OTLP_PROTOCOL` | `"http/protobuf"` | `OTLPProtocol: "grpc"` | `OTLPProtocol: "http/protobuf"` |
| `OTEL_SERVICE_NAME overrides` | `OTEL_SERVICE_NAME` | `"my-service"` | `ServiceName: "from-config"` | `ServiceName: "my-service"` |
| `SHARK_LOG_LEVEL overrides` | `SHARK_LOG_LEVEL` | `"debug"` | `LogLevel: "warn"` | `LogLevel: "debug"` |
| `SHARK_LOG_FORMAT overrides` | `SHARK_LOG_FORMAT` | `"text"` | `LogFormat: "json"` | `LogFormat: "text"` |
| `Unset env vars preserve config` | (none set) | — | `ServiceName: "from-config", LogLevel: "warn"` | `ServiceName: "from-config", LogLevel: "warn"` |

```go
func TestApplyEnvOverrides(t *testing.T) {
    tests := []struct {
        name      string
        envKey    string
        envVal    string
        initial   config.ObservabilityConfig
        assertFn  func(t *testing.T, cfg config.ObservabilityConfig)
    }{
        {
            name:    "SHARK_OTEL_ENABLED overrides false to true",
            envKey:  "SHARK_OTEL_ENABLED",
            envVal:  "true",
            initial: config.ObservabilityConfig{Enabled: false},
            assertFn: func(t *testing.T, cfg config.ObservabilityConfig) {
                assert.True(t, cfg.Enabled)
            },
        },
        // ... (all cases from table above)
        {
            name:   "Unset env vars preserve config values",
            envKey: "", // no env var set
            initial: config.ObservabilityConfig{
                ServiceName: "from-config",
                LogLevel:    "warn",
            },
            assertFn: func(t *testing.T, cfg config.ObservabilityConfig) {
                assert.Equal(t, "from-config", cfg.ServiceName)
                assert.Equal(t, "warn", cfg.LogLevel)
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if tt.envKey != "" {
                t.Setenv(tt.envKey, tt.envVal)
            }
            cfg := tt.initial
            applyEnvOverrides(&cfg) // call via exported test helper or white-box test
            tt.assertFn(t, cfg)
        })
    }
}
```

**Note on `applyEnvOverrides`:** Since this function is unexported, place `TestApplyEnvOverrides` in `package observability` (white-box test file) rather than `package observability_test`. Use `t.Setenv()` so environment variables are automatically unset after each subtest.

---

#### `TestInitProvider_SetsW3CPropagator`

**Purpose:** Verify that the W3C TraceContext propagator is installed globally after `InitProvider`.

**Expected Outputs:** `otel.GetTextMapPropagator()` returns a non-nil propagator.

---

#### `TestInitProvider_ServiceNameInResource`

**Purpose:** Verify the OTel resource contains the configured service name.

**Inputs:** `cfg.ServiceName = "my-test-service"`

**Expected Outputs:** The resource built by `buildResource` contains `semconv.ServiceNameKey = "my-test-service"`.

**Note:** Test `buildResource` directly if it is exported, or verify indirectly via span attributes in stdout exporter output captured on stderr.

---

### 4.2 `logger_test.go`

**Package:** `package observability_test` (black-box) with a companion white-box file `package observability` for `parseLogLevel`.

---

#### `TestInitLogger_DisabledDiscards`

**Purpose:** When `cfg.Enabled = false`, `slog.Info/Warn/Debug` produce zero bytes on stderr (REQ-F-002).

**Test Approach:** Inject a `bytes.Buffer` as the writer (requires the implementation to accept an `io.Writer` parameter in the internal helper, or use a test-local handler override). If the implementation calls `slog.SetDefault`, capture by temporarily replacing the slog default.

**Expected Outputs:** Buffer contains zero bytes after calling `slog.Info`, `slog.Warn`, `slog.Debug`.

```go
func TestInitLogger_DisabledDiscards(t *testing.T) {
    var buf bytes.Buffer
    cfg := config.ObservabilityConfig{Enabled: false}
    InitLoggerWithWriter(cfg, &buf) // implementation variant for testability
    // OR: call InitLogger(cfg) and assert slog level is above Error

    slog.Info("should be discarded")
    slog.Warn("also discarded")
    slog.Debug("debug also discarded")

    assert.Equal(t, 0, buf.Len(), "disabled logger must produce zero bytes")
}
```

**Alternative approach (if `InitLoggerWithWriter` is not exposed):** Check that the installed handler rejects all levels below `slog.LevelError + 1`:

```go
func TestInitLogger_DisabledHandler(t *testing.T) {
    cfg := config.ObservabilityConfig{Enabled: false}
    InitLogger(cfg)
    h := slog.Default().Handler()
    assert.False(t, h.Enabled(context.Background(), slog.LevelInfo))
    assert.False(t, h.Enabled(context.Background(), slog.LevelWarn))
    assert.False(t, h.Enabled(context.Background(), slog.LevelError))
}
```

---

#### `TestInitLogger_JSONFormat`

**Purpose:** When `cfg.LogFormat = "json"`, log lines are valid JSON (REQ-F-002).

**Inputs:** `cfg.Enabled = true, cfg.LogFormat = "json", cfg.ServiceName = "test-svc"`

**Expected Outputs:**
- Captured buffer line can be parsed by `json.Unmarshal` without error
- Parsed object contains a `"msg"` key
- Parsed object contains `"service.name": "test-svc"`

```go
func TestInitLogger_JSONFormat(t *testing.T) {
    var buf bytes.Buffer
    cfg := config.ObservabilityConfig{
        Enabled:     true,
        LogFormat:   "json",
        ServiceName: "test-svc",
    }
    InitLoggerWithWriter(cfg, &buf)

    slog.Info("test message")

    line := strings.TrimSpace(buf.String())
    require.NotEmpty(t, line)

    var parsed map[string]interface{}
    require.NoError(t, json.Unmarshal([]byte(line), &parsed), "log line must be valid JSON")
    assert.Equal(t, "test message", parsed["msg"])
    assert.Equal(t, "test-svc", parsed["service.name"])
}
```

---

#### `TestInitLogger_TextFormat`

**Purpose:** When `cfg.LogFormat = "text"`, log lines are key=value text format, not JSON (REQ-F-002).

**Expected Outputs:**
- Captured line does not start with `{`
- Line contains `msg=` substring

```go
func TestInitLogger_TextFormat(t *testing.T) {
    var buf bytes.Buffer
    cfg := config.ObservabilityConfig{
        Enabled:   true,
        LogFormat: "text",
        ServiceName: "txt-svc",
    }
    InitLoggerWithWriter(cfg, &buf)

    slog.Info("text message")

    line := strings.TrimSpace(buf.String())
    require.NotEmpty(t, line)
    assert.False(t, strings.HasPrefix(line, "{"), "text format must not produce JSON")
    assert.Contains(t, line, "msg=", "text format must contain msg= key")
}
```

---

#### `TestInitLogger_DefaultFormatIsJSON`

**Purpose:** When `cfg.LogFormat` is empty, the default format is JSON.

**Expected Outputs:** Captured line is valid JSON.

---

#### `TestInitLogger_ServiceNameAttribute`

**Purpose:** `service.name` attribute appears in every log line when enabled (REQ-F-002).

**Test Cases:**

| Name | ServiceName | Expected `service.name` |
|---|---|---|
| Explicit service name | `"myservice"` | `"myservice"` |
| Empty service name uses default | `""` | `"shark-task-manager"` |

```go
func TestInitLogger_ServiceNameAttribute(t *testing.T) {
    tests := []struct {
        name        string
        serviceName string
        expected    string
    }{
        {"explicit name", "myservice", "myservice"},
        {"default when empty", "", "shark-task-manager"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            cfg := config.ObservabilityConfig{
                Enabled:     true,
                LogFormat:   "json",
                ServiceName: tt.serviceName,
            }
            InitLoggerWithWriter(cfg, &buf)

            slog.Info("attr test")

            var parsed map[string]interface{}
            require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &parsed))
            assert.Equal(t, tt.expected, parsed["service.name"])
        })
    }
}
```

---

#### `TestInitLogger_LevelFiltering` (table-driven)

**Purpose:** Log level filtering suppresses messages below the configured level (REQ-F-002).

**Test Cases:**

| Name | LogLevel | Message Level | Should Appear |
|---|---|---|---|
| `warn level suppresses info` | `"warn"` | `slog.LevelInfo` | `false` |
| `warn level passes warn` | `"warn"` | `slog.LevelWarn` | `true` |
| `warn level passes error` | `"warn"` | `slog.LevelError` | `true` |
| `debug level passes debug` | `"debug"` | `slog.LevelDebug` | `true` |
| `error level suppresses warn` | `"error"` | `slog.LevelWarn` | `false` |
| `info level suppresses debug` | `"info"` | `slog.LevelDebug` | `false` |
| `info level passes info` | `"info"` | `slog.LevelInfo` | `true` |

```go
func TestInitLogger_LevelFiltering(t *testing.T) {
    tests := []struct {
        name        string
        logLevel    string
        msgLevel    slog.Level
        shouldAppear bool
    }{
        {"warn suppresses info", "warn", slog.LevelInfo, false},
        {"warn passes warn", "warn", slog.LevelWarn, true},
        {"warn passes error", "warn", slog.LevelError, true},
        {"debug passes debug", "debug", slog.LevelDebug, true},
        {"error suppresses warn", "error", slog.LevelWarn, false},
        {"info suppresses debug", "info", slog.LevelDebug, false},
        {"info passes info", "info", slog.LevelInfo, true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            var buf bytes.Buffer
            cfg := config.ObservabilityConfig{
                Enabled:   true,
                LogFormat: "json",
                LogLevel:  tt.logLevel,
            }
            InitLoggerWithWriter(cfg, &buf)

            slog.Log(context.Background(), tt.msgLevel, "level test message")

            if tt.shouldAppear {
                assert.NotEmpty(t, buf.String(), "expected log output but buffer was empty")
            } else {
                assert.Empty(t, buf.String(), "expected no log output but buffer had content")
            }
        })
    }
}
```

---

#### `TestInitLogger_OutputNeverGoesToStdout`

**Purpose:** Calling `InitLogger` with enabled=true and then writing logs produces zero bytes on stdout (REQ-F-009).

**Expected Outputs:** stdout buffer = 0 bytes after multiple `slog.*` calls.

---

#### `TestParseLogLevel` (table-driven, white-box)

**Purpose:** All valid and invalid log level strings are mapped correctly.

**Test Cases:**

| Input | Expected `slog.Level` |
|---|---|
| `"debug"` | `slog.LevelDebug` |
| `"DEBUG"` | `slog.LevelDebug` |
| `"info"` | `slog.LevelInfo` |
| `"warn"` | `slog.LevelWarn` |
| `"warning"` | `slog.LevelWarn` |
| `"error"` | `slog.LevelError` |
| `""` (empty) | `slog.LevelInfo` (default) |
| `"unknown"` | `slog.LevelInfo` (default) |
| `"TRACE"` | `slog.LevelDebug` (or a defined trace level) |

```go
// In package observability (white-box)
func TestParseLogLevel(t *testing.T) {
    tests := []struct {
        input    string
        expected slog.Level
    }{
        {"debug", slog.LevelDebug},
        {"DEBUG", slog.LevelDebug},
        {"info", slog.LevelInfo},
        {"warn", slog.LevelWarn},
        {"warning", slog.LevelWarn},
        {"error", slog.LevelError},
        {"", slog.LevelInfo},
        {"unknown", slog.LevelInfo},
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("input=%q", tt.input), func(t *testing.T) {
            result := parseLogLevel(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

---

### 4.3 `noop_test.go`

**Package:** `package observability_test`

---

#### `TestNoopProvider_ReturnsShutdownFunc`

**Purpose:** `NoopProvider()` returns a non-nil `ShutdownFunc`.

**Expected Outputs:** Return value is non-nil. Calling it returns `nil` error.

```go
func TestNoopProvider_ReturnsShutdownFunc(t *testing.T) {
    shutdown := NoopProvider()
    require.NotNil(t, shutdown)

    err := shutdown(context.Background())
    assert.NoError(t, err, "noop shutdown must return nil")
}
```

---

#### `TestNoopProvider_GlobalTracerDoesNotPanic`

**Purpose:** After `NoopProvider()`, `otel.Tracer("test").Start(ctx, "span")` does not panic and returns a valid span (REQ-F-003).

```go
func TestNoopProvider_GlobalTracerDoesNotPanic(t *testing.T) {
    NoopProvider()

    assert.NotPanics(t, func() {
        tracer := otel.Tracer("test")
        ctx, span := tracer.Start(context.Background(), "noop-span")
        require.NotNil(t, span)
        require.NotNil(t, ctx)
        span.End()
    })
}
```

---

#### `TestNoopProvider_IdempotentCallsSafe`

**Purpose:** Calling `NoopProvider()` three times in succession does not panic (REQ-F-003 — safe to call multiple times).

```go
func TestNoopProvider_IdempotentCallsSafe(t *testing.T) {
    assert.NotPanics(t, func() {
        NoopProvider()
        NoopProvider()
        NoopProvider()
    })
}
```

---

#### `TestNoopProvider_NoGoroutinesStarted`

**Purpose:** `NoopProvider()` does not start background goroutines.

**Test Approach:** Record `runtime.NumGoroutine()` before and after; assert the count is the same (±1 to allow for GC goroutine variance). Also use `goleak` if the project adopts it.

---

#### `TestNoopProvider_ShutdownReturnNil`

**Purpose:** The `ShutdownFunc` returned by `NoopProvider()` returns `nil` unconditionally, even when the context is already canceled.

```go
func TestNoopProvider_ShutdownReturnNil(t *testing.T) {
    shutdown := NoopProvider()

    canceledCtx, cancel := context.WithCancel(context.Background())
    cancel() // cancel immediately

    err := shutdown(canceledCtx)
    assert.NoError(t, err, "noop shutdown must return nil even with canceled context")
}
```

---

#### `TestNoopProvider_InstallsNoopMeterProvider`

**Purpose:** After `NoopProvider()`, `otel.GetMeterProvider()` is non-nil and meter operations don't panic.

```go
func TestNoopProvider_InstallsNoopMeterProvider(t *testing.T) {
    NoopProvider()
    mp := otel.GetMeterProvider()
    require.NotNil(t, mp)

    assert.NotPanics(t, func() {
        meter := mp.Meter("test")
        _, _ = meter.Float64Histogram("test.hist", metric.WithUnit("ms"))
    })
}
```

---

### 4.4 `metrics_test.go`

**Package:** `package observability_test`

---

#### `TestNewCommandMetrics_SucceedsWithNoopMeter`

**Purpose:** `NewCommandMetrics` succeeds with a no-op meter and returns no error (REQ-F-004).

```go
func TestNewCommandMetrics_SucceedsWithNoopMeter(t *testing.T) {
    NoopProvider()
    meter := otel.GetMeterProvider().Meter("test")

    cm, err := NewCommandMetrics(meter)
    require.NoError(t, err)
    assert.NotNil(t, cm)
}
```

---

#### `TestCommandMetrics_RecordDuration_OkStatus`

**Purpose:** `RecordDuration` with `err=nil` sets `status=ok` on metric attributes. No panic.

```go
func TestCommandMetrics_RecordDuration_OkStatus(t *testing.T) {
    NoopProvider()
    meter := otel.GetMeterProvider().Meter("test")
    cm, err := NewCommandMetrics(meter)
    require.NoError(t, err)

    assert.NotPanics(t, func() {
        cm.RecordDuration(context.Background(), "task list", 150*time.Millisecond, nil)
    })
}
```

---

#### `TestCommandMetrics_RecordDuration_ErrorStatus`

**Purpose:** `RecordDuration` with a non-nil error sets `status=error`. No panic.

```go
func TestCommandMetrics_RecordDuration_ErrorStatus(t *testing.T) {
    NoopProvider()
    meter := otel.GetMeterProvider().Meter("test")
    cm, err := NewCommandMetrics(meter)
    require.NoError(t, err)

    assert.NotPanics(t, func() {
        cm.RecordDuration(context.Background(), "task get", 50*time.Millisecond, errors.New("not found"))
    })
}
```

---

#### `TestCommandMetrics_RecordDuration_TableDriven`

**Purpose:** All combinations of command name and error state complete without panic.

**Test Cases:**

| Name | Command | Duration | Error | Expected Status |
|---|---|---|---|---|
| success | `"task list"` | `150ms` | `nil` | `"ok"` |
| error | `"task get"` | `50ms` | `errors.New("not found")` | `"error"` |
| zero duration | `"task create"` | `0` | `nil` | `"ok"` |
| long duration | `"epic get"` | `5000ms` | `nil` | `"ok"` |
| empty command | `""` | `100ms` | `nil` | `"ok"` |

---

#### `TestCommandMetrics_RecordInvocation`

**Purpose:** `RecordInvocation` completes without panic for both success and error cases.

---

#### `TestNewDBMetrics_SucceedsWithNoopMeter`

**Purpose:** `NewDBMetrics` succeeds with a no-op meter and returns no error (REQ-F-004).

---

#### `TestDBMetrics_RecordQueryDuration_TableDriven`

**Purpose:** All combinations of operation, table, duration, and error complete without panic.

**Test Cases:**

| Name | Operation | Table | Duration | Error |
|---|---|---|---|---|
| select success | `"SELECT"` | `"tasks"` | `10ms` | `nil` |
| insert error | `"INSERT"` | `"epics"` | `5ms` | `errors.New("constraint violation")` |
| zero duration | `"UPDATE"` | `"features"` | `0` | `nil` |
| empty table | `"DELETE"` | `""` | `1ms` | `nil` |

---

#### `TestCommandMetrics_InstrumentNames`

**Purpose:** Verify the exact instrument names match the specification.

**Note:** This test verifies the implementation registers instruments with the correct names. If the OTel SDK exposes the registered instrument names (via a test-only meter), assert them. Otherwise, this is validated by integration test against a real meter.

**Expected Instrument Names:**
- `shark.cli.command.duration`
- `shark.cli.command.invocations`
- `shark.cli.command.errors`

---

#### `TestDBMetrics_InstrumentNames`

**Purpose:** Verify DB metric instrument names match the specification.

**Expected Instrument Names:**
- `shark.db.query.duration`
- `shark.db.query.errors`

---

### 4.5 `config_observability_test.go`

**Package:** `package config_test`

**Location:** `internal/config/config_observability_test.go`

---

#### `TestObservabilityConfig_AbsentKeyProducesZeroValue`

**Purpose:** A config file without `"observability"` key produces a zero-value `ObservabilityConfig` (REQ-F-005).

```go
func TestObservabilityConfig_AbsentKeyProducesZeroValue(t *testing.T) {
    configJSON := `{
        "workflow_profile": "basic",
        "project_root": "/tmp/test"
    }`
    cfg := loadConfigFromString(t, configJSON)

    assert.False(t, cfg.Observability.Enabled)
    assert.False(t, cfg.Observability.TracingEnabled)
    assert.False(t, cfg.Observability.MetricsEnabled)
    assert.Empty(t, cfg.Observability.LogLevel)
    assert.Empty(t, cfg.Observability.LogFormat)
    assert.Empty(t, cfg.Observability.Exporter)
    assert.Empty(t, cfg.Observability.OTLPEndpoint)
    assert.Empty(t, cfg.Observability.OTLPProtocol)
    assert.Empty(t, cfg.Observability.ServiceName)
}
```

---

#### `TestObservabilityConfig_AllFieldsParsed`

**Purpose:** All nine `ObservabilityConfig` fields are correctly parsed from JSON (REQ-F-005).

```go
func TestObservabilityConfig_AllFieldsParsed(t *testing.T) {
    configJSON := `{
        "observability": {
            "enabled": true,
            "tracing_enabled": true,
            "metrics_enabled": true,
            "log_level": "debug",
            "log_format": "text",
            "exporter": "otlp",
            "otlp_endpoint": "localhost:4317",
            "otlp_protocol": "grpc",
            "service_name": "test-shark"
        }
    }`
    cfg := loadConfigFromString(t, configJSON)

    assert.True(t, cfg.Observability.Enabled)
    assert.True(t, cfg.Observability.TracingEnabled)
    assert.True(t, cfg.Observability.MetricsEnabled)
    assert.Equal(t, "debug", cfg.Observability.LogLevel)
    assert.Equal(t, "text", cfg.Observability.LogFormat)
    assert.Equal(t, "otlp", cfg.Observability.Exporter)
    assert.Equal(t, "localhost:4317", cfg.Observability.OTLPEndpoint)
    assert.Equal(t, "grpc", cfg.Observability.OTLPProtocol)
    assert.Equal(t, "test-shark", cfg.Observability.ServiceName)
}
```

---

#### `TestObservabilityConfig_EnabledFalse`

**Purpose:** `enabled: false` parses correctly and leaves all other fields at zero value.

```go
func TestObservabilityConfig_EnabledFalse(t *testing.T) {
    configJSON := `{"observability": {"enabled": false}}`
    cfg := loadConfigFromString(t, configJSON)
    assert.False(t, cfg.Observability.Enabled)
}
```

---

#### `TestObservabilityConfig_EnabledTrue_OtherFieldsDefault`

**Purpose:** `enabled: true` with no other fields leaves them at zero/empty value (not a failure).

```go
func TestObservabilityConfig_EnabledTrue_OtherFieldsDefault(t *testing.T) {
    configJSON := `{"observability": {"enabled": true}}`
    cfg := loadConfigFromString(t, configJSON)
    assert.True(t, cfg.Observability.Enabled)
    assert.Empty(t, cfg.Observability.Exporter)
    assert.Empty(t, cfg.Observability.ServiceName)
}
```

---

#### `TestObservabilityConfig_UnknownFieldsIgnored`

**Purpose:** Unknown fields in the `observability` block do not cause an error (forward-compatible design, UAT-07).

```go
func TestObservabilityConfig_UnknownFieldsIgnored(t *testing.T) {
    configJSON := `{
        "observability": {
            "enabled": false,
            "future_field_v2": "ignored",
            "another_unknown": 42
        }
    }`
    cfg, err := loadConfigFromStringWithError(t, configJSON)
    require.NoError(t, err)
    assert.False(t, cfg.Observability.Enabled)
}
```

---

#### `TestObservabilityConfig_OmitEmptyOnMarshal`

**Purpose:** Marshaling a `Config` with a zero-value `ObservabilityConfig` omits the `"observability"` key (REQ-F-005 — `omitempty`).

```go
func TestObservabilityConfig_OmitEmptyOnMarshal(t *testing.T) {
    cfg := Config{} // zero value, including zero ObservabilityConfig

    data, err := json.Marshal(cfg)
    require.NoError(t, err)

    var parsed map[string]interface{}
    require.NoError(t, json.Unmarshal(data, &parsed))
    _, present := parsed["observability"]
    assert.False(t, present, "observability key must be omitted when zero value")
}
```

---

#### `TestObservabilityConfig_ExistingConfigUnchanged`

**Purpose:** Adding `Observability` field to `Config` does not break existing config loading (REQ-NF-005, REQ-F-008).

**Test Approach:** Load a config that contains only fields present before F01 and assert all those fields are still correctly parsed.

---

#### Helper: `loadConfigFromString`

```go
// loadConfigFromString is a test helper that writes JSON to a temp file and loads it.
func loadConfigFromString(t *testing.T, jsonContent string) Config {
    t.Helper()
    dir := t.TempDir()
    path := filepath.Join(dir, ".sharkconfig.json")
    require.NoError(t, os.WriteFile(path, []byte(jsonContent), 0644))
    manager := NewManager(path)
    cfg, err := manager.Load()
    require.NoError(t, err)
    return *cfg
}
```

---

## 5. Integration Test Specifications

Integration tests live in `internal/observability/integration_test.go` and are gated by a build tag `//go:build integration` to prevent them from running in the standard `make test` cycle.

### 5.1 `TestIntegration_FullLifecycle_StdoutExporter`

**Purpose:** Verifies the complete lifecycle: `InitProvider` → start span → end span → `ShutdownFunc`. Confirms no panic, no error, and span data appears on stderr (not stdout).

**Steps:**
1. Call `InitProvider` with `Enabled=true, Exporter="stdout", ServiceName="integration-test"`
2. Start 3 spans with different names
3. End each span
4. Call `ShutdownFunc` with a 5-second context
5. Assert no error from shutdown
6. Assert stdout was empty throughout

**Build Tag:** `//go:build integration`

---

### 5.2 `TestIntegration_MetricsRecordedSuccessfully`

**Purpose:** Verifies that `CommandMetrics` and `DBMetrics` instruments can be created and used within a real (non-noop) MeterProvider lifecycle.

**Steps:**
1. Initialize full provider with stdout exporter
2. Create `CommandMetrics` and `DBMetrics`
3. Record several durations and invocations
4. Call shutdown
5. Assert no errors throughout

**Build Tag:** `//go:build integration`

---

### 5.3 `TestIntegration_Race_ConcurrentSpans`

**Purpose:** Verifies that concurrent span creation from multiple goroutines does not produce data races.

**Steps:**
1. Initialize provider
2. Launch 10 goroutines, each creating and ending 100 spans
3. Use `sync.WaitGroup` to wait for all goroutines
4. Call shutdown

**Run with:** `go test -race -tags integration ./internal/observability/`

---

## 6. Edge Cases and Error Scenarios

### 6.1 `InitProvider` Edge Cases

| Scenario | Input | Expected Behavior |
|---|---|---|
| Unknown exporter value | `Exporter: "zipkin"` | Return error with descriptive message |
| Empty ServiceName | `ServiceName: ""` | Default to `"shark-task-manager"` in resource |
| `Enabled=true` but empty Exporter | `Exporter: ""` | Default to `"stdout"` exporter or return error |
| Context already canceled at shutdown | Canceled ctx passed to `ShutdownFunc` | Return context error, no panic |
| `InitProvider` called after previous provider initialized | Second call | Overwrites global provider safely |

### 6.2 `InitLogger` Edge Cases

| Scenario | Input | Expected Behavior |
|---|---|---|
| Unknown log format | `LogFormat: "xml"` | Default to JSON, no panic |
| Unknown log level | `LogLevel: "verbose"` | Default to `"info"`, no panic |
| Empty log format | `LogFormat: ""` | Default to JSON |
| Empty log level | `LogLevel: ""` | Default to `"info"` |
| Concurrent `slog.*` calls after `InitLogger` | Multiple goroutines | No data race |

### 6.3 `NoopProvider` Edge Cases

| Scenario | Expected Behavior |
|---|---|
| Called before `InitProvider` | Safe, installs no-op globally |
| Called after `InitProvider` | Overwrites real provider, no panic |
| `ShutdownFunc` called with nil context | Return nil or panic with clear message (define in spec) |
| Concurrent calls from multiple goroutines | No data race |

### 6.4 Metrics Edge Cases

| Scenario | Expected Behavior |
|---|---|
| `NewCommandMetrics` with nil meter | Return error, not panic |
| `RecordDuration` with zero duration | No panic |
| `RecordDuration` with extremely large duration (overflow) | No panic |
| `RecordDuration` called on zero-value `CommandMetrics` | No panic (or documented panic behavior) |
| `NewDBMetrics` with nil meter | Return error, not panic |

### 6.5 Config Edge Cases

| Scenario | Expected Behavior |
|---|---|
| `observability` value is `null` in JSON | Zero value `ObservabilityConfig`, no error |
| `observability` value is a string (type mismatch) | Return error or zero value (define) |
| `enabled` value is a string `"true"` instead of bool | Documented behavior (error or parse) |
| Config file has only `"observability": {}` (empty object) | All fields zero value, no error |

---

## 7. Error Path Tests (100% Coverage Required)

Every function that returns an error must have at least one test that exercises the error path:

| Function | Error Trigger Scenario | Test Name |
|---|---|---|
| `InitProvider` | Unknown exporter type | `TestInitProvider_UnknownExporter_ReturnsError` |
| `InitProvider` | `buildResource` fails (simulated) | `TestInitProvider_ResourceBuildFailure` |
| `NewCommandMetrics` | Meter registration fails | `TestNewCommandMetrics_RegistrationError` |
| `NewDBMetrics` | Meter registration fails | `TestNewDBMetrics_RegistrationError` |
| `ShutdownFunc` (enabled path) | Context deadline exceeded during flush | `TestInitProvider_ShutdownContextTimeout` |
| Config `Load` | Malformed `observability` block | `TestConfigLoad_MalformedObservabilityBlock` |

**Note:** If the OTel SDK instruments do not return errors with a no-op meter, the `NewCommandMetrics` and `NewDBMetrics` error paths must be tested with a mock meter that returns an error.

---

## 8. Performance Tests

### 8.1 Benchmark: Disabled Path Overhead

**File:** `internal/observability/benchmarks_test.go`

```go
func BenchmarkInitProvider_Disabled(b *testing.B) {
    cfg := config.ObservabilityConfig{Enabled: false}
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        shutdown, _ := InitProvider(cfg)
        _ = shutdown(context.Background())
    }
}
```

**Target:** Less than 1% increase in `shark get E01` wall-clock time (REQ-NF-001).

### 8.2 Benchmark: Noop Span Creation

```go
func BenchmarkNoopSpan(b *testing.B) {
    NoopProvider()
    tracer := otel.Tracer("bench")
    ctx := context.Background()
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, span := tracer.Start(ctx, "bench-span")
        span.End()
    }
}
```

**Target:** Less than 200ns per span (comparable to no-op function call overhead).

---

## 9. Build and Static Analysis Tests

### 9.1 Import Cycle Check

```bash
# Must succeed with no import cycle errors (REQ-F-007)
go build ./internal/observability/...
```

### 9.2 Vet Check

```bash
go vet ./internal/observability/...
```

### 9.3 Backward Compatibility Check

```bash
# All existing callers compile without modification (REQ-NF-005)
go build ./...
```

---

## 10. Test Execution Plan

### Standard Run (included in `make test`)

```bash
make fmt && make lint && make test
```

All tests in `internal/observability/` and `internal/config/` run as part of the standard suite. No build tags required.

### Race Detection Run

```bash
go test -race ./internal/observability/...
go test -race ./internal/config/...
```

### Repeated Execution (idempotency)

```bash
go test -count=2 -race ./internal/observability/...
```

### Integration Tests (optional, requires running collector)

```bash
go test -tags integration -v ./internal/observability/...
```

### Coverage Check

```bash
go test -cover ./internal/observability/...
go test -cover ./internal/config/...
```

**Minimum coverage gates:**
- `internal/observability`: 80% overall, 100% error paths
- `internal/config` (observability additions): 100%

---

## 11. Sign-Off Criteria

All of the following must be true before F01 is marked as QA-passed:

- [ ] All unit tests in `provider_test.go` pass
- [ ] All unit tests in `logger_test.go` pass
- [ ] All unit tests in `noop_test.go` pass
- [ ] All unit tests in `metrics_test.go` pass
- [ ] All unit tests in `config_observability_test.go` pass
- [ ] `go test -race -count=2 ./internal/observability/...` exits 0
- [ ] `go test -cover ./internal/observability/...` reports 80%+
- [ ] Error path coverage is 100% (all error-returning functions have an error test)
- [ ] `go build ./...` exits 0 (no import cycles, no compile errors)
- [ ] `go vet ./internal/observability/...` exits 0
- [ ] `make fmt && make lint && make test` all exit 0
- [ ] No stdout contamination confirmed by `TestInitProvider_StdoutExporter_WritesToStderr`
- [ ] No stdout contamination confirmed by `TestInitLogger_OutputNeverGoesToStdout`
- [ ] UAT scenarios UAT-01 through UAT-30 have been manually verified (see `uat-plan.md`)

---

*Last Updated: 2026-03-22*
