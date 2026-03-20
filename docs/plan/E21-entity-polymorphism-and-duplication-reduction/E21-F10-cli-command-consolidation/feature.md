---
feature_key: E21-F10-cli-command-consolidation
epic_key: E21
title: CLI Command Consolidation
description: Replace duplicated per-entity CLI command files with generic polymorphic command handlers
---

# CLI Command Consolidation

**Feature Key**: E21-F10-cli-command-consolidation

---

## Epic

- **Epic PRD**: [Epic](../../epic.md)

---

## Goal

### Problem

The CLI layer has the highest duplication in the entire codebase. Common operations (notes, context, status transitions, resume) are implemented as **separate command files per entity type**, with 65-95% of the code being identical copy-paste with entity names swapped.

**Current State — Per-Entity Command Files:**

| Operation | Files | Total Lines | Structural Overlap |
|-----------|-------|------------|-------------------|
| Note add/list | `epic_note.go` (163), `feature_note.go` (163), `task_note.go` (358) | 684 | **86% identical** (epic vs feature) |
| Context get/set/clear | `epic_context.go` (237), `feature_context.go` (165), `task_context.go` (243) | 645 | ~65% identical |
| Next-status | `epic_next_status.go` (269), `feature_next_status.go` (166), `task_next_status.go` (324) | 759 | ~70% identical |
| Set-status | `epic_set_status.go` (99), `feature_set_status.go` (99) | 198 | **95% identical** |
| Resume | `epic_resume.go`, `feature_resume.go`, `task_resume.go` | ~400 | ~70% identical |
| **TOTAL** | **~15 files** | **~2,686** | |

**Concrete example**: `epic_note.go` and `feature_note.go` are 163 lines each and differ in exactly **23 lines** — every difference is just the word "epic" replaced with "feature" and `models.EntityTypeEpic` replaced with `models.EntityTypeFeature`.

This means **~1,500 lines** of the CLI codebase are redundant copies.

### Solution

Create **generic polymorphic command handlers** that accept entity type as a parameter, dispatched based on key format auto-detection (which already exists in the codebase). Each operation becomes a single implementation registered under multiple parent commands.

### Impact

- Eliminate **~1,500 lines** of duplicated CLI command code (~15 files consolidated to ~5)
- Bug fixes to note/context/status commands apply once instead of 3-5 times
- Adding a new entity type's CLI support requires **zero new command files** for shared operations
- Bugs and ChangeCards automatically get note/context/status commands (currently some are missing)

---

## User Personas

### Persona 1: Go Developer (CLI Maintainer)

**Goals**:
1. Fix a bug in note display formatting once, not in 3 files
2. Add `--format` flag to context commands without editing 3 files
3. Add CLI support for a new entity type without creating 5+ new command files

**Pain Points**:
- Changed note output formatting in `epic_note.go`, forgot to update `feature_note.go` — inconsistent behavior
- Adding ChangeCard note support required creating `change_card_note.go` as a copy of `epic_note.go` with find-and-replace
- PR reviews for simple changes touch 3-5 files that are near-identical

---

## Current State: Duplication Evidence

### Example: epic_note.go vs feature_note.go

Lines that differ (23 of 163):
```
Line 13: epicNoteCmd vs featureNoteCmd
Line 15: "Manage epic notes" vs "Manage feature notes"
Line 21: "add <epic-key>" vs "add <feature-key>"
Line 37: "shark epic" vs "shark feature"
Line 60: epicKey vs featureKey (variable name)
Line 76: models.EntityTypeEpic vs models.EntityTypeFeature
Line 78: "epic" vs "feature" (string)
Line 91: "epic" vs "feature" (output)
Line 99: epicKey vs featureKey (variable name)
Line 117: models.EntityTypeEpic vs models.EntityTypeFeature
Line 119: "epic" vs "feature" (string)
Line 128-132: "epic" vs "feature" (strings)
Line 150: epicCmd vs featureCmd (parent)
Line 154: epicNoteCmd vs featureNoteCmd
```

**140 of 163 lines are identical.** This is the definition of DRY violation.

### Example: epic_set_status.go vs feature_set_status.go

99 lines each, **95% identical**. Differences are only entity type enum and parent command registration.

---

## User Stories

### Must-Have Stories

**Story 1**: As a developer, I want a single `runNoteAdd` handler that works for any entity type so that note behavior is consistent and maintained in one place.

**Acceptance Criteria**:
- [ ] Single `note_generic.go` (or similar) contains note add/list logic
- [ ] Entity type determined from key format or parent command context
- [ ] `shark epic note add E21 --content="..."` and `shark feature note add E21-F07 --content="..."` both use the same handler
- [ ] Old per-entity note files (`epic_note.go`, `feature_note.go`) removed or reduced to registration-only stubs
- [ ] Bug and ChangeCard note commands work automatically (currently may be missing)

**Story 2**: As a developer, I want a single `runContextGetSetClear` handler that works for any entity type.

