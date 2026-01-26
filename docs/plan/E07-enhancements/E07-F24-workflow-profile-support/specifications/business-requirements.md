# Workflow Profile Feature - Business Requirements Specification

## Executive Summary

Add workflow profile support to `shark init` command, enabling users to quickly configure their task workflow from predefined templates while preserving existing configuration.

## Business Context

**Problem**: Users need flexibility in task workflow complexity - some want simple status tracking (todo → done), while others need sophisticated multi-stage workflows with refinement, review, QA, and approval stages.

**Solution**: Provide predefined workflow profiles that users can apply to their `.sharkconfig.json`, with intelligent merging to preserve existing customizations.

## User Stories

### US-1: Apply Basic Workflow Profile

**As a** solo developer
**I want** to apply a simple workflow profile
**So that** I can track task progress with minimal overhead

**Acceptance Criteria:**
- Command: `shark init update --workflow=basic`
- Applies 4-5 status workflow (todo, in_progress, ready_for_review, completed, blocked)
- Includes progress weights for visual indicators
- Preserves existing config fields (database, project settings)
- Reports what was added/updated

### US-2: Apply Advanced Workflow Profile

**As a** team lead
**I want** to apply a comprehensive TDD workflow profile
**So that** I can support multi-agent development with refinement, review, QA, and approval stages

**Acceptance Criteria:**
- Command: `shark init update --workflow=advanced`
- Applies 16+ status workflow with refinement, development, review, QA, approval stages
- Includes agent_types assignment (ba, tech_lead, developer, qa, product_owner)
- Includes progress weights (0.0 to 1.0 scale)
- Includes status flow definitions (valid transitions)
- Includes special status groups (_start_, _complete_)
- Includes blocking status configuration
- Preserves existing config fields
- Reports what was added/updated

### US-3: Merge with Existing Configuration

**As a** power user
**I want** to update my workflow without losing custom settings
**So that** I can incrementally adopt new workflow features

**Acceptance Criteria:**
- Command: `shark init update` (no --workflow flag)
- Adds missing config fields from default profile
- Preserves all existing values
- Does not overwrite existing status configurations
- Reports what was added (no changes to existing fields)

### US-4: Handle Partial Configuration

**As a** developer with corrupted config
**I want** to repair missing config sections
**So that** I can restore functionality without starting over

**Acceptance Criteria:**
- Command works with partial `.sharkconfig.json` files
- Merges new sections into existing structure
- Preserves valid sections (database, project_root, etc.)
- Reports sections added/repaired

## Workflow Profiles

### Basic Profile

**Target Users**: Solo developers, small teams, simple projects

**Status Flow** (5 statuses):
```
todo → in_progress → ready_for_review → completed
                   ↘ blocked ↗
```

**Status Configuration**:
| Status | Color | Phase | Progress Weight | Responsibility | Blocks Feature |
|--------|-------|-------|-----------------|----------------|----------------|
| todo | gray | planning | 0.0 | none | false |
| in_progress | yellow | development | 0.5 | agent | false |
| ready_for_review | magenta | review | 0.75 | human | true |
| completed | green | done | 1.0 | none | false |
| blocked | red | any | 0.0 | none | true |

**Features**:
- Simple status progression
- Progress indicator support (weighted progress)
- Human review checkpoint
- Blocking status for impediments

### Advanced Profile

**Target Users**: Multi-agent teams, TDD workflows, enterprise projects

**Status Flow** (19 statuses):
```
draft → ready_for_refinement_ba → in_refinement_ba →
ready_for_refinement_tech → in_refinement_tech →
ready_for_development → in_development →
ready_for_code_review → in_code_review → changes_requested →
ready_for_qa → in_qa → qa_failed →
ready_for_approval → in_approval → completed

blocked (from any status)
cancelled (terminal)
on_hold (suspended)
```

**Status Configuration** (abbreviated - see full spec below):
| Status | Agent Type | Progress Weight | Phase | Blocks Feature |
|--------|-----------|-----------------|-------|----------------|
| draft | ba | 0.0 | planning | false |
| ready_for_refinement_ba | none | 0.05 | planning | true |
| in_refinement_ba | ba | 0.10 | planning | false |
| ready_for_refinement_tech | none | 0.15 | planning | true |
| in_refinement_tech | tech_lead | 0.20 | planning | false |
| ready_for_development | none | 0.25 | development | true |
| in_development | developer | 0.50 | development | false |
| ready_for_code_review | none | 0.60 | review | true |
| in_code_review | tech_lead | 0.65 | review | false |
| changes_requested | developer | 0.55 | development | false |
| ready_for_qa | none | 0.70 | qa | true |
| in_qa | qa | 0.75 | qa | false |
| qa_failed | developer | 0.65 | development | false |
| ready_for_approval | none | 0.85 | approval | true |
| in_approval | product_owner | 0.90 | approval | false |
| completed | none | 1.0 | done | false |
| blocked | none | 0.0 | any | true |
| cancelled | none | 0.0 | done | false |
| on_hold | none | 0.0 | any | false |

