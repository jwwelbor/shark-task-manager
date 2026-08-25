# E32-F06 release-window and harness audit

**Audited:** 2026-08-25

## Release evidence

- `E32-F05` was completed in the live tracker on 2026-06-23 after its five
  task records completed. Its declared contract explicitly deprecates the same
  eight slash commands for removal in F06.
- `v2.0.0`: 2026-07-26 21:07:56 -0500
- `v2.0.1`: 2026-08-04 18:58:53 -0500

Both tags post-date the F05 completion. The v2.0.0 to v2.0.1 release window
spans nine calendar days and contains normal product use: merged work included
E38-F09 provider-neutral coordination (#149), E38-F12 team topology (#150),
the walkthrough safety fix (#151), and E34-F04 question management (#152).
Subsequent merged releases continued through 2026-08-24 (including E40 #183,
#187, #189, and #190). This records observable use of the post-F05 delivery,
not merely elapsed time.

## External harness audit

The eight deprecated paths were checked individually and are absent:
`run.md`, `feature.md`, `epic.md`, `task.md`, `prd.md`, `dispatch.md`,
`develop.md`, and `release.md` under `~/.claude/commands/`.

`~/.claude/hooks/` was inspected read-only. Repository `scripts/` contains no
`shark-templates` hardcoded path. No external file was deleted by this audit.

## Result

The release-window prerequisite for F06 command retirement is satisfied: F05's
named commands had a qualifying post-completion release and normal-use window;
the current absence of the exact eight paths satisfies the external command
state.