**Acceptance Criteria**:
- [ ] Single context command handler with entity type parameter
- [ ] `shark epic context set E21 --field=phase --value=dev` and `shark task context set E21-F07-001 --field=step --value=impl` use same handler
- [ ] Per-entity context files consolidated

**Story 3**: As a developer, I want a single `runSetStatus` handler that works for any entity type.

**Acceptance Criteria**:
- [ ] Single set-status handler
- [ ] `shark status set E21 active` and `shark status set E21-F07-001 in_development` use same handler
- [ ] Per-entity set-status files consolidated

**Story 4**: As a developer, I want a single `runNextStatus` handler that works for any entity type.

**Acceptance Criteria**:
- [ ] Single next-status/advance handler
- [ ] Per-entity next-status files consolidated
- [ ] Entity-specific display differences (e.g., Task shows agent, Feature shows progress) handled via formatting callbacks or templates

**Story 5**: As a developer, I want a single `runResume` handler that works for any entity type.

**Acceptance Criteria**:
- [ ] Single resume handler
- [ ] Per-entity resume files consolidated

---

### Should-Have Stories

**Story 6**: As a developer, I want the generic handlers to support entity-specific output formatting so that task output can show agent/priority while epic output shows feature counts.

**Acceptance Criteria**:
- [ ] Generic handler accepts optional formatting callback or display config per entity type
- [ ] Default formatting works for all entity types
- [ ] Entity-specific detail sections (e.g., task deps, epic rollups) added via hooks

---

## Requirements

### Functional Requirements

**Category: Generic Command Handlers**

1. **REQ-F-001**: Entity Type Resolution
   - **Description**: Generic handlers determine entity type from key format or command context
   - **Priority**: Must-Have
   - **Pattern**:
     ```go
     func resolveEntityType(key string) (models.EntityType, error) {
         // Already exists in codebase — key format auto-detection
         // E## -> Epic, E##-F## -> Feature, E##-F##-### -> Task, B### -> Bug, CC-### -> ChangeCard
     }
     ```

2. **REQ-F-002**: Generic Note Handler
   - **Description**: Single handler for note add and note list operations
   - **Priority**: Must-Have
   - **Pattern**:
     ```go
     func runGenericNoteAdd(cmd *cobra.Command, args []string) error {
         key := args[0]
         entityType, err := resolveEntityType(key)
         if err != nil { return err }

         content, _ := cmd.Flags().GetString("content")
         noteType, _ := cmd.Flags().GetString("type")

         svc := cli.GetNoteService()
         note, err := svc.AddNote(cmd.Context(), entityType, key, content, noteType)
         if err != nil { return err }

         if cli.GlobalConfig.JSON { return cli.OutputJSON(note) }
         cli.Success(fmt.Sprintf("Added %s note to %s %s", noteType, entityType, key))
         return nil
     }
     ```

3. **REQ-F-003**: Generic Context Handler
   - **Description**: Single handler for context get/set/clear operations
   - **Priority**: Must-Have

4. **REQ-F-004**: Generic Status Handlers
   - **Description**: Single handlers for set-status and next-status/advance operations
   - **Priority**: Must-Have

5. **REQ-F-005**: Generic Resume Handler
   - **Description**: Single handler for resume operations
   - **Priority**: Must-Have

6. **REQ-F-006**: Command Registration Pattern
   - **Description**: Generic handlers registered as subcommands under each entity parent command
   - **Priority**: Must-Have
   - **Pattern**:
     ```go
     // Registration — each entity parent gets the same subcommand pointing to generic handler
     func init() {
         epicCmd.AddCommand(makeNoteCmd("epic"))
         featureCmd.AddCommand(makeNoteCmd("feature"))
         taskCmd.AddCommand(makeNoteCmd("task"))
         bugCmd.AddCommand(makeNoteCmd("bug"))
     }

     func makeNoteCmd(entityName string) *cobra.Command {
         return &cobra.Command{
             Use:   "note",
             Short: fmt.Sprintf("Manage %s notes", entityName),
             // ... subcommands pointing to generic handlers
         }
     }
     ```

**Category: Entity-Specific Formatting**

7. **REQ-F-007**: Formatting Hooks
   - **Description**: Generic handlers support entity-specific output sections via formatting callbacks
   - **Priority**: Should-Have
   - **Pattern**:
     ```go
     type EntityDisplayConfig struct {
         EntityType    models.EntityType
         ExtraHeaders  []string                    // additional table columns
         ExtraFields   func(entity) []string       // additional field values
         DetailSection func(entity) string          // entity-specific detail text
     }
     ```

### Non-Functional Requirements

**Backward Compatibility**

1. **REQ-NF-001**: CLI Interface Preservation
   - **Description**: All existing command syntaxes must continue to work identically
   - **Measurement**: `shark epic note add E21 --content="test"` produces same output as before
   - **Target**: Zero user-facing changes

