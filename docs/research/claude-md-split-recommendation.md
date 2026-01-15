# CLAUDE.md Split Recommendation

## Executive Summary

The current CLAUDE.md is **1215 lines** with mixed concerns spanning project overview, architecture, CLI commands, database management, testing patterns, and development workflows. According to Claude Code best practices, this should be split into a modular structure using `.claude/rules/` for better context management and token efficiency.

## Key Principles from Research

### 1. Memory Hierarchy (from claude-memory.md)
- **Project memory**: `./CLAUDE.md` or `./.claude/CLAUDE.md` - Team-shared instructions
- **Project rules**: `./.claude/rules/*.md` - Modular, topic-specific instructions
- **Path-specific rules**: Use YAML frontmatter with `paths` field for conditional loading
- **Imports**: Use `@path/to/file` syntax to reference other files

### 2. Conciseness Principle (from skills-best-practices.md)
- "The context window is a public good"
- Only load what's needed for the current task
- Progressive disclosure: overview with links to detailed content
- Keep main files under 500 lines, ideally much less

### 3. Modular Organization (from claude-memory.md)
```
.claude/
├── CLAUDE.md           # Main project instructions (navigation hub)
└── rules/
    ├── code-style.md   # Code style guidelines
    ├── testing.md      # Testing conventions
    └── security.md     # Security requirements
```

### 4. Path-Specific Rules (from claude-memory.md)
Rules can be scoped to specific files using YAML frontmatter:
```markdown
---
paths: src/api/**/*.ts
---
# API Development Rules
```

## Current CLAUDE.md Analysis

### Content Breakdown by Section

| Section | Lines | Purpose | Frequency of Use |
|---------|-------|---------|------------------|
| Project Overview | 15 | Context | Every session |
| Build Commands | 30 | Quick reference | Frequent |
| Database Management | 60 | Critical warnings | When DB issues occur |
| Cloud Database (Turso) | 140 | Configuration | Setup only |
| Database Access Pattern | 60 | Implementation details | When modifying CLI |
| Project Root Auto-Detection | 60 | Implementation details | Rarely needed |
| Database Migrations | 10 | Implementation details | Rarely needed |
| Slug Architecture | 140 | Implementation details | When working with keys |
| Project Architecture | 140 | Architecture details | When understanding structure |
| Database Schema | 40 | Schema details | When modifying DB |
| CLI Command Structure | 100 | Command reference | Frequent |
| Task Creation Standards | 75 | Workflow | When creating tasks |
| Common Development Tasks | 45 | Development patterns | As needed |
| Important Patterns | 30 | Coding standards | When coding |
| Testing Architecture | 225 | Testing patterns | When writing tests |
| Performance Notes | 25 | Optimization details | Rarely needed |
| Development Workspace | 70 | Workspace patterns | Active development |

### Issues with Current Structure

1. **Context Overload**: All 1215 lines loaded on every session
2. **Mixed Concerns**: Architecture, commands, patterns, and workflows in one file
3. **No Progressive Disclosure**: No way to load only relevant sections
4. **Duplicate Information**: Some patterns repeated across sections
5. **No Path-Specific Rules**: Go-specific patterns not conditionally loaded

## Recommended Split Structure

### New File Organization

```
.claude/
├── CLAUDE.md                      # Main hub (~200 lines)
└── rules/
    ├── quickref.md                # Quick reference commands (~50 lines)
    ├── database-critical.md       # Critical DB warnings (~60 lines)
    ├── architecture.md            # Project architecture (~150 lines)
    ├── database-schema.md         # Database details (~200 lines)
    ├── cli-commands.md            # CLI command reference (~120 lines)
    ├── testing.md                 # Testing patterns (~250 lines)
    ├── development-workflows.md   # Task creation, workflows (~100 lines)
    ├── go-patterns.md             # Go-specific patterns (~80 lines)
    │                              # paths: internal/**/*.go, cmd/**/*.go
    └── go-testing.md              # Go test patterns (~100 lines)
                                   # paths: internal/**/*_test.go
```

### 1. Main CLAUDE.md (Navigation Hub)

**Purpose**: High-level overview and navigation to detailed rules
**Length**: ~200 lines

