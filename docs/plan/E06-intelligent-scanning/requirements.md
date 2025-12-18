# Requirements

**Epic**: [Intelligent Documentation Scanning](./epic.md)

---

## Overview

This document contains all functional and non-functional requirements for the intelligent scanning epic.

**Requirement Traceability**: Each requirement maps to specific [user journeys](./user-journeys.md) and [personas](./personas.md).

---

## Functional Requirements

### Priority Framework

We use **MoSCoW prioritization**:
- **Must Have**: Critical for launch; epic fails without these
- **Should Have**: Important but workarounds exist; target for initial release
- **Could Have**: Valuable but deferrable; include if time permits
- **Won't Have**: Explicitly out of scope (see [scope.md](./scope.md))

---

### Must Have Requirements

#### Epic Discovery & Import

**REQ-F-001**: epic-index.md Parsing
- **Description**: System must parse docs/plan/epic-index.md to extract epic keys, titles, and feature links
- **User Story**: As a Technical Lead, I want epic-index.md to be the source of truth so that I can explicitly control which epics are imported
- **Acceptance Criteria**:
  - [ ] Parse markdown links in format [Epic Name](./E##-epic-slug/) or [Feature Name](./E##-epic-slug/E##-F##-feature-slug/)
  - [ ] Extract epic keys from folder paths (E##-epic-slug format)
  - [ ] Extract feature keys from folder paths (E##-F##-feature-slug format)
  - [ ] Handle relative and absolute paths correctly
  - [ ] Log warnings for broken links or invalid formats
- **Related Journey**: Journey 1, Step 1; Journey 4, Step 1

**REQ-F-002**: Folder Structure Fallback
- **Description**: When epic-index.md is absent, system must discover epics by scanning docs/plan/ folder structure
- **User Story**: As a Technical Lead, I want to import documentation even without an index so that I can start using shark immediately on legacy projects
- **Acceptance Criteria**:
  - [ ] Recursively scan configured root directory (default: docs/plan/)
  - [ ] Recognize E##-epic-slug folders as epics
  - [ ] Recognize E##-F##-feature-slug folders as features within epics
  - [ ] Build epic/feature hierarchy from folder structure
  - [ ] Validate feature folders are nested within epic folders
- **Related Journey**: Journey 1, Step 1

**REQ-F-003**: Configurable Regex Pattern Matching
- **Description**: System must support configurable regex patterns for epic/feature/task folder and file name matching with named capture groups
- **User Story**: As a Product Manager, I want to define custom patterns for my documentation structure so that shark adapts to my existing conventions
- **Acceptance Criteria**:
  - [ ] Read patterns from .sharkconfig.json with separate regex for epic/feature/task folders and files
  - [ ] Support named capture groups: epic_id, epic_slug, feature_id, feature_slug, task_id, task_slug, number
  - [ ] Ship with comprehensive default patterns matching standard conventions (E##-slug) and common variations (tech-debt, bugs, change-cards)
  - [ ] Validate patterns on config load: ensure required capture groups present, valid regex syntax
  - [ ] Extract components from matched patterns to build database keys and metadata
  - [ ] Support multiple patterns per entity type (array), try in order until first match
  - [ ] Support adding new patterns mid-project (incremental import with new patterns)
- **Related Journey**: Journey 3, Steps 2-3; Journey 1, Step 3

**REQ-F-004**: Conflict Resolution Strategy
- **Description**: When epic-index.md and folder structure conflict, system must apply configured precedence strategy
- **User Story**: As a Technical Lead, I want to control which source wins during conflicts so that I can trust the import results
- **Acceptance Criteria**:
  - [ ] Support "index-precedence" strategy (default): epic-index.md is source of truth
  - [ ] Support "folder-precedence" strategy: folder structure overrides index
  - [ ] Support "merge" strategy: combine information from both sources
  - [ ] Detect conflicts: epics in index but not folders, epics in folders but not index
  - [ ] Log all conflicts with resolution applied
  - [ ] Provide --detect-conflicts flag for dry-run conflict reporting
- **Related Journey**: Journey 4, Steps 1-4

#### Feature File Discovery

**REQ-F-005**: Feature File Pattern Matching
- **Description**: System must use configurable regex patterns to recognize feature files within feature folders
- **User Story**: As a Technical Lead, I want to import features regardless of naming convention so that I don't have to rename 100+ files
- **Acceptance Criteria**:
  - [ ] Apply patterns.feature.file regex to files within feature folders
  - [ ] Default patterns recognize: prd.md, PRD_F##-descriptive-name.md, {feature-slug}.md
  - [ ] Support pattern ordering (first match wins), prioritize prd.md pattern first in defaults
  - [ ] Extract feature_id, feature_slug from named capture groups when present
  - [ ] Extract feature title from frontmatter (title:) or first H1 heading
  - [ ] Extract feature description from frontmatter (description:) or first paragraph
  - [ ] Allow users to add custom patterns for their conventions
- **Related Journey**: Journey 1, Step 1

**REQ-F-006**: Related Document Cataloging
- **Description**: System must identify and catalog related documents within feature folders (architecture, design specs)
- **User Story**: As a Product Manager, I want to track all documents related to a feature so that I can quickly find relevant context
- **Acceptance Criteria**:
  - [ ] Detect numbered design documents (02-architecture.md, 03-database-design.md, etc.)
  - [ ] Store related document paths in feature.related_docs JSON column
  - [ ] Detect documents without strict naming (any .md file in feature folder)
  - [ ] Exclude task files from related docs (files in tasks/ subfolder)
  - [ ] Support querying related docs via CLI (shark feature get {key} --show-docs)
- **Related Journey**: Journey 1, Step 5

#### Task File Discovery

**REQ-F-007**: Task File Pattern Matching
- **Description**: System must use configurable regex patterns to recognize task files within feature folders
- **User Story**: As an AI Agent, I want my generated task files to be automatically imported even if naming is slightly non-standard
- **Acceptance Criteria**:
  - [ ] Apply patterns.task.file regex to files in tasks/ or prps/ subfolders
  - [ ] Default patterns recognize: T-E##-F##-###.md, T-E##-F##-###-slug.md, ###-task-name.md, task-name.prp.md
  - [ ] Support pattern ordering (first match wins), prioritize full task key pattern first
  - [ ] Extract task_id (full T-E##-F##-###), epic_id, feature_id, number from named capture groups
  - [ ] Extract task key from frontmatter task_key: field if pattern matching fails
  - [ ] Generate task key if missing and epic/feature can be inferred from path
  - [ ] Allow users to add custom patterns for their task naming conventions
- **Related Journey**: Journey 2, Step 2; internal/sync/engine.go:150-175

**REQ-F-008**: Task Metadata Extraction
- **Description**: System must extract task title and description from multiple sources with fallback priority
- **User Story**: As a Technical Lead, I want task titles to be populated automatically so that I don't have to manually enter metadata
- **Acceptance Criteria**:
  - [ ] Priority 1: Extract title from frontmatter title: field
  - [ ] Priority 2: Extract title from filename descriptive part (T-E##-F##-###-descriptive-name.md)
  - [ ] Priority 3: Extract title from first H1 heading, removing "Task:" or "PRP:" prefixes
  - [ ] Extract description from frontmatter description: field
  - [ ] Validate title is non-empty before import
  - [ ] Log warning if no title can be extracted
- **Related Journey**: Journey 1, Step 2; internal/sync/engine.go:182-191

#### Incremental Sync

**REQ-F-009**: Modification Time Tracking
- **Description**: System must track file modification times to enable incremental sync
- **User Story**: As an AI Agent, I want sync to only process changed files so that session overhead is minimized
- **Acceptance Criteria**:
  - [ ] Record last_sync_time in .sharkconfig.json after each successful sync
  - [ ] Compare file mtime against last_sync_time to identify changed files
  - [ ] Support --incremental flag to enable modification-based filtering
  - [ ] Default to full scan when last_sync_time is not available
  - [ ] Handle clock skew gracefully (allow small negative time differences)
- **Related Journey**: Journey 2, Steps 1-2

**REQ-F-010**: Conflict Detection for Updates
- **Description**: System must detect when both file and database have changed since last sync
- **User Story**: As a Product Manager, I want to know when manual file edits conflict with database changes so that I can decide which to keep
- **Acceptance Criteria**:
  - [ ] Compare file mtime with database updated_at timestamp
  - [ ] Detect conflicts: both modified since last_sync_time
  - [ ] Apply configured resolution strategy (file-wins, db-wins, manual)
  - [ ] Log all conflicts with old values, new values, and resolution applied
  - [ ] Support --dry-run to report conflicts without applying changes
- **Related Journey**: Journey 2, Alt Path A

#### Configuration & Validation

**REQ-F-011**: Configurable Documentation Root
- **Description**: System must support per-project configuration of documentation root directory
- **User Story**: As a Technical Lead, I want to specify where documentation lives so that shark works with non-standard project structures
- **Acceptance Criteria**:
  - [ ] Read docs_root from .sharkconfig.json (default: docs/plan)
  - [ ] Support absolute and relative paths
  - [ ] Validate docs_root exists and is a directory before scanning
  - [ ] Use docs_root as base for all file path resolution
  - [ ] Support --path flag to override config for single scan
- **Related Journey**: Journey 1, Step 1; Journey 3, Step 4

**REQ-F-012**: Validation Strictness Levels
- **Description**: System must support configurable validation strictness for different project needs
- **User Story**: As a Technical Lead, I want balanced validation (strict IDs, flexible metadata) so that I maximize import success without sacrificing data integrity
- **Acceptance Criteria**:
  - [ ] Support validation_level in config: "strict", "balanced" (default), "permissive"
  - [ ] Strict mode: Require exact naming conventions, fail on any deviation, validate patterns match expected structure
  - [ ] Balanced mode: Strict epic/feature/task keys via pattern matching, flexible titles/descriptions
  - [ ] Permissive mode: Accept any reasonable interpretation, maximize import success, minimal pattern validation
  - [ ] Validate patterns on config load: regex syntax, required capture groups for relationship inference
  - [ ] Log validation level used during scan
  - [ ] Provide per-scan override via --validation-level flag
- **Related Journey**: Journey 1, Step 2

### Should Have Requirements

#### Pattern Management

**REQ-F-013**: Pattern Validation on Config Load
- **Description**: System should validate regex patterns when loading .sharkconfig.json to catch errors early
- **User Story**: As a Technical Lead, I want to know immediately if my custom patterns are invalid so that I can fix them before scanning
- **Acceptance Criteria**:
  - [ ] Validate regex syntax for all configured patterns
  - [ ] Ensure epic patterns include epic_id or epic_slug capture group
  - [ ] Ensure feature patterns include epic_id/epic_num AND feature_id/feature_slug capture groups
  - [ ] Ensure task patterns include epic_id, feature_id, AND task_id or number capture groups
  - [ ] Report specific validation errors with pattern name and missing/invalid groups
  - [ ] Allow --skip-pattern-validation flag for advanced users (at their own risk)
- **Related Journey**: Journey 3, Step 3

**REQ-F-014**: Generation Format Configuration
- **Description**: System should use configured generation formats when creating new epic/feature/task files via CLI
- **User Story**: As a Product Manager, I want new files created by shark to follow my team's conventions so that consistency is maintained
- **Acceptance Criteria**:
  - [ ] Read patterns.epic.generation.format, patterns.feature.generation.format, patterns.task.generation.format from config
  - [ ] Default generation formats: "E{number:02d}-{slug}", "E{epic:02d}-F{number:02d}-{slug}", "T-E{epic:02d}-F{feature:02d}-{number:03d}.md"
  - [ ] Support placeholders: {number}, {slug}, {epic}, {feature}, with optional formatting (02d = zero-padded 2 digits)
  - [ ] Apply generation format when creating new items via `shark epic create`, `shark feature create`, `shark task create`
  - [ ] Log generation format used for transparency
- **Related Journey**: Journey 3, Step 5

**REQ-F-015**: Pattern Preset Library
- **Description**: System should provide preset patterns for common documentation styles
- **User Story**: As a Technical Lead, I want to use preset patterns for common styles so that I don't have to write regex from scratch
- **Acceptance Criteria**:
  - [ ] Implement `shark config add-pattern --preset=<name>` command
  - [ ] Provide presets: "standard" (E##-slug), "special-epics" (tech-debt, bugs, change-cards), "numeric-only" (E001, F001), "legacy-prp" (prp files in prps/ folder)
  - [ ] Presets append patterns to existing config (don't replace)
  - [ ] Document presets in CLI help and documentation
  - [ ] Allow viewing preset patterns via `shark config show-preset <name>`
- **Related Journey**: Journey 1, Step 3; Journey 3, Step 3

#### Import Reporting

**REQ-F-016**: Detailed Scan Report
- **Description**: System should provide comprehensive report of scan results with actionable warnings
- **User Story**: As a Technical Lead, I want to see exactly what was imported and what was skipped so that I can verify correctness
- **Acceptance Criteria**:
  - [ ] Report total files scanned, matched, skipped
  - [ ] Break down by type: epics, features, tasks, related docs
  - [ ] List skipped files with reason for each (pattern mismatch, validation failure, etc.)
  - [ ] Show conflicts detected and resolution strategy applied
  - [ ] Provide file path and line number for parse errors
  - [ ] Support --output=json for programmatic processing
- **Related Journey**: Journey 1, Steps 1-2

**REQ-F-017**: Import Validation Command
- **Description**: System should provide command to validate import integrity after scan
- **User Story**: As a Technical Lead, I want to verify file paths and relationships after import so that I can catch any corruption
- **Acceptance Criteria**:
  - [ ] Implement `shark validate` command
  - [ ] Check all file_path entries point to existing files
  - [ ] Validate epic/feature/task relationships (foreign keys)
  - [ ] Detect orphaned records (features without epics, tasks without features)
  - [ ] Report broken references with specific IDs and paths
  - [ ] Suggest corrective actions (re-scan, manual fix)
- **Related Journey**: Journey 1, Step 6

#### Pattern Matching

**REQ-F-018**: Fuzzy Task Key Generation
- **Description**: System should generate task keys for PRP files lacking task_key frontmatter
- **User Story**: As an AI Agent, I want task keys generated automatically for new files so that I don't have to query database for next sequence number
- **Acceptance Criteria**:
  - [ ] Infer epic and feature from file path
  - [ ] Query database for next available task sequence number for that feature
  - [ ] Generate task key in T-E##-F##-### format
  - [ ] Write generated key back to file frontmatter
  - [ ] Fall back to in-memory-only key if frontmatter write fails
  - [ ] Log all generated keys for review
- **Related Journey**: Journey 2, Step 2; internal/sync/engine.go:150-175

**REQ-F-019**: Epic Title Extraction from Folder Name
- **Description**: System should generate human-readable epic titles from folder names when epic.md is absent
- **User Story**: As a Product Manager, I want epics to have readable titles even when documentation is incomplete so that CLI output is understandable
- **Acceptance Criteria**:
  - [ ] Convert epic-slug to Title Case (e.g., task-mgmt-cli-core → Task Mgmt CLI Core)
  - [ ] Expand common abbreviations (cli → CLI, api → API)
  - [ ] Mark auto-generated titles with prefix: "Auto: {title}"
  - [ ] Allow manual override via epic.md title: frontmatter
  - [ ] Log auto-generated titles for review
- **Related Journey**: Journey 1, Step 5

### Could Have Requirements

#### Advanced Scanning

**REQ-F-020**: Multi-Root Documentation Support
- **Description**: System could support scanning multiple documentation roots (e.g., docs/plan and docs/archived)
- **User Story**: As a Technical Lead, I want to import from multiple locations so that I can include archived epics
- **Acceptance Criteria**:
  - [ ] Accept multiple --path arguments
  - [ ] Scan each root independently
  - [ ] Merge results with conflict detection
  - [ ] Preserve source root in file_path for disambiguation
- **Related Journey**: Journey 1, Step 1 (extension)

**REQ-F-021**: Change Detection via Git
- **Description**: System could use git status to identify changed files instead of file mtime
- **User Story**: As an AI Agent, I want sync to use git tracking so that results are more accurate in version-controlled projects
- **Acceptance Criteria**:
  - [ ] Detect if working directory is a git repository
  - [ ] Use `git diff --name-only` to identify modified files since last commit
  - [ ] Fall back to mtime if not a git repository
  - [ ] Support --use-git flag to force git-based detection
- **Related Journey**: Journey 2, Step 1 (enhancement)

### Won't Have Requirements

See [scope.md](./scope.md) for explicitly excluded features.

---

## Non-Functional Requirements

### Performance

**REQ-NF-001**: Incremental Sync Speed
- **Description**: Incremental sync must complete in <5 seconds for 100 file changes
- **Measurement**: Time from `shark sync --incremental` command to completion
- **Target**:
  - <2 seconds for 1-10 file changes (typical AI agent session)
  - <5 seconds for 100 file changes (bulk updates)
  - <30 seconds for full project scan (500+ files)
- **Justification**: Agent sessions have limited time; sync cannot be a bottleneck

**REQ-NF-002**: Memory Efficiency
- **Description**: Scanner must operate within reasonable memory limits for large projects
- **Measurement**: Peak memory usage during scan
- **Target**: <100MB heap for scanning 1,000 files
- **Justification**: CLI tool should work on resource-constrained environments

### Reliability

**REQ-NF-003**: Transactional Import
- **Description**: All database modifications during scan must be atomic (all-or-nothing)
- **Implementation**: Wrap entire scan in database transaction with rollback on any error
- **Testing**: Simulate failures at various scan stages, verify no partial imports
- **Justification**: Prevents database corruption from interrupted scans

**REQ-NF-004**: Idempotent Scans
- **Description**: Running scan multiple times on same files must produce identical database state
- **Implementation**: Use upsert logic (insert or update based on key) for all entities
- **Testing**: Run scan 3 times, verify database state identical after each run
- **Justification**: Allows safe re-scanning without manual cleanup

**REQ-NF-005**: Error Recovery
- **Description**: Parser failures on individual files must not halt entire scan
- **Implementation**: Wrap file parsing in error handler, log error, continue with next file
- **Testing**: Inject invalid markdown files, verify scan completes with warnings
- **Justification**: One corrupted file shouldn't block importing 99 valid files

### Usability

**REQ-NF-006**: Actionable Error Messages
- **Description**: All errors and warnings must include file path, line number (if applicable), and suggested fix
- **Implementation**: Structured error types with context fields
- **Example**: "Cannot parse frontmatter in docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/tasks/T-E04-F07-003.md:5: missing closing '---'. Suggestion: Add '---' on line 8 to close frontmatter."
- **Justification**: Users need to fix issues quickly without trial-and-error

**REQ-NF-007**: Dry-Run Mode
- **Description**: All scan commands must support --dry-run flag for preview without database changes
- **Implementation**: Skip transaction commit in dry-run mode, generate full report as if committed
- **Testing**: Run scan with --dry-run, verify database unchanged but report complete
- **Justification**: Users need confidence before importing 200+ files

### Maintainability

**REQ-NF-008**: Pattern Extensibility
- **Description**: Adding new file patterns must require only configuration changes, no code modifications
- **Implementation**: Regex pattern definitions in .sharkconfig.json with named capture groups, pattern-matching engine generic and data-driven
- **Testing**: Add new pattern via config-only change, verify recognition and component extraction
- **Justification**: Future-proofs scanner for new documentation conventions, enables user customization without code changes

**REQ-NF-009**: Test Coverage
- **Description**: Scanner logic must have >80% unit test coverage
- **Implementation**: Go testing package with table-driven tests for pattern matching
- **Testing**: CI pipeline enforces coverage threshold
- **Justification**: Pattern matching is complex; tests prevent regressions

---

*See also*: [Success Metrics](./success-metrics.md), [Scope](./scope.md)