2. **REQ-NF-002**: Help Text Accuracy
   - **Description**: `shark epic note --help` still shows epic-specific examples and descriptions
   - **Measurement**: Help output reviewed for each entity type
   - **Target**: Help text is entity-aware despite using generic handlers

---

## Acceptance Criteria

### Feature-Level Acceptance

**Scenario 1: Note Add (Any Entity)**
- **Given** the generic note handler is registered
- **When** `shark epic note add E21 --content="test"` is run
- **Then** behavior is identical to the old `epic_note.go` implementation
- **And** `shark bug note add B001 --content="test"` also works (new capability)

**Scenario 2: Status Set (Any Entity)**
- **Given** the generic set-status handler is registered
- **When** `shark status set E21-F07 active` is run
- **Then** behavior is identical to the old `feature_set_status.go` implementation

**Scenario 3: File Count Reduction**
- **Given** per-entity command files existed for notes, context, status, resume
- **When** consolidation is complete
- **Then** ~15 per-entity command files are replaced by ~5 generic files + registration stubs

---

## Out of Scope

### Explicitly Excluded

1. **CRUD Command Consolidation** (create, get, list, update, delete)
   - **Why**: These have more entity-specific logic (Task has agent/priority/deps, Feature has progress, etc.)
   - **Future**: Could be partially consolidated once BaseEntity embedding (F07) reduces model differences

2. **Auto-Detection for All Commands**
   - **Why**: Some commands (e.g., `shark task deps`) are entity-specific by nature
   - **Future**: Only generic operations benefit from consolidation

---

## Design Notes

### Approach: Command Factory + Generic Handlers

Two patterns considered:

**Option A: Single command with auto-detection** (like `shark get <key>`)
```bash
shark note add E21 --content="..."         # auto-detects epic
shark note add E21-F07-001 --content="..." # auto-detects task
shark note add B001 --content="..."        # auto-detects bug
```

**Option B: Entity subcommand with generic handler** (keeps current CLI structure)
```bash
shark epic note add E21 --content="..."
shark task note add E21-F07-001 --content="..."
shark bug note add B001 --content="..."
```

**Recommendation**: Option B preserves backward compatibility while Option A can be added as an alias. The generic handler is the same either way — only registration differs.

### Migration Strategy

1. Create generic handlers alongside existing per-entity files
2. Re-point entity command registrations to generic handlers
3. Verify all tests pass
4. Remove old per-entity command files
5. Update any tests that import old command functions

---

## Estimated Impact

### File and Line Reduction

| Operation | Current Files | Current Lines | After | Savings |
|-----------|--------------|--------------|-------|---------|
| Note | 3 files | 684 | 1 file (~120) + 3 stubs (~15 each) | ~520 |
| Context | 3 files | 645 | 1 file (~150) + 3 stubs (~15 each) | ~450 |
| Next-status | 3 files | 759 | 1 file (~160) + 3 stubs (~15 each) | ~555 |
| Set-status | 2 files | 198 | 1 file (~60) + 2 stubs (~10 each) | ~118 |
| Resume | 3 files | ~400 | 1 file (~120) + 3 stubs (~15 each) | ~235 |
| **TOTAL** | **14 files** | **~2,686** | **5 files + ~14 stubs** | **~1,878** |

### Net Reduction

- **Before**: ~2,686 lines across 14+ files
- **After**: ~810 lines across 5 generic files + ~210 lines in registration stubs
- **Net savings**: ~1,666 lines (62% reduction)
- **File count**: 14 files reduced to 5 generic + registration = fewer files to maintain

---

## Dependencies & Integrations

### Dependencies

- **E21-F09** (Service Delegation): Generic handlers call unified service methods — works best after services are consolidated
- **Key format auto-detection**: Already exists in codebase (`internal/cli/commands/get.go` pattern)

### Ordering

- F10 can be done independently of F07-F09, but benefits from F09 (cleaner service calls)
- F10 is the **highest line-count win** and could be prioritized first if service consolidation is deferred

---

## Success Metrics

### Primary Metrics

1. **Line Count Reduction**
   - **Target**: ~1,666 lines eliminated from CLI commands
   - **Measurement**: `wc -l internal/cli/commands/*_note.go *_context.go *_next_status.go *_set_status.go *_resume.go` before/after

2. **File Count Reduction**
   - **Target**: 14 per-entity command files consolidated to 5 generic + registration stubs
   - **Measurement**: File count in `internal/cli/commands/`

3. **Bug Fix Locality**
   - **Target**: A bug in note formatting requires changing 1 file, not 3
   - **Measurement**: Code review of a hypothetical fix touches 1 file

4. **New Entity Type CLI Cost**
   - **Target**: Adding CLI support for shared operations on a new entity type requires only command registration (~15 lines), not new handler implementations
   - **Measurement**: Lines added for hypothetical 6th entity type

---

*Last Updated*: 2026-03-20
