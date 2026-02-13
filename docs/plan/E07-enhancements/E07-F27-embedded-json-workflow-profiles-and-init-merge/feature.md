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
2. Add a new `shark init merge` command with clear, safe semantics
3. Provide three profiles: basic, advanced, and orchestrated (the full AI-driven workflow)

### Impact

- Profiles are human-readable JSON files, viewable/downloadable directly from the repo
- Adding a new profile = adding a JSON file (no Go code changes)
- Clear command UX: `init merge` previews by default, requires `--force` to apply
- Current `.sharkconfig.json` accurately captured as the `orchestrated` profile

---

## Architecture

### Profile JSON Files

```
configs/profiles/
  basic.json          # 5-status solo dev workflow
  advanced.json       # 19-status team TDD workflow
  orchestrated.json   # Full AI-orchestrated workflow with agent routing
```

Each JSON file contains ONLY workflow-related fields:
- `status_metadata` - Status definitions, colors, phases, weights, orchestrator_action, agent_types
- `status_flow` - Valid status transitions
- `special_statuses` - Start/complete/blocked status groups
- `status_flow_version` - Version identifier

Profile JSON files do NOT contain project-specific fields (database, patterns, etc.).

### Embedding

```go
// internal/init/profiles_embedded.go
package init

import "embed"

//go:embed profiles/basic.json
var basicProfileJSON []byte

//go:embed profiles/advanced.json
var advancedProfileJSON []byte

//go:embed profiles/orchestrated.json
var orchestratedProfileJSON []byte
```

`GetProfile()` unmarshals the appropriate `[]byte` instead of returning a handcrafted struct. The JSON files are the source of truth.

### Profile Definitions

**basic.json** (5 statuses):
- `todo`, `in_progress`, `ready_for_review`, `completed`, `blocked`
- No status flow enforcement, single agent responsibility
- For: solo developers, simple projects, prototyping

**advanced.json** (19 statuses):
- Current `advancedProfile` Go struct content (as-is from `profiles.go`)
- Multi-stage workflow: draft -> refinement -> development -> review -> QA -> approval -> completed
- Agent types: ba, tech_lead, developer, qa, product_owner
- For: teams with formal review processes

**orchestrated.json** (23 statuses):
- Derived from current `.sharkconfig.json` (the actually-in-use workflow)
- Adds: `ready_for_research`, `in_research`, `ready_for_test_planning`, `in_test_planning`, `todo`
- Full `orchestrator_action` on all actionable statuses (spawn_agent, pause, wait_for_triage, archive)
- Rich `agent_types` on all `ready_for_*` statuses
- Research and test planning phases before development
- For: AI-orchestrated multi-agent development workflows

### Init Merge Command

```bash
# Preview what would change (default, safe)
shark init merge --workflow=orchestrated
# Output: "Would replace 23 statuses, 25 flow rules, 3 special groups. Database config preserved."

# Actually apply
shark init merge --workflow=orchestrated --force

# Re-apply current profile (refresh from embedded JSON)
shark init merge --force
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

### Init Command Update

Interactive profile selection during fresh `shark init`:

```
Select workflow profile:
  1. basic         - Simple 5-status workflow for solo developers
  2. advanced      - 19-status TDD workflow for teams
  3. orchestrated  - Full AI-orchestrated workflow with agent routing (Recommended)

Choice [3]:
```

Non-interactive: `shark init --workflow=orchestrated`

---

## Tasks (High-Level)

1. **Create profile JSON files** - Extract basic from current Go struct, advanced from current Go struct, orchestrated from current `.sharkconfig.json` (strip project-specific fields)
2. **Replace Go struct profiles with embed** - New `profiles_embedded.go` with `//go:embed`, update `GetProfile()` to unmarshal JSON, remove struct definitions from `profiles.go`
3. **Add `init merge` command** - Dry-run preview by default, `--force` to apply, `--workflow` flag, clear output showing what changes
4. **Update `shark init` for profile selection** - Interactive picker, `--workflow` flag for non-interactive
5. **Remove dead patterns from config creation** - `createConfig()` in `internal/init/config.go` still embeds patterns that aren't used; clean up `ConfigDefaults` and initial config to not include patterns
6. **Update tests** - Profile loading, merge behavior, init command with profile selection
7. **Update documentation** - CLAUDE.md workflow profiles section, CLI reference

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
