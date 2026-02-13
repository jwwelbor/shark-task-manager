---
feature_key: E07-F27-embedded-json-workflow-profiles-and-init-merge
epic_key: E07
title: Embedded JSON Workflow Profiles and Init Merge
description: Replace hardcoded Go struct workflow profiles with embedded JSON files and add a shark init merge command for safe profile switching.
---

# Embedded JSON Workflow Profiles and Init Merge

**Feature Key**: E07-F27

---

## Goal

### Problem

Workflow profiles (basic, advanced) are currently defined as Go structs in `internal/init/profiles.go`. This makes them hard to read, edit, and share. The advanced profile in code is outdated compared to the actual workflow in use (missing `ready_for_research`, `in_research`, `ready_for_test_planning`, `in_test_planning`, `orchestrator_action`, enriched `agent_types`, etc.). Adding or modifying profiles requires Go code changes and recompilation.

The current `shark init update` command has unclear UX: without `--force` it silently adds missing fields; the distinction between "update" and "apply profile" is confusing.

### Solution

1. Store workflow profiles as JSON files embedded into the binary via Go's `//go:embed`
2. Add a new `shark init merge` command with clear, safe semantics (dry-run by default)
3. Provide two profiles: basic (4-status solo dev) and advanced (full AI-orchestrated with epic/feature workflows)

### Impact

- Profiles are human-readable JSON files, viewable/downloadable directly from the repo
- Adding a new profile = adding a JSON file (no Go code changes)
- Clear command UX: `init merge` previews by default, requires `--force` to apply
- Current `.sharkconfig.json` accurately captured as the `advanced` profile
- Profiles can carry any JSON field (orchestrator_action, epic_workflow, feature_workflow) without Go struct changes

---

## Architecture

### Profile JSON Files

```
internal/init/profiles/
  basic.json          # 4-status solo dev workflow
  advanced.json       # Full AI-orchestrated workflow with agent routing
```

Each JSON file contains ONLY workflow-related fields:
- `status_metadata` - Status definitions, colors, phases, weights, orchestrator_action, agent_types
- `status_flow` - Valid status transitions
- `special_statuses` - Start/complete/blocked status groups
- `status_flow_version` - Version identifier
- `epic_workflow` (optional) - Epic-level status definitions and flows
- `feature_workflow` (optional) - Feature-level status definitions and flows

Profile JSON files do NOT contain project-specific fields (database, patterns, etc.).

### Embedding

```go
// internal/init/profiles_embedded.go
package init

import "embed"

//go:embed profiles/*.json
var embeddedProfiles embed.FS
```

`GetProfileMap(name)` reads the named JSON file from the embedded FS and unmarshals to `map[string]interface{}` directly — no Go struct intermediary needed. This allows profiles to carry any JSON fields without struct changes.

### Profile Definitions

**basic.json** (4 task statuses):
- `todo`, `in_progress`, `completed`, `blocked`
- Simple linear flow: todo → in_progress → completed (blocked can return to either)
- Weights: todo=0, in_progress=0.5, completed=1.0, blocked=0
- No epic_workflow or feature_workflow (uses system defaults)
- For: solo developers, simple projects, prototyping

**advanced.json** (18 task statuses + epic + feature workflows):
- Derived from current `.sharkconfig.json` (the actually-in-use workflow)
- Task statuses: draft, todo, ready_for_refinement_ba/tech, in_refinement_ba/tech, ready_for_development, in_development, ready_for_code_review, in_code_review, ready_for_qa, in_qa, ready_for_approval, in_approval, completed, cancelled, blocked, on_hold
- Full `orchestrator_action` on all actionable statuses (spawn_agent, pause, wait_for_triage, archive)
- Rich `agent_types` on all `ready_for_*` statuses
- `epic_workflow`: 14 statuses (draft → research → refinement → test_planning → decomposition → active → completed)
- `feature_workflow`: 18 statuses (draft → research → refinement_ba → refinement_tech → test_planning → task_generation → build → active → completed)
- For: AI-orchestrated multi-agent development workflows

### Init Merge Command

