# Feature: Task File Recognition & Import

**Feature Key**: E06-F03-task-recognition-import

---

## Epic

- [Epic PRD](/docs/plan/E06-intelligent-scanning/epic.md)
- [Epic Requirements](/docs/plan/E06-intelligent-scanning/requirements.md)

---

## Goal

### Problem

The current sync engine (internal/sync/engine.go) uses rigid pattern matching that only recognizes task files in the exact `T-E##-F##-###.md` format. This forces users to manually rename hundreds of existing task files or manually import them one-by-one before shark can track them. AI agents generating task files during development may produce slight variations (`001-implement-feature.md`, `feature-spec.prp.md`, `T-E04-F02-001-slug.md`) that fail to import, breaking the automated workflow. Product managers working with legacy documentation that uses numbered prefixes (`01-research.md`, `02-design.md`) or PRP suffixes cannot leverage shark's capabilities without extensive file restructuring.

This rigid pattern matching directly conflicts with E06's core value proposition: adapt to existing documentation patterns rather than forcing conformity. It creates a significant adoption barrier for existing projects and reduces AI agent reliability.

### Solution

Implement a configurable task file pattern matching system that recognizes multiple task file naming conventions through regex patterns defined in `.sharkconfig.json`. The system will use named capture groups to extract task components (epic_id, feature_id, task_id, slug, number), support multiple patterns per entity type (first match wins), and provide intelligent metadata extraction with multi-source fallbacks.

Key capabilities include:
- **Configurable Pattern Registry**: Users define task file patterns in `.sharkconfig.json` with named capture groups for component extraction
- **Multi-Source Metadata Extraction**: Extract title/description from frontmatter, filename, or H1 heading with priority-based fallbacks
- **Automatic Task Key Generation**: Generate missing task keys for PRP files by inferring epic/feature from path and querying database for next sequence number
- **Flexible Frontmatter Validation**: Balanced mode validates critical fields (task_key) while allowing optional fields with intelligent fallbacks
- **Integration with E04 Workflows**: Seamless integration with existing TaskRepository, key generation, and sync engine infrastructure

### Impact

- **Import Success Rate**: Increase task file import success from 40% (rigid matching) to 90% (flexible patterns) for existing projects
- **Agent Reliability**: Enable AI agents to create task files with naming variations without manual intervention
- **Migration Efficiency**: Reduce task file restructuring time from 2-4 hours to zero for projects with 100+ task files
- **Pattern Flexibility**: Support 5+ common task naming conventions out-of-box with defaults, unlimited via configuration
- **Developer Experience**: Eliminate "file not recognized" errors through clear pattern validation and actionable warnings

---

## User Personas

### Persona 1: Technical Lead (Legacy Project Migration)

**Profile**:
- Technical Lead managing migration of existing multi-epic project (200-500 task files)
- Task files use mixed conventions: numbered prefixes (`01-analysis.md`), descriptive names (`user-authentication.prp.md`), standard format (`T-E04-F02-001.md`)
- Limited time for migration (wants "import as-is")
- Needs visibility into what was imported vs. skipped
- Risk-averse: requires validation that relationships are correct

**Goals**:
1. Import existing task files without renaming 200+ files
2. Understand which patterns matched and which files were skipped
3. Validate that epic/feature relationships are correctly inferred from paths
4. Establish shark database as source of truth while preserving original filenames

**Pain Points**:
- Current sync only recognizes `T-E##-F##-###.md` format (maybe 20% of existing files)
- Manual renaming breaks git history and is error-prone at scale
- No way to configure custom patterns without modifying Go code
- Unclear why files are skipped (generic "pattern mismatch" errors)
- Fear of incorrect epic/feature associations during bulk import

**Success Scenario**:
Technical Lead adds task pattern to `.sharkconfig.json` matching their numbered prefix convention (`^\d{2}-.*\.md$`), runs `shark sync --dry-run`, sees 245 of 267 task files matched with clear warnings about 22 unmatched files, adjusts pattern to capture additional variation, re-runs sync, and successfully imports all 267 tasks with correct epic/feature relationships in under 10 minutes.

### Persona 2: AI Agent (Automated Task Creation)

**Profile**:
- Autonomous code generation agent (Claude Code) creating task files during development sessions
- Generates PRP files (Product Requirement Prompts) with descriptive names: `implement-auth-middleware.prp.md`
- Cannot manually query database for next task sequence number (stateless between sessions)
- Expects automatic task key generation when creating new files
- Needs immediate feedback if created file doesn't match expected patterns