**Features**:
- Multi-stage refinement (business analyst → technical lead)
- Development → Review → QA → Approval pipeline
- Agent type assignments for routing
- Failure recovery paths (changes_requested, qa_failed)
- Status flow enforcement (valid transitions)
- Special status groups (_start_, _complete_)
- Blocking status configuration

**Status Flow Definitions**:
```json
{
  "draft": ["ready_for_refinement_ba", "cancelled"],
  "ready_for_refinement_ba": ["in_refinement_ba", "blocked"],
  "in_refinement_ba": ["ready_for_refinement_tech", "draft", "blocked"],
  "ready_for_refinement_tech": ["in_refinement_tech", "blocked"],
  "in_refinement_tech": ["ready_for_development", "in_refinement_ba", "blocked"],
  "ready_for_development": ["in_development", "blocked"],
  "in_development": ["ready_for_code_review", "blocked"],
  "ready_for_code_review": ["in_code_review", "blocked"],
  "in_code_review": ["changes_requested", "ready_for_qa", "blocked"],
  "changes_requested": ["in_development", "blocked"],
  "ready_for_qa": ["in_qa", "blocked"],
  "in_qa": ["qa_failed", "ready_for_approval", "blocked"],
  "qa_failed": ["in_development", "blocked"],
  "ready_for_approval": ["in_approval", "blocked"],
  "in_approval": ["completed", "changes_requested", "blocked"],
  "blocked": ["<previous_status>"],
  "on_hold": ["<previous_status>"],
  "completed": [],
  "cancelled": []
}
```

**Special Status Groups**:
```json
{
  "_start_": ["draft", "ready_for_development"],
  "_complete_": ["completed", "cancelled"],
  "_blocked_": ["blocked", "on_hold"]
}
```

## Command Specification

### Command Syntax

```bash
shark init update [--workflow=<profile>] [--force] [--dry-run]
```

**Flags**:
- `--workflow=<profile>`: Apply workflow profile (basic, advanced)
- `--force`: Overwrite existing status configurations
- `--dry-run`: Preview changes without applying

**Aliases**:
- `shark config update --workflow=<profile>`

### Command Behavior

#### Case 1: No --workflow flag
```bash
shark init update
```
- Adds missing config fields from default profile
- Preserves all existing values
- Output: "Added 3 missing configuration fields"

#### Case 2: Basic profile
```bash
shark init update --workflow=basic
```
- Merges basic workflow into existing config
- Preserves database, project_root, etc.
- Overwrites `status_metadata` section
- Output: "Applied basic workflow profile (5 statuses)"

#### Case 3: Advanced profile
```bash
shark init update --workflow=advanced
```
- Merges advanced workflow into existing config
- Preserves database, project_root, etc.
- Overwrites `status_metadata`, `status_flow`, `special_statuses` sections
- Output: "Applied advanced workflow profile (19 statuses)"

#### Case 4: Dry-run
```bash
shark init update --workflow=advanced --dry-run
```
- Shows what would be changed
- Does not write to file
- Output: Diff showing additions/changes

#### Case 5: Force overwrite
```bash
shark init update --workflow=basic --force
```
- Overwrites existing status configurations
- Warning: "This will overwrite existing status configurations. Continue? [y/N]"

### Error Handling

| Error Condition | Exit Code | Message |
|----------------|-----------|---------|
| Invalid profile name | 1 | "Invalid workflow profile: {name}. Valid profiles: basic, advanced" |
| Missing .sharkconfig.json | 1 | ".sharkconfig.json not found. Run 'shark init' first." |
| Invalid JSON in config | 2 | "Failed to parse .sharkconfig.json: {error}" |
| Write permission denied | 2 | "Failed to update .sharkconfig.json: {error}" |
| Backup creation failed | 2 | "Failed to create config backup: {error}" |

### Output Format

**Human-readable** (default):
```
✓ Backed up config to .sharkconfig.json.backup
✓ Applied advanced workflow profile
  Added: 19 statuses
  Added: status_flow definitions
  Added: special_status_groups
  Preserved: database configuration
  Preserved: project settings
```