```bash
# Preview what would change (default, safe)
shark init merge --workflow=advanced
# Output: "Would replace 18 statuses, 18 flow rules, 3 special groups. Database config preserved."

# Actually apply
shark init merge --workflow=advanced --force

# Switch to basic
shark init merge --workflow=basic --force
```

**Preserved fields** (never touched by merge):
- `database` (backend, url, auth_token_file)
- `project_root`
- `last_sync_time`
- `interactive_mode`
- `require_rejection_reason`

**Replaced fields** (overwritten from profile JSON):
- `status_metadata`
- `status_flow`
- `special_statuses`
- `status_flow_version`
- `epic_workflow` (if present in profile)
- `feature_workflow` (if present in profile)

### Init Command Update

- Add `--workflow` flag to `shark init` (default: "basic")
- `shark init --workflow=advanced` applies advanced profile on fresh init

---

## Implementation Plan (TDD)

Each step follows: Write failing tests → Implement minimum code → Refactor

### Step 1: Create profile JSON files + embed infrastructure (tests first)

**Tests first** (`internal/init/profiles_test.go`):
- `TestGetProfileMap_Basic` - loads, has expected keys (status_metadata, status_flow, etc.)
- `TestGetProfileMap_Advanced` - loads, has epic_workflow, feature_workflow
- `TestGetProfileMap_InvalidName` - returns error
- `TestListProfiles` - returns ["basic", "advanced"]
- `TestGetProfileMap_BasicStatusCount` - 4 statuses
- `TestGetProfileMap_AdvancedHasOrchestratorAction` - orchestrator_action present
- `TestGetProfile_BackwardCompat` - GetProfile() still returns *WorkflowProfile

**Then implement:**
- `internal/init/profiles/basic.json` and `advanced.json`
- `internal/init/profiles_embedded.go` with `//go:embed profiles/*.json`
- Update `internal/init/profiles.go` - remove Go structs, keep GetProfile() shim

### Step 2: Update profile service (tests first)

**Tests first** (`internal/init/profile_service_test.go`):
- `TestApplyProfile_AdvancedIncludesEpicWorkflow` - epic_workflow in merged config
- `TestApplyProfile_AdvancedIncludesFeatureWorkflow` - feature_workflow in merged config
- `TestApplyProfile_BasicNoEpicWorkflow` - basic doesn't add epic/feature workflow
- `TestApplyProfile_PreservesProjectFields` - database, interactive_mode preserved
- `TestApplyProfile_OverwritesWorkflowFields` - status_metadata overwritten

**Then implement:**
- `profile_service.go` uses `GetProfileMap()` directly, remove `profileToMap()`
- Add epic_workflow, feature_workflow to OverwriteFields

### Step 3: Add `shark init merge` command (tests first)

**Tests first** (`internal/cli/commands/init_test.go`):
- `TestInitMergeCmd_DryRunByDefault` - no config changes without --force
- `TestInitMergeCmd_RequiresWorkflow` - error without --workflow
- `TestInitMergeCmd_ForceApplies` - config written with --force
- `TestInitMergeCmd_JSONOutput` - valid JSON output

**Then implement:**
- Add `initMergeCmd` in `init.go`, flags: --workflow (required), --force

### Step 4: Update `shark init` + clean up patterns

- Add `--workflow` flag to initCmd
- Remove dead patterns from `ConfigDefaults` and `createConfig()`

### Step 5: Quality gate

```bash
make fmt && make lint && make test
./bin/shark init merge --workflow=basic
./bin/shark init merge --workflow=advanced --force
```

---

## Out of Scope

- Configurable file naming patterns (epic.md/prd.md, task key format) - tracked in E15 epic doc
- Remote profile fetching from GitHub (all profiles embedded at build time)
- Custom user-defined profiles (users can manually edit .sharkconfig.json after init)
- Migration of existing tasks when switching profiles (task statuses remain as-is)

---

## Dependencies

- Current `ProfileService.ApplyProfile()` in `internal/init/profile_service.go` already handles the merge logic (preserve/overwrite fields). The `init merge` command is largely a UX wrapper around this.
- `ConfigMerger` in `internal/init/config_merger.go` handles the field-level merge.

---

*Last Updated*: 2026-02-12