**Goals**:
1. Create task files with descriptive names without worrying about exact format
2. Have task keys automatically generated and written to frontmatter
3. Complete sync in <2 seconds to maintain session efficiency
4. Receive clear errors if pattern matching fails with suggestions for fix

**Pain Points**:
- Current sync requires exact `T-E##-F##-###.md` format (agent must calculate task number)
- PRP files with descriptive names are rejected (agent workflow broken)
- No automatic task key generation (agent must query database manually)
- Sync failures are opaque (agent cannot determine root cause)
- Pattern variations during development cause import failures

**Success Scenario**:
AI Agent creates `implement-caching-layer.prp.md` in `docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/`, runs `shark sync`, system detects PRP pattern, infers epic E04 and feature F02 from path, queries database for next task number (003), generates task key `T-E04-F02-003`, writes key to frontmatter, imports task to database, and completes in 1.1 seconds with success message showing generated key.

### Persona 3: Product Manager (Multi-Convention Documentation)

**Profile**:
- Product Manager working with task files from multiple teams/time periods
- Documentation evolved organically: early tasks use numbered prefixes, later tasks use standard format, design tasks use `.prp.md` suffix
- Needs to track all tasks regardless of naming convention
- Occasionally hand-edits task files to update status/description
- Values flexibility over strict standardization

**Goals**:
1. Support multiple task naming conventions within same project
2. Configure patterns once and have all subsequent syncs recognize all conventions
3. Extract task metadata even when frontmatter is incomplete
4. Generate accurate progress reports across mixed task file styles

**Pain Points**:
- Different teams use different task file conventions (no standardization)
- Cannot track design tasks that use `.prp.md` suffix
- Tasks missing frontmatter are rejected (even though title/description exist in file)
- Reporting tools fail when some tasks don't match standard format
- Manual edits to task files require understanding rigid format rules

**Success Scenario**:
Product Manager configures three task patterns in `.sharkconfig.json`: standard task format, numbered prefix, and PRP suffix. Runs `shark sync`, system recognizes all three patterns across 150 task files, extracts metadata using fallbacks (frontmatter → filename → H1 heading), generates keys for 30 PRP files lacking frontmatter, imports all 150 tasks with correct epic/feature associations, and generates weekly status report combining data from all task conventions seamlessly.

---

## User Stories

### Must-Have Stories

**US-1**: As a Technical Lead, I want to define task file patterns in `.sharkconfig.json` using regex with named capture groups so that shark recognizes my existing task file naming conventions without code changes.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-1

**US-2**: As a Technical Lead, I want to specify multiple task patterns with precedence order so that shark tries each pattern until one matches, supporting projects with evolved naming conventions.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-2

**US-3**: As an AI Agent, I want task keys automatically generated for PRP files missing frontmatter so that I can create descriptive task files without querying the database for sequence numbers.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-3

**US-4**: As a Product Manager, I want task titles extracted from frontmatter, filename, or H1 heading with priority-based fallbacks so that tasks are imported even when frontmatter is incomplete.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-4

**US-5**: As a Technical Lead, I want detailed warnings when task files don't match any pattern so that I can adjust patterns or fix files with clear guidance.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-5

**US-6**: As a Product Manager, I want epic/feature inferred from file path using directory structure so that task relationships are established correctly during import.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-6

**US-7**: As a Technical Lead, I want pattern validation on config load so that I catch regex errors or missing capture groups before scanning thousands of files.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-7

**US-8**: As an AI Agent, I want generated task keys written back to file frontmatter so that future syncs use the stable key instead of regenerating.
- **Priority**: Must-have
- **Acceptance Criteria**: See AC-8

### Should-Have Stories

**US-9**: As a Product Manager, I want preset task patterns for common conventions (standard, numbered, PRP) so that I don't have to write regex from scratch.
- **Priority**: Should-have
- **Acceptance Criteria**: See AC-9

**US-10**: As a Technical Lead, I want pattern matching errors to include the attempted pattern and captured groups so that I can debug complex regex issues.
- **Priority**: Should-have
- **Acceptance Criteria**: See AC-10

