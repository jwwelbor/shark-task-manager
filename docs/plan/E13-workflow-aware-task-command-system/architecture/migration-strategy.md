# Migration Strategy: Deprecating Old Commands

**Epic**: [E13 Workflow-Aware Task Command System](../epic.md)

**Last Updated**: 2026-01-11

---

## Overview

This document defines the phased approach to migrating from hardcoded workflow commands (`start`, `complete`, `approve`, `reopen`) to workflow-aware commands (`claim`, `finish`, `reject`) while maintaining backward compatibility.

**Principle**: Gradual, non-breaking migration over 3 releases with clear user guidance.

---

## Migration Phases

### Phase 1: Non-Breaking Addition (Release 1)

**Timeline**: Release 1 (Weeks 1-4)

**Changes**:
- Add new commands: `claim`, `finish`, `reject`
- Keep all existing commands functional
- No deprecation warnings yet
- Both old and new commands work identically

**User Impact**: Zero breaking changes, optional adoption

**Implementation**:

```
NEW Commands Added:
  shark task claim <task-key> [--agent=<type>]
  shark task finish <task-key> [--notes="..."]
  shark task reject <task-key> --reason="..." [--to=<status>]

EXISTING Commands (Unchanged):
  shark task start <task-key>
  shark task complete <task-key>
  shark task approve <task-key>
  shark task reopen <task-key>
  shark task next-status <task-key>
```

**Documentation Updates**:
- CLI reference shows new commands as recommended
- Old commands marked as "Legacy (still supported)"
- Migration guide published
- Changelog announces new commands

---

### Phase 2: Deprecation Warnings (Release 2)

**Timeline**: Release 2 (6 weeks after Release 1)

**Changes**:
- Add deprecation warnings to old commands
- Update help text to suggest new commands
- Provide migration guide in error messages
- Track command usage (optional telemetry)

**User Impact**: Warnings displayed, but commands still work

**Deprecation Warning Format**:

```bash
$ shark task start T-E07-F20-001

╭─ DEPRECATION WARNING ─────────────────────────────────────────────╮
│                                                                    │
│  'shark task start' is deprecated and will be removed in v2.0.0   │
│                                                                    │
│  Use 'shark task claim' instead:                                  │
│    shark task claim T-E07-F20-001 --agent=<type>                  │
│                                                                    │
│  Migration guide: https://docs.shark.dev/migration-v2             │
│                                                                    │
│  To suppress this warning: export SHARK_SILENCE_DEPRECATION=1    │
│                                                                    │
╰────────────────────────────────────────────────────────────────────╯

Task T-E07-F20-001 started successfully
Status: todo → in_progress
```

**Implementation**:

```go
func runTaskStart(cmd *cobra.Command, args []string) error {
    // Show deprecation warning (suppressible)
    if !cli.IsSilenceDeprecation() {
        cli.DeprecationWarning("start", "claim", "shark task claim <task-key> --agent=<type>")
    }

    // Execute command as normal
    return executeTaskStart(cmd, args)
}

func (c *CLI) DeprecationWarning(oldCmd, newCmd, example string) {
    if c.JSON {
        return // Don't show in JSON mode
    }

    box := fmt.Sprintf(`
╭─ DEPRECATION WARNING %s─╮
│                                                                    │
│  'shark task %s' is deprecated and will be removed in v2.0.0     │
│                                                                    │
│  Use 'shark task %s' instead:                                     │
│    %s                                                             │
│                                                                    │
│  Migration guide: https://docs.shark.dev/migration-v2             │
│                                                                    │
│  To suppress this warning: export SHARK_SILENCE_DEPRECATION=1    │
│                                                                    │
╰─────────────────────────────────────────────────────────────────%s─╯
`, strings.Repeat("─", 44), oldCmd, newCmd, example, strings.Repeat("─", 44))

    fmt.Fprintln(os.Stderr, box)
}
```

**Command Mapping**:

| Old Command | New Command | Migration Example |
|-------------|-------------|-------------------|
| `start` | `claim` | `shark task claim <key> --agent=<type>` |
| `complete` | `finish` | `shark task finish <key> --notes="Done"` |
| `approve` | `finish` | `shark task finish <key> (from ready_for_approval)` |
| `next-status` | `finish` | `shark task finish <key>` |
| `reopen` | `reject` | `shark task reject <key> --reason="Rework needed"` |

---

### Phase 3: Removal (Release 3)

**Timeline**: Release 3 (6-8 weeks after Release 2, minimum 2 releases after deprecation)

**Changes**:
- Remove deprecated commands entirely
- Commands return helpful error with migration guide
- Clean up dead code

**User Impact**: Breaking change for users still using old commands

**Error Message for Removed Commands**:

