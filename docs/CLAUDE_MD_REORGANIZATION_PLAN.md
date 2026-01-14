# CLAUDE.md Reorganization Plan

Based on best practices from Claude Code documentation research.

## Problem Statement

Current CLAUDE.md is 888 lines - exceeds recommended 500-line limit for optimal performance.
Contains multiple distinct topics mixed together, making it hard to navigate and maintain.

## Best Practices Applied

From research in `docs/research/`:

1. **Modular rules** (`.claude/rules/`) - Split large files into focused topics
2. **Conciseness** - Keep main file under 200 lines as navigation hub
3. **Progressive disclosure** - Overview in main file, details in separate files
4. **Domain organization** - Group related content by topic
5. **Path-specific rules** - Use frontmatter for file-type specific guidance
6. **Descriptive filenames** - Each file covers one clear topic

## Proposed Structure

```
.claude/
├── CLAUDE.md                           # ~150-200 lines - navigation hub
└── rules/
    ├── database/
    │   ├── overview.md                 # Database management, critical warnings
    │   ├── turso-cloud.md              # Cloud database setup and migration
    │   ├── access-patterns.md          # Global DB instance, initialization
    │   └── migrations.md               # Auto-migration system
    ├── cli/
    │   ├── commands.md                 # CLI command structure and reference
    │   ├── task-commands.md            # Task management commands (primary AI interface)
    │   └── slug-architecture.md        # Dual key format system
    ├── architecture/
    │   ├── overview.md                 # Directory structure, data flow
    │   ├── design-patterns.md          # DI, repositories, unified file ops
    │   └── schema.md                   # Database schema, lifecycle states, progress
    ├── development/
    │   ├── workspace.md                # Dev workspace patterns and structure
    │   ├── task-creation.md            # Task and feature creation standards
    │   └── common-tasks.md             # Adding commands, repository methods
    ├── testing/
    │   ├── architecture.md             # Golden rule, test categories
    │   └── patterns.md                 # Repository vs CLI testing patterns
    └── go-specific.md                  # Go-specific patterns (path-specific rule)
```

## Content Mapping

### Main CLAUDE.md (150-200 lines)
- Project overview (1 paragraph)
- Key technologies (bullet list)
- Quick build/test commands
- Critical warnings (database, testing)
- Navigation links to all rule files
- Project root auto-detection

### .claude/rules/database/overview.md (~150 lines)
**Source**: CLAUDE.md lines 68-159, 274-287

Content:
- ⚠️ CRITICAL: DO NOT DELETE DATABASE warning
- Database reset procedure
- Sync error handling (UNIQUE constraint)
- Auto-migration system
- Common database issues

### .claude/rules/database/turso-cloud.md (~150 lines)
**Source**: CLAUDE.md lines 161-201

Content:
- Turso quick setup
- Configuration (.sharkconfig.json)
- Multi-machine usage
- Switching backends
- Migration from local to cloud
- Troubleshooting
- Documentation links

### .claude/rules/database/access-patterns.md (~70 lines)
**Source**: CLAUDE.md lines 203-244

Content:
- Global database instance pattern
- Implementation pattern (GetDB, ResetDB)
- Usage in commands
- Testing patterns
- Database backends (SQLite vs Turso)
- Architecture benefits

### .claude/rules/cli/commands.md (~120 lines)
**Source**: CLAUDE.md lines 503-567

Content:
- Root command and global flags
- Key format flexibility (case insensitive)
- Short task key format
- Command categories:
  - Initialization
  - Epic management
  - Feature management
  - Task management
  - Synchronization
  - Configuration

### .claude/rules/cli/task-commands.md (~80 lines)
**Source**: Extracted from CLAUDE.md lines 503-567 (task section)

Content:
- Task commands (primary AI interface)
- Positional vs flag syntax
- Status transitions
- File path organization
- Examples for each command

### .claude/rules/cli/slug-architecture.md (~120 lines)
**Source**: CLAUDE.md lines 289-392

Content:
- Key formats (numeric, slugged)
- Automatic slug generation
- Dual key lookup strategy
- Database schema changes
- Usage examples (epics, features, tasks)
- Benefits (for humans, AI agents, systems)
- Implementation details
- Slug migration

### .claude/rules/architecture/overview.md (~100 lines)
**Source**: CLAUDE.md lines 394-446

Content:
- Directory structure (tree view)
- Data flow (command → repository → database)
- Key packages and their responsibilities
- High-level architecture diagram (text)

