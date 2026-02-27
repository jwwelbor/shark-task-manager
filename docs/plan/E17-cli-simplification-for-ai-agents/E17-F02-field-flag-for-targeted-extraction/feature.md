---
feature_key: E17-F02
epic_key: E17
title: Field Flag for Targeted Extraction
description: Add --field <name> flag to get, list, next, progress, and status commands that returns only the raw value of the specified field, eliminating the need for external JSON post-processing tools like jq or Python.
execution_order: 5
phase: 1
complexity: S
status: draft
dependencies: []
depended_on_by:
  - E17-F06 (progress command supports --field)
epic_requirements:
  - F02 (Field Flag for Targeted Extraction)
  - NFR-1 (Backward Compatibility)
  - NFR-3 (Service Layer Integration)
  - NFR-4 (Testing)
---

# Field Flag for Targeted Extraction

**Feature Key**: E17-F02
**Phase**: 1 (Must-Have)
**Complexity**: S
**Execution Order**: 5 (after F04, F05, F03, F07; standalone but benefits from JSON error infrastructure)

---

## Scope

### Problem

15% of all AI agent CLI invocations require external post-processing (`jq`, Python) to extract a single field from JSON output. For example, to get a task's status an agent must run `shark get E18-F05-001 --json | jq -r '.status'`. This adds a dependency on external tools, introduces parsing fragility, and increases command complexity. Agent sessions that cannot install `jq` resort to regex parsing of human-readable output, which is brittle and error-prone.

### Solution

Add a `--field <name>` flag to `get`, `list`, `next`, `progress`, and `status` commands. When specified, the command returns only the raw value of the requested field -- no JSON wrapping, no quotes around strings, just the plain value. This eliminates the need for any external post-processing.

### What This Feature Does

- Adds `--field <name>` flag as a global or per-command flag on `get`, `list`, `next`, `progress`, and `status options` commands
- When `--field` is specified, output is the raw field value (no JSON envelope, no quotes for strings)
- For list commands, outputs one value per line (one per entity)
- For array fields (like `valid_transitions`), outputs comma-separated values
- Implicitly enables JSON processing internally (no need to also pass `--json`)
- Returns exit code 1 with a `FIELD_NOT_FOUND` error if the field does not exist on the entity
- Simple top-level field access only (no nested dotted paths in this phase)

### What This Feature Does NOT Do

- Does not support nested field paths (e.g., `--field progress.weighted`) -- that can be added later
- Does not change the behavior of any existing flag or output format
- Does not add new data fields -- only extracts existing ones
- Does not affect human-readable or JSON output when `--field` is not specified

---

## Acceptance Criteria

- [ ] `shark get E18-F05-001 --field status` returns `in_development` (raw string, no quotes)
- [ ] `shark get E18-F05-001 --field title` returns `Implement JWT tokens`
- [ ] `shark get E18-F05-001 --field key` returns `E18-F05-001`
- [ ] `shark next --agent developer --field key` returns `E18-F05-003`
- [ ] `shark list E18-F05 --field key` returns one key per line
- [ ] `shark progress E18-F05 --field progress_pct` returns `78.5` (when F06 exists)
- [ ] `shark status options E18-F05-001 --field valid_transitions` returns comma-separated list
- [ ] Exit code 1 if field does not exist in the response
- [ ] Structured JSON error (`FIELD_NOT_FOUND`) when field does not exist and JSON mode is active
- [ ] Implicitly enables JSON processing (no need for `--json` alongside `--field`)
- [ ] All existing tests pass without modification (`make test` green)

---

## Dependencies

### Depends On

None. This is standalone. However, the `FIELD_NOT_FOUND` structured error benefits from E17-F03 (Structured JSON Error Output) being implemented first.

### Depended On By

- **E17-F06 (Progress Command)**: The progress command supports `--field` for targeted extraction of progress metrics.

---

## Implementation Notes

- Implement as middleware in the output formatting layer, likely in `internal/cli/` or `internal/formatters/`
- Approach: marshal entity to JSON internally, then extract the requested top-level key from the JSON map
- For `list` commands, iterate over the result array and extract the field from each item
- Register `--field` as a persistent flag on the root command (available to all relevant subcommands)
- When `--field` is present, skip normal JSON/table formatting and write the raw value to stdout
- For the `FIELD_NOT_FOUND` error, return exit code 1 (consistent with the NOT_FOUND exit code from F03)
- Consider using `encoding/json` to marshal to `map[string]interface{}` and do a simple key lookup

---

## Success Metrics

- **Primary**: External post-processing reduced from 15% to less than 3% of agent commands
- **Measured by**: Count of `jq` and Python post-processing in agent logs
- **Backward Compatibility**: 100% -- no existing behavior changes when `--field` is not used

---

## UAT Scenarios

- J1-S01: Get task status with `--field status`
- J1-S04: Extract key from next task with `--field key`
- J4-S02: Extract progress metrics with `--field`
- BC-10: Existing `--json` output unchanged when `--field` not used

---

*Last Updated*: 2026-02-25
