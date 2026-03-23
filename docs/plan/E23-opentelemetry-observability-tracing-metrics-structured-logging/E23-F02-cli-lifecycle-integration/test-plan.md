# E23-F02 Test Plan: CLI Lifecycle Integration

**Feature Key**: E23-F02
**Date**: 2026-03-22
**Patterns**: See `.claude/rules/testing/architecture.md` — CLI tests use mocks; no real DB.

---

## AC Test Matrix

Each acceptance criterion from `specification.md` maps to one or more test cases.

### REQ-F-001: Observability Global Singleton

| AC | Test Case | Setup | Expected | Edge Case |
|----|-----------|-------|----------|-----------|
| InitObservability is idempotent | `TestInitObservability_Idempotent` | Call InitObservability twice with same cfg | Second call is no-op; obsInitOnce guards | |
| InitObservability stores ShutdownFunc | `TestInitObservability_StoresShutdown` | Call Init then Shutdown | Shutdown returns nil (noop cfg) | |
| InitObservability calls InitProvider + InitLogger | `TestInitObservability_CallsBoth` | Mock or spy cfg with enabled=false | Both paths execute; logger set to discard | enabled=true path also tested |
| ShutdownObservability is safe before Init | `TestShutdownObservability_BeforeInit` | Call Shutdown without Init | Returns nil (no panic) | |
| ResetObservability resets all globals | `TestResetObservability_ClearsState` | Init, Reset, Init again | Second Init succeeds from clean state | |
| GetTracer returns valid tracer | `TestGetTracer_ReturnsTracer` | Call GetTracer("test") | Non-nil tracer returned | Before Init (noop tracer) |

**File**: `internal/cli/observability_global_test.go`
**Pattern**: Tests in `package cli` calling package-level functions directly.
Each test calls `defer ResetObservability()` for cleanup.

---

### REQ-F-002: PersistentPreRunE Hook Integration

| AC | Test Case | Setup | Expected | Edge Case |
|----|-----------|-------|----------|-----------|
| InitObservability called after initConfig | `TestPersistentPreRunE_CallsObsInit` | Execute PreRunE with disabled cfg | InitObservability called; no error | |
| ObsCfg sourced from config file | `TestLoadObservabilityConfig_FromFile` | Config file with enabled=true | ObservabilityConfig has Enabled=true | Missing file → zero-value config |
| Init error does not abort command | `TestPersistentPreRunE_ObsInitError_NonFatal` | Force InitObservability to fail (bad exporter) | PreRunE returns nil; warning to stderr | |
| No-op path adds no overhead | `TestInitObservability_NoopPath` | cfg.Enabled=false | NoopProvider installed; no SDK init | |

**File**: `internal/cli/root_test.go` (existing file, add new test cases)
**Pattern**: Unit tests that call `loadObservabilityConfig` directly; integration test via Cobra
command execution with output capture.

---

### REQ-F-003: PersistentPostRunE Hook Integration

| AC | Test Case | Setup | Expected | Edge Case |
|----|-----------|-------|----------|-----------|
| ShutdownObservability called before CloseDB | `TestPersistentPostRunE_ShutdownBeforeClose` | Track call order via test flag | Shutdown recorded before DB close | |
| Shutdown uses 5-second timeout | `TestShutdownObservability_HasTimeout` | Verify ShutdownObservability creates timeout ctx | Function completes even if provider slow | |
| Shutdown errors are non-fatal | `TestPersistentPostRunE_ShutdownError_NonFatal` | ShutdownObservability returns error | PostRunE returns nil | |

**File**: `internal/cli/root_test.go`

---

### REQ-F-004: Test Teardown via ResetServices

| AC | Test Case | Setup | Expected | Edge Case |
|----|-----------|-------|----------|-----------|
| ResetServices calls ResetObservability | `TestResetServices_IncludesObservability` | Init obs, call ResetServices | Subsequent Init creates new singleton | |
| Post-reset Init reinitializes | `TestResetServices_ThenInit` | Reset, then Init with enabled=true | Clean initialization, no state from before | |

**File**: `internal/cli/services_global_test.go` (existing file)

---

### REQ-F-005: GetTracer Accessor

| AC | Test Case | Setup | Expected | Edge Case |
|----|-----------|-------|----------|-----------|
| Returns otel.Tracer(name) | `TestGetTracer_DelegatesToOtel` | Call GetTracer("services/task") | Tracer is non-nil | |
| Safe before Init | `TestGetTracer_BeforeInit` | Call without InitObservability | Returns noop tracer, no panic | |

