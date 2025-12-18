# Scope Boundaries

**Epic**: [Intelligent Documentation Scanning](./epic.md)

---

## Overview

This document explicitly defines what is **NOT** included in the intelligent scanning epic.

---

## Out of Scope

### Explicitly Excluded Features

**1. Bidirectional Sync (Database → Markdown)**
- **Why It's Out of Scope**: This epic focuses on importing markdown into database (filesystem → database). Writing database changes back to markdown files introduces complex merge conflicts, file locking issues, and race conditions.
- **Future Consideration**: Could be addressed in E07 (Bidirectional Sync) after intelligent import is proven stable
- **Workaround**: Users manually edit markdown files and re-sync to database (current workflow)

**2. Real-Time File Watching**
- **Why It's Out of Scope**: Implementing file system watchers (inotify, FSEvents, ReadDirectoryChangesW) adds significant complexity across platforms. Focus is on explicit sync commands triggered by users/agents.
- **Future Consideration**: Could add `shark watch` command in future epic if demand warrants
- **Workaround**: Run `shark sync --incremental` explicitly after file changes (agents can automate this in their workflow)

**3. Machine Learning for Pattern Recognition**
- **Why It's Out of Scope**: Using ML models to infer epic/feature/task structure from arbitrary markdown would require training data, model deployment, and ongoing maintenance. This epic uses deterministic pattern matching with configurable rules.
- **Future Consideration**: Not planned; deterministic patterns cover 90%+ of real-world cases
- **Workaround**: Users configure patterns and whitelists for edge cases

**4. Import from External Systems**
- **Why It's Out of Scope**: Importing from Jira, Linear, GitHub Issues, Notion, etc. requires system-specific adapters, authentication, and API integration. This epic focuses solely on markdown files.
- **Future Consideration**: E08 (External System Adapters) could provide import bridges
- **Workaround**: Export from external system to markdown, then use intelligent scanner

**5. Merge Conflict Resolution UI**
- **Why It's Out of Scope**: Building an interactive TUI or GUI for manual conflict resolution adds significant UI complexity. This epic provides automatic resolution strategies with detailed logging.
- **Future Consideration**: Could add `shark resolve` interactive command in future if automated strategies prove insufficient
- **Workaround**: Users review conflict logs and manually edit files or database, then re-sync

**6. Markdown Validation and Linting**
- **Why It's Out of Scope**: Checking markdown formatting, link validity, spelling, etc. is orthogonal to scanning. Scanner tolerates varied markdown styles.
- **Future Consideration**: Could integrate with existing markdown linters (markdownlint) as separate feature
- **Workaround**: Use standalone markdown linters before scanning

**7. Version Control Integration (Git Hooks)**
- **Why It's Out of Scope**: While REQ-F-018 mentions git-based change detection as "could have", automatic git hook installation and commit-triggered syncs are explicitly excluded from initial release.
- **Future Consideration**: Users can manually configure pre-commit/post-commit hooks to run `shark sync`
- **Workaround**: Run sync manually or via CI/CD pipeline

**8. Multi-Project / Workspace Support**
- **Why It's Out of Scope**: This epic assumes single project with one .sharkconfig.json and one project.db. Supporting multiple projects with shared configuration or cross-project dependencies is out of scope.
- **Future Consideration**: Could add workspace concept in future epic
- **Workaround**: Use separate shark databases per project

**9. Interactive Pattern Builder UI**
- **Why It's Out of Scope**: Building an interactive TUI or web UI for creating regex patterns would add significant complexity. This epic provides preset library and validation, users edit JSON config directly.
- **Future Consideration**: Could add `shark config pattern-wizard` interactive command in future if demand warrants
- **Workaround**: Use pattern presets (`shark config add-pattern --preset=special-epics`), copy examples from documentation, or use online regex testers (regex101.com) to validate patterns before adding to config

---

### Edge Cases & Scenarios Not Covered

**1. Circular Feature Dependencies**
- **Impact**: If feature folders reference each other circularly (Feature A → Feature B → Feature A), scanner may enter infinite loop or fail to resolve order
- **Rationale**: Rare edge case that indicates documentation structure problem
- **Mitigation**: Scanner detects circular references during validation and logs error without importing affected features

**2. Extremely Large Files (>10MB)**
- **Impact**: Scanning 50MB markdown files could cause memory issues or extreme slowdowns
- **Rationale**: Task files should be <100KB; anything larger indicates misuse (binary files, embeddings)
- **Mitigation**: Scanner skips files >5MB with warning (configurable threshold)

**3. Unicode and Emoji in Slugs**
- **Impact**: Epic slugs like "E01-🚀-launch-feature" or feature slugs with non-ASCII characters may not be handled correctly
- **Rationale**: File systems have varied Unicode support; kebab-case ASCII is safest
- **Mitigation**: Default patterns validate slugs contain only [a-z0-9-] and log warning for non-compliant slugs; users can define custom patterns for Unicode support if needed

