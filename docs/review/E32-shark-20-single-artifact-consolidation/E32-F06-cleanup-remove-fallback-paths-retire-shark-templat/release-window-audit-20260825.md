# E32-F06 release-window and harness audit

**Audited:** 2026-08-25

## Release evidence

- `v2.0.0`: 2026-07-26 21:07:56 -0500
- `v2.0.1`: 2026-08-04 18:58:53 -0500

Both tags post-date the F05 hold decision of 2026-06-22. The interval before
this audit is more than one normal-use day.

## External harness audit

The eight deprecated paths were checked individually and are absent:
`run.md`, `feature.md`, `epic.md`, `task.md`, `prd.md`, `dispatch.md`,
`develop.md`, and `release.md` under `~/.claude/commands/`.

`~/.claude/hooks/` was inspected read-only. Repository `scripts/` contains no
`shark-templates` hardcoded path. No external file was deleted by this audit.

## Result

The release-window prerequisite for F06 command retirement is satisfied; the
current absence of the exact eight paths satisfies the external command state.
