---
feature_key: E17-F04
epic_key: E17
title: SHARK_OUTPUT Environment Variable
description: Support SHARK_OUTPUT=json environment variable as a session-wide alternative to the --json flag, enabling AI agents to configure JSON output once per session.
execution_order: 1
phase: 1
complexity: XS
status: draft
dependencies: []
depended_on_by:
  - E17-F03 (JSON error output should be active when SHARK_OUTPUT=json)
epic_requirements:
  - F04 (SHARK_OUTPUT environment variable)
  - NFR-1 (Backward Compatibility)
---

# SHARK_OUTPUT Environment Variable

**Feature Key**: E17-F04
**Phase**: 1 (Must-Have)
**Complexity**: XS
**Execution Order**: 1 (implement first -- smallest change, immediate agent value)

---

## Scope

### Problem

AI agents must pass `--json` on every single shark command invocation. Analysis of 231 real agent interactions shows that agents frequently forget the `--json` flag, leading to unparseable human-readable output that causes downstream failures. There is no way to set JSON output mode for an entire agent session.

### Solution

Support `SHARK_OUTPUT=json` environment variable that sets `cli.GlobalConfig.JSON = true` for the entire session. Agents can set this once at session start and all subsequent commands return JSON automatically.

### What This Feature Does

- Reads `SHARK_OUTPUT` environment variable in the root command's initialization
- If value is `json`, enables JSON output mode globally
- The `--json` flag continues to work and takes precedence over the env var
- Also supports `SHARK_OUTPUT=table` to explicitly request table output (the default)
- Both `SHARK_OUTPUT` and `PM_OUTPUT` are supported for discoverability, with `SHARK_OUTPUT` taking precedence

### What This Feature Does NOT Do

- Does not change the default output format (table remains default when no env var is set)
- Does not introduce any new CLI flags
- Does not affect human-readable error output when env var is not set
- Does not change existing `--json` flag behavior

---

## Acceptance Criteria

- [ ] `SHARK_OUTPUT=json shark get E18-F05-001` returns JSON output
- [ ] `SHARK_OUTPUT=json` applies to all commands (get, list, next, status, etc.)
- [ ] `--json` flag overrides env var (both directions: `--json` forces JSON)
- [ ] `SHARK_OUTPUT=table` explicitly selects table output (default behavior)
- [ ] Invalid values for `SHARK_OUTPUT` are silently ignored (default to table)
- [ ] Documented in `--help` output for root command
- [ ] All existing tests pass without modification (`make test` green)
- [ ] When `SHARK_OUTPUT=json` is set, `--no-json` or explicit `--json=false` can override it
- [ ] `PM_OUTPUT=json` also works as a fallback (existing PM_ env prefix convention)

---

## Dependencies

### Depends On

None. This is a standalone feature that modifies only `internal/cli/root.go`.

### Depended On By

- **E17-F03 (Structured JSON Errors)**: When `SHARK_OUTPUT=json` is active, F03's structured error output should also be active.

---

## Implementation Notes

- Modify `internal/cli/root.go` in `PersistentPreRunE` or `initConfig()`
- Check `os.Getenv("SHARK_OUTPUT")` and `os.Getenv("PM_OUTPUT")`
- If either is `"json"`, set `GlobalConfig.JSON = true`
- `SHARK_OUTPUT` takes precedence over `PM_OUTPUT`
- Cobra flag parsing happens after PersistentPreRun, so `--json` flag naturally overrides env var
- Estimated: 5-10 lines of code change

---

## Success Metrics

- **Primary**: Agents can set `SHARK_OUTPUT=json` once and all commands return JSON
- **Measured by**: Zero occurrences of missing `--json` flags causing parse failures in agent logs
- **Performance**: Zero latency impact (env var read is negligible)
- **Backward Compatibility**: 100% -- no existing behavior changes

---

## UAT Scenarios

- J1-S02: Read task details with environment JSON mode
- J1-S07: Full journey command count (SHARK_OUTPUT eliminates per-command --json)

---

*Last Updated*: 2026-02-25
