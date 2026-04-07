# E25-F02 Research Report: Show File Location on Entity Create

## Executive Summary

Research into all entity creation commands reveals that task, feature, and epic creation already display the file path using two different patterns (a dedicated `FormatEntityCreationMessage()` helper and inline `cli.Info()` calls). Bug and change-card creation commands omit the file path entirely despite both services already computing and storing `FilePath` on the returned entity. The fix is a one-line addition in each affected command: read `entity.GetFilePath()` and emit it with `cli.Info()`. The tech-debt create command (E25-F01) must follow the same inline pattern when implemented.

---

## Research Questions

1. Which entity creation commands display the file path, and how?
2. Where is `FormatEntityCreationMessage()` defined and what does it do?
3. Which commands are missing file path display?
4. Do the bug and change-card services already return `FilePath` on the created entity?
5. What is the recommended unified pattern for the missing commands?
6. How should the tech-debt create command (E25-F01) integrate this?

---

## Findings

### Finding 1: FormatEntityCreationMessage() Helper

**Location:** `internal/cli/output.go:28`

The helper has this signature:

```go
func FormatEntityCreationMessage(
    entityType, entityKey, entityTitle, filePath, projectRoot string,
    fileWasLinked bool,
    requiredSections []string,
) string
```

It produces a multi-line output block including:
- A success header line (`Created <type> <key>: <title>`)
- Either a "LINKED TO EXISTING FILE" block or a "PLACEHOLDER FILE CREATED" block
- The file path on its own `File: <path>` line
- A list of required sections the user must fill in

This verbose format is appropriate for entity types that create markdown plan documents requiring human editing (epics, features, tasks). It is not appropriate as-is for bug or change-card creation, which produce self-contained markdown files and do not require section editing.

A companion `FormatEntityCreationJSON()` helper at `internal/cli/output.go:55` formats JSON creation responses including `file_path`, `file_state`, and `required_actions`.

---

### Finding 2: Current State Per Entity

#### Epic — SHOWS FILE PATH

**File:** `internal/cli/commands/epic_helpers.go`
**Handler:** `runEpicCreate` (called from `epic_helpers.go`)
**Output lines:** 691–697

```go
// Line 692-697
requiredSections := cli.GetRequiredSectionsForEntityType("epic")
if cli.GlobalConfig.JSON {
    jsonOutput := cli.FormatEntityCreationJSON("epic", nextKey, epicTitle, actualFilePath, projectRoot, requiredSections)
    return cli.OutputJSON(jsonOutput)
}

message := cli.FormatEntityCreationMessage("epic", nextKey, epicTitle, actualFilePath, projectRoot, fileWasLinked, requiredSections)
fmt.Print(message)
```

Pattern: `FormatEntityCreationMessage()` with verbose PLACEHOLDER/LINKED output block.

#### Feature — SHOWS FILE PATH

**File:** `internal/cli/commands/feature.go`
**Handler:** `runFeatureCreate`
**Output lines:** 295–299

```go
// Lines 295-299
requiredSections := cli.GetRequiredSectionsForEntityType("feature")
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(cli.FormatEntityCreationJSON("feature", feature.Key, featureTitle, featureFilePath, projectRoot, requiredSections))
}
fmt.Print(cli.FormatEntityCreationMessage("feature", feature.Key, featureTitle, featureFilePath, projectRoot, fileWasLinked, requiredSections))
```

Pattern: Same `FormatEntityCreationMessage()` helper.

#### Task — SHOWS FILE PATH

**File:** `internal/cli/commands/task.go`
**Handler:** `runTaskCreate`
**Output lines:** 221–228

```go
// Lines 221-228
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(task)
}
cli.Success(fmt.Sprintf("Created task %s", task.Key))
if fp := derefString(task.FilePath); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
```

Pattern: Inline `cli.Info()` call reading `task.FilePath` via the `derefString()` helper. Does NOT use `FormatEntityCreationMessage()`. The service populates `task.FilePath` before returning.

#### Bug — DOES NOT SHOW FILE PATH

**File:** `internal/cli/commands/bug.go`
**Handler:** `runBugCreate`
**Output lines:** 234–239

```go
// Lines 234-239
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
return nil
```

No file path output. The `bug.FilePath` field IS populated by the service (see below), but the command ignores it.