### .claude/rules/architecture/design-patterns.md (~120 lines)
**Source**: CLAUDE.md lines 394-446, 655-677

Content:
- Dependency injection via constructors
- Repository pattern
- Cobra command structure
- Unified file operations (fileops package)
- File-database sync
- Error handling patterns
- Database transactions
- CLI output patterns
- Validation patterns

### .claude/rules/architecture/schema.md (~80 lines)
**Source**: CLAUDE.md lines 448-501

Content:
- Core tables (epics, features, tasks, task_history)
- SQLite configuration (foreign keys, WAL, indexes)
- Task lifecycle states
- Progress calculation
- Database constraints and triggers

### .claude/rules/development/workspace.md (~50 lines)
**Source**: CLAUDE.md lines 822-888 (workspace section)

Content:
- Dev workspace structure pattern
- Date formatting
- Artifact types (analysis, scripts, verification, shared)
- Cleanup guidelines

### .claude/rules/development/task-creation.md (~80 lines)
**Source**: CLAUDE.md lines 569-619, 822-888 (patterns section)

Content:
- Creating tasks workflow (4 steps)
- Task status and lifecycle
- Development patterns (specifications, artifacts, debugging, migration, testing)
- Shark CLI usage for development work

### .claude/rules/development/common-tasks.md (~80 lines)
**Source**: CLAUDE.md lines 621-653

Content:
- Adding a new CLI command
- Adding a repository method
- Running a single test
- Database debugging
- Hot-reload development

### .claude/rules/testing/architecture.md (~120 lines)
**Source**: CLAUDE.md lines 679-796

Content:
- ⚠️ TESTING GOLDEN RULE
- Test categories:
  1. Repository tests (use real DB + cleanup)
  2. CLI command tests (use mocks)
  3. Service layer tests (use mocks)
  4. Unit tests (pure logic)
- Test organization
- Common testing mistakes (good/bad examples)
- Running tests
- Test database location

### .claude/rules/testing/patterns.md (~60 lines)
**Source**: CLAUDE.md lines 679-796 (examples section)

Content:
- Repository test pattern with cleanup
- CLI test pattern with mocks
- Mock interface examples
- Test isolation examples

### .claude/rules/go-specific.md (~40 lines)
**Source**: CLAUDE.md lines 655-677
**Frontmatter**: `paths: "**/*.go"`

Content:
- Error handling (explicit returns, wrapping)
- Database transactions (defer rollback)
- Validation (model layer before DB)
- File system sync patterns

## Migration Steps

1. **Create directory structure**
   ```bash
   mkdir -p .claude/rules/{database,cli,architecture,development,testing}
   ```

2. **Extract content to new files** (in order)
   - Start with database/ rules (most critical)
   - Then architecture/ (referenced frequently)
   - Then cli/ (command reference)
   - Then development/ (workflow guidance)
   - Then testing/ (patterns)
   - Finally go-specific.md

3. **Create new main CLAUDE.md**
   - High-level overview
   - Quick commands
   - Critical warnings (database, testing)
   - Navigation to all rule files

4. **Verify all content migrated**
   - Line count check (old should equal sum of new)
   - No orphaned content
   - All cross-references updated

5. **Test with Claude Code**
   - Run `/memory` to verify all files loaded
   - Test that rules are discovered
   - Verify path-specific rules work

## Benefits

1. **Performance**: Each file <500 lines, faster loading
2. **Navigation**: Clear topic separation, easy to find
3. **Maintenance**: Update specific topics without touching others
4. **Reusability**: Can share specific rule files across projects
5. **Progressive disclosure**: Claude loads only relevant rules
6. **Path-specific rules**: Go patterns only apply to .go files
7. **Team collaboration**: Easier to review/update specific topics

## Validation Checklist

- [ ] All content from original CLAUDE.md accounted for
- [ ] No duplicate content across files
- [ ] All internal references updated
- [ ] Main CLAUDE.md is <200 lines
- [ ] Each rule file is <500 lines
- [ ] Filenames are descriptive
- [ ] Directory structure is logical
- [ ] Path-specific frontmatter added where appropriate
- [ ] Critical warnings preserved and prominent
- [ ] Navigation links in main CLAUDE.md work
- [ ] `/memory` command shows all files

## Success Criteria

- Claude can find all guidance by topic
- Faster context loading (smaller files)
- Easier maintenance (focused files)
- Better team collaboration (granular updates)
- Preserved all critical warnings and patterns