**File**: `internal/cli/observability_global_test.go`

---

### REQ-NF-001: Zero Overhead When Disabled

| AC | Test Case | Setup | Expected |
|----|-----------|-------|----------|
| Noop path: no SDK allocations | `TestInitObservability_NoopNoAllocations` | Benchmark with enabled=false | <1μs per call, no heap allocs |

**Note**: Full benchmarks are manual (UAT-01). Unit test verifies noop path does not call
`sdktrace.NewTracerProvider`.

---

### REQ-NF-002: No stdout Contamination

| AC | Test Case | Setup | Expected |
|----|-----------|-------|----------|
| All obs output to stderr | Inherited from F01 tests | F01 enforces this in logger.go | F02 adds no new output paths |

**Note**: F01's `logger_test.go` already verifies stderr-only output. F02 tests capture
stdout/stderr separately to verify no contamination.

---

### REQ-NF-003: Graceful Degradation

Covered by `TestPersistentPreRunE_ObsInitError_NonFatal` (REQ-F-002 table) and UAT-04.

---

## Integration Scenarios

### Scenario A: Full Command Lifecycle (Init → Execute → Shutdown)

**Components involved**: `root.go` PersistentPreRunE + command + PersistentPostRunE + `observability_global.go`

**Verify**:
- InitObservability called once per command invocation
- ShutdownObservability called after command, before DB close
- Global OTel state is clean after shutdown

**Test**: Integration test using Cobra's `Execute()` with captured output.
See existing pattern in `internal/cli/commands/*_test.go`.

### Scenario B: Multi-Test Isolation

**Components involved**: `ResetServices()` + `ResetObservability()` + `sync.Once` reset

**Verify**:
- Test A initializes obs, calls ResetServices
- Test B can initialize obs fresh without state from Test A leaking

**Test**: Sequential subtests within `TestResetServices_ThenInit`.

### Scenario C: GetTracer Called From services_global.go (F04 Integration Point)

**Components involved**: `observability_global.go:GetTracer` + `services_global.go:GetTaskService`

**Verify**:
- When F04 adds `svc.SetTracer(GetTracer("services/task"))` to GetTaskService, it compiles
- When observability disabled, SetTracer receives a noop tracer
- When observability enabled, SetTracer receives a real tracer

**Test**: Compile-time verification (F04 scope); F02 only needs `GetTracer` to be exported.

---

## Test Infrastructure

### Existing Patterns to Follow

| Pattern | File | How Used |
|---------|------|----------|
| sync.Once reset in tests | `internal/cli/db_global.go:ResetDB()` | ResetObservability follows identical pattern |
| ResetServices in test cleanup | `internal/cli/services_global.go:ResetServices()` | Add `defer cli.ResetServices()` to CLI tests |
| Cobra command output capture | `internal/cli/commands/*_test.go` | Capture stdout/stderr in buffer |
| Config loading in tests | `internal/cli/root_test.go` | Use temp config file or viper mock |

### New Test Helpers Needed

**`obsTestHelper`** (local to `observability_global_test.go`):

```go
// resetObsForTest resets observability and installs noop providers.
// Defer this in every test touching global OTel state.
func resetObsForTest(t *testing.T) {
    t.Helper()
    ResetObservability()
    t.Cleanup(ResetObservability)
}
```

This is a trivial wrapper — no complex infrastructure needed.

---

## Epic UAT Traceability

| UAT Scenario | Test Coverage |
|--------------|---------------|
| UAT-01: No overhead default | REQ-NF-001 unit test + manual benchmark |
| UAT-02: Enabled stdout exporter | `TestInitObservability_CallsBoth` (enabled=true path) |
| UAT-03: Shutdown before DB close | `TestPersistentPostRunE_ShutdownBeforeClose` |
| UAT-04: OTLP failure graceful | `TestPersistentPreRunE_ObsInitError_NonFatal` |
| UAT-05: Test suite passes | `make test` (all existing tests + new tests) |
| UAT-06: No .sharkconfig panic | `TestLoadObservabilityConfig_FromFile` (missing file case) |
| UAT-07: Verbose warnings | `TestPersistentPostRunE_ShutdownError_NonFatal` with verbose flag |

---

*Last Updated: 2026-03-22*
