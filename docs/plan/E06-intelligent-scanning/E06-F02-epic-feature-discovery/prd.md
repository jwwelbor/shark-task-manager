# Feature: Epic & Feature Discovery Engine

## Epic

- [Epic PRD](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/epic.md)
- [Epic Requirements](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/requirements.md)
- [Epic User Journeys](/home/jwwelbor/projects/shark-task-manager/docs/plan/E06-intelligent-scanning/user-journeys.md)

## Feature Name

Epic & Feature Discovery Engine

## Goal

### Problem

When importing existing project documentation into shark, the current system assumes a rigid folder structure with exact naming conventions (E##-epic-slug, E##-F##-feature-slug). This fails to accommodate real-world documentation that has evolved organically over months or years. Projects may have:

- Epic folders using descriptive names (tech-debt, bugs, notifications, identity) instead of E## prefixes
- Mixed epic naming conventions where some use E## and others use semantic names
- Epic-index.md files that explicitly define project structure with markdown links, but current sync ignores this metadata
- Feature folders with varied naming patterns (descriptive names, different number formats, legacy patterns)
- Multiple related documents within feature folders (architecture.md, 02-database-design.md, security-notes.md) that provide critical context but aren't cataloged
- Conflicts between what epic-index.md declares and what folder structure shows (obsolete folders, work-in-progress epics not yet documented)

Technical Leads attempting to migrate 200+ existing files face a choice: spend hours manually restructuring documentation to match shark's expectations, or lose the ability to use shark's query and tracking capabilities. Neither option scales for enterprise projects with extensive documentation histories.

The current sync engine (internal/sync/engine.go) handles task-level synchronization effectively but lacks the intelligence to discover and catalog epic/feature hierarchies from varied documentation patterns. There's no mechanism to leverage epic-index.md as an explicit source of truth, no fallback strategy when the index is absent, and no way to resolve conflicts between declared structure and actual folder organization.

### Solution

Implement an intelligent Epic & Feature Discovery Engine that adapts to existing documentation patterns rather than enforcing rigid conventions. The engine will:

**1. Dual-Source Discovery with Precedence Strategy**
- Parse epic-index.md (if present) as the primary, explicit source of truth for project structure
- Extract epic and feature keys from markdown links: `[Epic Name](./E##-epic-slug/)`, `[Feature Name](./E##-epic-slug/E##-F##-feature-slug/)`
- Fall back to folder structure scanning when epic-index.md is absent or incomplete
- Apply configurable conflict resolution when index and folders disagree (default: index-precedence)
- Merge information from both sources to build comprehensive epic/feature catalog

**2. Flexible Pattern Recognition**
- Consume regex patterns from F01 configuration (.sharkconfig.json patterns.epic.folder, patterns.feature.folder)
- Use named capture groups to extract components: epic_id, epic_slug, feature_id, feature_slug
- Support multiple patterns per entity type (try in order, first match wins)
- Recognize standard E## conventions alongside special types (tech-debt, bugs, change-cards)
- Build epic/feature hierarchy from path structure and captured components

**3. Related Document Cataloging**
- Identify non-PRD documents within feature folders using glob patterns (02-*.md, 03-*.md, architecture.md, etc.)
- Store document paths in feature.related_docs JSON column for quick reference
- Exclude task files (in tasks/ subfolder) from related docs catalog
- Enable querying related docs via CLI: `shark feature get E04-F07 --show-docs`

**4. Conflict Detection and Resolution**
- Detect epics listed in epic-index.md but absent from folder structure
- Detect epic folders not mentioned in epic-index.md (potential obsolete or work-in-progress)
- Apply configured precedence strategy: index-precedence (default), folder-precedence, or merge
- Generate detailed conflict reports with warnings and resolution actions taken
- Support --detect-conflicts dry-run flag for pre-import analysis

**5. Validation and Error Reporting**
- Apply validation strictness levels from F01 configuration (strict, balanced, permissive)
- Validate epic/feature relationships (features must belong to existing epics)
- Report broken references, invalid patterns, and missing required metadata
- Provide actionable error messages with file paths and suggested fixes

### Impact

- **Migration Time Reduction**: Reduce existing project onboarding from 4+ hours (manual restructuring) to 10-15 minutes (intelligent discovery with conflict review)
- **Documentation Flexibility**: Support 90% of existing documentation patterns without requiring file restructuring or renaming
- **Discovery Performance**: Complete epic/feature discovery for 100+ folders in <5 seconds, enabling fast incremental scans
- **Data Accuracy**: Maintain 100% consistency between documentation hierarchy and database with explicit conflict resolution strategies
- **Catalog Completeness**: Track 100% of related documents within features, providing instant access to architecture and design context
- **Developer Confidence**: Dry-run mode with detailed conflict reports enables safe preview before database modification

## User Personas

### Primary Persona: Technical Lead (Existing Project Migration)

**Reference**: [E06 Persona 1](../personas.md#persona-1-technical-lead-existing-project-migration)

**Role**: Technical Lead or Engineering Manager migrating multi-epic project (100-300 markdown files) to shark

**Experience Level**: 5+ years in role, high technical proficiency with CLI tools and Git

**Key Characteristics**:
- Manages 3-5 concurrent epics with varied documentation styles
- Documentation evolved organically over 6-18 months with changing team standards
- Has epic-index.md defining project structure but some folders don't match index
- Limited time for migration (needs "works out of the box" with minimal configuration)
- Risk-averse: requires validation, conflict reports, and rollback capabilities

**Goals Related to This Feature**:
1. Import all epics and features from existing documentation without manual restructuring
2. Leverage epic-index.md as source of truth while handling folder discrepancies
3. Understand what was imported vs. skipped with detailed conflict reports
4. Catalog all design documents and architecture files for easy reference
5. Validate discovery results before committing to database

**Pain Points This Feature Addresses**:
- Current sync doesn't recognize epic-index.md, forcing manual folder-based import only
- Special epic types (tech-debt, bugs) are ignored because they lack E## prefix
- No visibility into conflicts between documented structure (index) and actual folders
- Related documents (architecture.md, design specs) are not cataloged or easily queryable
- No dry-run mode to preview discovery results before database modification

**Success Looks Like**:
Run `shark scan --dry-run --detect-conflicts` and receive report showing 15 epics discovered (12 from index, 3 folder-only), 87 features found, 2 conflicts detected (E04 and E05 in folders but not in index). Review conflict report, decide E04/E05 are obsolete WIP folders, run `shark scan --execute` with index-precedence. Database now contains accurate project structure with 13 epics (one special type: tech-debt) and 87 features, all related docs cataloged. Total time: 12 minutes including review.

### Secondary Persona: Product Manager (Multi-Style Documentation)

**Reference**: [E06 Persona 3](../personas.md#persona-3-product-manager-multi-style-documentation)

**Role**: Product Manager managing projects across multiple teams with different documentation conventions

**Experience Level**: 3-5 years in role, moderate technical proficiency, comfortable with markdown and Git

**Key Characteristics**:
- Works with documentation from different teams/time periods with varied conventions
- Some teams use E## epics, others use semantic names (identity, notifications, tech-debt)
- Has epic-index.md but some teams haven't updated it with new features
- Values flexibility over strict standardization
- Occasionally hand-edits markdown files directly

**Goals Related to This Feature**:
1. Discover and import special epic types (tech-debt, bugs, change-cards) alongside standard E## epics
2. Handle features that use team-specific naming conventions (non-standard patterns)
3. Merge information from epic-index.md with actual folder structure
4. Track all related documents within features for context during status reporting
5. Generate accurate progress reports across mixed documentation styles

**Pain Points This Feature Addresses**:
- Cannot track tech-debt or bug epics because current system only recognizes E## format
- Epic-index.md lists some features but folders contain additional unlisted features (incomplete index)
- No merge strategy to combine index metadata with folder discoveries
- Related architecture documents are scattered and hard to find when preparing reports

**Success Looks Like**:
Configure .sharkconfig.json with merge conflict strategy and special epic patterns. Run `shark scan --strategy=merge` and system discovers 8 epics (5 from index + 3 from folders including tech-debt), 42 features (35 from index, 7 from folders). All related docs cataloged. Can now run `shark epic list` and see unified view including tech-debt epic. Weekly status reports combine data from standardized and special epic types seamlessly. Run `shark feature get E04-F07 --show-docs` and instantly see list of 5 related design documents.

### Tertiary Persona: AI Agent (Incremental Development Sync)

**Reference**: [E06 Persona 2](../personas.md#persona-2-ai-agent-incremental-development-sync)

**Role**: Autonomous code generation and task execution agent (Claude Code)

**Experience Level**: Stateless between sessions, relies on explicit project state, optimizes for token efficiency

**Key Characteristics**:
- Creates new feature folders during epic implementation
- Updates epic-index.md when creating features (if instructed by workflow)
- Cannot manually validate or restructure documentation
- Needs fast discovery to maintain database consistency between sessions
- Works within configured patterns, cannot debug pattern mismatches independently

**Goals Related to This Feature**:
1. Automatically detect new feature folders created during work session
2. Sync epic/feature hierarchy changes after creating new features
3. Complete discovery in <2 seconds for incremental scans (modified folders only)
4. Receive clear errors if created folder doesn't match expected patterns

**Pain Points This Feature Addresses**:
- Current sync is slow (rescans entire tree) and only handles tasks, not epic/feature structure
- Agent-generated folders may have slight variations from ideal format (needs flexible parsing)
- No incremental discovery (must rescan all folders even if only one changed)
- Sync failures are opaque (agent doesn't understand pattern mismatch errors)

**Success Looks Like**:
Agent creates new feature folder `E04-F09-recommended-improvements/` with prd.md during session. Updates epic-index.md to add feature link. Runs `shark sync` before ending session. Discovery engine detects new feature from index, validates folder structure matches, catalogs prd.md and related docs. Sync completes in 1.8 seconds, reports 1 new feature discovered. Next agent session has accurate epic/feature structure without re-reading all markdown files.

## User Stories

### Must-Have User Stories

**Story 1: Parse epic-index.md for Explicit Structure**
- As a Technical Lead, I want epic-index.md to be parsed for epic and feature links so that my explicitly documented project structure is the primary source of truth.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-001
- **Related Journey**: Journey 1 (Alt Path A), Journey 4

**Story 2: Discover Epics from Folder Structure**
- As a Technical Lead, I want epics discovered from folder structure when epic-index.md is absent so that I can import legacy projects without creating an index first.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-002
- **Related Journey**: Journey 1

**Story 3: Apply Configurable Regex Patterns**
- As a Product Manager, I want epic and feature discovery to use configurable regex patterns from .sharkconfig.json so that shark adapts to my team's documentation conventions.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-003
- **Related Journey**: Journey 3

**Story 4: Resolve Conflicts Between Index and Folders**
- As a Technical Lead, I want conflicts between epic-index.md and folder structure to be detected and resolved using my chosen strategy so that I control which source wins.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-004
- **Related Journey**: Journey 4

**Story 5: Recognize Feature Files with Pattern Matching**
- As a Technical Lead, I want feature files (prd.md, PRD_F##-name.md) to be recognized using configurable patterns so that I don't have to rename 100+ files.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-005
- **Related Journey**: Journey 1

**Story 6: Catalog Related Documents**
- As a Product Manager, I want all related documents within feature folders (architecture.md, design docs) to be cataloged so that I can quickly find relevant context.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-006
- **Related Journey**: Journey 1, Step 5

**Story 7: Extract Epic Metadata from Multiple Sources**
- As a Technical Lead, I want epic titles and descriptions extracted from epic.md, epic-index.md links, or folder names (with fallback priority) so that metadata is populated automatically.
- **Priority**: Must-Have
- **Related Requirement**: REQ-F-019
- **Related Journey**: Journey 1, Step 5

**Story 8: Validate Epic/Feature Relationships**
- As a Technical Lead, I want discovery to validate that features belong to existing epics so that orphaned features are not created.
- **Priority**: Must-Have
- **Related Requirement**: Implied by REQ-F-002, REQ-F-004
- **Related Journey**: Journey 1, Step 6

### Should-Have User Stories

**Story 9: Generate Detailed Discovery Report**
- As a Technical Lead, I want a comprehensive discovery report showing epics found, features discovered, conflicts detected, and files skipped so that I can verify import correctness.
- **Priority**: Should-Have
- **Related Requirement**: REQ-F-016
- **Related Journey**: Journey 1, Steps 1-2

**Story 10: Preview Conflicts Before Import**
- As a Technical Lead, I want to run `shark scan --dry-run --detect-conflicts` to preview conflicts without modifying the database so that I can plan resolution strategy.
- **Priority**: Should-Have
- **Related Requirement**: REQ-F-004, REQ-NF-007
- **Related Journey**: Journey 4, Step 1

**Story 11: Query Related Documents via CLI**
- As a Product Manager, I want to run `shark feature get E04-F07 --show-docs` to see all related documents for a feature so that I can access architecture context quickly.
- **Priority**: Should-Have
- **Related Requirement**: REQ-F-006
- **Related Journey**: Journey 1, Step 5 (extension)

**Story 12: Support Multiple Conflict Resolution Strategies**
- As a Product Manager, I want to choose between index-precedence, folder-precedence, or merge strategies so that I control how conflicts are resolved.
- **Priority**: Should-Have
- **Related Requirement**: REQ-F-004
- **Related Journey**: Journey 4, Alt Path B

**Story 13: Extract Feature Metadata from Files and Paths**
- As a Technical Lead, I want feature titles and descriptions extracted from prd.md frontmatter, first H1 heading, or folder names so that metadata is populated without manual entry.
- **Priority**: Should-Have
- **Related Requirement**: REQ-F-005
- **Related Journey**: Journey 1, Step 1

### Could-Have User Stories

**Story 14: Validate Pattern Configuration on Load**
- As a Technical Lead, I want regex patterns validated when loading .sharkconfig.json so that I know immediately if my custom patterns are invalid.
- **Priority**: Could-Have
- **Related Requirement**: REQ-F-013
- **Related Journey**: Journey 3, Step 4

**Story 15: Support Incremental Epic/Feature Discovery**
- As an AI Agent, I want discovery to only scan modified epic/feature folders (based on mtime) so that incremental sync is fast.
- **Priority**: Could-Have
- **Related Requirement**: REQ-F-009 (applies to epic/feature discovery)
- **Related Journey**: Journey 2, Step 1

**Story 16: Auto-Create Missing Epics During Feature Import**
- As a Technical Lead, I want missing epics to be auto-created (with minimal metadata) when features reference them so that I can import features before manually creating parent epics.
- **Priority**: Could-Have
- **Related Requirement**: Similar to REQ-F-018 (task-level fuzzy key generation)
- **Related Journey**: Not explicitly covered

## Requirements

### Functional Requirements

#### Epic Discovery (REQ-F-001, REQ-F-002)

**FR-001: epic-index.md Parsing**
The system must parse `docs/plan/epic-index.md` to extract epic keys, titles, and feature links as the primary source of project structure.

- Parse markdown links in format `[Epic Name](./E##-epic-slug/)` to extract epic keys
- Parse markdown links in format `[Feature Name](./E##-epic-slug/E##-F##-feature-slug/)` to extract feature keys
- Extract epic titles from link text (e.g., "Epic Name" from `[Epic Name](./path/)`)
- Handle relative paths (./E##-epic-slug/) and absolute paths (/docs/plan/E##-epic-slug/)
- Support nested list structures (epics as top-level list items, features as nested items)
- Log warnings for broken links (path doesn't exist on filesystem)
- Log warnings for invalid link formats (missing parentheses, malformed paths)

**FR-002: Folder Structure Fallback Scanning**
When epic-index.md is absent or incomplete, the system must discover epics by recursively scanning the configured documentation root (default: docs/plan/).

- Recursively walk configured root directory (from .sharkconfig.json docs_root)
- Apply patterns.epic.folder regex to directory names
- Extract epic_id and epic_slug from named capture groups
- Recognize E##-epic-slug folders as standard epics
- Recognize special epic types based on configured patterns (tech-debt, bugs, change-cards)
- Build epic list from matched folders
- Skip hidden directories (starting with .)
- Skip non-directory files during epic discovery

**FR-003: Epic Metadata Extraction**
The system must extract epic metadata from multiple sources with fallback priority.

- **Priority 1**: Extract title from epic-index.md link text (if epic listed in index)
- **Priority 2**: Extract title and description from `docs/plan/{epic-key}/epic.md` frontmatter (title:, description:)
- **Priority 3**: Extract title from first H1 heading in epic.md
- **Priority 4**: Generate title from folder name (convert epic-slug to Title Case, expand abbreviations like cli → CLI)
- Mark auto-generated titles with prefix: "Auto: {title}" for transparency
- Store file_path to epic.md (if exists) for reference
- Validate epic_key is non-empty before import

#### Feature Discovery (REQ-F-005, REQ-F-006)

**FR-004: Feature Folder Pattern Matching**
The system must discover feature folders within epic directories using configurable regex patterns.

- Scan subdirectories within each discovered epic folder
- Apply patterns.feature.folder regex to directory names
- Extract epic_id, epic_num, feature_id, feature_slug from named capture groups
- Validate extracted epic_id matches parent epic folder
- Support multiple pattern attempts (first match wins)
- Default patterns recognize: E##-F##-feature-slug, F##-feature-slug (infer epic from parent)
- Log warnings for folders that match neither epic nor feature patterns
- Build epic → feature relationship hierarchy from path structure

**FR-005: Feature File Pattern Matching**
The system must recognize feature PRD files within feature folders using configurable regex patterns.

- Apply patterns.feature.file regex to files within feature folders
- Default patterns recognize: prd.md, PRD_F##-descriptive-name.md, {feature-slug}.md
- Support pattern ordering (first match wins)
- Prioritize prd.md pattern first in default configuration
- Extract feature_id, feature_slug from named capture groups (if pattern includes them)
- Validate at most one PRD file matches per feature folder
- Log warnings if multiple PRD files found (ambiguous which to use)
- Log warnings if no PRD file found but feature folder exists

**FR-006: Feature Metadata Extraction**
The system must extract feature metadata from multiple sources with fallback priority.

- **Priority 1**: Extract title from epic-index.md link text (if feature listed in index)
- **Priority 2**: Extract title and description from PRD file frontmatter (title:, description:)
- **Priority 3**: Extract title from first H1 heading in PRD file
- **Priority 4**: Generate title from folder name (convert feature-slug to Title Case)
- Store file_path to PRD file for reference
- Validate feature_key is non-empty before import
- Validate feature belongs to valid parent epic

**FR-007: Related Document Cataloging**
The system must identify and catalog all related documents within feature folders for quick reference.

- Scan all .md files within feature folder (excluding tasks/ subfolder)
- Recognize numbered design documents using glob patterns: 02-*.md, 03-*.md, 04-*.md, etc.
- Recognize common document names: architecture.md, database-design.md, security-design.md, test-criteria.md
- Detect any additional .md files not matching PRD or task patterns
- Exclude PRD file itself from related docs list
- Exclude task files (in tasks/ or prps/ subfolders) from related docs
- Store document paths relative to project root in feature.related_docs JSON column
- Support querying related docs via `shark feature get {key} --show-docs` CLI command
- Report count of related documents in discovery report

#### Conflict Detection and Resolution (REQ-F-004)

**FR-008: Index vs. Folder Conflict Detection**
The system must detect conflicts between epic-index.md declarations and actual folder structure.

- **Epic Conflicts**:
  - Detect epics listed in index but folder doesn't exist (broken reference)
  - Detect epic folders not mentioned in index (undocumented or obsolete)
- **Feature Conflicts**:
  - Detect features listed in index but folder doesn't exist
  - Detect feature folders not mentioned in index
  - Detect features in index with wrong parent epic (index says E04, folder is in E05/)
- Log all conflicts with specific paths and keys
- Generate conflict report showing: index-only items, folder-only items, mismatched relationships

**FR-009: Conflict Resolution Strategies**
The system must apply configurable precedence strategy when index and folder structure conflict.

- **index-precedence strategy (default)**:
  - Epic-index.md is source of truth for which epics/features to import
  - Ignore folders not mentioned in index (log as warnings)
  - Fail on broken index references (epic in index but folder missing)
- **folder-precedence strategy**:
  - Folder structure is source of truth
  - Ignore index entries without matching folders
  - Import all discovered folders even if not in index
- **merge strategy**:
  - Import epics/features from BOTH index and folders
  - Use index metadata (titles from link text) when available
  - Fall back to folder-based metadata for items not in index
  - Resolve metadata conflicts: index-provided metadata wins
- Support --strategy flag to override config for single scan
- Log resolution strategy applied for each conflict

**FR-010: Conflict Reporting**
The system must provide detailed conflict reports showing what was found, what conflicts occurred, and how they were resolved.

- Report format:
  ```
  Conflict: Epic E04 found in folders but not in epic-index.md
    Path: docs/plan/E04-task-mgmt-cli-core/
    Resolution: Skipped (index-precedence strategy)
    Suggestion: Add E04 to epic-index.md or use merge strategy
  ```
- Group conflicts by type: broken references, undocumented folders, relationship mismatches
- Provide actionable suggestions for each conflict type
- Support --detect-conflicts flag for dry-run conflict-only analysis (no import)

#### Pattern Configuration (REQ-F-003)

**FR-011: Regex Pattern Application**
The system must use regex patterns from .sharkconfig.json to match epic and feature folders/files.

- Read patterns.epic.folder from config (array of regex strings)
- Read patterns.feature.folder from config (array of regex strings)
- Read patterns.feature.file from config (array of regex strings)
- Try patterns in order, use first match
- Extract components using named capture groups: (?P<epic_id>...), (?P<epic_slug>...), etc.
- Required epic capture groups: epic_id OR epic_slug (at least one)
- Required feature capture groups: feature_id OR feature_slug (at least one), AND epic_id/epic_num (for relationship)
- Support multiple patterns per entity type to handle varied conventions
- Log which pattern matched for each discovered item (for debugging)

**FR-012: Default Pattern Definitions**
The system must ship with comprehensive default patterns matching standard conventions and common variations.

- **Default epic.folder patterns**:
  1. `(?P<epic_id>E\d{2})-(?P<epic_slug>[a-z0-9-]+)` (E##-epic-slug)
  2. `(?P<epic_id>tech-debt|bugs|change-cards)` (special types)
- **Default feature.folder patterns**:
  1. `(?P<epic_id>E(?P<epic_num>\d{2}))-(?P<feature_id>F(?P<feature_num>\d{2}))-(?P<feature_slug>[a-z0-9-]+)` (E##-F##-feature-slug)
  2. `(?P<feature_id>F(?P<feature_num>\d{2}))-(?P<feature_slug>[a-z0-9-]+)` (F##-feature-slug, infer epic from parent)
- **Default feature.file patterns**:
  1. `^prd\.md$` (prd.md - highest priority)
  2. `^PRD_(?P<feature_id>F\d{2})-(?P<feature_slug>[a-z0-9-]+)\.md$` (PRD_F##-name.md)
  3. `^(?P<feature_slug>[a-z0-9-]+)\.md$` (feature-slug.md - lowest priority, exclude if matches task patterns)

**FR-013: Pattern Component Extraction**
The system must extract epic/feature components from matched patterns to build database keys and metadata.

- Extract epic_id from epic pattern match (e.g., "E04" or "tech-debt")
- Extract epic_slug from epic pattern match (e.g., "task-mgmt-cli-core")
- Build epic_key: Use epic_id as canonical key
- Extract feature_id from feature pattern match (e.g., "F07")
- Extract epic_id/epic_num from feature pattern match to infer parent epic
- Build feature_key: Combine epic_id + feature_id (e.g., "E04-F07")
- If epic_id not in feature pattern, infer from parent folder epic_id
- Validate all extracted keys are non-empty before import

#### Validation and Configuration (REQ-F-011, REQ-F-012)

**FR-014: Configurable Documentation Root**
The system must support per-project configuration of documentation root directory.

- Read docs_root from .sharkconfig.json (default: "docs/plan")
- Support absolute paths (e.g., "/home/user/project/docs/plan")
- Support relative paths (resolved relative to project root)
- Validate docs_root exists and is a directory before scanning
- Log error and fail early if docs_root doesn't exist
- Use docs_root as base for all file path resolution
- Support --path flag to override config for single scan

**FR-015: Validation Strictness Levels**
The system must support configurable validation strictness for different project needs.

- Read validation_level from config: "strict", "balanced" (default), "permissive"
- **Strict mode**:
  - Require exact E##-F## naming conventions (reject special types unless explicitly whitelisted)
  - Fail on any pattern mismatch
  - Require PRD file for every feature folder
  - Fail on relationship mismatches (feature in wrong epic folder)
- **Balanced mode** (default):
  - Accept patterns defined in config (standard + special types)
  - Warn on pattern mismatches but continue
  - Require epic/feature keys via pattern matching
  - Allow features without PRD files (generate minimal metadata from folder name)
- **Permissive mode**:
  - Accept any reasonable folder structure
  - Maximize import success, minimal pattern validation
  - Auto-generate keys if pattern extraction fails
  - Allow orphaned features (create placeholder epics)
- Support --validation-level flag to override config for single scan

**FR-016: Epic-Index.md Format Support**
The system must support flexible markdown formats for epic-index.md.

- Support unordered lists (- or * bullet points)
- Support ordered lists (1., 2., etc.)
- Support nested lists (epics as top-level, features indented)
- Support flat lists (all epics/features at same level, infer relationships from paths)
- Support headings as epic sections (## Epic: E04 Task Management CLI Core)
- Extract links regardless of surrounding text formatting (bold, italic, etc.)
- Skip non-link list items (plain text without markdown links)
- Handle multiple links per line (extract all)

#### Database Integration

**FR-017: Epic Record Creation**
The system must create database records for discovered epics with complete metadata.

- Insert into epics table with fields: epic_key, title, description, file_path, status
- Set epic_key to extracted epic_id (e.g., "E04" or "tech-debt")
- Set title from extracted metadata (see FR-003)
- Set description from epic.md frontmatter or first paragraph
- Set file_path to epic.md location (if exists, else NULL)
- Set status to "active" by default for newly discovered epics
- Use upsert logic (INSERT OR REPLACE) to handle re-scans
- Record created_at and updated_at timestamps

**FR-018: Feature Record Creation**
The system must create database records for discovered features with complete metadata and relationships.

- Insert into features table with fields: feature_key, epic_key, title, description, file_path, related_docs, status
- Set feature_key to extracted key (e.g., "E04-F07")
- Set epic_key to parent epic (foreign key relationship)
- Set title from extracted metadata (see FR-006)
- Set description from PRD frontmatter or first paragraph
- Set file_path to PRD file location (if exists, else NULL)
- Set related_docs to JSON array of related document paths (see FR-007)
- Set status to "planning" by default for newly discovered features
- Validate epic_key references existing epic (foreign key constraint)
- Use upsert logic to handle re-scans (update if feature_key exists)
- Record created_at and updated_at timestamps

**FR-019: Transaction Safety**
All database modifications during discovery must be atomic (all-or-nothing).

- Wrap entire discovery operation in database transaction
- Commit transaction only if all epics and features imported successfully
- Rollback transaction on any database error (constraint violation, connection failure)
- Rollback transaction on user cancellation (Ctrl+C)
- Log transaction status (committed or rolled back) in discovery report

#### Discovery Reporting (REQ-F-016)

**FR-020: Comprehensive Discovery Report**
The system must provide detailed report of discovery results with actionable warnings.

- Report total folders scanned (epic folders, feature folders)
- Report total files analyzed (epic.md, prd.md, related docs)
- Break down by type:
  - Epics discovered: X (Y from index, Z from folders)
  - Features discovered: X (Y from index, Z from folders)
  - Related documents cataloged: X
- List skipped folders/files with reason for each:
  - Pattern mismatch (folder name doesn't match any configured pattern)
  - Validation failure (invalid epic/feature key format)
  - Relationship error (feature references non-existent epic)
- Show conflicts detected and resolution strategy applied
- Provide file paths and line numbers for parse errors
- Support --output=json for programmatic processing

**FR-021: JSON Output Format**
The system must support JSON output for programmatic integration.

- Structure:
  ```json
  {
    "folders_scanned": 47,
    "files_analyzed": 123,
    "epics_discovered": 15,
    "epics_from_index": 12,
    "epics_from_folders": 5,
    "features_discovered": 87,
    "features_from_index": 80,
    "features_from_folders": 10,
    "related_docs_cataloged": 234,
    "conflicts_detected": 2,
    "conflicts": [
      {
        "type": "epic_folder_only",
        "key": "E04",
        "path": "docs/plan/E04-task-mgmt-cli-core/",
        "resolution": "skipped",
        "strategy": "index-precedence"
      }
    ],
    "warnings": ["Warning: Feature E05-F03 listed in index but folder not found"],
    "errors": []
  }
  ```
- Use --json flag to enable JSON output
- Disable colored output when --json is used
- Write JSON to stdout, logs/errors to stderr

### Non-Functional Requirements

#### Performance (REQ-NF-001, REQ-NF-002)

**NFR-001: Discovery Speed**
Epic and feature discovery must complete within performance targets based on project size.

- <2 seconds for 1-20 epic/feature folders (small projects)
- <5 seconds for 100 epic/feature folders (medium projects)
- <15 seconds for 500+ epic/feature folders (large enterprise projects)
- Measured from scan command invocation to discovery report display
- Target applies to full discovery (not incremental)

**NFR-002: Memory Efficiency**
Discovery engine must operate within reasonable memory limits.

- <50MB heap for scanning 100 epic/feature folders
- <200MB heap for scanning 1,000 epic/feature folders
- Use streaming file reading (don't load entire files into memory)
- Use database batch operations for bulk inserts (not one-by-one)

**NFR-003: Incremental Discovery (Future Enhancement)**
Discovery engine should support incremental discovery based on modification time (could-have).

- Track last_discovery_time in .sharkconfig.json
- Compare folder mtime against last_discovery_time
- Skip folders not modified since last discovery
- Support --incremental flag to enable mtime-based filtering
- Default to full discovery when last_discovery_time not available

#### Reliability (REQ-NF-003, REQ-NF-004, REQ-NF-005)

**NFR-004: Transactional Discovery**
All database modifications during discovery must be atomic.

- Use database transactions (Go database/sql BEGIN/COMMIT/ROLLBACK)
- Rollback on any error: constraint violation, parse error, filesystem error
- Ensure no partial imports (if 50 features discovered but error on #30, rollback all)
- Log transaction status clearly in report

**NFR-005: Idempotent Discovery**
Running discovery multiple times on same documentation must produce identical database state.

- Use upsert logic (INSERT OR REPLACE) for epic and feature records
- Compare existing record metadata with discovered metadata
- Update records only if metadata changed (preserve timestamps if unchanged)
- Running `shark scan` 3 times consecutively should result in identical database state

**NFR-006: Error Recovery**
Parser failures on individual files must not halt entire discovery.

- Wrap epic.md parsing in error handler
- Wrap prd.md parsing in error handler
- Log parse error with file path and line number
- Continue discovery with next folder/file
- Include parse errors in discovery report warnings section
- Commit successfully discovered items even if some fail

#### Usability (REQ-NF-006, REQ-NF-007)

**NFR-007: Actionable Error Messages**
All errors and warnings must include file path, line number (if applicable), and suggested fix.

- **Example**: "Cannot parse frontmatter in docs/plan/E04-task-mgmt-cli-core/epic.md:5: missing closing '---'. Suggestion: Add '---' on line 8 to close frontmatter."
- **Example**: "Folder E04-task-mgmt-cli-core matches epic pattern but not listed in epic-index.md. Suggestion: Add epic to index or use merge strategy."
- **Example**: "Feature E04-F07 listed in index but folder docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/ not found. Suggestion: Create folder or remove from index."
- Include suggestion in every warning/error message
- Provide command examples for fixes where applicable

**NFR-008: Dry-Run Mode**
All discovery commands must support --dry-run flag for preview without database changes.

- Scan documentation and detect epics/features as normal
- Generate discovery report as if committed
- Display conflicts and resolution strategy
- DO NOT modify database
- DO NOT update .sharkconfig.json
- Exit with code 0 (success)
- Clearly indicate preview mode: "Dry-run mode: No changes will be made"

**NFR-009: Progress Indication**
For large projects, discovery should show progress during scan.

- Display progress for operations taking >2 seconds
- Show current step: "Scanning epic folders...", "Parsing epic-index.md...", "Discovering features...", "Cataloging related documents..."
- Show progress percentage or count: "Processed 50/200 folders"
- Suppress progress output when --json flag is used
- Suppress progress output when --quiet flag is used

#### Maintainability (REQ-NF-008, REQ-NF-009)

**NFR-010: Pattern Extensibility**
Adding new epic/feature patterns must require only configuration changes, no code modifications.

- Regex pattern definitions in .sharkconfig.json
- Pattern-matching engine generic and data-driven
- Named capture groups define which components to extract
- No hardcoded patterns in Go code (except shipped defaults)
- Test new pattern by adding to config and running scan

**NFR-011: Test Coverage**
Discovery engine logic must have high unit test coverage.

- >80% unit test coverage for pattern matching logic
- >80% unit test coverage for conflict resolution logic
- Table-driven tests for pattern variations
- Test cases for all conflict scenarios
- Integration tests for full discovery workflow
- CI pipeline enforces coverage threshold

**NFR-012: Logging and Observability**
Discovery engine must provide detailed logging for debugging.

- Log level configuration: debug, info, warn, error
- Debug logs show: patterns matched, capture groups extracted, metadata sources used
- Info logs show: discovery progress, summary stats
- Warn logs show: parse failures, validation warnings, conflicts
- Error logs show: database errors, filesystem errors, transaction failures
- Support --verbose flag to enable debug logging
- Write logs to stderr (stdout reserved for report/JSON)

#### Security and Data Integrity

**NFR-013: Path Traversal Protection**
Discovery engine must protect against path traversal attacks.

- Validate all file paths are within configured docs_root
- Reject paths containing ".." (parent directory references)
- Canonicalize paths before validation (resolve symlinks)
- Log warning if path traversal attempt detected

**NFR-014: Database Constraint Enforcement**
Discovery must respect database constraints to maintain referential integrity.

- Foreign key constraints: features.epic_key must reference existing epics.epic_key
- Unique constraints: epic_key and feature_key must be unique
- NOT NULL constraints: epic_key, feature_key, title must not be NULL
- Validate constraints before database insertion
- Catch constraint violation exceptions and provide clear error messages

**NFR-015: Data Validation**
Discovery must validate extracted metadata before database insertion.

- Validate epic_key format matches expected pattern (alphanumeric, hyphens, underscores)
- Validate feature_key format matches expected pattern
- Validate title is non-empty (at least 3 characters)
- Validate file_path is relative to project root (not absolute system path)
- Truncate descriptions to maximum length (if database column has length limit)
- Sanitize metadata extracted from markdown (remove potentially dangerous characters)

## Acceptance Criteria

### Epic Discovery from epic-index.md

**Given** docs/plan/epic-index.md exists with markdown links to 5 epics
**When** I run `shark scan`
**Then** discovery engine parses epic-index.md
**And** extracts 5 epic keys from link paths
**And** extracts 5 epic titles from link text
**And** creates 5 epic records in database
**And** discovery report shows "Epics discovered: 5 (5 from index, 0 from folders)"

### Epic Discovery from Folder Structure (Fallback)

**Given** docs/plan/epic-index.md does NOT exist
**And** docs/plan/ contains 3 folders: E04-task-mgmt-cli-core/, E05-advanced-querying/, tech-debt/
**When** I run `shark scan`
**Then** discovery engine scans docs/plan/ recursively
**And** applies epic.folder patterns to directory names
**And** matches E04-task-mgmt-cli-core (standard pattern)
**And** matches E05-advanced-querying (standard pattern)
**And** matches tech-debt (special type pattern)
**And** creates 3 epic records in database
**And** discovery report shows "Epics discovered: 3 (0 from index, 3 from folders)"

### Feature Discovery with Pattern Matching

**Given** epic E04-task-mgmt-cli-core/ exists with subfolders: E04-F01-database-schema/, E04-F02-cli-infrastructure/, E04-F07-initialization-sync/
**When** I run `shark scan`
**Then** discovery engine scans E04 subfolders
**And** applies feature.folder patterns to directory names
**And** extracts epic_id=E04, feature_id=F01 from E04-F01-database-schema/
**And** extracts epic_id=E04, feature_id=F02 from E04-F02-cli-infrastructure/
**And** extracts epic_id=E04, feature_id=F07 from E04-F07-initialization-sync/
**And** creates 3 feature records with parent epic_key=E04
**And** discovery report shows "Features discovered: 3"

### Feature PRD File Recognition

**Given** feature folder E04-F07-initialization-sync/ contains files: prd.md, 02-architecture.md, tasks/
**When** I run `shark scan`
**Then** discovery engine applies feature.file patterns to .md files
**And** matches prd.md (first priority pattern)
**And** does NOT match 02-architecture.md (cataloged as related doc instead)
**And** does NOT match tasks/ folder contents (task files, not feature files)
**And** creates feature record with file_path="docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/prd.md"

### Related Document Cataloging

**Given** feature folder E04-F07-initialization-sync/ contains: prd.md, 02-architecture.md, 04-backend-design.md, 09-test-criteria.md
**When** I run `shark scan`
**Then** discovery engine catalogs related documents
**And** identifies 02-architecture.md (numbered design doc)
**And** identifies 04-backend-design.md (numbered design doc)
**And** identifies 09-test-criteria.md (numbered design doc)
**And** stores related_docs JSON: ["docs/plan/E04-.../02-architecture.md", "docs/plan/E04-.../04-backend-design.md", "docs/plan/E04-.../09-test-criteria.md"]
**And** excludes prd.md from related_docs (PRD itself)
**And** discovery report shows "Related documents cataloged: 3"

### Conflict Detection: Epic in Index but Folder Missing

**Given** epic-index.md lists epic E05-advanced-querying with link ./E05-advanced-querying/
**And** docs/plan/ does NOT contain E05-advanced-querying/ folder
**When** I run `shark scan --detect-conflicts`
**Then** discovery engine detects broken reference
**And** reports conflict: "Epic E05 listed in index but folder not found"
**And** includes suggestion: "Create folder or remove from index"
**And** discovery fails (index-precedence strategy requires folder for index entries)

### Conflict Detection: Epic Folder Not in Index

**Given** epic-index.md lists 5 epics
**And** docs/plan/ contains E04-task-mgmt-cli-core/ folder (not listed in index)
**When** I run `shark scan --detect-conflicts --strategy=index-precedence`
**Then** discovery engine detects undocumented folder
**And** reports conflict: "Epic E04 found in folders but not in epic-index.md"
**And** includes suggestion: "Add epic to index or use merge strategy"
**And** skips E04 import (index-precedence strategy ignores folder-only epics)
**And** logs warning in discovery report

### Conflict Resolution: Merge Strategy

**Given** epic-index.md lists 5 epics
**And** docs/plan/ contains 7 epic folders (5 matching index + 2 extra)
**When** I run `shark scan --strategy=merge`
**Then** discovery engine imports ALL 7 epics (5 from index + 2 from folders)
**And** uses index-provided titles for 5 matching epics
**And** generates titles from folder names for 2 folder-only epics
**And** discovery report shows "Epics discovered: 7 (5 from index, 2 from folders)"
**And** logs 2 conflicts resolved via merge

### Metadata Extraction: Title Fallback Priority

**Given** epic E04-task-mgmt-cli-core/ has:
- epic-index.md link text: "Task Management CLI Core"
- docs/plan/E04-task-mgmt-cli-core/epic.md frontmatter: title="Task Mgmt CLI Core"
**When** I run `shark scan`
**Then** discovery engine extracts title with priority:
**And** uses "Task Management CLI Core" from epic-index.md (highest priority)
**And** ignores epic.md frontmatter title (lower priority)
**And** creates epic record with title="Task Management CLI Core"

**Given** epic E06-intelligent-scanning/ is NOT in epic-index.md
**And** docs/plan/E06-intelligent-scanning/epic.md frontmatter: title="Intelligent Documentation Scanning"
**When** I run `shark scan --strategy=folder-precedence`
**Then** discovery engine uses epic.md frontmatter title (index not available)
**And** creates epic record with title="Intelligent Documentation Scanning"

**Given** epic tech-debt/ has no epic.md file
**When** I run `shark scan`
**Then** discovery engine generates title from folder name
**And** converts "tech-debt" to "Tech Debt"
**And** creates epic record with title="Auto: Tech Debt" (marked as auto-generated)

### Dry-Run Mode

**Given** I run `shark scan --dry-run`
**When** discovery engine finds 5 epics and 20 features
**Then** discovery report shows "Epics discovered: 5, Features discovered: 20"
**And** message displays "Dry-run mode: No changes will be made"
**And** database is NOT modified (no epic/feature records created)
**And** command exits with code 0 (success)

### Transaction Rollback on Error

**Given** discovery engine is processing 10 features
**And** feature #5 has invalid epic_key reference (foreign key violation)
**When** database insert fails with constraint error
**Then** entire transaction is rolled back
**And** features #1-4 are NOT in database (rollback successful)
**And** error message explains: "Foreign key violation: feature E99-F03 references non-existent epic E99"
**And** command exits with code 2 (database error)

### JSON Output Format

**Given** I run `shark scan --json`
**When** discovery completes
**Then** output is valid JSON
**And** JSON contains: epics_discovered, features_discovered, related_docs_cataloged, conflicts_detected
**And** JSON contains conflicts array with details for each conflict
**And** JSON contains warnings array
**And** command exits with code 0

### Query Related Documents via CLI

**Given** feature E04-F07 has 3 related documents cataloged
**When** I run `shark feature get E04-F07 --show-docs`
**Then** CLI displays feature metadata
**And** displays related documents section:
```
Related Documents:
  - docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/02-architecture.md
  - docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/04-backend-design.md
  - docs/plan/E04-task-mgmt-cli-core/E04-F07-initialization-sync/09-test-criteria.md
```
**And** command exits with code 0

### Pattern Validation (Could-Have)

**Given** .sharkconfig.json contains invalid regex: `(?P<epic_id>E\d{2`
**When** I run `shark scan`
**Then** config loading fails with validation error
**And** error message shows: "Invalid regex syntax in patterns.epic.folder: missing closing parenthesis"
**And** command exits with code 1 (config error)

### Validation Strictness: Balanced Mode

**Given** .sharkconfig.json has validation_level="balanced"
**And** docs/plan/ contains folder "legacy-epic" (doesn't match any pattern)
**When** I run `shark scan`
**Then** discovery engine logs warning: "Folder legacy-epic doesn't match any epic pattern, skipping"
**And** continues discovery with other folders
**And** discovery report includes warning but completes successfully
**And** command exits with code 0

### Validation Strictness: Strict Mode

**Given** .sharkconfig.json has validation_level="strict"
**And** docs/plan/ contains folder "tech-debt" (special type, not E##)
**When** I run `shark scan`
**Then** discovery engine rejects folder (strict mode only accepts E## format)
**And** logs error: "Folder tech-debt doesn't match strict E##-slug pattern"
**And** discovery fails
**And** command exits with code 2 (validation error)

### Epic/Feature Relationship Validation

**Given** feature folder E04-F07-initialization-sync/ exists
**And** epic E04 does NOT exist in database
**When** I run `shark scan`
**Then** discovery engine detects orphaned feature
**And** logs warning: "Feature E04-F07 references non-existent epic E04"
**And** skips feature import (cannot create orphaned feature)
**And** suggests: "Ensure epic E04 is discovered before feature E04-F07"

## Out of Scope

### Explicitly NOT Included in This Feature

**1. Task Discovery**
Task file discovery and import is handled by F03 (Task Discovery Engine), not this feature. E06-F02 focuses exclusively on epic and feature level discovery.

**2. Configuration Management and Pattern Definition**
Defining, validating, and managing regex patterns in .sharkconfig.json is handled by F01 (Configuration & Pattern Management). This feature consumes patterns from F01 but does not manage pattern configuration itself.

**3. Incremental Discovery Based on Modification Time**
Incremental discovery (only scanning modified folders since last discovery) is Could-Have priority and deferred to future enhancement. Initial implementation performs full discovery on every scan.

**4. Interactive Conflict Resolution**
Prompting user interactively for each conflict (choose index or folder for this epic?) is Could-Have, deferred. Initial implementation uses configured strategy for all conflicts non-interactively.

**5. Bidirectional Sync (Database to Files)**
Updating epic-index.md or epic.md files based on database changes is out of scope. Discovery is unidirectional: documentation → database only.

**6. Epic/Feature Deletion Detection**
Detecting and removing epics/features from database when folders are deleted is out of scope. Discovery only adds/updates, never deletes.

**7. Git Integration for Change Detection**
Using `git diff` or `git status` to identify changed folders is Could-Have, deferred. Initial implementation relies on full folder scans or mtime-based filtering (future).

**8. Pattern Preset Library**
Providing `shark config add-pattern --preset=special-epics` command for preset patterns is Should-Have but deferred to F01 (Configuration Management).

**9. Epic-Index.md Generation**
Auto-generating epic-index.md from folder structure (reverse operation) is out of scope. Discovery only reads index, doesn't write it.

**10. Multi-Root Documentation Support**
Scanning multiple documentation roots (e.g., docs/plan and docs/archived) is Could-Have (REQ-F-020), deferred. Initial implementation supports single docs_root.

**11. Related Document Content Parsing**
Extracting metadata or content from related documents (architecture.md, design docs) is out of scope. Discovery only catalogs paths, doesn't parse content.

**12. Automatic Epic/Feature Creation from Task References**
Auto-creating missing epics when tasks reference them (task fuzzy key generation from REQ-F-018) is task-level functionality, not epic/feature discovery.

**13. Feature Status Inference**
Automatically setting feature status based on folder contents (e.g., status="completed" if all tasks done) is out of scope. Discovery sets default status="planning" for all newly discovered features.

**14. Duplicate Detection Across Roots**
Detecting duplicate epic/feature keys across different documentation roots is out of scope (single root only in initial implementation).

**15. Schema Migration Integration**
Upgrading database schema during discovery (adding new columns, migrating data) is out of scope. Discovery assumes schema is already up-to-date.
