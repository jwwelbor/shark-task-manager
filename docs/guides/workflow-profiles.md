# Workflow Profiles Guide

Shark supports workflow profiles to quickly configure task workflows for different team sizes and development methodologies.

## Overview

Workflow profiles are predefined configurations that set up:
- Status metadata (colors, phases, progress weights)
- Status flow (valid transitions)
- Special status groups
- Agent type assignments
- Responsibility tracking

## Available Profiles

### Basic Profile

**Use for:**
- Solo developers
- Simple projects
- Prototyping and exploration
- Teams learning Shark

**Statuses (5):**
1. `todo` - Task not started (gray, planning)
2. `in_progress` - Work in progress (yellow, development)
3. `ready_for_review` - Awaiting review (magenta, review)
4. `completed` - Task finished (green, done)
5. `blocked` - External dependency (red, any phase)

**Characteristics:**
- No status flow enforcement
- Single agent responsibility
- Simple progress tracking
- Minimal overhead

**Status Flow:**
```
todo → in_progress → ready_for_review → completed
                   ↘ blocked ↗
```

### Advanced Profile

**Use for:**
- Development teams (3+ people)
- Test-driven development (TDD)
- Multi-stage review processes
- Quality-focused workflows

**Statuses (19):**

**Planning Phase:**
- `draft` - Initial task creation
- `ready_for_refinement` - Ready for BA refinement
- `in_refinement` - Being refined by BA

**Development Phase:**
- `ready_for_development` - Ready to start coding
- `in_development` - Implementation in progress

**Review Phase:**
- `ready_for_code_review` - Code complete, needs review
- `in_code_review` - Tech lead reviewing
- `changes_requested` - Revisions needed

**QA Phase:**
- `ready_for_qa` - Ready for testing
- `in_qa` - QA testing in progress
- `qa_failed` - Tests failed, needs fixes

**Approval Phase:**
- `ready_for_approval` - Ready for final approval
- `in_approval` - Product owner reviewing

**Terminal States:**
- `completed` - Work done and approved
- `cancelled` - Work not needed
- `blocked` - Waiting on external dependency
- `on_hold` - Temporarily suspended

**Characteristics:**
- Status flow enforcement
- Multiple agent types (ba, tech_lead, developer, qa, product_owner)
- Granular progress tracking
- Responsibility assignment
- Special status groups

**Agent Types:**
- **ba** (Business Analyst) - Refinement statuses
- **tech_lead** - Code review, architecture
- **developer** - Implementation
- **qa** - Testing and validation
- **product_owner** - Final approval

## Choosing a Profile

### Choose Basic if:
- ✅ You're working solo or with a small team (1-2 people)
- ✅ Your workflow is simple and flexible
- ✅ You don't need formal review processes
- ✅ You're prototyping or exploring ideas
- ✅ You want minimal configuration

### Choose Advanced if:
- ✅ You have a team with defined roles (3+ people)
- ✅ You practice test-driven development
- ✅ You need formal code review and QA processes
- ✅ You want granular progress tracking
- ✅ You need status flow enforcement

## Applying Profiles

### Fresh Installation

```bash
# Initialize with basic profile (default)
shark init

# Or apply advanced profile
shark init
shark init update --workflow=advanced
```

### Existing Installation

```bash
# Preview changes first
shark init update --workflow=advanced --dry-run

# Apply if satisfied
shark init update --workflow=advanced
```

### Migrating Between Profiles

**Basic to Advanced:**
```bash
# Tasks in old statuses remain valid
# New statuses become available
shark init update --workflow=advanced
```

**Advanced to Basic:**
```bash
# Warning: Tasks in advanced-only statuses need migration
# Use --force to overwrite
shark init update --workflow=basic --force
```

## Configuration Preservation

When applying profiles, these fields are ALWAYS preserved:
- Database configuration
- Viewer settings
- Project root path
- Last sync time
- Any custom fields not in profile

## Backup and Recovery

### Automatic Backup

Every update creates a timestamped backup:
```
.sharkconfig.json.backup.20260125-143022
```

### Manual Recovery

```bash
# Restore from backup
cp .sharkconfig.json.backup.20260125-143022 .sharkconfig.json
```

## Customization

After applying a profile, you can customize:
- Add custom statuses
- Modify colors and descriptions
- Adjust progress weights
- Add agent types
- Extend status flow

Changes are preserved when running `shark init update` without `--workflow` flag.

## Best Practices

1. **Start simple** - Use basic profile initially, upgrade to advanced when needed
2. **Preview first** - Always use `--dry-run` before applying profiles
3. **Backup manually** - Keep important configs in version control
4. **Customize gradually** - Add customizations after understanding defaults
5. **Document changes** - Note why you deviated from standard profile

## Troubleshooting

### Tasks stuck in old statuses after profile change

**Solution:** Manually update task statuses or create migration script.

### Lost custom statuses after applying profile

**Solution:** Restore from backup, then reapply customizations.

### Profile not applying

**Solution:** Check for write permissions on `.sharkconfig.json`.

## Examples

See `docs/examples/` for complete configuration examples:
- `basic-profile-config.json` - Complete basic profile
- `advanced-profile-config.json` - Complete advanced profile

## Related Documentation

- [CLI Reference - Initialization](../cli-reference/initialization.md) - Full command documentation
- [Workflow Configuration](../cli-reference/workflow-config.md) - Advanced customization
- [CLAUDE.md](../CLAUDE.md) - Agent routing and development workflow