**Contents**:
- Project overview (what is Shark Task Manager)
- Key technologies
- Critical warnings (database management - abbreviated)
- Navigation guide to other rules
- Quick command reference (most common commands only)
- When to use which rules file

**Example Structure**:
```markdown
# CLAUDE.md

Shark Task Manager is a Go-based CLI tool for managing project tasks, features,
and epics with AI-driven development workflows.

## Quick Start

**Build**: `make build` or `make shark`
**Test**: `make test`
**Run**: `./bin/shark task list`

See @.claude/rules/quickref.md for complete command reference.

## ⚠️ Critical: Database Management

**NEVER delete shark-tasks.db** - it's the source of truth for all project data.

See @.claude/rules/database-critical.md for recovery procedures.

## Navigation Guide

When working on specific areas, refer to these detailed guides:

- **Architecture & Structure**: @.claude/rules/architecture.md
- **Database Schema & Cloud**: @.claude/rules/database-schema.md
- **CLI Commands**: @.claude/rules/cli-commands.md
- **Testing Patterns**: @.claude/rules/testing.md
- **Development Workflows**: @.claude/rules/development-workflows.md
- **Go Coding Patterns**: @.claude/rules/go-patterns.md (auto-loaded for .go files)
- **Go Testing Patterns**: @.claude/rules/go-testing.md (auto-loaded for *_test.go files)

## Key Concepts

[Brief overview of: dual key format, slug architecture, task lifecycle]

## Documentation References

- Architecture details: @.claude/rules/architecture.md
- Complete CLI reference: @docs/CLI_REFERENCE.md
- Turso migration: @docs/TURSO_MIGRATION.md
```

### 2. .claude/rules/quickref.md

**Purpose**: Quick command reference
**Length**: ~50 lines

**Contents**:
- Build commands
- Test commands
- Common shark commands
- Quick examples

### 3. .claude/rules/database-critical.md

**Purpose**: Critical database management warnings and recovery
**Length**: ~60 lines

**Contents**:
- DO NOT delete database warning
- Recovery procedures
- Sync troubleshooting
- What NOT to do

### 4. .claude/rules/architecture.md

**Purpose**: Project architecture and design patterns
**Length**: ~150 lines

**Contents**:
- Directory structure
- Data flow
- Design patterns (Dependency Injection, Repository, Cobra, fileops, Sync)
- Module descriptions

### 5. .claude/rules/database-schema.md

**Purpose**: Database details, schema, configuration
**Length**: ~200 lines

**Contents**:
- Core tables
- SQLite configuration
- Database migrations
- Cloud database (Turso) support
- Database access patterns
- Progress calculation

### 6. .claude/rules/cli-commands.md

**Purpose**: CLI command reference
**Length**: ~120 lines

**Contents**:
- Root command and global flags
- Key format flexibility
- Command categories (Epic, Feature, Task management)
- Synchronization
- Configuration

### 7. .claude/rules/testing.md

**Purpose**: Testing architecture and patterns
**Length**: ~250 lines

**Contents**:
- Testing golden rule (repository tests only use DB)
- Test categories
- Test organization
- Common testing mistakes
- Running tests
- Test database

### 8. .claude/rules/development-workflows.md

**Purpose**: Development workflows and task creation
**Length**: ~100 lines

**Contents**:
- Task & feature creation standards
- Task status & lifecycle
- Development workspace structure
- Development patterns
- Common development tasks

### 9. .claude/rules/go-patterns.md (Path-Specific)

**Purpose**: Go-specific coding patterns
**Length**: ~80 lines

**YAML Frontmatter**:
```yaml
---
paths: "{internal,cmd}/**/*.go"
---
```

**Contents**:
- Error handling patterns
- Database transactions
- CLI output patterns
- Validation patterns
- File system sync patterns

### 10. .claude/rules/go-testing.md (Path-Specific)

**Purpose**: Go testing patterns
**Length**: ~100 lines

**YAML Frontmatter**:
```yaml
---
paths: "{internal,cmd}/**/*_test.go"
---
```

**Contents**:
- Testing patterns specific to Go
- Mock creation
- Table-driven tests
- Cleanup patterns
- Assertion patterns

