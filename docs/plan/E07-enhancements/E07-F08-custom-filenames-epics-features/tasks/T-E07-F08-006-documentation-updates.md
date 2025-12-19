---
task_key: T-E07-F08-006
epic_key: E07
feature_key: E07-F08
title: Update CLI reference and user documentation
status: created
priority: 4
agent_type: documentation
depends_on: ["T-E07-F08-004", "T-E07-F08-005"]
---

# Task: Update CLI reference and user documentation

## Objective

Update project documentation to reflect the new `--filename` and `--force` flags for epic and feature creation, ensuring users understand how to use custom file paths across all entity types.

## Context

**Why this task exists**: Documentation is the primary interface for users learning the CLI. Without updated docs, users won't discover the new functionality, and the feature's value is diminished.

**Design reference**:
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (lines 507-518, 577-658)
- `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (lines 43-198)

## What to Build

Update the following documentation files:

### 1. CLI Reference (`docs/CLI_REFERENCE.md`)

Add or update documentation for:

**shark epic create**:
- Add `--filename` flag documentation with description, type, default, and example
- Add `--force` flag documentation with description and use case
- Add examples showing:
  - Default behavior (backward compatible)
  - Custom filename usage
  - Collision detection error
  - Force reassignment

**shark feature create**:
- Add `--filename` flag documentation with description, type, default, and example
- Add `--force` flag documentation with description and use case
- Add examples showing:
  - Default behavior (backward compatible)
  - Custom filename usage
  - Collision detection error
  - Force reassignment

**Format Pattern** (follow existing CLI_REFERENCE.md structure):

```markdown
### shark epic create

Create a new epic with optional custom file location.

**Usage:**
```
shark epic create [flags] <title>
```

**Flags:**
| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--filename` | string | (empty) | Custom file path relative to project root. Must end in `.md`. Example: `docs/roadmap/2025.md` |
| `--force` | bool | false | Force reassignment if file path already claimed by another entity |
| `--description` | string | (empty) | Epic description |
| `--priority` | string | medium | Priority: high, medium, or low |
| `--business-value` | string | (empty) | Business value: high, medium, or low |

**Examples:**
```bash
# Default location (backward compatible)
shark epic create "User Authentication System"
# Output: Created epic E04 at docs/plan/E04/epic.md

# Custom location
shark epic create --filename="docs/roadmap/2025-platform.md" "Platform Roadmap"
# Output: Created epic E09 at docs/roadmap/2025-platform.md

# Force reassignment
shark epic create --filename="docs/shared/auth.md" --force "SSO Integration"
# Output: Created epic E10 at docs/shared/auth.md (reassigned from epic E04)
```
```

### 2. CLAUDE.md

Update the project overview section to mention feature parity:

**Section to Update**: "CLI Command Structure" → "Epic Management" and "Feature Management"

Add:
- `--filename` and `--force` flags to epic and feature creation command syntax
- Note that these flags follow the same patterns as task creation
- Reference to CLI_REFERENCE.md for detailed examples

**Example Addition**:

```markdown
#### Epic Management
- `shark epic create --title="..." [--filename=<path>] [--force] [--priority=...] [--business-value=...] [--json]`
  - `--filename`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another epic or feature
- `shark epic list [--json]`
- `shark epic get <epic-key> [--json]`

#### Feature Management
- `shark feature create --epic=<epic-key> --title="..." [--filename=<path>] [--force] [--execution-order=...] [--json]`
  - `--filename`: Custom file path (relative to root, must include .md)
  - `--force`: Reassign file if already claimed by another feature or epic
- `shark feature list --epic=<epic-key> [--json]`
- `shark feature get <feature-key> [--json]`
```

### 3. README.md (if applicable)

Check if README.md has a "Features" or "CLI Commands" section. If so, add a bullet point:

```markdown
- **Custom File Paths**: Organize epics, features, and tasks anywhere in your project with the `--filename` flag
```

## Success Criteria

- [ ] `docs/CLI_REFERENCE.md` documents `--filename` and `--force` for both epic and feature creation
- [ ] Examples show default behavior, custom paths, collision errors, and force reassignment
- [ ] `CLAUDE.md` updated with new flag syntax in CLI command reference section
- [ ] Documentation is consistent with actual CLI behavior (verified via manual testing)
- [ ] Examples use realistic file paths (e.g., `docs/roadmap/...`, `docs/specs/...`)
- [ ] Error messages shown in examples match actual CLI output

## Validation Gates

1. **Accuracy Check**:
   - Run each example command in the documentation
   - Verify outputs match documented examples
   - If discrepancies found, update documentation to match reality

2. **Consistency Check**:
   - Compare epic and feature documentation for parallel structure
   - Ensure flag descriptions are identical where applicable (e.g., `--filename` has same description for both)

3. **Markdown Syntax**:
   - Render `CLI_REFERENCE.md` in a markdown viewer
   - Verify code blocks, tables, and formatting display correctly

## Dependencies

**Prerequisite Tasks**:
- T-E07-F08-004 (epic CLI implementation must be complete to document accurately)
- T-E07-F08-005 (feature CLI implementation must be complete to document accurately)

**Blocks**: None (documentation is final task)

## Implementation Notes

### Documentation Style

Follow existing patterns in `docs/CLI_REFERENCE.md`:
- Use tables for flag documentation
- Use code blocks for examples with comments showing output
- Include both success and error scenarios in examples

### Example Selection

Choose examples that demonstrate:
1. **Default behavior**: Show backward compatibility
2. **Custom path**: Show new functionality
3. **Collision without --force**: Show error handling
4. **Collision with --force**: Show reassignment
5. **Validation error**: Show at least one validation failure (e.g., absolute path)

### Testing Examples

Before finalizing documentation, test each example command:

```bash
# For each example in the documentation
./bin/shark epic create "User Authentication System"
./bin/shark epic create --filename="docs/roadmap/2025-platform.md" "Platform Roadmap"
./bin/shark epic create --filename="docs/roadmap/2025-platform.md" "Another Epic"
# (Should fail with collision error)
./bin/shark epic create --filename="docs/roadmap/2025-platform.md" --force "Another Epic"
# (Should succeed with reassignment message)

# Repeat for feature examples
```

Capture actual output and paste into documentation examples.

### Cross-References

Add cross-references between documents:
- In `CLI_REFERENCE.md`, reference `CLAUDE.md` for development context
- In `CLAUDE.md`, reference `CLI_REFERENCE.md` for detailed CLI usage

## References

- **PRD Appendix A**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/prd.md` (Section: Example User Workflows)
- **Backend Design**: `docs/plan/E07-enhancements/E07-F08-custom-filenames-epics-features/04-backend-design.md` (CLI Commands section)
- **Existing Documentation**: `docs/CLI_REFERENCE.md` (for style consistency)