**JSON** (`--json` flag):
```json
{
  "success": true,
  "profile": "advanced",
  "changes": {
    "added": ["status_metadata", "status_flow", "special_status_groups"],
    "preserved": ["database", "project_root"],
    "overwritten": []
  },
  "stats": {
    "statuses_added": 19,
    "flows_added": 17,
    "groups_added": 3
  }
}
```

## Technical Implementation Notes

### Configuration Merge Strategy

1. **Read existing config**: Load `.sharkconfig.json`
2. **Backup**: Copy to `.sharkconfig.json.backup`
3. **Merge profile**: Deep merge profile into existing config
4. **Preserve sections**: Never overwrite database, project_root, viewer
5. **Overwrite workflow**: Replace status_metadata, status_flow (unless --force=false)
6. **Write config**: Save merged config with pretty-print
7. **Report changes**: Show what was added/changed

### Profile Storage

Profiles stored in `internal/cli/commands/workflow_profiles.go`:

```go
var workflowProfiles = map[string]WorkflowProfile{
    "basic": {
        Name: "basic",
        Description: "Simple workflow for solo developers",
        StatusMetadata: basicStatusMetadata,
    },
    "advanced": {
        Name: "advanced",
        Description: "Comprehensive TDD workflow for teams",
        StatusMetadata: advancedStatusMetadata,
        StatusFlow: advancedStatusFlow,
        SpecialStatusGroups: advancedSpecialGroups,
    },
}
```

### Testing Requirements

**Unit Tests**:
- Profile loading from map
- Config merge logic (preserve vs overwrite)
- JSON validation after merge
- Backup creation

**Integration Tests**:
- Apply basic profile to fresh config
- Apply advanced profile to fresh config
- Apply profile to partial config (merge)
- Dry-run mode (no changes)
- Force mode (overwrite existing)

**Error Tests**:
- Invalid profile name
- Missing config file
- Corrupted JSON
- Write permission denied

## Non-Functional Requirements

### Performance
- Command executes in < 100ms
- Config file size < 50KB (advanced profile)

### Usability
- Clear progress indicators
- Helpful error messages with resolution steps
- Dry-run mode for preview
- Automatic backup before changes

### Maintainability
- Profiles stored as Go constants (easy to version)
- Validation on profile load
- Comprehensive test coverage (>80%)

### Compatibility
- Works with existing .sharkconfig.json files
- Backward compatible (old configs still work)
- No database schema changes required

## Success Metrics

1. **Adoption**: 50%+ of new users apply a workflow profile
2. **Error Rate**: < 1% of profile applications fail
3. **User Feedback**: Positive feedback on workflow flexibility
4. **Support Tickets**: Reduction in workflow configuration questions

## Dependencies

- Existing `shark init` command
- Config management (`internal/config/`)
- JSON merge utilities
- File backup utilities

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Corrupt existing config | Low | High | Automatic backup before changes |
| Invalid JSON in profile | Low | Medium | Validation on profile load |
| User overwrites custom config | Medium | Low | Default to merge mode, --force required |
| Profile doesn't fit user needs | Medium | Low | Document customization in CLI reference |

## Future Enhancements

1. **Custom Profiles**: User-defined profiles in `~/.shark/profiles/`
2. **Profile Validation**: `shark init validate` command
3. **Profile Export**: `shark init export --workflow=my-custom`
4. **Profile Repository**: Community-contributed profiles
5. **Interactive Profile Selection**: `shark init update --interactive`

## Documentation Requirements

1. Update `CLI_REFERENCE.md` with `shark init update` command
2. Add workflow profile guide: `docs/guides/workflow-profiles.md`
3. Add advanced workflow diagram
4. Update CLAUDE.md with agent status routing
5. Add example configs for both profiles

## Acceptance Checklist

- [ ] Command `shark init update --workflow=basic` works
- [ ] Command `shark init update --workflow=advanced` works
- [ ] Command `shark init update` (no flag) adds missing fields only
- [ ] Existing config values preserved during merge
- [ ] Automatic backup created before changes
- [ ] Dry-run mode shows preview without changes
- [ ] JSON output mode works (`--json`)
- [ ] Error messages are helpful and actionable
- [ ] Unit tests pass (>80% coverage)
- [ ] Integration tests pass
- [ ] Documentation updated
- [ ] User stories accepted by product owner

---

**Document Version**: 1.0
**Created**: 2026-01-25
**Status**: Draft
**Owner**: BusinessAnalyst Agent