```bash
$ shark task start T-E07-F20-001

Error: Command 'shark task start' has been removed in v2.0.0

This command was deprecated in v1.8.0 (released 2026-03-15)

Use the replacement command:
  shark task claim T-E07-F20-001 --agent=<type>

Migration Guide:
  https://docs.shark.dev/migration-v2

Available commands:
  shark task claim     - Claim task for current phase
  shark task finish    - Complete current phase
  shark task reject    - Send task back for rework
  shark task --help    - View all task commands

Exit Code: 1
```

**Implementation**:

```go
// Replace old command with error stub
var taskStartCmd = &cobra.Command{
    Use:    "start <task-key>",
    Short:  "[REMOVED] Use 'shark task claim' instead",
    Hidden: true, // Hide from help by default
    RunE: func(cmd *cobra.Command, args []string) error {
        return removedCommandError("start", "claim", "shark task claim <task-key> --agent=<type>")
    },
}

func removedCommandError(oldCmd, newCmd, example string) error {
    return fmt.Errorf(`Command 'shark task %s' has been removed in v2.0.0

This command was deprecated in v1.8.0 (released 2026-03-15)

Use the replacement command:
  %s

Migration Guide:
  https://docs.shark.dev/migration-v2

Exit Code: 1`, oldCmd, example)
}
```

---

## Command Consolidation

### Deprecated Commands to Remove

**Primary Deprecations** (Status Management):
- `shark task start` → `claim`
- `shark task complete` → `finish`
- `shark task approve` → `finish`
- `shark task next-status` → `finish`
- `shark task reopen` → `reject`

**Command Consolidations** (Simplification):
- `shark task blocks` + `shark task blocked-by` → `shark task deps --type=<type>`
- `shark task note` (add) + `shark task notes` (list) → `shark task notes [add|list]`
- `shark task timeline` → `shark task history --format=timeline`

**Total Command Reduction**: 25 commands → 18 commands (28% reduction)

---

## Migration Guide Document

### Location

`docs/MIGRATION_V2.md`

### Content Structure

```markdown
# Migration Guide: Shark v2.0 (Workflow-Aware Commands)

## Overview

Shark v2.0 introduces workflow-aware task commands that work with any custom
workflow configuration. Old hardcoded commands are deprecated.

## Quick Migration Table

| Old Command | New Command | Key Differences |
|-------------|-------------|-----------------|
| `task start` | `task claim --agent=<type>` | Requires agent type, works with any workflow phase |
| `task complete` | `task finish --notes="..."` | Automatically determines next phase from workflow |
| `task approve` | `task finish` (from approval phase) | Same command, different phase |
| `task reopen` | `task reject --reason="..."` | Requires rejection reason |

## Detailed Command Changes

### start → claim

**Before**:
```bash
shark task start T-E07-F20-001
```

**After**:
```bash
shark task claim T-E07-F20-001 --agent=backend
```

**Why the change?**
- Works with custom workflows (any ready_for_X → in_X transition)
- Tracks work sessions automatically
- Agent type enables filtering and assignment

### complete → finish

**Before**:
```bash
shark task complete T-E07-F20-001
```

**After**:
```bash
shark task finish T-E07-F20-001 --notes="Implementation complete"
```

**Why the change?**
- Workflow determines next phase (not hardcoded)
- Works across all phases (development, review, QA, approval)
- Clearer intent: "finish this phase" vs "task is complete"

## Workflow Configuration

New commands read `.sharkconfig.json` workflow configuration.

**Default workflow** (if no config):
```json
{
  "status_flow": {
    "todo": ["in_progress"],
    "in_progress": ["ready_for_review", "todo"],
    "ready_for_review": ["completed", "in_progress"],
    "completed": []
  }
}
```

**Custom workflow example**:
```json
{
  "status_flow": {
    "ready_for_development": ["in_development"],
    "in_development": ["ready_for_code_review", "ready_for_refinement"],
    "ready_for_code_review": ["in_code_review"],
    "in_code_review": ["ready_for_qa", "in_development"],
    "ready_for_qa": ["in_qa"],
    "in_qa": ["ready_for_approval", "in_development"],
    "ready_for_approval": ["in_approval"],
    "in_approval": ["completed"],
    "completed": []
  }
}
```

## For AI Orchestrator Users

**Old Pattern**:
```bash
shark task next --agent=backend --json
shark task start T-E07-F20-001
# ... work ...
shark task complete T-E07-F20-001
```

**New Pattern**:
```bash
shark task next --agent=backend --json
shark task claim T-E07-F20-001 --agent=backend --json
# ... work ...
shark task finish T-E07-F20-001 --json
```

**Benefits**:
- Works with any workflow configuration
- Tracks work sessions automatically
- JSON output includes next phase info

## Migration Timeline

- **v1.7 (Feb 2026)**: New commands available, old commands work
- **v1.8 (Mar 2026)**: Deprecation warnings added
- **v2.0 (May 2026)**: Old commands removed

## Getting Help

- Migration issues: https://github.com/yourorg/shark/issues
- Community: https://discord.gg/shark-dev
```

