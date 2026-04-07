# Technical Specification: E25-F02 — Show File Location on Entity Create

**Feature Key**: E25-F02-show-file-location-on-entity-create
**Epic**: E25 — Add Tech-Debt Entity Type
**Status**: Ready for Development
**Complexity**: XS (3 one-line additions across 3 files, zero service changes)

---

## Overview

Bug and change-card creation commands do not display the file path of the created markdown file. Task, feature, and epic creation already do. This spec defines the minimal changes to close that gap, and documents the pattern the tech-debt create command (E25-F01) must follow.

Delivers SC-6 from the epic success criteria.

---

## Functional Requirements

### FR-01: Bug create displays file path

After `shark bug create` succeeds (human-readable output path), the CLI must print the file path of the created markdown file on the line immediately following the success message.

**Acceptance criteria:**
- [ ] `shark bug create "Title"` outputs a line beginning with `File:` containing the relative file path
- [ ] File path is non-empty (service already populates `bug.FilePath`)
- [ ] When `--json` is used, no change — `file_path` is already serialized in the JSON output via `BaseEntity`

### FR-02: Change-card create displays file path

Both command paths that create a change-card must display the file path:

- `shark change create "Title"` — handled by `runChangeCreate` in `change.go`
- `shark create change "Title"` — handled by `runChangeCardCreate` in `change_card_commands.go`

**Acceptance criteria:**
- [ ] Both `shark change create` and `shark create change` output a `File:` line after the success message
- [ ] Both paths display the same file path for the same entity
- [ ] `--json` output is unchanged

### FR-03: Tech-debt create displays file path (dependency on E25-F01)

When the tech-debt create command is implemented in E25-F01, it must follow the same inline pattern. This feature documents the requirement; E25-F01 implements it.

**Acceptance criteria:**
- [ ] `shark td create "Title"` outputs a `File:` line after the success message
- [ ] The tech-debt service populates `td.FilePath` before returning (E25-F01 responsibility)

---

## Non-Functional Requirements

### NFR-01: Consistent output pattern

All entity creation commands that handle file creation in the service layer must use the same inline output pattern:

```
<success line>
File: <relative-path>
```

This matches the existing task create pattern. The `FormatEntityCreationMessage()` pattern (verbose PLACEHOLDER block) is reserved for epic and feature creation, where the output file requires human editing of required sections.

---

## Architecture

### Files to Modify

| File | Handler | Line | Change |
|------|---------|------|--------|
| `internal/cli/commands/bug.go` | `runBugCreate` | ~238 | Add `cli.Info` after success message |
| `internal/cli/commands/change.go` | `runChangeCreate` | ~228 | Add `cli.Info` after success message |
| `internal/cli/commands/change_card_commands.go` | `runChangeCardCreate` | ~57 | Add `cli.Info` after success message |

### No service changes required

Both `BugService.CreateBug()` and `ChangeCardService.CreateChangeCard()` already populate `FilePath` on the returned entity. The commands simply never read it for output.

### Pattern

The three-line addition is identical for all affected files:

```go
if fp := entity.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
```

`GetFilePath()` is defined on `BaseEntity` (`internal/models/entity.go:68`) and is nil-safe. No import changes are needed in any of the three files.

### Existing patterns for reference

**Task create (correct pattern to follow):**
```go
// internal/cli/commands/task.go:221-228
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(task)
}
cli.Success(fmt.Sprintf("Created task %s", task.Key))
if fp := derefString(task.FilePath); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
```

Note: Task uses `derefString(task.FilePath)` while the recommended pattern uses `task.GetFilePath()`. Both are equivalent. Use `GetFilePath()` for the new additions as it is more expressive and nil-safe.

**Bug create (before):**
```go
// internal/cli/commands/bug.go:234-239
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
return nil
```

**Bug create (after):**
```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
if fp := bug.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

Apply the same diff to `change.go` (variable name `card`) and `change_card_commands.go` (variable name `card`).

### Tech-debt create pattern (E25-F01 guidance)

When `tech_debt.go` is implemented:

```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(td)
}
cli.Success(fmt.Sprintf("Created tech-debt %s: %s", td.Key, td.Title))
if fp := td.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

E25-F01 must also set `td.FilePath` in the service before returning, following the bug service pattern at `internal/services/bug_service.go:138-145`.

---

## Test Plan

### Manual smoke tests (required)

1. `shark bug create "Test bug for file path display"` — verify output includes `File:` line with a relative path ending in `.md`
2. `shark change create "Test change-card for file path display"` — verify `File:` line present
3. `shark create change "Test change via create command"` — verify `File:` line present (second code path)
4. `shark bug create "Test bug" --json` — verify JSON output is unchanged (file_path field was already present)

### Regression tests (required)

5. `shark task create E01 F01 "Regression task"` — verify file path still displayed (existing behavior unchanged)
6. `shark epic create "Regression epic"` — verify full PLACEHOLDER block still displayed (existing behavior unchanged)
7. `shark feature create E01 "Regression feature"` — verify full PLACEHOLDER block still displayed

### Path consistency check

8. Note the file path printed by bug create and verify the file exists at that path: `ls <printed-path>`
9. Note the file path printed by change create and verify the file exists at that path: `ls <printed-path>`

---

## Dependencies

- **E25-F01**: Tech-debt create command and service must be available for FR-03 testing. FR-01 and FR-02 are independent of E25-F01 and can be developed in parallel.
- **No schema changes**: This feature requires no database changes and no migration.

---

## Out of Scope

- Displaying file path in `shark bug get`, `shark change get` — those commands already show file information in their detail output.
- Modifying `shark epic create` or `shark feature create` — those commands already use `FormatEntityCreationMessage()` which includes the file path.
- Changing the JSON output format — `file_path` is already present in JSON output for all entity types via `BaseEntity`.

---

*Last Updated*: 2026-04-05