**US-11**: As a Technical Lead, I want dry-run mode to show which pattern matched each file so that I can validate pattern precedence is correct before importing.
- **Priority**: Should-have
- **Acceptance Criteria**: See AC-11

### Could-Have Stories

**US-12**: As a Product Manager, I want to test task patterns against filenames via CLI command so that I can validate regex before adding to config.
- **Priority**: Could-have
- **Acceptance Criteria**: See AC-12

**US-13**: As a Technical Lead, I want task pattern statistics (match count per pattern) in sync report so that I understand which conventions are most common in my project.
- **Priority**: Could-have
- **Acceptance Criteria**: See AC-13

---

## Requirements

### Functional Requirements

#### Pattern Configuration & Validation

**REQ-F-001**: Configurable Task Pattern Definitions
- System must read task file patterns from `.sharkconfig.json` under `patterns.task.file` field (array of pattern objects)
- Each pattern object must include: `name` (string), `regex` (string), `enabled` (boolean), `description` (string, optional)
- Default configuration must include three patterns: "standard" (T-E##-F##-###), "numbered-prefix" (##-*), "prp-suffix" (*.prp.md)
- Patterns must be evaluated in array order (first match wins)
- Missing `patterns.task.file` field must fall back to standard pattern only for backward compatibility

**REQ-F-002**: Named Capture Groups for Component Extraction
- System must require specific named capture groups in task patterns based on pattern type:
  - Standard pattern: `(?P<task_key>T-E\d{2}-F\d{2}-\d{3})` (full task key)
  - Numbered pattern: `(?P<number>\d{2,3})` (task sequence number only)
  - PRP pattern: `(?P<slug>[a-z0-9-]+)` (descriptive slug only)
- System must extract epic/feature from file path when task key not captured in pattern (required for numbered/PRP patterns)
- System must validate parent directory structure matches epic/feature pattern: `docs/plan/E##-epic-slug/E##-F##-feature-slug/tasks/`
- System must fail validation if pattern captures `task_key` but path-based epic/feature extraction is inconsistent

**REQ-F-003**: Pattern Validation on Configuration Load
- System must validate all enabled patterns when loading `.sharkconfig.json`
- Validation must check: (1) regex syntax is valid, (2) required capture groups present, (3) capture group names follow conventions
- System must log specific validation errors: pattern name, missing/invalid capture groups, regex syntax error details
- System must exit with error code 1 if any enabled pattern fails validation (prevent invalid scans)
- System must support `--skip-pattern-validation` flag for advanced users (at their own risk, logged as warning)

**REQ-F-004**: Pattern Precedence and Fallback
- System must evaluate patterns in configured array order when matching task files
- System must use first matching pattern (do not evaluate subsequent patterns)
- System must log which pattern matched each file in verbose mode (`--verbose` flag)
- System must track unmatched files separately with list of attempted patterns
- System must provide default pattern fallback if config is malformed (log warning about using defaults)

#### Metadata Extraction

**REQ-F-005**: Multi-Source Task Title Extraction
- System must extract task title using priority-based fallback:
  1. **Priority 1**: Frontmatter `title:` field (if present and non-empty)
  2. **Priority 2**: Filename descriptive part after task key/number (e.g., `T-E04-F02-001-implement-caching.md` → "Implement Caching")
  3. **Priority 3**: First H1 heading (`# Task: ...` or `# PRP: ...`), removing prefixes
- Title extraction from filename must:
  - Convert hyphens to spaces
  - Capitalize first letter of each word (Title Case)
  - Remove file extension
  - Remove pattern-matched prefix (task key or number)
- Title extraction from H1 must:
  - Remove common prefixes: "Task:", "PRP:", "TODO:", "WIP:" (case-insensitive)
  - Trim whitespace
  - Preserve original capitalization
- System must log warning if no title can be extracted from any source (file path included in warning)
- System must NOT skip file if title missing; use placeholder: "Untitled Task" with warning

**REQ-F-006**: Multi-Source Task Description Extraction
- System must extract task description using priority-based fallback:
  1. **Priority 1**: Frontmatter `description:` field (if present and non-empty)
  2. **Priority 2**: First paragraph after frontmatter or H1 heading (up to 500 characters)
  3. **Priority 3**: Empty string (description is optional field)
- Description extraction from markdown body must:
  - Skip frontmatter block (between `---` delimiters)
  - Skip H1 heading line
  - Extract first continuous paragraph (stop at blank line or next heading)
  - Trim to 500 characters maximum
  - Preserve line breaks within paragraph
- System must NOT log warning if description is empty (optional field)

**REQ-F-007**: Frontmatter Field Parsing
- System must parse YAML frontmatter between `---` delimiters at file start
- System must extract optional fields: `title`, `description`, `task_key`, `status`, `agent_type`, `priority`, `assigned_agent`, `blocked_reason`
- System must use default values for missing optional fields: `status="todo"`, `agent_type=null`, `priority=2`, `assigned_agent=null`, `blocked_reason=null`
- System must validate frontmatter YAML syntax and log parsing errors with file path and line number
- System must continue with fallback extraction if frontmatter is invalid or missing (log warning, do not skip file)

#### Task Key Generation

**REQ-F-008**: Automatic Task Key Generation for PRP Files
- System must generate task keys for files matching PRP pattern when `task_key` frontmatter field is missing
- Key generation must:
  1. Extract epic key and feature key from parent directory path (`docs/plan/{epic}/{feature}/tasks/{file}`)
  2. Validate epic/feature exist in database (fail with clear error if orphaned)
  3. Query database for next available task sequence number for that feature (SELECT MAX(sequence) WHERE feature_id=...)
  4. Generate task key in format: `T-{epic_key}-{feature_key}-{sequence:03d}` (e.g., `T-E04-F02-003`)
  5. Return generated key for use in current sync
- System must handle concurrent key generation safely using database transactions
- System must NOT generate keys for files matching standard pattern (task_key must be in filename or frontmatter)

**REQ-F-009**: Frontmatter Update with Generated Keys
- System must write generated task keys back to file frontmatter after generation
- Frontmatter update must:
  1. Read current file contents
  2. Add or update `task_key: {generated_key}` field in frontmatter YAML
  3. Preserve all other frontmatter fields and markdown content
  4. Write updated content back to file atomically (temp file + rename)
  5. Maintain file permissions and ownership
- System must log warning if frontmatter write fails (permission denied, disk full)
- System must continue with in-memory key if write fails (sync completes successfully but future syncs regenerate key)
- System must NOT fail entire sync if single file frontmatter update fails (isolation per REQ-NF-005)

**REQ-F-010**: Epic and Feature Inference from Path
- System must extract epic key and feature key from file path for numbered/PRP patterns
- Path parsing must match structure: `{docs_root}/{epic_folder}/{feature_folder}/tasks/{filename}` or `{docs_root}/{epic_folder}/{feature_folder}/prps/{filename}`
- Epic folder must match epic pattern (default: `E\d{2}-[a-z0-9-]+` or special epic names per epic whitelist)
- Feature folder must match feature pattern (default: `E\d{2}-F\d{2}-[a-z0-9-]+`)
- System must log clear error if path structure doesn't match expected format (include expected structure in error)
- System must skip file if epic/feature cannot be inferred from path (cannot establish relationship)

#### Integration with E04 Workflows

**REQ-F-011**: TaskRepository Integration
- System must use existing `TaskRepository.Create()` method for task insertion (no direct SQL in sync engine)
- Task creation must include all standard fields: `feature_id`, `key`, `title`, `description`, `status`, `agent_type`, `priority`, `assigned_agent`, `file_path`, `blocked_reason`
- System must resolve `feature_id` from feature_key using `FeatureRepository.GetByKey()` before task creation
- System must handle duplicate key errors gracefully (treat as update instead of create)
- System must validate task data using `task.Validate()` before insertion (per E04 validation rules)

**REQ-F-012**: File Path Recording
- System must record absolute file path in `tasks.file_path` column during import
- File path must be relative to project root (strip prefix up to project base directory)
- System must normalize path separators to forward slashes (cross-platform compatibility)
- System must validate file still exists before recording path (symlink resolution supported)
- System must update file path if task file is moved (detected via task_key match during subsequent sync)

**REQ-F-013**: Sync Engine Integration
- System must extend existing sync engine's `scanTaskFiles()` function (internal/sync/engine.go:145-195)
- System must replace hardcoded pattern matching with configurable `PatternRegistry.MatchesAnyPattern()`
- System must preserve existing modification time tracking and incremental sync logic
- System must maintain transaction boundaries (all task imports in single transaction with rollback on error)
- System must integrate with existing conflict detection and resolution (REQ-F-009 and REQ-F-010 per epic requirements)

### Non-Functional Requirements

#### Performance

**REQ-NF-001**: Pattern Matching Performance
- Pattern matching must complete in <1ms per file for up to 5 configured patterns
- Regex compilation must occur once at config load (cached for all files)
- Pattern evaluation must short-circuit on first match (do not evaluate remaining patterns)
- System must handle 1,000 task files in single scan without performance degradation

**REQ-NF-002**: Task Key Generation Performance
- Task key generation must complete in <20ms per file (including database query for next sequence)
- System must batch-query sequence numbers when generating multiple keys for same feature (single query for max sequence)
- Database transaction for key generation must not hold locks longer than 50ms (prevent sync bottlenecks)

**REQ-NF-003**: Metadata Extraction Performance
- Title/description extraction must complete in <5ms per file
- Markdown parsing must use streaming approach (do not load entire file into memory)
- Frontmatter parsing must stop at closing `---` (do not parse entire file for frontmatter)

#### Reliability

**REQ-NF-004**: Pattern Matching Error Isolation
- Invalid regex in single pattern must not prevent other patterns from being evaluated
- System must log error for invalid pattern and continue with remaining enabled patterns
- Pattern validation errors must be accumulated and reported at end (do not fail fast for better visibility)

**REQ-NF-005**: File-Level Error Recovery
- Parsing failure on individual file must not abort entire sync (per epic REQ-NF-005)
- System must wrap file parsing in error handler, log detailed error, continue with next file
- Sync report must show successful imports + failed files with specific error messages
- Transaction must commit successfully for all valid files even if some files failed

**REQ-NF-006**: Transactional Task Key Generation
- Task key generation for multiple files must be atomic per feature (all keys generated or none)
- Database transaction must include both sequence number query and task insertion
- System must handle race conditions (concurrent key generation) using appropriate isolation level
- Generated keys must be sequential without gaps (use database auto-increment or explicit gap checking)

#### Usability

**REQ-NF-007**: Actionable Error Messages
- Pattern validation errors must include: pattern name, regex, specific capture group missing, valid example
- Path inference errors must include: file path, expected directory structure, actual structure
- Metadata extraction warnings must include: file path, attempted extraction sources, suggestions
- Example error format: "Pattern 'numbered-prefix' in file 'docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/01-setup.md' missing required capture group 'number'. Expected regex: `(?P<number>\d{2,3})-.*\.md`. Suggestion: Add (?P<number>...) capture group to extract task number."

**REQ-NF-008**: Verbose Logging for Debugging
- System must support `--verbose` flag to enable detailed pattern matching logs
- Verbose mode must log: (1) pattern evaluation order, (2) matched pattern name, (3) captured groups, (4) metadata extraction sources used
- Verbose logs must be structured for easy parsing (consider JSON format with `--json` flag)
- Default mode must only log warnings/errors (minimal output for successful imports)

**REQ-NF-009**: Dry-Run Mode Support
- System must support `--dry-run` flag to preview pattern matching without database changes
- Dry-run must show: (1) files that would be imported, (2) matched patterns, (3) generated task keys, (4) warnings for unmatched files
- Dry-run must NOT write generated keys to frontmatter (read-only mode)
- Dry-run must NOT modify database (skip transaction commit)

#### Maintainability

**REQ-NF-010**: Pattern Configuration Schema
- System must use well-documented JSON schema for pattern configuration
- Schema must include examples for each pattern type with explanations
- System must validate configuration against schema on load (JSON schema validation library)
- Schema validation errors must include: field path, expected type/format, actual value

**REQ-NF-011**: Test Coverage for Pattern Matching
- Pattern matching logic must have >90% unit test coverage
- Tests must include: (1) each default pattern, (2) custom pattern configurations, (3) precedence order, (4) capture group extraction
- Tests must use table-driven approach with variety of filename examples
- Integration tests must validate end-to-end: pattern config → file scan → database insertion

**REQ-NF-012**: Extensibility for New Pattern Types
- Adding new pattern type must require only: (1) config update, (2) capture group documentation
- No code changes required for new pattern types if capture groups follow conventions
- System must log clear error if unknown capture group used (with documentation link)

---

## Acceptance Criteria

### AC-1: Configurable Task Pattern Definitions (US-1)

**Given** a `.sharkconfig.json` file with custom task pattern
**When** I add the following configuration:
```json
{
  "patterns": {
    "task": {
      "file": [
        {
          "name": "standard",
          "regex": "^(?P<task_key>T-E\\d{2}-F\\d{2}-\\d{3})(?:-(?P<slug>[a-z0-9-]+))?\\.md$",
          "enabled": true,
          "description": "Standard task format with optional slug"
        }
      ]
    }
  }
}
```
**Then** running `shark sync` must recognize task files matching the pattern
**And** unrecognized files must be logged with "pattern mismatch" warning
**And** pattern validation must succeed on config load without errors

### AC-2: Multiple Task Patterns with Precedence (US-2)

**Given** configuration with three task patterns in order: standard, numbered-prefix, prp-suffix
**When** I run `shark sync` on directory with mixed task file naming conventions
**Then** file `T-E04-F02-001.md` must match "standard" pattern (first)
**And** file `01-research.md` must match "numbered-prefix" pattern (second)
**And** file `implement-auth.prp.md` must match "prp-suffix" pattern (third)
**And** sync report must show match count per pattern type
**And** verbose mode (`--verbose`) must log which pattern matched each file

### AC-3: Automatic Task Key Generation (US-3)

**Given** file `implement-caching.prp.md` in `docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/`
**And** file has no `task_key` in frontmatter
**When** I run `shark sync`
**Then** system must infer epic E04 and feature F02 from path
**And** system must query database for next task number in feature F02 (e.g., 003)
**And** system must generate task key `T-E04-F02-003`
**And** system must write `task_key: T-E04-F02-003` to file frontmatter
**And** system must import task to database with key `T-E04-F02-003`
**And** sync report must show "Generated task key: T-E04-F02-003 for file: implement-caching.prp.md"

### AC-4: Multi-Source Title Extraction (US-4)

**Given** task file with frontmatter `title: "Implement User Authentication"`
**When** I run `shark sync`
**Then** task title must be extracted from frontmatter (Priority 1)

**Given** task file without frontmatter, filename `T-E04-F02-001-implement-user-authentication.md`
**When** I run `shark sync`
**Then** task title must be extracted from filename as "Implement User Authentication" (Priority 2)

**Given** task file without frontmatter or descriptive filename, markdown content `# Task: Implement User Authentication`
**When** I run `shark sync`
**Then** task title must be extracted from H1 as "Implement User Authentication" with prefix removed (Priority 3)

**Given** task file with no title in any source
**When** I run `shark sync`
**Then** task title must default to "Untitled Task"
**And** system must log warning: "No title found for file: {path}. Using default title."

### AC-5: Detailed Pattern Mismatch Warnings (US-5)

**Given** task file `my-random-file.txt` that doesn't match any pattern
**When** I run `shark sync`
**Then** system must log warning: "File 'docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/my-random-file.txt' did not match any task patterns. Attempted patterns: standard, numbered-prefix, prp-suffix. Suggestion: Ensure filename matches one of the configured patterns or add custom pattern to .sharkconfig.json."
**And** file must be listed in "Skipped Files" section of sync report

### AC-6: Epic/Feature Inference from Path (US-6)

**Given** file `01-research.md` in path `docs/plan/E04-task-mgmt-cli-core/E04-F02-cli-infrastructure/tasks/`
**When** I run `shark sync`
**Then** system must extract epic key "E04-task-mgmt-cli-core" from path
**And** system must extract feature key "E04-F02-cli-infrastructure" from path
**And** system must resolve feature_id from feature key using database query
**And** task must be associated with correct feature in database

**Given** file in invalid path `docs/plan/random-folder/01-research.md`
**When** I run `shark sync`
**Then** system must log error: "Cannot infer epic/feature from path 'docs/plan/random-folder/01-research.md'. Expected structure: docs/plan/{E##-epic-slug}/{E##-F##-feature-slug}/tasks/{file}. Suggestion: Move file to valid epic/feature folder or add task_key to frontmatter."
**And** file must be skipped (not imported)

### AC-7: Pattern Validation on Config Load (US-7)

**Given** configuration with invalid regex pattern:
```json
{
  "patterns": {
    "task": {
      "file": [
        {
          "name": "invalid",
          "regex": "(?P<task_key>T-E\\d{2}-F\\d{2}-\\d{3",
          "enabled": true
        }
      ]
    }
  }
}
```
**When** I run any `shark` command
**Then** system must exit with error code 1
**And** system must log: "Pattern validation failed for 'invalid': Invalid regex syntax - missing closing parenthesis. Regex: '(?P<task_key>T-E\\d{2}-F\\d{2}-\\d{3'. Suggestion: Fix regex syntax or disable pattern."

**Given** configuration with missing capture group:
```json
{
  "patterns": {
    "task": {
      "file": [
        {
          "name": "no-capture",
          "regex": "^T-E\\d{2}-F\\d{2}-\\d{3}\\.md$",
          "enabled": true
        }
      ]
    }
  }
}
```
**When** I run any `shark` command
**Then** system must exit with error code 1
**And** system must log: "Pattern validation failed for 'no-capture': Missing required capture group 'task_key' or 'number'. Suggestion: Add named capture group (?P<task_key>...) or (?P<number>...) to extract task identifier."

### AC-8: Generated Key Frontmatter Update (US-8)

**Given** file `implement-caching.prp.md` with content:
```markdown
---
description: Add caching layer for API responses
---

# Implement Caching

...content...
```
**And** file has no `task_key` in frontmatter
**When** I run `shark sync` (first time)
**Then** system must generate key `T-E04-F02-003`
**And** file must be updated to:
```markdown
---
description: Add caching layer for API responses
task_key: T-E04-F02-003
---

# Implement Caching

...content...
```
**And** file permissions/ownership must be preserved

**When** I run `shark sync` again (second time)
**Then** system must read existing `task_key: T-E04-F02-003` from frontmatter
**And** system must NOT generate new key (use existing)
**And** system must NOT modify file

### AC-9: Preset Task Patterns (US-9)

**Given** I want to add common task patterns without writing regex
**When** I run `shark config add-pattern --preset=standard-with-slug`
**Then** system must append preset pattern to `.sharkconfig.json`:
```json
{
  "name": "standard-with-slug",
  "regex": "^(?P<task_key>T-E\\d{2}-F\\d{2}-\\d{3})-(?P<slug>[a-z0-9-]+)\\.md$",
  "enabled": true,
  "description": "Standard task format with required descriptive slug"
}
```
**And** system must not replace existing patterns (append only)

**When** I run `shark config show-preset standard-with-slug`
**Then** system must display preset details: name, regex, description, example filenames

### AC-10: Pattern Matching Error Details (US-10)

**Given** file `T-E04-F02-001-my-task.md` with pattern configuration that captures incorrect groups
**When** I run `shark sync --verbose`
**Then** verbose log must include:
```
[DEBUG] Attempting to match file: T-E04-F02-001-my-task.md
[DEBUG]   Pattern 1: standard (^(?P<task_key>T-E\\d{2}-F\\d{2}-\\d{3})(?:-(?P<slug>[a-z0-9-]+))?\\.md$)
[DEBUG]     Match: true
[DEBUG]     Captured groups: task_key=T-E04-F02-001, slug=my-task
[DEBUG]   Selected pattern: standard
```

### AC-11: Dry-Run Pattern Matching (US-11)

**Given** directory with 10 task files using mixed conventions
**When** I run `shark sync --dry-run --verbose`
**Then** system must display:
```
Dry-run mode: No database changes will be made

Task Files Scanned: 10
  Matched: 8
    - standard pattern: 5 files
    - numbered-prefix pattern: 2 files
    - prp-suffix pattern: 1 file
  Unmatched: 2

Matched Files:
  T-E04-F02-001.md (pattern: standard, task_key: T-E04-F02-001)
  01-research.md (pattern: numbered-prefix, generated_key: T-E04-F02-003)
  implement-auth.prp.md (pattern: prp-suffix, generated_key: T-E04-F02-004)
  ...

Unmatched Files:
  random-notes.txt (attempted patterns: standard, numbered-prefix, prp-suffix)
  old-backup.md~ (attempted patterns: standard, numbered-prefix, prp-suffix)

Would import 8 tasks. Run without --dry-run to execute.
```
**And** database must remain unchanged
**And** no frontmatter updates must occur

### AC-12: Pattern Testing CLI Command (US-12)

**Given** I want to test a pattern before adding to config
**When** I run `shark config test-pattern --regex="^(?P<number>\d{2})-.*\.md$" --file="01-research.md"`
**Then** system must display:
```
Pattern Test Result:
  Pattern: ^(?P<number>\d{2})-.*\.md$
  Test File: 01-research.md
  Match: true
  Captured Groups:
    - number: 01

Validation: PASSED
  Required capture groups present: number
```

**When** I run `shark config test-pattern --regex="^(?P<number>\d{2})-.*\.md$" --file="T-E04-F02-001.md"`
**Then** system must display:
```
Pattern Test Result:
  Pattern: ^(?P<number>\d{2})-.*\.md$
  Test File: T-E04-F02-001.md
  Match: false

Validation: PASSED
  Required capture groups present: number
```

### AC-13: Pattern Match Statistics (US-13)

**Given** directory with 50 task files using multiple conventions
**When** I run `shark sync`
**Then** sync report must include:
```
Task Import Summary:
  Total Files: 50
  Matched: 47 (94%)
  Unmatched: 3 (6%)

Pattern Statistics:
  standard: 35 files (74%)
  numbered-prefix: 8 files (17%)
  prp-suffix: 4 files (9%)

Generated Task Keys: 12
  Feature E04-F02: 5 keys
  Feature E04-F07: 7 keys
```

---

## Out of Scope

### Explicitly Excluded from This Feature

1. **Task Dependency Parsing**: Parsing `depends_on` frontmatter field and establishing task dependencies is E05-F02 (Dependency Management). This feature only imports task metadata, not relationships beyond epic/feature.

2. **Task Status Transition Validation**: Validating that imported task status values follow allowed state transitions is E04-F03 (Task Lifecycle Operations). This feature imports status as-is from frontmatter.

3. **Epic and Feature Pattern Matching**: Configurable patterns for epic/feature folders and files are separate E06 features (E06-F01, E06-F02). This feature assumes standard epic/feature folder naming when inferring from path.

4. **Historic Task File Versioning**: Tracking changes to task files over time via git history or audit trail is E05-F03 (History & Audit Trail). This feature only imports current file state.

5. **Task File Content Validation**: Deep validation of task file markdown structure (e.g., required sections, formatting rules) is out of scope. This feature only extracts metadata, not validates content completeness.

6. **Multi-Root Documentation Support**: Scanning task files from multiple documentation roots (e.g., `docs/plan` and `docs/archived`) is E06-F01 (Multi-Root Documentation). This feature assumes single docs root.

7. **Git-Based Change Detection**: Using `git diff` to identify modified task files instead of file mtime is REQ-F-021 (Could Have). This feature uses modification timestamps per E06 incremental sync requirements.

8. **Task File Generation**: Creating new task files via CLI (`shark task create`) is E04-F06 (Task Creation & Templating). This feature only imports existing task files.

9. **Bulk Task Updates**: Batch operations to update multiple task statuses or metadata fields are E05-F04+ (Advanced Features). This feature is import-only, not bidirectional sync.

10. **Integration with External Tools**: Importing tasks from Jira, Linear, GitHub Issues, or other project management tools is explicitly excluded from E04 and E06. This feature only processes local markdown files.

### Deferred to Future Features

- **Custom Metadata Fields**: Supporting arbitrary custom fields in task frontmatter beyond standard schema (e.g., `estimated_hours`, `tags`, `reviewers`) may be addressed in future extensibility epic.

- **Task File Templates**: Configurable templates for different task types (research, implementation, review) is related to E04-F06 but with pattern-aware template selection.

- **Pattern Migration Tools**: CLI commands to help migrate files from one pattern to another (e.g., rename all numbered-prefix files to standard format) could be valuable but adds significant scope.

- **Pattern Performance Optimization**: Advanced regex optimization (compilation caching, pattern ordering optimization based on match frequency) can be deferred until performance issues identified.

---

## Summary

This feature transforms shark from a rigid, standardized-only task tracker into a flexible system that adapts to real-world documentation patterns. By implementing configurable pattern matching with intelligent metadata extraction and automatic key generation, we remove the primary adoption barrier for existing projects while maintaining reliability through balanced validation and error isolation.

The feature directly addresses E06 requirements REQ-F-007 (Task File Pattern Matching), REQ-F-008 (Task Metadata Extraction), REQ-F-018 (Fuzzy Task Key Generation), and integrates seamlessly with E04's task repository and sync engine infrastructure. Success is measured by achieving 90% import success rate on existing projects without manual file restructuring.

---

**Last Updated**: 2025-12-17