#### Change-Card (via shark change create) — DOES NOT SHOW FILE PATH

**File:** `internal/cli/commands/change.go`
**Handler:** `runChangeCreate`
**Output lines:** 224–229

```go
// Lines 224-229
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
return nil
```

**File:** `internal/cli/commands/change_card_commands.go`
**Handler:** `runChangeCardCreate` (called via `shark create change` and `shark create change-card`)
**Output lines:** 53–58

```go
// Lines 53-58
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
return nil
```

Both handlers have the same gap. No file path output.

---

### Finding 3: Service-Layer FilePath Population

Both bug and change-card services already compute and store `FilePath` on the entity before returning it.

**Bug service** (`internal/services/bug_service.go`, lines 138–145):
```go
var filePath string
if input.FilePath != nil && *input.FilePath != "" {
    filePath = *input.FilePath
} else {
    filePath = filepath.Join("docs", "plan", "bugs", key+".md")
}
bug.FilePath = &filePath
```

**Change-card service** (`internal/services/change_card_service.go`, line 151):
```go
card.FilePath = &filePath
```

Both services return entities with `FilePath` populated. The commands simply never read it for output. This means the fix requires zero service changes.

---

### Finding 4: FilePath on BaseEntity

`FilePath *string` is defined on `BaseEntity` (`internal/models/entity.go:42`) with an accessor:

```go
func (b *BaseEntity) GetFilePath() string {
    if b.FilePath != nil {
        return *b.FilePath
    }
    return ""
}
```

All entity types (Bug, ChangeCard, TechDebt) embed `BaseEntity` and thus all have `GetFilePath()` available. The commands can call it directly without type assertion or nil checking.

---

### Finding 5: Tech-Debt Integration (E25-F01)

The `TechDebt` model (`internal/models/tech_debt.go:55`) embeds `BaseEntity`:

```go
type TechDebt struct {
    BaseEntity
    Status   TechDebtStatus   `json:"status" db:"status"`
    Category TechDebtCategory `json:"category" db:"category"`
    Severity TechDebtSeverity `json:"severity" db:"severity"`
    EffortEstimate *string    `json:"effort_estimate,omitempty" db:"effort_estimate"`
}
```

The tech-debt service (to be implemented in E25-F01) must set `td.FilePath` before returning, following the exact same pattern as the bug service. The tech-debt create CLI command will then display the file path using the inline `cli.Info()` pattern.

---

## Pattern Analysis

Three distinct patterns exist across the codebase:

| Pattern | Used By | Verbosity | Appropriate For |
|---------|---------|-----------|-----------------|
| `FormatEntityCreationMessage()` | Epic, Feature | High — full PLACEHOLDER/LINKED block with required sections | Plan docs requiring human editing |
| Inline `cli.Info("File: %s", fp)` | Task | Low — one extra line | Entities where service returns FilePath |
| No file path at all | Bug, Change-Card, (TD — not built yet) | None | Gap to fix |

The `FormatEntityCreationMessage()` pattern is NOT appropriate for bug and change-card because:
1. Those entities do not have "required sections" in the same sense (no `GetRequiredSectionsForEntityType("bug")` section called).
2. Their file creation is handled entirely in the service layer; there is no separate `fileWasLinked` flag available in the command.
3. The verbose PLACEHOLDER block is designed for entities where the user must manually edit the file before implementation can proceed. Bug and change-card files are self-contained at creation.

The **inline `cli.Info()` pattern from task.go** is the correct approach for bug, change-card, and tech-debt.

---

## What Needs to Change

### bug.go — `runBugCreate` (line 238)

**Before (lines 234–239):**
```go
// Step 3: Format output
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
return nil
```

**After:**
```go
// Step 3: Format output
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(bug)
}
cli.Success(fmt.Sprintf("Created bug %s: %s", bug.Key, bug.Title))
if fp := bug.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

No import changes needed. `GetFilePath()` is inherited from `BaseEntity`. `cli.Info` is already imported.

---

### change.go — `runChangeCreate` (line 228)

**Before (lines 224–229):**
```go
// Step 3: Format output
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
return nil
```

**After:**
```go
// Step 3: Format output
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
if fp := card.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

---

### change_card_commands.go — `runChangeCardCreate` (line 57)

