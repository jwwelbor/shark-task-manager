---
feature_key: E07-F40-file-logging-destination-for-observability
epic_key: E07
title: File logging destination for observability
description: Add log_file field to observability config so slog output can be written to a file instead of stderr.
---

# File logging destination for observability

**Feature Key**: E07-F40

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

Today, the observability subsystem only writes structured logs (`slog`) to
stderr (`internal/observability/logger.go:41,44`). There is no built-in way to
capture logs to a file. Users who want a persistent log for `shark run` (or any
command) must rely on shell redirection (`2> shark.log`). Shell redirection
works but:

- Requires the user to remember to set it up every invocation.
- Cannot be configured per-project via `.sharkconfig.json`.
- Is invisible in the config surface — users don't discover the capability.

### Solution

Add a `log_file` string field to `ObservabilityConfig`. When set, `InitLogger`
opens the file and uses it as the writer for the slog handler instead of
stderr. When unset (empty string), current stderr behavior is preserved
exactly.

Path resolution: absolute paths are used as-is; relative paths resolve against
the project root. The parent directory is created with `MkdirAll` if missing.
Failure to open the file falls back to stderr with a single raw warning
printed to stderr — the command never fails over logging.

Env override: `SHARK_LOG_FILE` overrides the config field, consistent with
existing env override conventions (`SHARK_LOG_LEVEL`, `SHARK_LOG_FORMAT`).

### Impact

- Users can enable persistent file logging by editing
  `.sharkconfig.json` once — no shell gymnastics per invocation.
- Discoverable: the field ships in the default scaffolded config (with empty
  string, i.e., inactive).
- Zero behavior change for users who don't set the field.

---

## Scope

### In Scope

- `log_file` field on `config.ObservabilityConfig` (optional, `omitempty`).
- File writer wiring in `observability.InitLogger`.
- Path resolution (abs/rel), parent directory creation, append-on-open.
- Fallback to stderr on open failure with single raw warning.
- `SHARK_LOG_FILE` env override in `observability.ApplyEnvOverrides`.
- Cleanup hook: close the file on `ShutdownObservability` and
  `ResetObservability`.
- Inclusion of the field in the default config scaffold written by
  `shark admin init` (value: `""`, so field is visible but inactive).
- Docs: field reference entry, example block, env var table row, and link from
  the main configuration reference doc to the observability guide (closes a
  pre-existing documentation gap).

### Out of Scope — Explicit

1. **Log rotation**. Users can point `logrotate` at the file externally. We
   may revisit with `lumberjack.v2` in a future feature if demand emerges.
2. **Tee mode** (file + stderr simultaneously). Users who want both can use
   shell tee: `shark run ... 2> >(tee shark.log >&2)`. Revisit later.
3. **Capturing stdout output** (human-readable progress from `run.go:138-159`
   and pterm output). Only `slog` output is captured. Documented as a known
   limitation.
4. **Truncate-on-open mode**. Append-only for this iteration.
5. **Log file permissions beyond 0644 / dir 0755**. No per-field overrides.

### Alternative Approaches Rejected

- **Full logger rewrite to pipe stdout through slog**: massive blast radius,
  would conflict with pterm styling. Not worth it for this feature.
- **Add a third exporter option (`file`)**: mixes concerns. Exporter governs
  OTel traces/metrics; logging destination is orthogonal. A dedicated
  `log_file` field is clearer.

---

## Design

### Scope Decisions (Explicit)

- **File-only** (no tee to stderr by default).
- **No rotation** (external tooling).
- **Path resolution**: absolute as-is; relative against project root.
- **Parent directory**: `os.MkdirAll(parent, 0755)`.
- **Open mode**: `O_APPEND|O_CREATE|O_WRONLY`, perm `0644`.
- **Failure policy**: fallback to stderr, print one `fmt.Fprintln(os.Stderr, ...)`
  warning. Never fail startup over logging.
- **Fallback warning must be raw** (not via slog) to avoid recursion when the
  slog handler itself is the thing being installed.

### Files to Change

