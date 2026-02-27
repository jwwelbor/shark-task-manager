---
feature_key: E17-F05
epic_key: E17
title: Flag Normalization
description: Normalize inconsistent flag names across all commands, making --order the primary name (replacing --execution-order) and --all the primary name (replacing --show-all), with backward-compatible hidden aliases.
execution_order: 2
phase: 1
complexity: XS
status: draft
dependencies: []
depended_on_by:
  - E17-F08 (Phase 2 unified create uses normalized flags)
epic_requirements:
  - F05 (Flag Normalization)
  - NFR-1 (Backward Compatibility)
---

# Flag Normalization

**Feature Key**: E17-F05
**Phase**: 1 (Must-Have)
**Complexity**: XS
**Execution Order**: 2 (mechanical change, zero risk, builds on existing --order alias)

---

## Scope

### Problem

Flag names are inconsistent across commands. Features use `--execution-order` while tasks use `--order` as an alias. List commands use `--show-all` instead of the shorter `--all`. AI agents, which have no persistent memory of flag names between sessions, encounter these inconsistencies repeatedly, leading to trial-and-error flag usage.

### Solution

Normalize flag names using Cobra's built-in `MarkDeprecated()` mechanism:
- `--order` becomes the primary name everywhere (replaces `--execution-order`)
- `--all` becomes the primary name on list commands (replaces `--show-all`)
- Old flag names continue to work as hidden aliases with deprecation warnings on stderr

### What This Feature Does

- Swaps `--execution-order` to be the deprecated alias and `--order` to be the primary name
- Adds `--all` as primary flag on list commands, deprecates `--show-all`
- Uses Cobra's `MarkDeprecated()` which shows stderr warnings but keeps the old flags functional
- Deprecation warnings appear on stderr only, never on stdout, never in JSON mode

### What This Feature Does NOT Do

- Does not remove any existing flags
- Does not change flag semantics or values
- Does not add new functionality beyond renaming
- Does not affect JSON output format

---

## Acceptance Criteria

- [ ] `--order` accepted everywhere that `--execution-order` was accepted
- [ ] `--execution-order` kept as hidden alias, produces deprecation warning on stderr
- [ ] `--all` replaces `--show-all` on list commands
- [ ] `--show-all` kept as hidden alias, produces deprecation warning on stderr
- [ ] Deprecation warnings go to stderr only (never stdout, never in JSON output)
- [ ] No deprecation warnings when `--json` or `SHARK_OUTPUT=json` is active
- [ ] All existing tests pass without modification (`make test` green)
- [ ] `--help` output shows new primary flag names

---

## Dependencies

### Depends On

None. This is a standalone flag renaming exercise.

### Depended On By

- **E17-F08 (Phase 2)**: Unified create dispatcher uses normalized flag names.

---

## Implementation Notes

- Audit all commands in `internal/cli/commands/` for `--execution-order` and `--show-all` usage
- Use `cmd.Flags().MarkDeprecated("execution-order", "use --order instead")`
- Use `cmd.Flags().MarkDeprecated("show-all", "use --all instead")`
- Check `internal/cli/commands/shared_flags.go` for centralized flag definitions
- Existing `--order` alias on task create already exists -- swap primary/deprecated
- Cobra's deprecation mechanism is well-tested and handles the warning output automatically

---

## Success Metrics

- **Primary**: 100% flag name consistency across all commands
- **Measured by**: Audit of all commands shows `--order` and `--all` as primary names
- **Backward Compatibility**: 100% -- old flags still work with deprecation warning

---

## UAT Scenarios

- J3-S03: Deprecated flag still works
- J3-S01: Unified create feature with normalized flags
- BC-08: `--execution-order` still accepted (deprecated but functional)
- BC-09: `--show-all` still accepted (deprecated but functional)

---

*Last Updated*: 2026-02-25