## Benefits of This Structure

### 1. Context Efficiency
- **Before**: 1215 lines always loaded
- **After**: ~200 lines main + only relevant rules loaded
- **Savings**: 80-90% reduction in baseline context usage

### 2. Progressive Disclosure
- Main CLAUDE.md provides navigation
- Claude loads detailed rules only when needed
- Path-specific rules auto-load for relevant files

### 3. Maintenance
- Easier to update individual topics
- Clear separation of concerns
- No duplicate content

### 4. Discoverability
- Navigation guide in main file
- Clear file names indicate content
- Path-specific rules load automatically

### 5. Team Collaboration
- Easier for team members to contribute to specific areas
- Clearer ownership of documentation sections
- Better git history for changes

## Implementation Plan

### Phase 1: Create Directory Structure
```bash
mkdir -p .claude/rules
```

### Phase 2: Split Content
1. Create new CLAUDE.md as navigation hub
2. Extract sections to individual rule files
3. Add path-specific frontmatter to go-patterns.md and go-testing.md
4. Use @imports in main CLAUDE.md to reference other files

### Phase 3: Test
1. Test that rules load correctly
2. Verify path-specific rules activate for .go files
3. Check that navigation works with imports
4. Measure context usage reduction

### Phase 4: Refinement
1. Adjust file sizes if needed
2. Add cross-references between rule files
3. Update as needed based on usage

## Migration Considerations

### Backward Compatibility
- Keep a backup of original CLAUDE.md
- Can temporarily keep both structures
- Gradual migration: create rules/, keep CLAUDE.md as is initially

### Import Strategy
Main CLAUDE.md can import other files:
```markdown
## Database Management

See @.claude/rules/database-critical.md for critical warnings.
See @.claude/rules/database-schema.md for schema details.
```

### Path-Specific Rule Testing
Test that go-patterns.md loads when editing Go files:
```bash
# Edit a .go file and check context
# Should see go-patterns.md loaded
```

## Alternative Considerations

### Option 1: Skills Instead of Rules
Could create Skills for complex workflows:
- `.claude/skills/debugging/` - Debugging workflow
- `.claude/skills/testing/` - Testing workflow
- `.claude/skills/code-review/` - Code review workflow

**Pros**:
- Can include scripts and utilities
- Better for complex multi-step workflows
- Can bundle reference materials

**Cons**:
- More overhead for simple instructions
- Skills better for optional capabilities, not core project knowledge
- Rules are better for "always-on" project context

**Recommendation**: Use rules for now. Consider skills later for optional workflows like code review automation, debugging helpers, etc.

### Option 2: Hybrid Approach
- Rules for core project knowledge (architecture, patterns, testing)
- Skills for optional workflows (debugging, code review, performance analysis)
- Slash commands for frequently-used prompts (commit, review PR, etc.)

**Recommendation**: Start with rules, add skills/slash-commands as needs emerge.

## Next Steps

1. **Review this recommendation** with team/project owner
2. **Create backup** of current CLAUDE.md
3. **Implement Phase 1-2**: Create directory structure and split content
4. **Test Phase 3**: Verify loading and context usage
5. **Iterate Phase 4**: Refine based on real usage

## Appendix: Rules vs Slash Commands vs Skills

| Feature | Rules | Slash Commands | Skills |
|---------|-------|----------------|--------|
| **Purpose** | Always-on project context | Quick frequently-used prompts | Complex optional capabilities |
| **Structure** | Markdown files in .claude/rules/ | Markdown files in .claude/commands/ | Directory with SKILL.md + resources |
| **Loading** | Auto-loaded (all or path-specific) | Explicit invocation (/command) | Auto-discovered based on context |
| **Complexity** | Medium (organized docs) | Simple (single file prompts) | High (multi-file + scripts) |
| **Use Case** | Project architecture, patterns | "Review code", "Create commit" | PDF processing, Data analysis |

**For Shark Task Manager**:
- **Rules**: Architecture, database, testing patterns ✅
- **Slash Commands**: Common tasks like "/review-pr", "/debug" (future)
- **Skills**: Complex workflows like "/analyze-performance" (future)
