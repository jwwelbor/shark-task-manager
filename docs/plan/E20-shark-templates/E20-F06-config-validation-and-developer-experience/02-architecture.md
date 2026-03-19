# E20-F06 Architecture: Config Validation and Developer Experience

## Overview

This feature extends 4 existing capabilities in the config subsystem. All changes follow established patterns from E20-F04 (workflow file loading).

## Key Decisions

### D1: Extend existing config.go commands (not new service)
- `shark config validate` and `shark config show` are existing commands in `internal/cli/commands/config.go`
- These already operate at the CLI level reading config files directly
- No service layer needed -- config validation is a CLI-only concern (not shared with HTTP API)
- Follow the existing pattern in config.go

### D2: Workflow export as subcommand of `shark admin workflow` -- DESCOPED
> **DESCOPED**: The export command (REQ-F-021) was removed from scope during implementation. The existing `shark init update --workflow=<profile>` command provides sufficient migration capability.

~~- `shark admin workflow export` fits the existing `workflow.go` command group~~
~~- Already has `workflowCmd` parent in `internal/cli/commands/workflow.go`~~
~~- New subcommand `workflowExportCmd` added to the existing group~~

### D3: Deprecation warnings in workflow_parser.go loading path
- Deprecation warnings fire during `LoadMultiLevelWorkflow()` when legacy keys coexist with `task_workflow` block
- Emitted via `fmt.Fprintf(os.Stderr, ...)` to stderr (not stdout) to avoid polluting JSON output
- Suppressed when `cli.GlobalConfig.JSON` is true (check at call site, not in parser)
- One-time per invocation via `sync.Once`

### D4: Source tracking via existing MultiLevelWorkflow struct
- `MultiLevelWorkflow` already tracks which file each entity workflow came from
- Add a `Sources map[string]string` field to track per-entity source
- Populated during `LoadMultiLevelWorkflow()` as each entity is resolved
- Consumed by `config show` and `config validate` commands

## File Impact

| File | Change Type | Description |
|------|-------------|-------------|
| `internal/config/workflow_parser.go` | Modify | Add source tracking to `LoadMultiLevelWorkflow`, add deprecation detection |
| `internal/config/workflow_multilevel.go` | Modify | Add `Sources` field to `MultiLevelWorkflow` struct |
| `internal/cli/commands/config.go` | Modify | Extend `configValidateCmd` and `configShowCmd` |
| ~~`internal/cli/commands/workflow.go`~~ | ~~Modify~~ | ~~Add `workflowExportCmd` subcommand~~ **(DESCOPED)** |
| `internal/config/workflow_parser_test.go` | New | Tests for source tracking and deprecation detection |
| `internal/cli/commands/config_test.go` | New | Tests for extended validate/show commands |

## Implementation Plan

### Story 1: Config Validate (REQ-F-010) -- Should Have
1. In `LoadMultiLevelWorkflow()`, populate `Sources` map with source path per entity
2. Add `ValidateWorkflowFile(path string) []ValidationResult` function in `workflow_parser.go`
3. Extend `configValidateCmd` to call `ValidateWorkflowFile` and report findings
4. Check for duplicates (same entity defined in both files)
5. Check required sub-keys per entity workflow block

### Story 2: Config Show Source (REQ-F-011) -- Should Have
1. Read `Sources` from loaded `MultiLevelWorkflow`
2. In human output: append "(source: .sharkworkflow.json)" per entity type
3. In JSON output: add `_source` field per workflow block

### Story 3: Deprecation Warnings (REQ-F-020) -- Could Have
1. During `LoadMultiLevelWorkflow`, detect legacy keys + `task_workflow` coexistence
2. Store detection result in `MultiLevelWorkflow.HasLegacyTaskKeys bool`
3. Emit warning at CLI command level (not in parser) via `sync.Once`
4. Skip when `cli.GlobalConfig.JSON` is true

### Story 4: Workflow Export (REQ-F-021) -- DESCOPED
> **DESCOPED**: Removed from scope during implementation. See feature.md Out of Scope section.

~~1. Add `workflowExportCmd` to `workflow.go`~~
~~2. Read all entity workflows from current config~~
~~3. Build `.sharkworkflow.json` with all 5 entity blocks~~
~~4. Convert legacy task keys to `task_workflow` block format~~
~~5. Support `--dry-run` flag for preview~~

## Testing Strategy

- Unit tests for source tracking in `workflow_parser.go`
- Unit tests for validation logic (duplicate detection, required key checks)
- ~~Unit tests for export command (mock config files via temp dirs)~~ **(DESCOPED)**
- ~~Integration test: validate -> show -> export roundtrip~~ **(DESCOPED)**