| Path | Change |
|---|---|
| `internal/config/config.go` | Add `LogFile string \`json:"log_file,omitempty"\`` to `ObservabilityConfig`. |
| `internal/observability/logger.go` | Introduce `InitLoggerWithRoot(cfg, projectRoot string) io.Closer`; keep `InitLogger(cfg)` as wrapper. Open file when `cfg.LogFile != ""`; return closer (or nil). |
| `internal/observability/provider.go` | Add `SHARK_LOG_FILE` env override in `ApplyEnvOverrides`. |
| `internal/cli/observability_global.go` | Store returned `io.Closer` on `obsContainer.logFile`; close it in `ShutdownObservability` and `ResetObservability`; pass project root to `InitLoggerWithRoot`. |
| `internal/init/types.go` | Add `LogFile string \`json:"log_file"\`` to `ObservabilityConfigDefault`. |
| `internal/init/config.go` | Default value `""` in the scaffolded observability block. |
| `internal/observability/logger_test.go` | New cases for file destination behavior (see Test Plan). |
| `internal/observability/provider_test.go` | New case in `TestApplyEnvOverrides` for `SHARK_LOG_FILE`. |
| `internal/cli/observability_global_test.go` | New case asserting file closer runs on shutdown. |
| `internal/init/config_test.go` | Assert scaffolded `Observability.LogFile == ""`. |
| `docs/guides/observability-config-reference.md` | New `### log_file` field reference section; row in env var table. |
| `docs/guides/observability.md` | One example block for file destination. |
| `docs/cli-reference/configuration.md` | Add link to the observability reference guides (closes pre-existing gap). |

### API Shape

```go
// internal/config/config.go
type ObservabilityConfig struct {
    // ... existing fields ...
    LogFile string `json:"log_file,omitempty"`
}
```

```go
// internal/observability/logger.go
// InitLoggerWithRoot configures the global slog default logger based on cfg.
// If cfg.LogFile is set, writes to that file (relative paths resolved against
// projectRoot). Returns an io.Closer for the opened file, or nil if writing
// to stderr.
func InitLoggerWithRoot(cfg config.ObservabilityConfig, projectRoot string) io.Closer

// InitLogger preserves the existing signature; delegates to InitLoggerWithRoot
// with projectRoot == "".
func InitLogger(cfg config.ObservabilityConfig)
```

### Behavior Matrix

| `enabled` | `log_file` | Open succeeds? | Writer | Handler |
|---|---|---|---|---|
| `false` | * | — | (discard) | discard handler (unchanged) |
| `true`  | `""` | — | `os.Stderr` | JSON/text per `log_format` (unchanged) |
| `true`  | path | yes | `*os.File` | JSON/text per `log_format` |
| `true`  | path | no | `os.Stderr` | JSON/text + one-shot raw stderr warning |

---

## Requirements

### Functional

- **REQ-F-001**: `log_file` field exists on `ObservabilityConfig` and
  round-trips through JSON marshal/unmarshal.
- **REQ-F-002**: When `log_file` is an absolute path, logs are written to that
  exact path.
- **REQ-F-003**: When `log_file` is relative, it is resolved against project
  root (passed into `InitLoggerWithRoot`).
- **REQ-F-004**: Missing parent directories are created with perm `0755`.
- **REQ-F-005**: File is opened in append mode; repeated `InitLogger` calls do
  not truncate prior content.
- **REQ-F-006**: On file open failure, logger falls back to stderr and prints
  exactly one warning line to stderr. The command completes normally.
- **REQ-F-007**: `SHARK_LOG_FILE` env var overrides `cfg.LogFile` when set to
  a non-empty value.
- **REQ-F-008**: On `ShutdownObservability`, the opened log file is closed.
- **REQ-F-009**: `shark admin init` writes an `observability` block that
  includes `"log_file": ""`.

### Non-Functional

- **REQ-NF-001**: Zero runtime overhead when `log_file == ""` — code path
  unchanged vs. today.
- **REQ-NF-002**: No new third-party dependencies.
- **REQ-NF-003**: Code passes `make fmt && make lint && make test`.

---

## Acceptance Criteria

**Scenario 1: Happy path — file destination**
- **Given** `.sharkconfig.json` with `observability.enabled: true` and
  `observability.log_file: "./logs/shark.log"`
- **When** `shark run <anything>` executes a slog call at or above the
  configured level
- **Then** the log record appears in `./logs/shark.log` (newline-delimited
  JSON by default)
- **And** nothing is written to stderr for that record

**Scenario 2: Relative path resolution**
- **Given** a project rooted at `/home/user/proj` and `log_file: "shark.log"`
- **When** any command runs from a subdirectory
- **Then** logs are written to `/home/user/proj/shark.log` (not the CWD)