**7. Complex Regex Pattern Debugging**
- **Impact**: Users may write invalid regex patterns or patterns missing required capture groups, causing import failures
- **Rationale**: Regex is powerful but can be confusing for non-technical users
- **Mitigation**: Provide pattern validation on config load, comprehensive preset library, detailed error messages with examples, and `shark config validate-patterns` command for testing

**4. Concurrent Scan Operations**
- **Impact**: Two users/agents running `shark scan` simultaneously on same database could cause lock contention or race conditions
- **Rationale**: Single-developer focus with occasional AI agent assistance (E04 scope)
- **Mitigation**: Database transaction locks prevent corruption; second scan waits or fails with lock timeout error

**5. Symbolic Links and Hardlinks**
- **Impact**: Scanner may follow symlinks leading to infinite loops (symlink cycles) or double-counting files
- **Rationale**: Complex filesystem structures are rare in documentation
- **Mitigation**: Scanner follows symlinks but tracks visited inodes to prevent cycles (platform-dependent, may not work on Windows)

**6. Case-Insensitive File Systems (macOS, Windows)**
- **Impact**: Scanning on macOS/Windows may treat E01-epic and e01-epic as same file; Linux treats as different
- **Rationale**: Documentation should use consistent casing per conventions
- **Mitigation**: Scanner normalizes paths to lowercase for comparison, logs warning if case-only duplicates detected

---

## Alternative Approaches Considered But Rejected

### Alternative 1: YAML-Based Epic Index Instead of epic-index.md

**Description**:
Use structured YAML file (epic-index.yaml) instead of markdown with links for explicit epic/feature declarations.

**Pros**:
- Easier to parse programmatically (no regex for markdown links)
- Can include richer metadata (epic priority, status, dependencies)
- Validates against schema (JSON Schema or similar)

**Cons**:
- Less human-readable than markdown with links
- Requires users to learn YAML syntax and schema
- Doesn't leverage existing markdown-based documentation ecosystem
- Adds tooling complexity (schema validation, YAML parser)

**Decision Rationale**:
Markdown-first approach aligns with git-friendly documentation philosophy. epic-index.md can be generated by `shark index generate` command and is easily editable by humans. YAML adds barrier for non-technical users.

---

### Alternative 2: Database-First with Markdown Generation

**Description**:
Make database the authoritative source of truth, generate markdown files from database on demand.

**Pros**:
- Eliminates sync complexity (single source of truth)
- No conflict resolution needed
- Guarantees consistency

**Cons**:
- Breaks git workflow (markdown files are ephemeral, not version-controlled)
- Loses context and prose from handwritten documentation
- Requires custom editors or tooling for all modifications
- Violates E04 design principle: "markdown files are authoritative"

**Decision Rationale**:
Markdown-first approach is core to shark philosophy. Database is optimization layer for queries, not replacement for documentation. Generated markdown loses valuable context (design rationale, discussions, links to external resources).

---

### Alternative 3: Strict Schema Enforcement with Migration Tools

**Description**:
Require all documentation to match rigid schema (E##-epic-slug only, no special types), provide `shark migrate` tool to automatically restructure files.

**Pros**:
- Simpler scanner (no pattern matching complexity)
- Guaranteed consistency
- Clear migration path for legacy projects

**Cons**:
- High barrier to adoption (forces restructuring before usage)
- Migration tool is complex (must preserve git history, update links)
- Breaks existing documentation URLs and bookmarks
- Loses semantic meaning of tech-debt, bugs categories

**Decision Rationale**:
Flexibility is key value proposition. Users want shark to adapt to their workflow, not force workflow changes. Pattern matching and whitelists provide "works out of the box" experience while encouraging best practices over time.

---

### Alternative 4: Plugin System for Custom Scanners

**Description**:
Provide plugin API allowing users to write custom scanner plugins for their unique documentation structures.

**Pros**:
- Maximum flexibility for edge cases
- Community-driven scanner ecosystem
- Doesn't bloat core with niche patterns

**Cons**:
- Complex to implement (plugin API design, sandboxing, versioning)
- High barrier for users (must write code to use shark)
- Fragments ecosystem (incompatible plugins, maintenance burden)
- Overkill for current need (configurable patterns + whitelists cover most cases)

**Decision Rationale**:
Configuration-driven approach (patterns, whitelists, validation levels) provides sufficient flexibility without plugin complexity. Can revisit plugin system in future if clear demand emerges for highly specialized scanning.

---

## Future Epic Candidates

Features/capabilities that are natural follow-ons to intelligent scanning:

| Future Epic Concept | Priority | Dependency |
|---------------------|----------|------------|
| E07: Bidirectional Sync (Database → Markdown) | Medium | Depends on E06 stability |
| E08: External System Import Adapters (Jira, Linear, GitHub Issues) | Medium | Builds on E06 patterns |
| E09: Real-Time File Watching (`shark watch`) | Low | Complements E06 incremental sync |
| E10: Interactive Conflict Resolution TUI | Low | Extends E06 conflict detection |
| E11: Markdown Quality Tools (validation, linting, auto-formatting) | Low | Optional enhancement to E06 |

---

*See also*: [Requirements](./requirements.md)