---

## Backward Compatibility Testing

### Test Matrix

| Scenario | Phase 1 | Phase 2 | Phase 3 |
|----------|---------|---------|---------|
| Old commands work | ✓ | ✓ (warning) | ✗ (error) |
| New commands work | ✓ | ✓ | ✓ |
| JSON output unchanged | ✓ | ✓ | ✓ |
| AI orchestrator compatible | ✓ | ✓ | ✓ |
| Custom workflows | ✓ | ✓ | ✓ |

### Automated Tests

```go
func TestBackwardCompatibility_OldCommands(t *testing.T) {
    tests := []struct{
        phase       int    // 1, 2, or 3
        command     string
        shouldWork  bool
        hasWarning  bool
    }{
        {1, "task start T-001", true, false},
        {2, "task start T-001", true, true},
        {3, "task start T-001", false, false},
        {1, "task claim T-001", true, false},
        {2, "task claim T-001", true, false},
        {3, "task claim T-001", true, false},
    }

    for _, tt := range tests {
        t.Run(fmt.Sprintf("Phase%d_%s", tt.phase, tt.command), func(t *testing.T) {
            setMigrationPhase(tt.phase)
            output, err := runCommand(tt.command)

            if tt.shouldWork && err != nil {
                t.Errorf("Expected success, got error: %v", err)
            }
            if !tt.shouldWork && err == nil {
                t.Error("Expected error, got success")
            }
            if tt.hasWarning && !containsWarning(output) {
                t.Error("Expected deprecation warning, not found")
            }
        })
    }
}
```

---

## User Communication

### Release Notes Template

**v1.7.0 Release Notes**:

```markdown
## New Features

### Workflow-Aware Task Commands

We're excited to introduce new task commands that work seamlessly with
custom workflow configurations!

**New Commands**:
- `shark task claim` - Claim task for current workflow phase
- `shark task finish` - Complete current phase and advance
- `shark task reject` - Send task back for rework

**Why?**
- Works with any custom workflow (no hardcoded statuses)
- Automatic work session tracking
- Better AI orchestrator integration

**Migration**: Existing commands (`start`, `complete`, `approve`) still
work. See [Migration Guide](docs/MIGRATION_V2.md) for details.

**Try it now**:
```bash
shark task claim T-E07-F20-001 --agent=backend
shark task finish T-E07-F20-001 --notes="Done"
```
```

**v1.8.0 Release Notes**:

```markdown
## Deprecation Notices

Task commands `start`, `complete`, `approve`, `reopen`, and `next-status`
are now deprecated and will be removed in v2.0.0.

**What you need to do**:
1. Update scripts to use new commands (see migration guide)
2. Test with deprecation warnings enabled
3. Update AI orchestrator code if applicable

**Timeline**:
- Now (v1.8): Warnings displayed
- May 2026 (v2.0): Commands removed

See [Migration Guide](docs/MIGRATION_V2.md) for full details.
```

**v2.0.0 Release Notes**:

```markdown
## Breaking Changes

Deprecated task commands have been removed. Use workflow-aware commands:
- `claim` instead of `start`
- `finish` instead of `complete`/`approve`/`next-status`
- `reject` instead of `reopen`

See [Migration Guide](docs/MIGRATION_V2.md) for upgrade instructions.

If you encounter issues, open a GitHub issue or contact support.
```

---

## Rollback Plan

### If Migration Issues Arise

**Phase 2 → Phase 1 Rollback**:
- Remove deprecation warnings (hotfix release)
- Extend deprecation period by 2 releases
- Gather user feedback

**Phase 3 → Phase 2 Rollback**:
- Re-add old commands with deprecation warnings
- Mark as "temporarily restored due to user feedback"
- Extend removal timeline by 6 months

**Implementation**:
```bash
# Restore old commands (emergency rollback)
git revert <commit-removing-old-commands>
# Add clear messaging
echo "Old commands temporarily restored due to migration issues"
# Release hotfix
```

---

## Success Criteria

### Phase 1 Success

- [ ] All new commands functional
- [ ] All existing commands still work
- [ ] AI orchestrator integration tested
- [ ] Documentation published
- [ ] Zero reported regressions

### Phase 2 Success

- [ ] Deprecation warnings displayed correctly
- [ ] 50%+ users migrated to new commands (telemetry)
- [ ] Migration guide updated with user feedback
- [ ] No critical issues reported

### Phase 3 Success

- [ ] Old commands removed cleanly
- [ ] 90%+ users migrated (based on lack of complaints)
- [ ] No rollback needed
- [ ] Command count reduced as expected (25 → 18)

---

## References

- [System Architecture](./system-architecture.md)
- [Command Specifications](./command-specifications.md)
- [Epic Requirements](../requirements.md) - REQ-NF-003
- [Success Metrics](../success-metrics.md) - Metric 4 (Migration Adoption Rate)