**Scenario 3: Parent dir creation**
- **Given** `log_file: "nested/deep/shark.log"` and `nested/` does not exist
- **When** the logger initializes
- **Then** `nested/deep/` is created with mode `0755` and the file is created

**Scenario 4: Fallback on open failure**
- **Given** `log_file` points to an unwritable path (e.g., `/proc/1/noperm`)
- **When** the logger initializes
- **Then** exactly one warning line is printed to stderr
- **And** subsequent slog records go to stderr in the configured format
- **And** the command proceeds normally (exit code unchanged)

**Scenario 5: Env override**
- **Given** config has `log_file: "a.log"` and env `SHARK_LOG_FILE=b.log`
- **When** the logger initializes
- **Then** logs are written to `b.log`

**Scenario 6: Scaffold discoverability**
- **Given** a fresh project
- **When** the user runs `shark admin init --non-interactive`
- **Then** `.sharkconfig.json` contains `observability.log_file: ""`

**Scenario 7: File closed on shutdown**
- **Given** an initialized logger with an open log file
- **When** `ShutdownObservability` runs
- **Then** the file descriptor is closed (follow-up writes would fail)

---

## Test Plan

### Unit Tests

**`internal/observability/logger_test.go`** — new cases:

1. `TestInitLogger_FileDestination_WritesToFile` — set `LogFile` to a temp
   path; emit a log record; assert file contents contain the record in the
   configured format.
2. `TestInitLogger_FileDestination_Appends` — call `InitLoggerWithRoot`
   twice with the same path, emit a record after each; assert both records
   present.
3. `TestInitLogger_FileDestination_CreatesParentDir` — nested path under a
   temp dir; assert intermediate dirs created with `0755`.
4. `TestInitLogger_FileDestination_BadPath_FallsBackToStderr` — use an
   unwritable path; capture stderr; assert single warning line and that
   subsequent records land on stderr (not the bad path).
5. `TestInitLogger_FileDestination_RelativePath_ResolvedToRoot` — relative
   path + project root; assert resolved absolute write location.
6. `TestInitLogger_FileDestination_Disabled_NoFileOpened` — `enabled: false`
   with `log_file` set; assert no file is created.

**`internal/observability/provider_test.go`** — extend `TestApplyEnvOverrides`:

7. `SHARK_LOG_FILE=/tmp/foo.log` → `cfg.LogFile == "/tmp/foo.log"`.

**`internal/cli/observability_global_test.go`** — new case:

8. `TestShutdownObservability_ClosesLogFile` — configure `log_file`,
   initialize, shutdown, assert the underlying file descriptor is closed
   (e.g., `Write` returns `os.ErrClosed`).

**`internal/init/config_test.go`** — extend `TestCreateConfig`:

9. Assert `cfg.Observability.LogFile == ""` in scaffolded output.
10. Assert `"log_file"` present in the top-level JSON keys under
    `observability`.

### Integration / Smoke (manual)

- Build `make shark`; create a tmp project; run
  `./bin/shark admin init --non-interactive`; set `enabled: true` and
  `log_file: "./shark.log"`; run `./bin/shark status`; verify `shark.log`
  exists and contains a JSON record.
- Repeat with `SHARK_LOG_FILE` env var.

### Quality Gate

- `make fmt && make lint && make test` passes with no new warnings.
- No new lint exceptions introduced.

---

## Dependencies

- **None.** Purely additive changes inside `internal/config`,
  `internal/observability`, `internal/cli`, and `internal/init`.

---

## Known Limitations (documented)

- Only `slog` output is captured. Human-readable stdout output from commands
  (e.g., the progress printer in `run.go:138-159`, pterm spinners) is NOT
  written to the log file. Users who need a full transcript should still use
  shell redirection: `shark run ... > full.log 2>&1`.
- No log rotation. Use `logrotate` (or equivalent) if the file grows large.

---

## Rollout

1. Implement and land behind no feature flag — `log_file == ""` is the safe
   default and preserves today's behavior exactly.
2. Users adopt by editing `.sharkconfig.json` or setting `SHARK_LOG_FILE`.
3. Follow-up features (Tier 2 tee, Tier 3 rotation) tracked separately if
   demand materializes.

---

*Last Updated*: 2026-04-18