**Before (lines 53–58):**
```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
return nil
```

**After:**
```go
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(card)
}
cli.Success(fmt.Sprintf("Created change-card %s: %s", card.Key, card.Title))
if fp := card.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

---

### tech_debt.go — `runTechDebtCreate` (E25-F01, not yet built)

When the tech-debt create command is implemented, it must follow the same inline pattern:

```go
// Step 3: Format output
if cli.GlobalConfig.JSON {
    return cli.OutputJSON(td)
}
cli.Success(fmt.Sprintf("Created tech-debt %s: %s", td.Key, td.Title))
if fp := td.GetFilePath(); fp != "" {
    cli.Info(fmt.Sprintf("File: %s", fp))
}
return nil
```

The tech-debt service must also set `td.FilePath` before returning, following the bug service pattern in `internal/services/bug_service.go:138–145`.

---

## Recommended Unified Approach

Use the **inline `cli.Info()` pattern** for all entities whose file creation is handled by the service layer (bug, change-card, tech-debt). Reserve the `FormatEntityCreationMessage()` pattern for entities whose file is created in the command layer with explicit `fileWasLinked` tracking (epic, feature).

Tasks use the inline pattern even though file creation happens in the service, which is consistent — the inline pattern is the simpler, correct approach for service-managed entities.

Summary of recommended output format for each entity type:

```
# Bug
✅ Created bug B001: Login page crashes on Safari
File: docs/plan/bugs/B001.md

# Change-Card
✅ Created change-card CC-001: Migrate auth to OAuth2
File: docs/plan/change-cards/CC-001.md

# Tech-Debt (E25-F01)
✅ Created tech-debt TD-001: Fix N+1 queries in task list
File: docs/plan/tech-debt/TD-001.md
```

---

## Integration with E25-F01

E25-F01 implements the tech-debt service and CLI commands. E25-F02 can be developed in parallel for bug and change-card. The tech-debt create command output is only testable end-to-end once `tech_debt.go` is available from E25-F01.

**Dependency:** The tech-debt service (E25-F01) must set `td.FilePath` before returning from `CreateTechDebt()`. This is the only service-layer requirement for E25-F02.

---

## Constraints Identified

- **Technical:** Zero service-layer changes needed for bug and change-card. Both services already return `FilePath` populated.
- **Technical:** No new helper functions needed. `GetFilePath()` is on `BaseEntity`.
- **Technical:** No import changes needed in bug.go, change.go, or change_card_commands.go.
- **External:** Tech-debt CLI display (the TD-### case) depends on E25-F01 delivering `tech_debt.go`.

---

## Risks and Unknowns

- **Risk (Low):** The JSON output path (`cli.OutputJSON(bug)`) already serializes `file_path` via the struct's JSON tags because `BaseEntity.FilePath` has `json:"file_path,omitempty"`. No change needed for JSON callers — they already receive the file path. The gap is human-readable output only.
- **Unknown:** Whether change-card service populates a relative or absolute path. The bug service produces a relative path (`docs/plan/bugs/B001.md`). Consistency check recommended when running manual smoke tests after the fix.

---

## References

- `internal/cli/output.go` — `FormatEntityCreationMessage()` and `FormatEntityCreationJSON()` definitions
- `internal/cli/commands/task.go:214–229` — Task create inline pattern
- `internal/cli/commands/epic_helpers.go:688–698` — Epic create FormatEntityCreationMessage pattern
- `internal/cli/commands/feature.go:278–300` — Feature create FormatEntityCreationMessage pattern
- `internal/cli/commands/bug.go:202–239` — Bug create handler (gap)
- `internal/cli/commands/change.go:211–229` — Change create handler (gap)
- `internal/cli/commands/change_card_commands.go:21–58` — Change-card create via `shark create change` (gap)
- `internal/models/entity.go:42,68–72` — BaseEntity FilePath field and GetFilePath() accessor
- `internal/services/bug_service.go:138–145` — Bug service FilePath population
- `internal/services/change_card_service.go:151` — Change-card service FilePath population
- `internal/models/tech_debt.go:55–61` — TechDebt model struct (inherits BaseEntity)
- `docs/plan/E25-add-tech-debt-entity-type/epic.md` — SC-6 success criterion
